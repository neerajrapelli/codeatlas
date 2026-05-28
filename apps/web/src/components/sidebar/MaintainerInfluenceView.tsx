import { useStore } from '../../store';
import { useMaintainerInfluenceQuery } from '../../hooks/queries/useArchitectureIntelligence';
import { EmptyState } from '../ui/EmptyState';
import { ViewSkeleton } from '../ui/ViewSkeleton';

export function MaintainerInfluenceView() {
  const activeRepoId = useStore((s) => s.activeRepoId);
  const { data, isLoading } = useMaintainerInfluenceQuery(activeRepoId);
  const rows = data ?? [];

  if (activeRepoId == null) {
    return (
      <div className="sidebar-view">
        <h3 className="sidebar-section-title">MAINTAINER INFLUENCE</h3>
        <EmptyState title="No repository" description="Select a repository first." />
      </div>
    );
  }

  return (
    <div className="sidebar-view">
      <h3 className="sidebar-section-title">MAINTAINER INFLUENCE</h3>
      {isLoading && rows.length === 0 ? <ViewSkeleton rows={6} /> : null}
      {!isLoading && rows.length === 0 ? (
        <EmptyState title="No maintainer influence data" description="Run discussion ingestion to compute influence." />
      ) : null}
      {rows.map((m) => (
        <div key={m.login} className="sidebar-card">
          <div className="sidebar-card__title">{m.displayName || m.login}</div>
          <div className="sidebar-card__meta">
            {m.decisionsShaped} decisions · {m.acceptedProposals} accepted · {m.rejectedProposals} rejected
          </div>
        </div>
      ))}
    </div>
  );
}
