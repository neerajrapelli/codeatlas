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
  const viol = data.violationSeverity as string | undefined;
  const highlight = Boolean(data.highlight);
  const hoverPing = Boolean(data.hoverPing);
  const metaParts = [
    `${String(data.symbolCount ?? 0)} symbols`,
    data.dominantOwnerLogin ? `@${String(data.dominantOwnerLogin)}` : '',
    isHotspot ? 'churn HIGH' : '',
  ].filter(Boolean);

  return (
    <div
      className={[
        'graph-file-node',
        selected ? 'graph-file-node--selected' : '',
        highlight ? 'graph-file-node--highlight' : '',
        hoverPing ? 'graph-file-node--ping' : '',
        isHotspot ? 'graph-file-node--hotspot' : '',
        hasBus ? 'graph-file-node--bus' : '',
        dimmed ? 'graph-file-node--dimmed' : '',
        blastDepth != null ? `graph-file-node--blast-d${String(Math.min(blastDepth, 3))}` : '',
        data.blastTarget ? 'graph-file-node--blast-target' : '',
        viol === 'error' ? 'graph-file-node--violation-error' : '',
        viol === 'warning' ? 'graph-file-node--violation-warning' : '',
      ]
        .filter(Boolean)
        .join(' ')}
      title={path}
    >
      <Handle type="target" position={Position.Left} />
      <div className="graph-file-node__row">
        <FileTypeIcon type={ft} label={fileTypeLabel(ft)} />
        <span className="graph-file-node__path">{basename(path)}</span>
        {isHotspot ? (
          <i className="codicon codicon-warning graph-file-node__warn" aria-hidden />
        ) : null}
        {signals > 0 ? (
          <span className="graph-file-node__signals" title="Signals">
            ◆{signals}
          </span>
        ) : null}
      </div>
      <div className="graph-file-node__meta">{metaParts.join(' · ')}</div>
      <Handle type="source" position={Position.Right} />
    </div>
  );
});
