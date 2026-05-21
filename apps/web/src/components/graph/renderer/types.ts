import type { MouseEvent, ComponentType } from 'react';
import type { Edge, Node, NodeProps } from 'reactflow';

export type GraphRendererId = 'reactflow' | 'webgl';

export interface GraphRendererProps {
  nodes: Node[];
  edges: Edge[];
  loading?: boolean;
  error?: string | null;
  onNodeClick: (event: MouseEvent, node: Node) => void;
  onNodeContextMenu: (event: MouseEvent, node: Node) => void;
  onPaneClick: () => void;
  nodeTypes: Record<string, ComponentType<NodeProps>>;
}
