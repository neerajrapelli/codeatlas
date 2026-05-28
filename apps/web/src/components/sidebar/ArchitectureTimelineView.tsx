import { useStore } from '../../store';
import { EmptyState } from '../ui/EmptyState';
import { ViewSkeleton } from '../ui/ViewSkeleton';
import { useArchitectureTimelineQuery } from '../../hooks/queries/useArchitectureIntelligence';

export function ArchitectureTimelineView() {
  const activeRepoId = useStore((s) => s.activeRepoId);
  const { data, isLoading } = useArchitectureTimelineQuery(activeRepoId);

  if (activeRepoId == null) {
    return (
      <div className="sidebar-view">
        <h3 className="sidebar-section-title">ARCHITECTURE TIMELINE</h3>
        <EmptyState title="No repository" description="Select a repository first." />
      </div>
    );
  }
  const items = data ?? [];
  return (
    <div className="sidebar-view">
      <h3 className="sidebar-section-title">ARCHITECTURE TIMELINE</h3>
      {isLoading && items.length === 0 ? <ViewSkeleton rows={6} /> : null}
      {!isLoading && items.length === 0 ? (
        <EmptyState title="No timeline entries" description="Run socio sync to populate architecture events." />
      ) : null}
      {items.map((item) => (
        <div key={item.id} className="sidebar-card">
          <div className="sidebar-card__title">{item.title}</div>
          <div className="sidebar-card__meta">
            {new Date(item.occurredAt).toLocaleString()} · {item.kind}
          </div>
          <p className="sidebar-card__desc">{item.summary}</p>
        </div>
      ))}
    </div>
  );
}
