import type { ClusterLayer } from '../../types';

/** At repo root, show only module/folder clusters — files load after drill-down. */
export function applySemanticZoom(layer: ClusterLayer, prefix: string): ClusterLayer {
  const atRoot = prefix === '';
  if (!atRoot) {
    return layer;
  }

  const clusterIds = new Set(layer.clusters.map((c) => c.id));
  const edges = (layer.edges ?? []).filter((e) => clusterIds.has(e.from) && clusterIds.has(e.to));

  return {
    ...layer,
    files: [],
    edges,
  };
}
