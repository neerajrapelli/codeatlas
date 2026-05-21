import type { GraphRendererProps } from './types';

/**
 * Placeholder for a WebGL macro-graph renderer (Sigma.js, Cosmograph, etc.).
 * Swap `VITE_GRAPH_RENDERER=webgl` once a canvas implementation is wired.
 */
export function WebGLGraphView({ nodes, edges, error }: GraphRendererProps) {
  return (
    <div className="graph-webgl-placeholder">
      <p className="empty-state">
        WebGL renderer ({nodes.length} nodes, {edges.length} edges) — enable when Cosmograph/Sigma
        integration lands. Set <code>VITE_GRAPH_RENDERER=reactflow</code> to use the DOM view.
      </p>
      {error ? <p className="empty-state">{error}</p> : null}
    </div>
  );
}
