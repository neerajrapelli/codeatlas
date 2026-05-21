import { useEffect, useState } from 'react';

import { api } from '../../lib/api';
import { BlastRadiusSummary } from '../graph/BlastRadiusSummary';
import { basename, detectFileType, fileTypeLabel } from '../../lib/fileType';
import { useStore } from '../../store';
import { FileTypeIcon } from '../ui/FileTypeIcon';
import { RiskBadge } from '../ui/RiskBadge';

export function InspectorPanel() {
  const inspectorOpen = useStore((s) => s.inspectorOpen);
  const setInspectorOpen = useStore((s) => s.setInspectorOpen);
  const activeRepoId = useStore((s) => s.activeRepoId);
  const selectedNodeId = useStore((s) => s.selectedNodeId);
  const selectedNodePath = useStore((s) => s.selectedNodePath);
  const fileDetail = useStore((s) => s.fileDetail);
  const setFileDetail = useStore((s) => s.setFileDetail);
  const hotspots = useStore((s) => s.hotspots);
  const setSelectedNode = useStore((s) => s.setSelectedNode);
  const setBlastRadius = useStore((s) => s.setBlastRadius);
  const [blastBusy, setBlastBusy] = useState(false);

  useEffect(() => {
    if (!inspectorOpen || activeRepoId == null || !selectedNodeId) {
      setFileDetail(null);
      return;
    }
    void api.getGraphFile(activeRepoId, selectedNodeId).then(setFileDetail).catch(() => setFileDetail(null));
  }, [inspectorOpen, activeRepoId, selectedNodeId, setFileDetail]);

  if (!inspectorOpen) return null;

  const path = selectedNodePath ?? fileDetail?.path ?? '';
  const hot = hotspots.find((h) => String(h.fileId) === selectedNodeId);
  const ft = path ? detectFileType(path) : 'other';

  return (
    <aside className="inspector-panel">
      <div className="inspector-section">
        <div style={{ display: 'flex', justifyContent: 'space-between' }}>
          <div className="graph-file-node__row">
            <FileTypeIcon type={ft} label={fileTypeLabel(ft)} />
            <span className="mono">{basename(path) || '—'}</span>
          </div>
          <button type="button" className="btn-icon" onClick={() => setInspectorOpen(false)} aria-label="Close">
            <i className="codicon codicon-close" />
          </button>
        </div>
        {hot ? (
          <p style={{ margin: '8px 0 0', fontSize: 'var(--font-size-xs)' }}>
            <RiskBadge level={hot.riskLevel} /> · {hot.commitCount90d} commits/90d · bus {hot.busFactor}
          </p>
        ) : (
          <p className="empty-state" style={{ padding: '8px 0' }}>
            No file metrics yet
          </p>
        )}
      </div>

      <div className="inspector-section">
        <h4>IMPORTS ({fileDetail?.imports.length ?? 0})</h4>
        {(fileDetail?.imports ?? []).slice(0, 8).map((imp) => (
          <div key={imp} className="mono" style={{ padding: '2px 0', cursor: 'pointer' }}>
            → {imp}
          </div>
        ))}
      </div>

      <div className="inspector-section">
        <h4>EXPORTS ({fileDetail?.exports.length ?? 0})</h4>
        {(fileDetail?.exports ?? []).slice(0, 8).map((ex) => (
          <div key={ex} className="mono" style={{ padding: '2px 0' }}>
            {ex}
          </div>
        ))}
      </div>

      <div className="inspector-section">
        <h4>BLAST RADIUS</h4>
        {path ? (
          <button
            type="button"
            className="btn-secondary"
            disabled={blastBusy || activeRepoId == null}
            onClick={() => {
              if (activeRepoId == null || !path) return;
              setBlastBusy(true);
              void api
                .getBlastRadius(activeRepoId, path, { depth: 3 })
                .then(setBlastRadius)
                .catch(() => undefined)
                .finally(() => setBlastBusy(false));
            }}
          >
            {blastBusy ? 'Analyzing…' : 'Analyze blast radius'}
          </button>
        ) : (
          <p className="empty-state">Select a file node.</p>
        )}
        <BlastRadiusSummary />
      </div>

      <div className="inspector-section">
        <h4>SIGNALS</h4>
        <p className="empty-state">Phase 2 — not ingested yet.</p>
      </div>
    </aside>
  );
}
