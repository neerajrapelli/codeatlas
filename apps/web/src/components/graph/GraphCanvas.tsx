import 'reactflow/dist/style.css';

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type { MouseEvent } from 'react';
import { useReactFlow, type Edge, type Node, type NodeProps } from 'reactflow';

import { useClusterLayerQuery } from '../../hooks/queries/useClusterLayer';
import { useIngestionProgress } from '../../hooks/useIngestionProgress';
import { layoutClusterLayer } from '../../lib/graphLayout';
import { api } from '../../lib/api';
import { useStore } from '../../store';
import { IngestionBar } from '../shell/IngestionBar';
import { GraphErrorBoundary } from './GraphErrorBoundary';
import { GraphFileNode } from './GraphFileNode';
import { GraphNodeContextMenu } from './GraphNodeContextMenu';
import { GraphLayoutOverlay, GraphSkeleton } from './GraphSkeleton';
import { GraphToolbar } from './GraphToolbar';
import { GraphWelcome } from './GraphWelcome';
import { GraphRendererView } from './renderer';

const ClusterNode = ({ data }: NodeProps) => (
  <div className="graph-cluster-node">
    <div className="graph-cluster-node__label">{String(data.label)}</div>
    <div className="graph-cluster-node__meta">{String(data.fileCount)} files</div>
  </div>
);

const nodeTypes = { cluster: ClusterNode, graphFile: GraphFileNode };

const LAYOUT_TIMEOUT_MS = 35_000;

export function GraphCanvas() {
  const { fitView } = useReactFlow();
  const fitViewRef = useRef(fitView);
  fitViewRef.current = fitView;
  const layoutGenRef = useRef(0);

  const activeRepoId = useStore((s) => s.activeRepoId);
  const graphPrefix = useStore((s) => s.graphPrefix);
  const setGraphPrefix = useStore((s) => s.setGraphPrefix);
  const setClusterLayer = useStore((s) => s.setClusterLayer);
  const selectedNodeId = useStore((s) => s.selectedNodeId);
  const setSelectedNode = useStore((s) => s.setSelectedNode);
  const highlightedFileIds = useStore((s) => s.highlightedFileIds);
  const graphHoverFileId = useStore((s) => s.graphHoverFileId);
  const blastRadius = useStore((s) => s.blastRadius);
  const blastDepthByPath = useStore((s) => s.blastDepthByPath);
  const blastTargetPath = useStore((s) => s.blastTargetPath);
  const setBlastRadius = useStore((s) => s.setBlastRadius);
  const repositories = useStore((s) => s.repositories);
  const setGraphLoading = useStore((s) => s.setGraphLoading);

  const [nodes, setNodes] = useState<Node[]>([]);
  const [edges, setEdges] = useState<Edge[]>([]);
  const [layoutPending, setLayoutPending] = useState(false);
  const [layoutError, setLayoutError] = useState<string | null>(null);
  const [graphEpoch, setGraphEpoch] = useState(0);
  const [contextMenu, setContextMenu] = useState<{
    x: number;
    y: number;
    path: string;
  } | null>(null);
  const [blastLoading, setBlastLoading] = useState(false);

  const {
    data: layer,
    error: fetchError,
    isLoading: layerLoading,
    isFetching,
    refetch,
  } = useClusterLayerQuery(activeRepoId, graphPrefix, activeRepoId != null);

  const ruleViolations = useStore((s) => s.ruleViolations);
  const violationSeverity = useMemo(() => {
    const m: Record<string, 'error' | 'warning' | 'info'> = {};
    for (const v of ruleViolations) {
      const cur = m[v.sourceFile];
      if (!cur || v.severity === 'error' || (v.severity === 'warning' && cur === 'info')) {
        m[v.sourceFile] = v.severity;
      }
      const curT = m[v.targetFile];
      if (!curT || v.severity === 'error' || (v.severity === 'warning' && curT === 'info')) {
        m[v.targetFile] = v.severity;
      }
    }
    return m;
  }, [ruleViolations]);

  const blastOverlay = useMemo(
    () => ({
      active: blastRadius != null,
      targetPath: blastTargetPath,
      depthByPath: blastDepthByPath,
    }),
    [blastRadius, blastTargetPath, blastDepthByPath],
  );

  const repo = repositories.find((r) => r.id === activeRepoId);
  const { progress, fading } = useIngestionProgress(activeRepoId);
  const activelyIndexing =
    progress?.status === 'running' ||
    progress?.status === 'queued' ||
    (repo != null && repo.status !== 'ready' && repo.status !== 'failed');

  const layerHasContent = (layer?.files?.length ?? 0) + (layer?.clusters?.length ?? 0) > 0;
  const showFullSkeleton =
    layerLoading ||
    (isFetching && !layer) ||
    (activelyIndexing && !layerHasContent && (repo?.filesIndexed ?? 0) === 0);

  useEffect(() => {
    setGraphLoading(isFetching && activeRepoId != null);
  }, [isFetching, activeRepoId, setGraphLoading]);

  useEffect(() => {
    if (layer) setClusterLayer(layer);
  }, [layer, setClusterLayer]);

  useEffect(() => {
    if (!layer || activeRepoId == null) {
      setNodes([]);
      setEdges([]);
      setLayoutPending(false);
      setLayoutError(null);
      return;
    }

    const fileCount = layer.files?.length ?? 0;
    const clusterCount = layer.clusters?.length ?? 0;
    if (fileCount === 0 && clusterCount === 0) {
      setNodes([]);
      setEdges([]);
      setLayoutPending(false);
      setLayoutError(null);
      return;
    }

    const gen = ++layoutGenRef.current;
    setLayoutPending(true);
    setLayoutError(null);
    const overlays = layer.socioOverlay?.fileOverlays ?? {};

    const timeoutId = window.setTimeout(() => {
      if (layoutGenRef.current !== gen) return;
      setLayoutPending(false);
      setLayoutError('Layout computation timed out. Click Retry or drill into a smaller folder.');
    }, LAYOUT_TIMEOUT_MS);

    void layoutClusterLayer(
      layer,
      overlays,
      highlightedFileIds,
      graphHoverFileId,
      selectedNodeId,
      blastOverlay,
      violationSeverity,
    )
      .then((laid) => {
        if (layoutGenRef.current !== gen) return;
        window.clearTimeout(timeoutId);
        setNodes(laid.nodes);
        setEdges(laid.edges);
        setLayoutError(null);
        setLayoutPending(false);
        requestAnimationFrame(() => {
          try {
            fitViewRef.current({ padding: 0.12, duration: 300 });
          } catch {
            /* fitView can fail if canvas unmounted */
          }
        });
      })
      .catch((e) => {
        if (layoutGenRef.current !== gen) return;
        window.clearTimeout(timeoutId);
        setLayoutError(e instanceof Error ? e.message : 'Layout failed');
        setNodes([]);
        setEdges([]);
        setLayoutPending(false);
      });

    return () => {
      layoutGenRef.current += 1;
      window.clearTimeout(timeoutId);
    };
  }, [
    layer,
    highlightedFileIds,
    graphHoverFileId,
    selectedNodeId,
    blastOverlay,
    violationSeverity,
    activeRepoId,
    graphEpoch,
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
        setLayoutError(e instanceof Error ? e.message : 'Blast radius failed');
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

  const displayError =
    fetchError instanceof Error ? fetchError.message : fetchError ? String(fetchError) : layoutError;

  const layerSettled = !layerLoading && !isFetching;
  const showNoGraphData =
    !showFullSkeleton &&
    !layoutPending &&
    !displayError &&
    layerSettled &&
    activeRepoId != null &&
    (layer == null || !layerHasContent || nodes.length === 0);

  if (activeRepoId == null) {
    return (
      <div className="graph-area">
        <GraphWelcome />
      </div>
    );
  }

  return (
    <div className="graph-area">
      <IngestionBar
        progress={progress}
        fading={fading}
        repoName={repo?.name}
        repoReady={repo?.status === 'ready'}
      />
      {showFullSkeleton ? (
        <GraphSkeleton
          message={
            activelyIndexing && !layerHasContent
              ? 'Indexing repository…'
              : 'Loading architecture map…'
          }
        />
      ) : (
        <GraphErrorBoundary
          onReset={() => {
            setGraphEpoch((n) => n + 1);
            void refetch();
          }}
        >
          <GraphToolbar />
          <div className="graph-canvas">
            {displayError ? (
              <div className="graph-canvas__error">
                <p>{displayError}</p>
                <button type="button" className="btn-primary" onClick={() => void refetch()}>
                  Retry load
                </button>
              </div>
            ) : showNoGraphData ? (
              <div className="graph-canvas__empty">
                <p>No graph data available for this repository.</p>
                <p className="muted">
                  {activelyIndexing
                    ? 'Indexing is still in progress — check back shortly.'
                    : 'Try another folder prefix or re-index if ingestion failed.'}
                </p>
              </div>
            ) : (
              <GraphRendererView
                nodes={nodes}
                edges={edges}
                nodeTypes={nodeTypes}
                onNodeClick={onNodeClick}
                onNodeContextMenu={onNodeContextMenu}
                onPaneClick={() => {
                  setSelectedNode(null, null);
                  setContextMenu(null);
                }}
              />
            )}
            {layoutPending ? <GraphLayoutOverlay /> : null}
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
        </GraphErrorBoundary>
      )}
    </div>
  );
}
