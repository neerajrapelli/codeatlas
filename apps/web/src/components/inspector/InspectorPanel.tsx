import { useEffect, useState } from 'react';

import { api } from '../../lib/api';
import { BlastRadiusSummary } from '../graph/BlastRadiusSummary';
import { basename, detectFileType, fileTypeLabel } from '../../lib/fileType';
import { useStore } from '../../store';
import { EmptyState } from '../ui/EmptyState';
import { FileTypeIcon } from '../ui/FileTypeIcon';
import { RiskBadge } from '../ui/RiskBadge';
import { ViewSkeleton } from '../ui/ViewSkeleton';

export function InspectorPanel() {
  const inspectorOpen = useStore((s) => s.inspectorOpen);
  const setInspectorOpen = useStore((s) => s.setInspectorOpen);
  const activeRepoId = useStore((s) => s.activeRepoId);
  const selectedNodeId = useStore((s) => s.selectedNodeId);
  const selectedNodePath = useStore((s) => s.selectedNodePath);
  const fileDetail = useStore((s) => s.fileDetail);
  const setFileDetail = useStore((s) => s.setFileDetail);
  const hotspots = useStore((s) => s.hotspots);
  const setBlastRadius = useStore((s) => s.setBlastRadius);
  const pushToast = useStore((s) => s.pushToast);
  const [blastBusy, setBlastBusy] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);

  useEffect(() => {
    if (!inspectorOpen || activeRepoId == null || !selectedNodeId) {
      setFileDetail(null);
      return;
    }
    setDetailLoading(true);
    void api
      .getGraphFile(activeRepoId, selectedNodeId)
      .then(setFileDetail)
      .catch(() => {
        setFileDetail(null);
        pushToast('Could not load file details from API', 'error');
      })
      .finally(() => setDetailLoading(false));
  }, [inspectorOpen, activeRepoId, selectedNodeId, setFileDetail, pushToast]);

  if (!inspectorOpen) return null;

  if (!selectedNodeId) {
    return (
      <aside className="inspector-panel">
        <div className="inspector-panel__header">
          <span>Inspector</span>
          <button type="button" className="btn-icon" onClick={() => setInspectorOpen(false)} aria-label="Close">
            <i className="codicon codicon-close" />
          </button>
        </div>
        <EmptyState
          icon="codicon-file"
          title="No file selected"
          description="Click a file node on the architecture map to inspect imports, exports, and blast radius."
        />
      </aside>
    );
  }

  const path = selectedNodePath ?? fileDetail?.path ?? '';
  const hot = hotspots.find((h) => String(h.fileId) === selectedNodeId);
  const ft = path ? detectFileType(path) : 'other';

  return (
    <aside className="inspector-panel">
      <div className="inspector-panel__header">
        <div className="graph-file-node__row">
          <FileTypeIcon type={ft} label={fileTypeLabel(ft)} />
          <span className="mono inspector-panel__path">{basename(path) || '—'}</span>
        </div>
        <button type="button" className="btn-icon" onClick={() => setInspectorOpen(false)} aria-label="Close">
          <i className="codicon codicon-close" />
        </button>
      </div>

      {detailLoading ? (
        <ViewSkeleton rows={3} />
      ) : (
        <>
          <div className="inspector-section">
            {hot ? (
              <p className="inspector-meta">
                <RiskBadge level={hot.riskLevel} /> · {hot.commitCount90d} commits/90d · bus {hot.busFactor}
              </p>
            ) : (
              <p className="inspector-meta inspector-meta--muted">No socio metrics for this file yet</p>
            )}
          </div>

          <div className="inspector-section">
            <h4>IMPORTS ({fileDetail?.imports.length ?? 0})</h4>
            {(fileDetail?.imports ?? []).slice(0, 12).map((imp) => (
              <div key={imp} className="mono inspector-line">
                → {imp}
              </div>
            ))}
          </div>

          <div className="inspector-section">
            <h4>EXPORTS ({fileDetail?.exports.length ?? 0})</h4>
            {(fileDetail?.exports ?? []).slice(0, 12).map((ex) => (
              <div key={ex} className="mono inspector-line">
                {ex}
              </div>
            ))}
          </div>
        </>
      )}

      <div className="inspector-section">
        <h4>BLAST RADIUS</h4>
        {path ? (
          <button
            type="button"
            className="btn-secondary btn-primary--block"
            disabled={blastBusy || activeRepoId == null}
            onClick={() => {
              if (activeRepoId == null || !path) return;
              setBlastBusy(true);
              void api
                .getBlastRadius(activeRepoId, path, { depth: 3 })
                .then(setBlastRadius)
                .catch((e) => pushToast(e instanceof Error ? e.message : 'Blast radius failed', 'error'))
                .finally(() => setBlastBusy(false));
            }}
          >
            {blastBusy ? 'Analyzing…' : 'Analyze blast radius'}
          </button>
        ) : null}
        <BlastRadiusSummary />
      </div>
    </aside>
  );
}
