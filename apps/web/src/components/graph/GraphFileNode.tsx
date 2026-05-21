import { memo } from 'react';
import { Handle, Position, type NodeProps } from 'reactflow';

import { basename, detectFileType, fileTypeLabel } from '../../lib/fileType';
import { FileTypeIcon } from '../ui/FileTypeIcon';

export const GraphFileNode = memo(function GraphFileNode({ data, selected }: NodeProps) {
  const path = String(data.path ?? '');
  const ft = detectFileType(path);
  const isHotspot = Boolean(data.isHotspot);
  const hasBus = Boolean(data.hasBusFactorRisk);
  const signals = Number(data.architectureSignals ?? 0);
  const blastDepth = data.blastDepth != null ? Number(data.blastDepth) : null;
  const dimmed = Boolean(data.dim);

  return (
    <div
      className={[
        'graph-file-node',
        selected ? 'graph-file-node--selected' : '',
        isHotspot ? 'graph-file-node--hotspot' : '',
        hasBus ? 'graph-file-node--bus' : '',
        dimmed ? 'graph-file-node--dimmed' : '',
        blastDepth != null ? `graph-file-node--blast-d${String(Math.min(blastDepth, 3))}` : '',
        data.blastTarget ? 'graph-file-node--blast-target' : '',
      ]
        .filter(Boolean)
        .join(' ')}
    >
      <Handle type="target" position={Position.Left} />
      <div className="graph-file-node__row">
        <FileTypeIcon type={ft} label={fileTypeLabel(ft)} />
        <span className="graph-file-node__path">{basename(path)}</span>
        {isHotspot ? <i className="codicon codicon-warning" style={{ color: 'var(--color-error)' }} /> : null}
        {signals > 0 ? (
          <span style={{ fontSize: 10, color: 'var(--color-info)' }} title="Signals">
            ◆{signals}
          </span>
        ) : null}
      </div>
      <div className="graph-file-node__meta">
        {String(data.symbolCount ?? 0)} symbols
        {data.dominantOwnerLogin ? ` · @${String(data.dominantOwnerLogin)}` : ''}
        {isHotspot ? <span className="graph-file-node__churn"> · churn HIGH</span> : null}
      </div>
      <Handle type="source" position={Position.Right} />
    </div>
  );
});
