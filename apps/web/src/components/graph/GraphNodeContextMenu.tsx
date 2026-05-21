import type { MouseEvent } from 'react';

interface GraphNodeContextMenuProps {
  x: number;
  y: number;
  path: string;
  onAnalyze: () => void;
  onClose: () => void;
}

export function GraphNodeContextMenu({ x, y, path, onAnalyze, onClose }: GraphNodeContextMenuProps) {
  const stop = (e: MouseEvent) => e.stopPropagation();

  return (
    <>
      <div className="graph-context-menu-backdrop" role="presentation" onClick={onClose} />
      <menu
        className="graph-context-menu"
        style={{ left: x, top: y }}
        onClick={stop}
        onContextMenu={(e) => e.preventDefault()}
      >
        <li className="graph-context-menu__label mono">{path}</li>
        <li>
          <button type="button" onClick={onAnalyze}>
            Analyze blast radius
          </button>
        </li>
      </menu>
    </>
  );
}
