import { useQuery } from '@tanstack/react-query';

import { applySemanticZoom } from '../../components/graph/semanticZoom';
import { api } from '../../lib/api';
import { queryKeys } from '../../lib/queryKeys';
import type { ClusterLayer } from '../../types';

function normalizeLayer(raw: ClusterLayer): ClusterLayer {
  return {
    prefix: raw.prefix ?? '',
    clusters: raw.clusters ?? [],
    files: raw.files ?? [],
    edges: raw.edges ?? [],
    socioOverlay: raw.socioOverlay,
  };
}

export function useClusterLayerQuery(
  repositoryId: number | null,
  prefix: string,
  enabled = true,
) {
  return useQuery({
    queryKey:
      repositoryId != null
        ? queryKeys.clusterLayer(repositoryId, prefix)
        : ['graph', 'clusters', 'none'],
    queryFn: async () => {
      if (repositoryId == null) throw new Error('no repository');
      const raw = normalizeLayer(await api.getClusters(repositoryId, prefix));
      return applySemanticZoom(raw, prefix);
    },
    enabled: enabled && repositoryId != null,
  });
}
