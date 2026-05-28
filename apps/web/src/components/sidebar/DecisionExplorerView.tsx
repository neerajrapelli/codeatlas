import { useStore } from '../../store';
import { useArchitectureDecisionsQuery } from '../../hooks/queries/useArchitectureIntelligence';
import { EmptyState } from '../ui/EmptyState';
import { ViewSkeleton } from '../ui/ViewSkeleton';

export function DecisionExplorerView() {
  const activeRepoId = useStore((s) => s.activeRepoId);
  const { data, isLoading } = useArchitectureDecisionsQuery(activeRepoId);

  if (activeRepoId == null) {
    return (
      <div className="sidebar-view">
        <h3 className="sidebar-section-title">DECISION EXPLORER</h3>
        <EmptyState title="No repository" description="Select a repository first." />
      </div>
    );
  }
  const decisions = data ?? [];
  return (
    <div className="sidebar-view">
      <h3 className="sidebar-section-title">DECISION EXPLORER</h3>
      {isLoading && decisions.length === 0 ? <ViewSkeleton rows={6} /> : null}
      {!isLoading && decisions.length === 0 ? (
        <EmptyState
          title="No architecture decisions"
          description="Decisions are extracted from discussions, RFCs, ADRs and PR reviews."
        />
      ) : null}
      {decisions.map((d) => (
        <div key={d.id} className="sidebar-card">
          <div className="sidebar-card__title">{d.title}</div>
          <div className="sidebar-card__meta">
            {d.status} · confidence {Math.round(d.confidence * 100)}%
          </div>
          <p className="sidebar-card__desc">{d.summary}</p>
        </div>
      ))}
    </div>
  );
}
