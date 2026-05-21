import { useStore } from '../../store';
import { basename } from '../../lib/fileType';
import { RiskBadge } from '../ui/RiskBadge';

export function HotspotsView() {
  const hotspots = useStore((s) => s.hotspots);
  const setSelectedNode = useStore((s) => s.setSelectedNode);
  const setSidebarView = useStore((s) => s.setSidebarView);

  return (
    <div className="sidebar-view">
      <h3 className="sidebar-section-title">HOTSPOTS</h3>
      {hotspots.length === 0 ? (
        <p className="empty-state">No hotspot metrics yet. Complete GitHub socio sync.</p>
      ) : null}
      {hotspots.map((h) => {
        const pct = Math.min(100, Math.round(h.hotspotScore * 100));
        return (
          <div
            key={h.fileId}
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
            <div style={{ fontSize: 'var(--font-size-xs)', color: 'var(--text-secondary)', marginTop: 4 }}>
              {h.commitCount90d} commits · {h.busFactor} owners · <RiskBadge level={h.riskLevel} />
            </div>
          </div>
        );
      })}
    </div>
  );
}
