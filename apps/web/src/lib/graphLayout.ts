import { MarkerType, type Edge, type Node } from 'reactflow';

import { runElkLayout } from './elkLayout';
import type { ClusterLayer, FileOverlay } from '../types';

/** ELK layered layout is too slow above this; use a grid instead. */
const ELK_NODE_LIMIT = 48;
const ELK_TIMEOUT_MS = 20_000;

function gridLayoutPositions(
  children: Array<{ id: string; width: number; height: number }>,
): Record<string, { x: number; y: number }> {
  const cols = Math.max(1, Math.ceil(Math.sqrt(children.length)));
  const gapX = 28;
  const gapY = 28;
  const positions: Record<string, { x: number; y: number }> = {};
  children.forEach((c, i) => {
    const col = i % cols;
    const row = Math.floor(i / cols);
    positions[c.id] = {
      x: col * (c.width + gapX),
      y: row * (c.height + gapY),
    };
  });
  return positions;
}

async function layoutPositions(
  children: Array<{ id: string; width: number; height: number }>,
  elkEdges: Array<{ id: string; sources: string[]; targets: string[] }>,
): Promise<Record<string, { x: number; y: number }>> {
  if (children.length === 0) return {};
  if (children.length > ELK_NODE_LIMIT) {
    return gridLayoutPositions(children);
  }
  try {
    return await Promise.race([
      runElkLayout(children, elkEdges, {
        'elk.algorithm': 'box',
        'elk.direction': 'RIGHT',
      }),
      new Promise<Record<string, { x: number; y: number }>>((_, reject) => {
        window.setTimeout(() => reject(new Error('ELK timeout')), ELK_TIMEOUT_MS);
      }),
    ]);
  } catch {
    return gridLayoutPositions(children);
  }
}

export async function layoutClusterLayer(
  layer: ClusterLayer,
  overlays: Record<string, FileOverlay>,
  highlighted: Set<string>,
  hoverFileId: string | null,
  selectedId: string | null,
  blast: {
    active: boolean;
    targetPath: string | null;
    depthByPath: Record<string, number>;
  },
  violationSeverity: Record<string, 'error' | 'warning' | 'info'>,
): Promise<{ nodes: Node[]; edges: Edge[] }> {
  const rfNodes: Node[] = [];
  const children: Array<{ id: string; width: number; height: number }> = [];

  for (const c of layer.clusters) {
    children.push({ id: c.id, width: 200, height: 56 });
    rfNodes.push({
      id: c.id,
      type: 'cluster',
      position: { x: 0, y: 0 },
      data: { label: c.label, fileCount: c.fileCount, pathPrefix: c.pathPrefix },
    });
  }

  for (const f of layer.files) {
    const nid = `f:${f.id}`;
    const ov = overlays[f.id];
    const isTarget = blast.active && blast.targetPath === f.path;
    const blastDepth = blast.depthByPath[f.path];
    const inBlast = blast.active && (isTarget || blastDepth != null);
    const dim = blast.active && !inBlast;
    const viol = violationSeverity[f.path];
    children.push({ id: nid, width: 180, height: 52 });
    rfNodes.push({
      id: nid,
      type: 'graphFile',
      position: { x: 0, y: 0 },
      data: {
        path: f.path,
        symbolCount: f.symbolCount,
        isHotspot: ov?.isHotspot,
        hasBusFactorRisk: ov?.hasBusFactorRisk,
        architectureSignals: ov?.architectureSignalCount ?? 0,
        dominantOwnerLogin: ov?.dominantOwnerLogin,
        highlight: highlighted.has(f.id),
        hoverPing: hoverFileId === f.id,
        dim,
        blastDepth: blastDepth ?? (isTarget ? 0 : undefined),
        blastTarget: isTarget,
        violationSeverity: viol,
      },
      selected: selectedId === f.id,
    });
  }

  const elkEdges = (layer.edges ?? []).map((e, i) => ({
    id: `e-${String(i)}`,
    sources: [e.from],
    targets: [e.to],
  }));

  const positions = await layoutPositions(children, elkEdges);
  for (const n of rfNodes) {
    const p = positions[n.id];
    if (p) n.position = p;
  }

  const edges: Edge[] = [];
  const selNid = selectedId ? `f:${selectedId}` : null;
  for (let i = 0; i < (layer.edges ?? []).length; i += 1) {
    const e = layer.edges[i];
    if (!e) continue;
    const outDep = selNid != null && e.from === selNid;
    const inDep = selNid != null && e.to === selNid;
    const hi = outDep || inDep;
    edges.push({
      id: `edge-${String(i)}`,
      source: e.from,
      target: e.to,
      style: {
        stroke: outDep ? 'var(--accent-blue)' : inDep ? 'var(--color-warning)' : '#333',
        strokeWidth: hi ? 1.5 : 1,
      },
      markerEnd: { type: MarkerType.ArrowClosed, color: hi ? 'var(--accent-blue)' : '#555' },
    });
  }

  return { nodes: rfNodes, edges };
}
