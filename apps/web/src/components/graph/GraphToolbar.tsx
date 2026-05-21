import { useReactFlow } from 'reactflow';

export function GraphToolbar() {
  const { zoomIn, zoomOut, fitView } = useReactFlow();

  return (
    <div className="graph-toolbar-float">
      <button type="button" className="active" title="Cluster view">
        Cluster
      </button>
      <button type="button" onClick={() => zoomOut()} title="Zoom out">
        −
      </button>
      <button type="button" onClick={() => fitView({ padding: 0.15 })} title="Reset zoom">
        100%
      </button>
      <button type="button" onClick={() => zoomIn()} title="Zoom in">
        +
      </button>
    </div>
  );
}
