import { useEffect, useState } from 'react';

import { getApiBase } from '../apiBase';

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

  useEffect(() => {
    if (repoId == null) {
      setProgress(null);
      setFading(false);
      return;
    }

    const base = getApiBase();
    const es = new EventSource(`${base}/repositories/${String(repoId)}/ingestion/stream`);

    es.onmessage = (e: MessageEvent<string>) => {
      try {
        const next = JSON.parse(e.data) as IngestionProgress;
        setProgress(next);
        if (next.status === 'complete') {
          setFading(true);
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
  }, [repoId]);

  return { progress, fading };
}
