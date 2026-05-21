import { useStore } from '../../store';
import { basename } from '../../lib/fileType';
import { RiskBadge } from '../ui/RiskBadge';
import { ViewSkeleton } from '../ui/ViewSkeleton';
import { EmptyState } from '../ui/EmptyState';

export function OwnershipView() {
  const ownershipRows = useStore((s) => s.ownershipRows);
  const socioLoading = useStore((s) => s.socioLoading);
  const activeRepoId = useStore((s) => s.activeRepoId);
  const setSelectedNode = useStore((s) => s.setSelectedNode);
  const setSidebarView = useStore((s) => s.setSidebarView);

  if (activeRepoId == null) {
    return (
      <div className="sidebar-view">
        <h3 className="sidebar-section-title">OWNERSHIP</h3>
        <EmptyState title="No repository" description="Select or add a repository first." />
      </div>
    );
  }

  return (
    <div className="sidebar-view">
      <h3 className="sidebar-section-title">OWNERSHIP</h3>
      {socioLoading && ownershipRows.length === 0 ? <ViewSkeleton rows={6} /> : null}
      {!socioLoading && ownershipRows.length === 0 ? (
        <EmptyState
          icon="codicon-person"
          title="No ownership data"
          description="Index a GitHub repo with GITHUB_TOKEN configured on the API."
        />
      ) : null}
      {ownershipRows.slice(0, 40).map((row) => (
        <button
          key={row.fileId}
          type="button"
          className="hotspot-row"
          onClick={() => {
            setSelectedNode(String(row.fileId), row.path);
            setSidebarView('map');
          }}
        >
          <div className="mono">{basename(row.path)}</div>
          <div className="hotspot-row__meta">
            {row.dominantOwner?.login ? `@${row.dominantOwner.login}` : '—'} · bus {row.busFactor} ·{' '}
            <RiskBadge level={row.riskLevel} />
          </div>
        </button>
      ))}
    </div>
  );
}
