import type { SidebarView } from '../../types';
import { useStore } from '../../store';

const ITEMS: Array<{ view: SidebarView; icon: string; title: string }> = [
  { view: 'repos', icon: 'codicon-folder-opened', title: 'Repositories' },
  { view: 'map', icon: 'codicon-type-hierarchy-sub', title: 'Architecture map' },
  { view: 'hotspots', icon: 'codicon-warning', title: 'Hotspots' },
  { view: 'signals', icon: 'codicon-comment-discussion', title: 'Signals' },
  { view: 'ownership', icon: 'codicon-person', title: 'Ownership' },
  { view: 'teams', icon: 'codicon-group-by-ref-type', title: 'Teams' },
  { view: 'docs', icon: 'codicon-book', title: 'Living docs' },
  { view: 'onboarding', icon: 'codicon-mortar-board', title: 'Onboarding' },
  { view: 'drift', icon: 'codicon-shield', title: 'Architecture drift' },
  { view: 'timeline', icon: 'codicon-history', title: 'Architecture timeline' },
  { view: 'decisions', icon: 'codicon-git-pull-request-closed', title: 'Decision explorer' },
  { view: 'module_intel', icon: 'codicon-symbol-module', title: 'Module intelligence' },
  { view: 'pr_insights', icon: 'codicon-git-pull-request', title: 'PR insights' },
  { view: 'maintainer_influence', icon: 'codicon-account', title: 'Maintainer influence' },
  { view: 'mcp', icon: 'codicon-plug', title: 'MCP' },
];

export function ActivityBar() {
  const sidebarView = useStore((s) => s.sidebarView);
  const setSidebarView = useStore((s) => s.setSidebarView);

  return (
    <nav className="activity-bar" aria-label="Primary">
      {ITEMS.map((item) => (
        <button
          key={item.view}
          type="button"
          className={`activity-bar__btn ${sidebarView === item.view ? 'activity-bar__btn--active' : ''}`}
          title={item.title}
          aria-label={item.title}
          aria-current={sidebarView === item.view ? 'page' : undefined}
          onClick={() => setSidebarView(item.view)}
        >
          <i className={`codicon ${item.icon} activity-bar__icon`} aria-hidden />
        </button>
      ))}
      <div className="activity-bar__spacer" />
      <button
        type="button"
        className={`activity-bar__btn ${sidebarView === 'settings' ? 'activity-bar__btn--active' : ''}`}
        title="Settings (⌘,)"
        aria-label="Settings"
        onClick={() => setSidebarView('settings')}
      >
        <i className="codicon codicon-settings-gear activity-bar__icon" aria-hidden />
      </button>
    </nav>
  );
}
