import { useReactFlow } from 'reactflow';

import { useStore } from '../../store';

export function GraphToolbar() {
  const { zoomIn, zoomOut, fitView } = useReactFlow();
  const graphPrefix = useStore((s) => s.graphPrefix);
  const setGraphPrefix = useStore((s) => s.setGraphPrefix);

  const parentPrefix = graphPrefix.includes('/')
    ? graphPrefix.slice(0, graphPrefix.lastIndexOf('/'))
    : '';

  return (
    <div className="graph-toolbar-float" role="toolbar" aria-label="Graph controls">
      {graphPrefix ? (
        <button
          type="button"
          className="graph-breadcrumb"
          title="Up one level"
          onClick={() => setGraphPrefix(parentPrefix)}
        >
          ↑ {graphPrefix}
        </button>
      ) : (
        <button type="button" className="active" title="Top-level modules">
          Modules
        </button>
      )}
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
