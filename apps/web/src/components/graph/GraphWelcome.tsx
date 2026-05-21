import { useStore } from '../../store';
import { EmptyState } from '../ui/EmptyState';

export function GraphWelcome() {
  const setSidebarView = useStore((s) => s.setSidebarView);

  return (
    <div className="graph-welcome">
      <EmptyState
        icon="codicon-type-hierarchy-sub"
        title="Architecture map"
        description="Add a repository, wait for indexing to finish, then explore clusters and file dependencies on the canvas."
        action={
          <button type="button" className="btn-primary" onClick={() => setSidebarView('repos')}>
            Add repository
          </button>
        }
      />
    </div>
  );
}
