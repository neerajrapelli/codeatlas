import { useQueryClient } from '@tanstack/react-query';
import { useEffect, useState } from 'react';

import { api } from '../lib/api';
import { queryKeys } from '../lib/queryKeys';

export interface IngestionStep {
  name: string;
  status: 'pending' | 'running' | 'complete' | 'failed';
  duration_ms: number | null;
}

export interface IngestionProgress {
  phase: number;
  status: 'queued' | 'running' | 'complete' | 'failed';
  current_step: string;
  steps: IngestionStep[];
  progress: {
    total_files: number;
    processed_files: number;
    percent: number;
  };
  eta_seconds: number | null;
}

export function useIngestionProgress(repoId: string | number | null) {
  const [progress, setProgress] = useState<IngestionProgress | null>(null);
  const [fading, setFading] = useState(false);
  const queryClient = useQueryClient();

  useEffect(() => {
    if (repoId == null) {
      setProgress(null);
      setFading(false);
      return;
    }

    const id = typeof repoId === 'number' ? repoId : Number(repoId);
    const es = new EventSource(api.ingestionStreamUrl(id));

    es.onmessage = (e: MessageEvent<string>) => {
      try {
        const next = JSON.parse(e.data) as IngestionProgress;
        setProgress(next);
        if (next.status === 'complete') {
          setFading(true);
          void queryClient.invalidateQueries({ queryKey: queryKeys.repositories });
          if (!Number.isNaN(id)) {
            void queryClient.invalidateQueries({ queryKey: queryKeys.repoSync(id) });
            void queryClient.invalidateQueries({ queryKey: ['graph', 'clusters', id] });
          }
          window.setTimeout(() => {
            setProgress(null);
            setFading(false);
          }, 600);
          es.close();
        }
        if (next.status === 'failed') {
          es.close();
        }
      } catch {
        /* ignore malformed events */
      }
    };

    es.onerror = () => {
      es.close();
    };

    return () => {
      es.close();
    };
  }, [repoId, queryClient]);

  return { progress, fading };
}
