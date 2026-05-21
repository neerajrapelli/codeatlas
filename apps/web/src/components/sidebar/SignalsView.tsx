import { EmptyState } from '../ui/EmptyState';

export function SignalsView() {
  return (
    <div className="sidebar-view">
      <h3 className="sidebar-section-title">SIGNALS</h3>
      <EmptyState
        icon="codicon-comment-discussion"
        title="Signals coming soon"
        description="Engineering memory from PRs and issues will surface here in a future release."
      />
    </div>
  );
}
