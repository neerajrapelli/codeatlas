import type { DependencyEdge } from '@codeatlas/shared-types';

export type AdjacencyList = ReadonlyMap<string, ReadonlySet<string>>;

/**
 * Builds an undirected adjacency view for reachability queries.
 * For directed impact analysis, keep edge direction in a separate structure later.
 */
export function buildUndirectedAdjacency(edges: readonly DependencyEdge[]): AdjacencyList {
  const map = new Map<string, Set<string>>();

  const add = (a: string, b: string) => {
    let setA = map.get(a);
    if (!setA) {
      setA = new Set();
      map.set(a, setA);
    }

    let setB = map.get(b);
    if (!setB) {
      setB = new Set();
      map.set(b, setB);
    }

    setA.add(b);
    setB.add(a);
  };

  for (const edge of edges) {
    add(edge.fromSymbolId, edge.toSymbolId);
  }

  return map;
}

/** BFS from `startId` over undirected adjacency; returns visited node ids. */
export function reachableFrom(startId: string, adjacency: AdjacencyList): ReadonlySet<string> {
  const visited = new Set<string>();
  const queue: string[] = [];

  if (!adjacency.has(startId)) return visited;

  visited.add(startId);
  queue.push(startId);

  while (queue.length > 0) {
    const current = queue.shift();
    if (current === undefined) break;

    for (const neighbor of adjacency.get(current) ?? []) {
      if (!visited.has(neighbor)) {
        visited.add(neighbor);
        queue.push(neighbor);
      }
    }
  }

  return visited;
}
