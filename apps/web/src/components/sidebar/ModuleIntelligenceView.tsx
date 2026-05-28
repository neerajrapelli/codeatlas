import { useState } from 'react';

import { useStore } from '../../store';
import { useArchitectureModuleIntelQuery } from '../../hooks/queries/useArchitectureIntelligence';
import { EmptyState } from '../ui/EmptyState';

export function ModuleIntelligenceView() {
  const activeRepoId = useStore((s) => s.activeRepoId);
  const [modulePath, setModulePath] = useState('');
  const { data, isLoading } = useArchitectureModuleIntelQuery(activeRepoId, modulePath);

  return (
    <div className="sidebar-view">
      <h3 className="sidebar-section-title">MODULE INTELLIGENCE</h3>
      {activeRepoId == null ? (
        <EmptyState title="No repository" description="Select a repository first." />
      ) : (
        <>
          <input
            className="repo-input"
            value={modulePath}
            placeholder="Enter folder/module path (e.g. apps/api/internal)"
            onChange={(e) => setModulePath(e.target.value)}
          />
          {modulePath.trim().length === 0 ? (
            <EmptyState title="Enter module path" description="Query architecture context for a module." />
          ) : null}
          {isLoading ? <div className="muted">Loading module intelligence…</div> : null}
          {data ? (
            <div className="sidebar-card">
              <div className="sidebar-card__title">{data.modulePath}</div>
              <div className="sidebar-card__meta">{data.decisionCount} decisions</div>
              <p className="sidebar-card__desc">
                {data.relatedPRs.length} related PRs · {data.topMaintainers.length} maintainers involved
              </p>
            </div>
          ) : null}
        </>
      )}
    </div>
  );
}
