import { useEffect, useState } from 'react';

import { api } from '../../lib/api';
import { useStore } from '../../store';
import type { BoundaryViolationRow, TeamRow } from '../../types';

export function TeamsView() {
  const activeRepoId = useStore((s) => s.activeRepoId);
  const [teams, setTeams] = useState<TeamRow[]>([]);
  const [violations, setViolations] = useState<BoundaryViolationRow[]>([]);
  const [gaps, setGaps] = useState<{ filePath: string; message?: string }[]>([]);

  useEffect(() => {
    if (activeRepoId == null) return;
    void api.listTeams(activeRepoId).then(setTeams).catch(() => setTeams([]));
    void api.getBoundaryViolations(activeRepoId).then(setViolations).catch(() => setViolations([]));
    void api.getOwnershipGaps(activeRepoId).then(setGaps).catch(() => setGaps([]));
  }, [activeRepoId]);

  if (activeRepoId == null) {
    return <p className="empty-state">Select a repository.</p>;
  }

  return (
    <div className="sidebar-view">
      <h3 className="sidebar-section-title">TEAMS</h3>
      {teams.length === 0 ? (
        <p className="empty-state">No teams synced yet. Run ingestion with CODEOWNERS in the repo.</p>
      ) : (
        <ul className="sidebar-list">
          {teams.map((t) => (
            <li key={t.id}>
              <span style={{ color: t.color || '#888' }}>●</span> {t.displayName}{' '}
              <span className="muted">{String(t.fileCount)} files</span>
            </li>
          ))}
        </ul>
      )}
      <h3 className="sidebar-section-title">BOUNDARY VIOLATIONS ({violations.length})</h3>
      {violations.length === 0 ? (
        <p className="empty-state">None detected.</p>
      ) : (
        violations.slice(0, 20).map((v, i) => (
          <div key={i} className="sidebar-card">
            <div className="muted">
              {v.sourceTeam} → {v.targetTeam}
            </div>
            <div>{v.sourceFile}</div>
            <div className="muted">→ {v.targetFile}</div>
          </div>
        ))
      )}
      {gaps.length > 0 ? (
        <>
          <h3 className="sidebar-section-title">OWNERSHIP GAPS ({gaps.length})</h3>
          {gaps.slice(0, 15).map((g) => (
            <div key={g.filePath} className="muted">
              ? {g.filePath}
            </div>
          ))}
        </>
      ) : null}
    </div>
  );
}
