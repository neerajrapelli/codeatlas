import { useCallback, useEffect, useState } from 'react';

import { api } from '../../lib/api';
import { basename } from '../../lib/fileType';
import { useStore } from '../../store';
import type { BoundaryViolationRow, TeamRow } from '../../types';
import { EmptyState } from '../ui/EmptyState';
import { ViewSkeleton } from '../ui/ViewSkeleton';

export function TeamsView() {
  const activeRepoId = useStore((s) => s.activeRepoId);
  const setSelectedNode = useStore((s) => s.setSelectedNode);
  const setSidebarView = useStore((s) => s.setSidebarView);
  const [teams, setTeams] = useState<TeamRow[]>([]);
  const [violations, setViolations] = useState<BoundaryViolationRow[]>([]);
  const [gaps, setGaps] = useState<{ filePath: string; message?: string }[]>([]);
  const [loading, setLoading] = useState(false);
  const [expandedTeamId, setExpandedTeamId] = useState<string | null>(null);
  const [teamFiles, setTeamFiles] = useState<string[]>([]);
  const clusterLayer = useStore((s) => s.clusterLayer);
  const [filesLoading, setFilesLoading] = useState(false);

  useEffect(() => {
    if (activeRepoId == null) {
      setTeams([]);
      setViolations([]);
      setGaps([]);
      return;
    }
    setLoading(true);
    void Promise.all([
      api.listTeams(activeRepoId),
      api.getBoundaryViolations(activeRepoId),
      api.getOwnershipGaps(activeRepoId),
    ])
      .then(([t, v, g]) => {
        setTeams(t);
        setViolations(v);
        setGaps(g);
      })
      .finally(() => setLoading(false));
  }, [activeRepoId]);

  const toggleTeam = useCallback(
    async (teamId: string) => {
      if (expandedTeamId === teamId) {
        setExpandedTeamId(null);
        setTeamFiles([]);
        return;
      }
      setExpandedTeamId(teamId);
      if (activeRepoId == null) return;
      setFilesLoading(true);
      try {
        const files = await api.listTeamFiles(activeRepoId, teamId);
        setTeamFiles(files);
      } finally {
        setFilesLoading(false);
      }
    },
    [activeRepoId, expandedTeamId],
  );

  const openOnMap = (path: string) => {
    const f = clusterLayer?.files?.find((x) => x.path === path);
    if (f) setSelectedNode(f.id, f.path);
    else setSelectedNode(null, path);
    setSidebarView('map');
  };

  if (activeRepoId == null) {
    return (
      <div className="sidebar-view">
        <h3 className="sidebar-section-title">TEAMS</h3>
        <EmptyState title="No repository" description="Select or add a repository first." />
      </div>
    );
  }

  return (
    <div className="sidebar-view">
      <h3 className="sidebar-section-title">TEAMS</h3>
      {loading ? <ViewSkeleton rows={4} /> : null}
      {!loading && teams.length === 0 ? (
        <EmptyState
          icon="codicon-group-by-ref-type"
          title="No teams yet"
          description="Ingest a repo with CODEOWNERS or team metadata to populate boundaries."
        />
      ) : null}
      {!loading
        ? teams.map((t) => (
            <div key={t.id} className="team-block">
              <button type="button" className="team-block__head" onClick={() => void toggleTeam(t.id)}>
                <span className="team-block__dot" style={{ color: t.color || 'var(--text-muted)' }}>
                  ●
                </span>
                <span className="team-block__name">{t.displayName}</span>
                <span className="muted">{String(t.fileCount)} files</span>
                <i
                  className={`codicon codicon-chevron-${expandedTeamId === t.id ? 'down' : 'right'}`}
                  aria-hidden
                />
              </button>
              {expandedTeamId === t.id ? (
                <div className="team-block__files">
                  {filesLoading ? <ViewSkeleton rows={3} /> : null}
                  {!filesLoading && teamFiles.length === 0 ? (
                    <p className="muted">No files mapped to this team.</p>
                  ) : null}
                  {teamFiles.slice(0, 40).map((path) => (
                    <button
                      key={path}
                      type="button"
                      className="team-file-row"
                      onClick={() => openOnMap(path)}
                    >
                      {basename(path)}
                    </button>
                  ))}
                </div>
              ) : null}
            </div>
          ))
        : null}

      <h3 className="sidebar-section-title">BOUNDARY VIOLATIONS ({violations.length})</h3>
      {violations.length === 0 ? (
        <p className="sidebar-hint">No cross-team dependency violations detected.</p>
      ) : (
        violations.slice(0, 20).map((v, i) => (
          <div key={`${v.sourceFile}-${v.targetFile}-${String(i)}`} className="sidebar-card">
            <div className="muted">
              {v.sourceTeam} → {v.targetTeam}
            </div>
            <div className="mono">{basename(v.sourceFile)}</div>
            <div className="muted">→ {basename(v.targetFile)}</div>
            {v.message ? <div className="sidebar-hint">{v.message}</div> : null}
          </div>
        ))
      )}

      {gaps.length > 0 ? (
        <>
          <h3 className="sidebar-section-title">OWNERSHIP GAPS ({gaps.length})</h3>
          {gaps.slice(0, 15).map((g) => (
            <div key={g.filePath} className="muted mono">
              ? {g.filePath}
            </div>
          ))}
        </>
      ) : null}
    </div>
  );
}
