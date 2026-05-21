import type { SidebarView } from '../../types';
import { useStore } from '../../store';

const ITEMS: Array<{ view: SidebarView; icon: string; title: string }> = [
  { view: 'repos', icon: 'codicon-folder-opened', title: 'Repositories' },
  { view: 'map', icon: 'codicon-type-hierarchy-sub', title: 'Architecture Map' },
  { view: 'hotspots', icon: 'codicon-warning', title: 'Hotspots' },
  { view: 'signals', icon: 'codicon-comment-discussion', title: 'Signals' },
  { view: 'ownership', icon: 'codicon-organization', title: 'Ownership' },
  { view: 'drift', icon: 'codicon-shield', title: 'Architecture Drift' },
  { view: 'mcp', icon: 'codicon-plug', title: 'MCP' },
  { view: 'timeline', icon: 'codicon-history', title: 'Timeline' },
];

export function ActivityBar() {
  const sidebarView = useStore((s) => s.sidebarView);
  const setSidebarView = useStore((s) => s.setSidebarView);

  return (
    <nav className="activity-bar" aria-label="Activity Bar">
      {ITEMS.map((item, i) => (
        <button
          key={item.view}
          type="button"
          className={`activity-bar__btn ${sidebarView === item.view ? 'activity-bar__btn--active' : ''}`}
          title={item.title}
          aria-label={item.title}
          onClick={() => setSidebarView(item.view)}
        >
          <i className={`codicon ${item.icon}`} />
        </button>
      ))}
      <div className="activity-bar__spacer" />
      <button type="button" className="activity-bar__btn" title="Settings" aria-label="Settings">
        <i className="codicon codicon-settings-gear" />
      </button>
    </nav>
  );
}
