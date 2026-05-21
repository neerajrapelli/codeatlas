import type { GraphRendererId } from './types';
import { ReactFlowGraphView } from './ReactFlowGraphView';
import { WebGLGraphView } from './WebGLGraphView';

export type { GraphRendererProps, GraphRendererId } from './types';
export { ReactFlowGraphView, WebGLGraphView };

const envRenderer = (import.meta.env.VITE_GRAPH_RENDERER as GraphRendererId | undefined) ?? 'reactflow';

export function resolveGraphRenderer(): GraphRendererId {
  return envRenderer === 'webgl' ? 'webgl' : 'reactflow';
}

export function GraphRendererView(
  props: import('./types').GraphRendererProps & { renderer?: GraphRendererId },
) {
  const id = props.renderer ?? resolveGraphRenderer();
  if (id === 'webgl') {
    return WebGLGraphView(props);
  }
  return ReactFlowGraphView(props);
}
