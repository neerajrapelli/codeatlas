import ELK from 'elkjs/lib/elk.bundled.js';
import { useCallback, useEffect, useMemo, useState } from 'react';
import type { MouseEvent } from 'react';
import ReactFlow, {
  Background,
  Controls,
  MarkerType,
  MiniMap,
  useReactFlow,
  type Edge,
  type Node,
  type NodeProps,
} from 'reactflow';

import { useIngestionProgress } from '../../hooks/useIngestionProgress';
import { api } from '../../lib/api';
import { useStore } from '../../store';
import type { ClusterLayer, FileOverlay } from '../../types';
import { IngestionBar } from '../shell/IngestionBar';
import { GraphFileNode } from './GraphFileNode';
import { GraphNodeContextMenu } from './GraphNodeContextMenu';
import { GraphSkeleton } from './GraphSkeleton';
import { GraphToolbar } from './GraphToolbar';
import { GraphWelcome } from './GraphWelcome';

const elk = new ELK();

const ClusterNode = ({ data }: NodeProps) => (
  <div
    style={{
      width: 200,
      height: 56,
      background: 'var(--bg-3)',
      border: '1px solid var(--border-subtle)',
      borderRadius: 'var(--radius-md)',
      padding: 8,
      fontSize: 'var(--font-size-sm)',
    }}
  >
    <div style={{ fontWeight: 600 }}>{String(data.label)}</div>
    <div style={{ color: 'var(--text-secondary)', fontSize: 'var(--font-size-xs)' }}>
      {String(data.fileCount)} files
    </div>
  </div>
);

const nodeTypes = { cluster: ClusterNode, graphFile: GraphFileNode };

function normalizeLayer(raw: ClusterLayer): ClusterLayer {
  return {
    prefix: raw.prefix ?? '',
    clusters: raw.clusters ?? [],
    files: raw.files ?? [],
    edges: raw.edges ?? [],
    socioOverlay: raw.socioOverlay,
  };
}

async function layoutLayer(
  layer: ClusterLayer,
  overlays: Record<string, FileOverlay>,
  highlighted: Set<string>,
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
    const inBlast =
      blast.active &&
      (isTarget || blastDepth != null);
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

  const graph = {
    id: 'root',
    layoutOptions: {
      'elk.algorithm': 'layered',
      'elk.direction': 'RIGHT',
      'elk.spacing.nodeNode': '72',
    },
    children,
    edges: elkEdges,
  };

  const laid = await elk.layout(graph);
  for (const ch of laid.children ?? []) {
    if (ch.x != null && ch.y != null) {
      const n = rfNodes.find((x) => x.id === ch.id);
      if (n) n.position = { x: ch.x, y: ch.y };
    }
  }

  const edges: Edge[] = [];
  const selNid = selectedId ? `f:${selectedId}` : null;
  for (let i = 0; i < (layer.edges ?? []).length; i += 1) {
    const e = layer.edges[i];
    if (!e) continue;
    const outDep = selNid && e.from === selNid;
    const inDep = selNid && e.to === selNid;
    const hi = outDep || inDep;
    edges.push({
      id: `edge-${String(i)}`,
      source: e.from,
      target: e.to,
      style: {
        stroke: outDep ? 'var(--accent-blue)' : inDep ? 'var(--color-warning)' : '#333',
        strokeWidth: hi ? 1.5 : 1,
        strokeDasharray: undefined,
      },
      markerEnd: { type: MarkerType.ArrowClosed, color: hi ? 'var(--accent-blue)' : '#555' },
    });
  }

  return { nodes: rfNodes, edges };
}

export function GraphCanvas() {
  const reactFlow = useReactFlow();
  const activeRepoId = useStore((s) => s.activeRepoId);
  const graphPrefix = useStore((s) => s.graphPrefix);
  const setGraphPrefix = useStore((s) => s.setGraphPrefix);
  const setClusterLayer = useStore((s) => s.setClusterLayer);
  const selectedNodeId = useStore((s) => s.selectedNodeId);
  const setSelectedNode = useStore((s) => s.setSelectedNode);
  const highlightedFileIds = useStore((s) => s.highlightedFileIds);
  const blastRadius = useStore((s) => s.blastRadius);
  const blastDepthByPath = useStore((s) => s.blastDepthByPath);
  const blastTargetPath = useStore((s) => s.blastTargetPath);
  const setBlastRadius = useStore((s) => s.setBlastRadius);
  const repositories = useStore((s) => s.repositories);
  const graphLoading = useStore((s) => s.graphLoading);
  const setGraphLoading = useStore((s) => s.setGraphLoading);

  const [nodes, setNodes] = useState<Node[]>([]);
  const [edges, setEdges] = useState<Edge[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [contextMenu, setContextMenu] = useState<{
    x: number;
    y: number;
    path: string;
  } | null>(null);
  const [blastLoading, setBlastLoading] = useState(false);

  const ruleViolations = useStore((s) => s.ruleViolations);
  const violationSeverity = useMemo(() => {
    const m: Record<string, 'error' | 'warning' | 'info'> = {};
    for (const v of ruleViolations) {
      const cur = m[v.sourceFile];
      if (!cur || (v.severity === 'error') || (v.severity === 'warning' && cur === 'info')) {
        m[v.sourceFile] = v.severity;
      }
      const curT = m[v.targetFile];
      if (!curT || v.severity === 'error' || (v.severity === 'warning' && curT === 'info')) {
        m[v.targetFile] = v.severity;
      }
    }
    return m;
  }, [ruleViolations]);

  const blastOverlay = {
    active: blastRadius != null,
    targetPath: blastTargetPath,
    depthByPath: blastDepthByPath,
  };

  const repo = repositories.find((r) => r.id === activeRepoId);
  const { progress, fading } = useIngestionProgress(activeRepoId);
  const activelyIndexing =
    progress?.status === 'running' ||
    progress?.status === 'queued' ||
    (repo != null && repo.status !== 'ready' && repo.status !== 'failed');
  const showSkeleton =
    graphLoading || !repo || (activelyIndexing && (repo?.filesIndexed ?? 0) === 0);

  useEffect(() => {
    if (activeRepoId == null) {
      setNodes([]);
      setEdges([]);
      return;
    }
    const ac = new AbortController();
    setGraphLoading(true);
    void (async () => {
      try {
        const layer = normalizeLayer(await api.getClusters(activeRepoId, graphPrefix));
        if (ac.signal.aborted) return;
        setClusterLayer(layer);
        const overlays = layer.socioOverlay?.fileOverlays ?? {};
        const laid = await layoutLayer(
          layer,
          overlays,
          highlightedFileIds,
          selectedNodeId,
          blastOverlay,
          violationSeverity,
        );
        if (ac.signal.aborted) return;
        setNodes(laid.nodes);
        setEdges(laid.edges);
        setError(null);
        requestAnimationFrame(() => reactFlow.fitView({ padding: 0.12, duration: 300 }));
      } catch (e) {
        if ((e as Error).name === 'AbortError') return;
        setError(e instanceof Error ? e.message : 'Failed to load graph');
      } finally {
        if (!ac.signal.aborted) setGraphLoading(false);
      }
    })();
    return () => ac.abort();
  }, [activeRepoId, graphPrefix, setClusterLayer, setGraphLoading, reactFlow]);

  useEffect(() => {
    const layer = useStore.getState().clusterLayer;
    if (!layer || activeRepoId == null) return;
    const overlays = layer.socioOverlay?.fileOverlays ?? {};
    void layoutLayer(
      layer,
      overlays,
      highlightedFileIds,
      selectedNodeId,
      blastOverlay,
      violationSeverity,
    ).then((laid) => {
      setNodes(laid.nodes);
      setEdges(laid.edges);
    });
  }, [
    highlightedFileIds,
    selectedNodeId,
    activeRepoId,
    blastRadius,
    blastDepthByPath,
    blastTargetPath,
    violationSeverity,
  ]);

  useEffect(() => {
    if (activeRepoId == null) return;
    void api.getViolations(activeRepoId).then(useStore.getState().setRuleViolations);
  }, [activeRepoId]);

  const runBlastRadius = useCallback(
    async (filePath: string) => {
      if (activeRepoId == null) return;
      setBlastLoading(true);
      setContextMenu(null);
      try {
        const result = await api.getBlastRadius(activeRepoId, filePath, { depth: 3 });
        setBlastRadius(result);
      } catch (e) {
        setError(e instanceof Error ? e.message : 'Blast radius failed');
      } finally {
        setBlastLoading(false);
      }
    },
    [activeRepoId, setBlastRadius],
  );

  const onNodeContextMenu = useCallback((e: MouseEvent, node: Node) => {
    if (node.type !== 'graphFile') return;
    e.preventDefault();
    const path = String((node.data as { path?: string }).path ?? '');
    if (!path) return;
    setContextMenu({ x: e.clientX, y: e.clientY, path });
  }, []);

  const onNodeClick = useCallback(
    (_: MouseEvent, node: Node) => {
      if (node.type === 'cluster') {
        const pathPrefix = String((node.data as { pathPrefix?: string }).pathPrefix ?? '');
        setGraphPrefix(pathPrefix);
        return;
      }
      if (node.type === 'graphFile') {
        const id = node.id.startsWith('f:') ? node.id.slice(2) : node.id;
        const path = String((node.data as { path?: string }).path ?? '');
        setSelectedNode(id, path);
      }
    },
    [setGraphPrefix, setSelectedNode],
  );

  if (activeRepoId == null) {
    return (
      <div className="graph-area">
        <GraphWelcome />
      </div>
    );
  }

  return (
    <div className="graph-area">
      <IngestionBar progress={progress} fading={fading} repoName={repo?.name} />
      {showSkeleton ? (
        <GraphSkeleton />
      ) : (
        <>
          <GraphToolbar />
          <div className="graph-canvas" style={{ flex: 1, minHeight: 0 }}>
            {error ? <p className="empty-state" style={{ padding: 12 }}>{error}</p> : null}
            <ReactFlow
              nodes={nodes}
              edges={edges}
              nodeTypes={nodeTypes}
              onNodeClick={onNodeClick}
              onNodeContextMenu={onNodeContextMenu}
              fitView
              minZoom={0.1}
              maxZoom={2}
              proOptions={{ hideAttribution: true }}
              onPaneClick={() => {
                setSelectedNode(null, null);
                setContextMenu(null);
              }}
            >
              <MiniMap
                nodeColor={() => 'var(--border-default)'}
                maskColor="var(--minimap-mask)"
                style={{ background: 'var(--bg-1)' }}
              />
              <Controls />
              <Background gap={24} size={1} color="var(--graph-dot)" />
            </ReactFlow>
          </div>
          {contextMenu ? (
            <GraphNodeContextMenu
              x={contextMenu.x}
              y={contextMenu.y}
              path={contextMenu.path}
              onAnalyze={() => void runBlastRadius(contextMenu.path)}
              onClose={() => setContextMenu(null)}
            />
          ) : null}
          {blastLoading ? (
            <div className="graph-blast-loading">Computing blast radius…</div>
          ) : null}
        </>
      )}
    </div>
  );
}
