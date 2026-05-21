import type { ComponentType, MouseEvent } from 'react';
import ReactFlow, {
  Background,
  Controls,
  MiniMap,
  type Edge,
  type Node,
  type NodeProps,
} from 'reactflow';

import type { GraphRendererProps } from './types';

export function ReactFlowGraphView({
  nodes,
  edges,
  onNodeClick,
  onNodeContextMenu,
  onPaneClick,
  nodeTypes,
}: GraphRendererProps) {
  return (
    <ReactFlow
      className="graph-canvas__flow"
      nodes={nodes}
      edges={edges}
      nodeTypes={nodeTypes as Record<string, ComponentType<NodeProps>>}
      onNodeClick={onNodeClick}
      onNodeContextMenu={onNodeContextMenu}
      minZoom={0.1}
      maxZoom={2}
      proOptions={{ hideAttribution: true }}
      onPaneClick={onPaneClick}
      nodesDraggable={false}
      nodesConnectable={false}
      elementsSelectable
    >
      <MiniMap
        className="graph-minimap"
        nodeColor={() => 'var(--border-default)'}
        maskColor="var(--minimap-mask)"
        pannable
        zoomable
      />
      <Controls className="graph-controls" showInteractive={false} />
      <Background gap={24} size={1} color="var(--graph-dot)" />
    </ReactFlow>
  );
}
