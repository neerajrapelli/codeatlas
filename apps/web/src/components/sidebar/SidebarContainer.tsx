import { useStore } from '../../store';
import { HotspotsView } from './HotspotsView';
import { OwnershipView } from './OwnershipView';
import { RepositoriesView } from './RepositoriesView';
import { SignalsView } from './SignalsView';

export function SidebarContainer() {
  const sidebarView = useStore((s) => s.sidebarView);
  const sidebarVisible = useStore((s) => s.sidebarVisible);

  if (!sidebarVisible) return null;

  return (
    <aside className="primary-sidebar">
      {sidebarView === 'repos' ? <RepositoriesView /> : null}
      {sidebarView === 'map' ? (
        <div className="sidebar-view">
          <h3 className="sidebar-section-title">ARCHITECTURE MAP</h3>
          <p className="empty-state">
            Use the canvas to drill clusters. Select a file node to open the inspector. ⌘F focuses search via command
            palette.
          </p>
        </div>
      ) : null}
      {sidebarView === 'hotspots' ? <HotspotsView /> : null}
      {sidebarView === 'signals' ? <SignalsView /> : null}
      {sidebarView === 'ownership' ? <OwnershipView /> : null}
      {sidebarView === 'timeline' ? (
        <div className="sidebar-view">
          <h3 className="sidebar-section-title">TIMELINE</h3>
          <p className="empty-state">File timeline API (Phase 2) not available yet.</p>
        </div>
      ) : null}
    </aside>
  );
}
