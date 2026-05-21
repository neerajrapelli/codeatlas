import { useStore } from '../../store';
import { basename } from '../../lib/fileType';
import { RiskBadge } from '../ui/RiskBadge';

export function OwnershipView() {
  const ownershipRows = useStore((s) => s.ownershipRows);
  const setSelectedNode = useStore((s) => s.setSelectedNode);
  const setSidebarView = useStore((s) => s.setSidebarView);

  return (
    <div className="sidebar-view">
      <h3 className="sidebar-section-title">OWNERSHIP</h3>
      {ownershipRows.length === 0 ? (
        <p className="empty-state">No ownership data. Index a GitHub repo with GITHUB_TOKEN.</p>
      ) : null}
      {ownershipRows.slice(0, 40).map((row) => (
        <div
          key={row.fileId}
          className="hotspot-row"
          onClick={() => {
            setSelectedNode(String(row.fileId), row.path);
            setSidebarView('map');
          }}
        >
          <div className="mono">{basename(row.path)}</div>
          <div style={{ fontSize: 'var(--font-size-xs)', color: 'var(--text-secondary)' }}>
            {row.dominantOwner?.login ? `@${row.dominantOwner.login}` : '—'} · bus {row.busFactor} ·{' '}
            <RiskBadge level={row.riskLevel} />
          </div>
        </div>
      ))}
    </div>
  );
}
