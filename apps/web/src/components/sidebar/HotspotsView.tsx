import { useStore } from '../../store';
import { basename } from '../../lib/fileType';
import { RiskBadge } from '../ui/RiskBadge';
import { ViewSkeleton } from '../ui/ViewSkeleton';
import { EmptyState } from '../ui/EmptyState';

export function HotspotsView() {
  const hotspots = useStore((s) => s.hotspots);
  const socioLoading = useStore((s) => s.socioLoading);
  const activeRepoId = useStore((s) => s.activeRepoId);
  const setSelectedNode = useStore((s) => s.setSelectedNode);
  const setSidebarView = useStore((s) => s.setSidebarView);

  if (activeRepoId == null) {
    return (
      <div className="sidebar-view">
        <h3 className="sidebar-section-title">HOTSPOTS</h3>
        <EmptyState title="No repository" description="Select or add a repository first." />
      </div>
    );
  }

  return (
    <div className="sidebar-view">
      <h3 className="sidebar-section-title">HOTSPOTS</h3>
      {socioLoading && hotspots.length === 0 ? <ViewSkeleton rows={6} /> : null}
      {!socioLoading && hotspots.length === 0 ? (
        <EmptyState
          icon="codicon-warning"
          title="No hotspot data"
          description="Complete GitHub socio sync (GITHUB_TOKEN on API) after indexing finishes."
        />
      ) : null}
      {hotspots.map((h) => {
        const pct = Math.min(100, Math.round(h.hotspotScore * 100));
        return (
          <button
            key={h.fileId}
            type="button"
            className="hotspot-row"
            onClick={() => {
              setSelectedNode(String(h.fileId), h.path);
              setSidebarView('map');
            }}
          >
            <div className="mono">{basename(h.path)}</div>
            <div className="hotspot-bar">
              <div className="hotspot-bar__fill" style={{ width: `${String(pct)}%` }} />
            </div>
            <div className="hotspot-row__meta">
              {h.commitCount90d} commits · {h.busFactor} owners · <RiskBadge level={h.riskLevel} />
            </div>
          </button>
        );
      })}
    </div>
  );
}
