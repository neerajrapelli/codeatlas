import { useStore } from '../../store';
import { useArchitecturePRInsightsQuery } from '../../hooks/queries/useArchitectureIntelligence';
import { EmptyState } from '../ui/EmptyState';
import { ViewSkeleton } from '../ui/ViewSkeleton';

export function PRInsightView() {
  const activeRepoId = useStore((s) => s.activeRepoId);
  const { data, isLoading } = useArchitecturePRInsightsQuery(activeRepoId);
  const insights = data ?? [];

  if (activeRepoId == null) {
    return (
      <div className="sidebar-view">
        <h3 className="sidebar-section-title">PR INSIGHTS</h3>
        <EmptyState title="No repository" description="Select a repository first." />
      </div>
    );
  }

  return (
    <div className="sidebar-view">
      <h3 className="sidebar-section-title">PR INSIGHTS</h3>
      {isLoading && insights.length === 0 ? <ViewSkeleton rows={5} /> : null}
      {!isLoading && insights.length === 0 ? (
        <EmptyState title="No PR insights yet" description="Sync engineering memory to ingest PR reviews and discussions." />
      ) : null}
      {insights.map((pr) => (
        <div key={pr.pullRequestId} className="sidebar-card">
          <div className="sidebar-card__title">#{pr.number} {pr.title}</div>
          <div className="sidebar-card__meta">{pr.author} · {pr.reviewDisagreementCount} review events</div>
          <p className="sidebar-card__desc">{pr.summary}</p>
        </div>
      ))}
    </div>
  );
}
