import { useStore } from '../../store';
import { OnboardingPanel } from './OnboardingPanel';
import { DocsView } from './DocsView';
import { DriftView } from './DriftView';
import { McpView } from './McpView';
import { SettingsView } from './SettingsView';
import { TeamsView } from './TeamsView';
import { HotspotsView } from './HotspotsView';
import { OwnershipView } from './OwnershipView';
import { RepositoriesView } from './RepositoriesView';
import { SignalsView } from './SignalsView';
import { EmptyState } from '../ui/EmptyState';

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
          <EmptyState
            icon="codicon-type-hierarchy-sub"
            title="Explore on the canvas"
            description="Click clusters to drill down. Select a file to inspect symbols and blast radius. Open ⌘K for quick actions."
          />
        </div>
      ) : null}
      {sidebarView === 'hotspots' ? <HotspotsView /> : null}
      {sidebarView === 'signals' ? <SignalsView /> : null}
      {sidebarView === 'ownership' ? <OwnershipView /> : null}
      {sidebarView === 'teams' ? <TeamsView /> : null}
      {sidebarView === 'docs' ? <DocsView /> : null}
      {sidebarView === 'onboarding' ? <OnboardingPanel /> : null}
      {sidebarView === 'drift' ? <DriftView /> : null}
      {sidebarView === 'settings' ? <SettingsView /> : null}
      {sidebarView === 'mcp' ? <McpView /> : null}
    </aside>
  );
}
