import { memo, useCallback, useEffect, useMemo, useState } from 'react';
import type { MouseEvent } from 'react';
import ELK from 'elkjs/lib/elk.bundled.js';

import { getApiBase } from '../apiBase';
import ReactFlow, {
  Background,
  Controls,
  Handle,
  MarkerType,
  MiniMap,
  Position,
  useReactFlow,
  useViewport,
  type Edge,
  type Node,
  type NodeProps,
} from 'reactflow';

const elk = new ELK();

export interface ClusterLayer {
  prefix: string;
  clusters: Array<{
    id: string;
    label: string;
    pathPrefix: string;
    level: number;
    fileCount: number;
    internalEdges: number;
    density: number;
    hasChildren: boolean;
  }>;
  files: Array<{ id: string; path: string; symbolCount: number }>;
  edges: Array<{ from: string; to: string; count: number }>;
}

interface HierarchyGraphProps {
  repositoryId: number;
  highlightedFileIds: Set<string>;
  /** File node open in the inspector (persistent selection ring). */
  selectedFileId?: string | null;
  /** Bump when AI wants to jump to file paths (paths from repo root). */
  focusGeneration: number;
  focusPaths: string[];
  onPrefixChange?: (prefix: string) => void;
  onSelectFile: (fileId: string, path: string) => void;
}

function basename(p: string): string {
  const i = Math.max(p.lastIndexOf('/'), p.lastIndexOf('\\'));
  return i >= 0 ? p.slice(i + 1) : p;
}

export function parentPrefix(path: string): string {
  const norm = path.replace(/\\/g, '/');
  const i = norm.lastIndexOf('/');
  if (i <= 0) return '';
  return norm.slice(0, i);
}

const ClusterNode = memo(function ClusterNodeInner({ data }: NodeProps) {
  return (
    <div className={`cluster-node ${data.active ? 'active' : ''}`}>
      <Handle type="target" position={Position.Left} />
      <div className="cluster-node-glow" />
      <div className="cluster-node-title">{String(data.label)}</div>
      <div className="cluster-node-meta">
        <span>{String(data.fileCount)} files</span>
        <span>{(Number(data.density) * 100).toFixed(1)} dense</span>
      </div>
      <Handle type="source" position={Position.Right} />
    </div>
  );
});

const FileNode = memo(function FileNodeInner({ data }: NodeProps) {
  return (
    <div
      className={`file-node ${data.highlight ? 'highlight' : ''} ${data.selected ? 'file-node--selected' : ''} ${data.dim ? 'file-node--dim' : ''}`}
    >
      <Handle type="target" position={Position.Left} />
      <div className="file-node-title">{String(data.title)}</div>
      <div className="file-node-meta">{String(data.meta)}</div>
      <Handle type="source" position={Position.Right} />
    </div>
  );
});

const nodeTypes = { cluster: ClusterNode, file: FileNode };

function normalizeClusterLayer(raw: ClusterLayer | Record<string, unknown>): ClusterLayer {
  const o = raw as Record<string, unknown>;
  return {
    prefix: typeof o.prefix === 'string' ? o.prefix : '',
    clusters: Array.isArray(o.clusters) ? (o.clusters as ClusterLayer['clusters']) : [],
    files: Array.isArray(o.files) ? (o.files as ClusterLayer['files']) : [],
    edges: Array.isArray(o.edges) ? (o.edges as ClusterLayer['edges']) : [],
  };
}

async function layoutWithElk(
  layer: ClusterLayer,
  highlightedFileIds: Set<string>,
  selectedFileId: string | null,
): Promise<{ nodes: Node[]; edges: Edge[] }> {
  const clusters = layer.clusters ?? [];
  const files = layer.files ?? [];
  const edgesList = layer.edges ?? [];

  const children: Array<{ id: string; width: number; height: number }> = [];
  const rfNodes: Node[] = [];

  for (const c of clusters) {
    children.push({ id: c.id, width: 220, height: 76 });
    rfNodes.push({
      id: c.id,
      type: 'cluster',
      position: { x: 0, y: 0 },
      data: {
        label: c.label,
        fileCount: c.fileCount,
        density: c.density,
        pathPrefix: c.pathPrefix,
        active: false,
      },
    });
  }
  for (const f of files) {
    const nid = `f:${f.id}`;
    children.push({ id: nid, width: 200, height: 64 });
    rfNodes.push({
      id: nid,
      type: 'file',
      position: { x: 0, y: 0 },
      data: {
        title: basename(f.path),
        meta: `${String(f.symbolCount)} symbols`,
        path: f.path,
        highlight: highlightedFileIds.has(f.id),
        selected: selectedFileId !== null && selectedFileId === f.id,
      },
    });
  }

  const elkEdges = edgesList.map((e, i) => ({
    id: `e-${String(i)}`,
    sources: [e.from],
    targets: [e.to],
  }));

  const graph = {
    id: 'root',
    layoutOptions: {
      'elk.algorithm': 'layered',
      'elk.direction': 'RIGHT',
      'elk.spacing.nodeNode': '80',
      'elk.layered.spacing.nodeNodeBetweenLayers': '96',
    },
    children,
    edges: elkEdges,
  };

  const laidOut = await elk.layout(graph);

  const posById = new Map<string, { x: number; y: number }>();
  for (const ch of laidOut.children ?? []) {
    if (ch.x !== undefined && ch.y !== undefined) {
      posById.set(ch.id, { x: ch.x, y: ch.y });
    }
  }

  for (const n of rfNodes) {
    const p = posById.get(n.id);
    if (p) {
      n.position = p;
    }
  }

  const rfEdges: Edge[] = [];
  for (let i = 0; i < edgesList.length; i += 1) {
    const e = edgesList[i];
    if (!e) continue;
    const fromHi = e.from.startsWith('f:') && highlightedFileIds.has(e.from.slice(2));
    const toHi = e.to.startsWith('f:') && highlightedFileIds.has(e.to.slice(2));
    const hi = fromHi || toHi;
    rfEdges.push({
      id: `edge-${String(i)}-${e.from}-${e.to}`,
      source: e.from,
      target: e.to,
      animated: hi,
      label: String(e.count),
      labelStyle: { fill: '#64748b', fontSize: 11 },
      style: hi
        ? { stroke: '#2563eb', strokeWidth: 2, opacity: 0.95 }
        : { stroke: '#cbd5e1', strokeWidth: 1, opacity: 0.9 },
      markerEnd: { type: MarkerType.ArrowClosed, color: hi ? '#2563eb' : '#94a3b8' },
    });
  }

  return { nodes: rfNodes, edges: rfEdges };
}

/** Must render under a single ReactFlowProvider (parent App). */
export function HierarchyGraph({
  repositoryId,
  highlightedFileIds,
  selectedFileId = null,
  focusGeneration,
  focusPaths,
  onPrefixChange,
  onSelectFile,
}: HierarchyGraphProps) {
  const reactFlow = useReactFlow();
  const { zoom } = useViewport();
  const [prefix, setPrefix] = useState('');
  const [layer, setLayer] = useState<ClusterLayer | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [nodes, setNodes] = useState<Node[]>([]);
  const [edges, setEdges] = useState<Edge[]>([]);
  const [hoveredEdgeId, setHoveredEdgeId] = useState<string | null>(null);
  const [hoveredNodeId, setHoveredNodeId] = useState<string | null>(null);
  const [archSearch, setArchSearch] = useState('');

  const apiBase = useMemo(() => getApiBase(), []);

  useEffect(() => {
    if (focusGeneration <= 0 || focusPaths.length === 0) return;
    const path = focusPaths[0];
    if (!path) return;
    const pp = parentPrefix(path);
    setPrefix(pp);
    onPrefixChange?.(pp);
  }, [focusGeneration, focusPaths, onPrefixChange]);

  useEffect(() => {
    const ac = new AbortController();
    void (async () => {
      try {
        const url = `${apiBase}/graph/clusters?repositoryId=${String(repositoryId)}&prefix=${encodeURIComponent(prefix)}`;
        const res = await fetch(url, { signal: ac.signal });
        if (!res.ok) throw new Error(`clusters ${String(res.status)}`);
        const data = normalizeClusterLayer((await res.json()) as ClusterLayer);
        setLayer(data);
        setLoadError(null);
        const laid = await layoutWithElk(data, highlightedFileIds, selectedFileId);
        setNodes(laid.nodes);
        setEdges(laid.edges);
        requestAnimationFrame(() => {
          reactFlow.fitView({ padding: 0.15, duration: 450 });
        });
      } catch (e) {
        if ((e as Error).name === 'AbortError') return;
        setLoadError(e instanceof Error ? e.message : 'Failed to load clusters');
        setLayer(null);
        setNodes([]);
        setEdges([]);
      }
    })();
    return () => ac.abort();
  }, [apiBase, repositoryId, prefix, reactFlow]);

  useEffect(() => {
    if (!layer) return;
    void layoutWithElk(layer, highlightedFileIds, selectedFileId).then((laid) => {
      setNodes(laid.nodes);
      setEdges(laid.edges);
    });
  }, [highlightedFileIds, layer, selectedFileId]);

  const vizEdges = useMemo(() => {
    return edges.map((e) => {
      const fromHi = e.source.startsWith('f:') && highlightedFileIds.has(e.source.slice(2));
      const toHi = e.target.startsWith('f:') && highlightedFileIds.has(e.target.slice(2));
      const aiHi = fromHi || toHi;
      const edgeHover = hoveredEdgeId === e.id;
      const touchesHover =
        hoveredNodeId !== null && (e.source === hoveredNodeId || e.target === hoveredNodeId);
      const glow = aiHi || edgeHover || touchesHover;
      return {
        ...e,
        animated: glow,
        style: glow
          ? { stroke: '#2563eb', strokeWidth: aiHi ? 2.25 : 2.8, opacity: 0.96 }
          : { stroke: '#cbd5e1', strokeWidth: 1.15, opacity: 0.88 },
        markerEnd: {
          type: MarkerType.ArrowClosed,
          color: glow ? '#2563eb' : '#94a3b8',
        },
      };
    });
  }, [edges, highlightedFileIds, hoveredEdgeId, hoveredNodeId]);

  const vizNodes = useMemo(() => {
    if (!hoveredNodeId) return nodes;
    return nodes.map((n) => {
      if (n.type === 'cluster') {
        return { ...n, data: { ...n.data, dim: false } };
      }
      const touched =
        n.id === hoveredNodeId ||
        edges.some(
          (ed) =>
            (ed.source === hoveredNodeId || ed.target === hoveredNodeId) &&
            (ed.source === n.id || ed.target === n.id),
        );
      return {
        ...n,
        data: {
          ...n.data,
          dim: !touched,
        },
      };
    });
  }, [nodes, edges, hoveredNodeId]);

  const onNodeClick = useCallback(
    (_: MouseEvent, node: Node) => {
      if (node.type === 'cluster') {
        const pathPrefix = String((node.data as { pathPrefix?: string }).pathPrefix ?? '');
        setPrefix(pathPrefix);
        onPrefixChange?.(pathPrefix);
        return;
      }
      if (node.type === 'file') {
        const id = node.id.startsWith('f:') ? node.id.slice(2) : node.id;
        const p = String((node.data as { path?: string }).path ?? '');
        onSelectFile(id, p);
      }
    },
    [onPrefixChange, onSelectFile],
  );

  const onNodeMouseEnter = useCallback(
    (_: MouseEvent, node: Node) => {
      if (node.type === 'file') {
        setHoveredNodeId(node.id);
      }
    },
    [],
  );

  const onNodeMouseLeave = useCallback(() => {
    setHoveredNodeId(null);
  }, []);

  const onEdgeMouseEnter = useCallback((_: MouseEvent, edge: Edge) => {
    setHoveredEdgeId(edge.id);
  }, []);

  const onEdgeMouseLeave = useCallback(() => {
    setHoveredEdgeId(null);
  }, []);

  const jumpToArchitectureMatch = useCallback(() => {
    if (!layer) return;
    const q = archSearch.trim().toLowerCase();
    if (!q) return;
    const hit = layer.files.find((f) => f.path.toLowerCase().includes(q));
    if (!hit) return;
    const pp = parentPrefix(hit.path);
    setPrefix(pp);
    onPrefixChange?.(pp);
    onSelectFile(hit.id, hit.path);
    requestAnimationFrame(() => {
      reactFlow.fitView({ padding: 0.35, duration: 420, maxZoom: 1.8 });
    });
  }, [layer, archSearch, onPrefixChange, onSelectFile, reactFlow]);

  const breadcrumb = useMemo(() => {
    if (!prefix) return [{ label: 'Repository root', prefix: '' }];
    const parts = prefix.split('/');
    const crumbs: Array<{ label: string; prefix: string }> = [{ label: 'Repository root', prefix: '' }];
    let acc = '';
    for (const part of parts) {
      acc = acc ? `${acc}/${part}` : part;
      crumbs.push({ label: part, prefix: acc });
    }
    return crumbs;
  }, [prefix]);

  const detailHint =
    zoom < 0.45 ? 'Zoom in for module detail' : zoom > 1.2 ? 'Fine-grained view' : 'Architecture map';

  return (
    <div className="hierarchy-graph-wrap">
      <div className="hierarchy-toolbar">
        <div className="breadcrumb">
          {breadcrumb.map((c, i) => (
            <button
              key={`${c.prefix}-${String(i)}`}
              type="button"
              className={`crumb ${c.prefix === prefix ? 'active' : ''}`}
              onClick={() => {
                setPrefix(c.prefix);
                onPrefixChange?.(c.prefix);
              }}
            >
              {c.label}
              {i < breadcrumb.length - 1 ? <span className="crumb-sep">/</span> : null}
            </button>
          ))}
        </div>
        <div className="hierarchy-toolbar__search" role="search">
          <input
            type="search"
            value={archSearch}
            onChange={(e) => setArchSearch(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                e.preventDefault();
                jumpToArchitectureMatch();
              }
            }}
            placeholder="Find file in map…"
            aria-label="Find file path in architecture map"
          />
          <button type="button" className="hierarchy-find-btn" onClick={jumpToArchitectureMatch}>
            Go
          </button>
        </div>
        <span className="zoom-hint">{detailHint}</span>
      </div>
      {loadError ? <p className="warning hierarchy-warning">{loadError}</p> : null}
      {!layer ? (
        <div className="graph-loading">
          <p>Loading architecture map…</p>
          <div className="skeleton" style={{ height: 120, width: 'min(420px, 100%)' }} />
        </div>
      ) : (
        <ReactFlow
          nodes={vizNodes}
          edges={vizEdges}
          nodeTypes={nodeTypes}
          fitView
          minZoom={0.15}
          maxZoom={2.4}
          onlyRenderVisibleElements
          onNodeClick={onNodeClick}
          onNodeMouseEnter={onNodeMouseEnter}
          onNodeMouseLeave={onNodeMouseLeave}
          onEdgeMouseEnter={onEdgeMouseEnter}
          onEdgeMouseLeave={onEdgeMouseLeave}
          proOptions={{ hideAttribution: true }}
          defaultEdgeOptions={{
            type: 'smoothstep',
            style: { stroke: '#cbd5e1', strokeWidth: 1.25 },
            interactionWidth: 20,
          }}
        >
          <MiniMap
            pannable
            zoomable
            nodeStrokeWidth={1}
            nodeColor="#e2e8f0"
            maskColor="rgba(243, 244, 246, 0.85)"
          />
          <Controls showInteractive />
          <Background gap={28} size={1} color="#d1d5db" style={{ opacity: 0.45 }} />
        </ReactFlow>
      )}
    </div>
  );
}
