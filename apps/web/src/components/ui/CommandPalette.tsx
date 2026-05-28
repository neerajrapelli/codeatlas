import { Command } from 'cmdk';
import Fuse, { type FuseResult, type FuseOptionKey } from 'fuse.js';
import { useEffect, useMemo, useState } from 'react';

import { basename } from '../../lib/fileType';
import { loadRecentCommands, pushRecentCommand } from '../../lib/recentCommands';
import type { SidebarView } from '../../types';
import { useStore } from '../../store';

type CommandGroup = 'recent' | 'repos' | 'ai' | 'actions' | 'views' | 'files';

type PaletteItem = {
  id: string;
  label: string;
  sub?: string;
  group: CommandGroup;
  icon: string;
  searchText: string;
  action: () => void;
};

const GROUP_LABELS: Record<CommandGroup, string> = {
  recent: 'Recent',
  repos: 'Repositories',
  ai: 'AI',
  actions: 'Actions',
  views: 'Views',
  files: 'Files & hotspots',
};

const AI_PROMPTS = [
  { id: 'ai-breaks', label: 'What breaks if auth changes?', icon: 'codicon-sparkle' },
  { id: 'ai-risk', label: 'Show highest risk files', icon: 'codicon-sparkle' },
  { id: 'ai-owner', label: 'Who owns checkout?', icon: 'codicon-sparkle' },
  { id: 'ai-trace', label: 'Trace this dependency', icon: 'codicon-sparkle' },
];

const VIEW_COMMANDS: Array<{ view: SidebarView; label: string; icon: string }> = [
  { view: 'repos', label: 'Repositories', icon: 'codicon-folder-opened' },
  { view: 'map', label: 'Architecture map', icon: 'codicon-type-hierarchy-sub' },
  { view: 'hotspots', label: 'Hotspots', icon: 'codicon-warning' },
  { view: 'ownership', label: 'Ownership', icon: 'codicon-person' },
  { view: 'teams', label: 'Teams', icon: 'codicon-group-by-ref-type' },
  { view: 'docs', label: 'Living docs', icon: 'codicon-book' },
  { view: 'onboarding', label: 'Onboarding', icon: 'codicon-mortar-board' },
  { view: 'drift', label: 'Architecture drift', icon: 'codicon-shield' },
  { view: 'timeline', label: 'Architecture timeline', icon: 'codicon-history' },
  { view: 'decisions', label: 'Decision explorer', icon: 'codicon-git-pull-request-closed' },
  { view: 'module_intel', label: 'Module intelligence', icon: 'codicon-symbol-module' },
  { view: 'pr_insights', label: 'PR insights', icon: 'codicon-git-pull-request' },
  { view: 'maintainer_influence', label: 'Maintainer influence', icon: 'codicon-account' },
  { view: 'mcp', label: 'MCP', icon: 'codicon-plug' },
  { view: 'settings', label: 'Settings', icon: 'codicon-settings-gear' },
];

const FUSE_OPTS = { threshold: 0.4, ignoreLocation: true, minMatchCharLength: 1 };

function filterWithFuse<T extends { searchText: string }>(
  items: T[],
  query: string,
  extraKeys: FuseOptionKey<T>[] = [],
): T[] {
  const q = query.trim();
  if (!q) return items;
  const fuse = new Fuse(items, {
    ...FUSE_OPTS,
    keys: ['searchText', 'label', 'sub', ...extraKeys],
  });
  return fuse.search(q).map((r: FuseResult<T>) => r.item);
}

function useModKey(): string {
  return typeof navigator !== 'undefined' && /Mac/i.test(navigator.platform) ? '⌘' : 'Ctrl+';
}

function PaletteGroups({
  items,
  renderLabel,
}: {
  items: PaletteItem[];
  renderLabel?: (item: PaletteItem) => string;
}) {
  const groups: CommandGroup[] = [];
  for (const item of items) {
    if (!groups.includes(item.group)) groups.push(item.group);
  }
  return (
    <>
      {groups.map((group) => (
        <Command.Group key={group} heading={GROUP_LABELS[group]} className="command-palette__group">
          {items
            .filter((i) => i.group === group)
            .map((item) => {
              const label = renderLabel ? renderLabel(item) : item.label;
              return (
                <Command.Item
                  key={item.id}
                  value={item.searchText}
                  onSelect={item.action}
                  className="command-palette__item"
                >
                  <span className="command-palette__item-main">
                    <i className={`codicon ${item.icon}`} aria-hidden />
                    <span className="command-palette__label" title={item.label}>
                      {label}
                    </span>
                  </span>
                  {item.sub ? <span className="command-palette__sub">{item.sub}</span> : null}
                </Command.Item>
              );
            })}
        </Command.Group>
      ))}
    </>
  );
}

export function CommandPalette() {
  const open = useStore((s) => s.commandPaletteOpen);
  const mode = useStore((s) => s.paletteMode);
  const setOpen = useStore((s) => s.setCommandPaletteOpen);
  const clusterLayer = useStore((s) => s.clusterLayer);
  const hotspots = useStore((s) => s.hotspots);
  const repositories = useStore((s) => s.repositories);
  const activeRepoId = useStore((s) => s.activeRepoId);
  const selectedNodePath = useStore((s) => s.selectedNodePath);
  const setSelectedNode = useStore((s) => s.setSelectedNode);
  const setSidebarView = useStore((s) => s.setSidebarView);
  const setFocusRepoInput = useStore((s) => s.setFocusRepoInput);
  const setActiveRepo = useStore((s) => s.setActiveRepo);
  const toggleBottomPanel = useStore((s) => s.toggleBottomPanel);
  const bottomPanelOpen = useStore((s) => s.bottomPanelOpen);
  const setAiPanelDraft = useStore((s) => s.setAiPanelDraft);
  const toggleInspector = useStore((s) => s.toggleInspector);
  const [q, setQ] = useState('');
  const [recentVersion, setRecentVersion] = useState(0);
  const modKey = useModKey();

  const close = () => setOpen(false);

  const runCommand = (id: string, label: string, action: () => void) => {
    return () => {
      pushRecentCommand({ id, label });
      setRecentVersion((v) => v + 1);
      action();
    };
  };

  const openAiPrompt = (prompt: string) => {
    if (!bottomPanelOpen) toggleBottomPanel();
    setAiPanelDraft(prompt);
    setSidebarView('map');
    close();
  };

  const allCommands = useMemo((): PaletteItem[] => {
    const list: PaletteItem[] = [];
    const add = (
      group: CommandGroup,
      id: string,
      label: string,
      icon: string,
      action: () => void,
      sub?: string,
    ) => {
      list.push({
        id,
        label,
        sub,
        group,
        icon,
        searchText: `${label} ${sub ?? ''}`,
        action: runCommand(id, label, action),
      });
    };

    for (const r of repositories) {
      const active = r.id === activeRepoId;
      add(
        'repos',
        `repo-${String(r.id)}`,
        r.name,
        active ? 'codicon-check' : 'codicon-repo',
        () => {
          setActiveRepo(r.id);
          setSidebarView('map');
          close();
        },
        active ? 'Active repository' : 'Switch repository',
      );
    }

    for (const p of AI_PROMPTS) {
      add('ai', p.id, `Ask AI: ${p.label}`, p.icon, () => openAiPrompt(p.label), 'Architecture assistant');
    }
    if (selectedNodePath) {
      add(
        'ai',
        'ai-selected-file',
        `Ask AI about ${basename(selectedNodePath)}`,
        'codicon-sparkle',
        () => openAiPrompt(`Explain architecture and risks for file ${selectedNodePath}`),
        selectedNodePath,
      );
    }

    add('actions', 'add-repo', 'Add repository', 'codicon-add', () => {
      setSidebarView('repos');
      setFocusRepoInput(true);
      close();
    });
    add(
      'actions',
      'refresh',
      'Refresh repository data',
      'codicon-refresh',
      () => {
        if (activeRepoId != null) {
          void import('../../lib/syncRepository').then((m) => m.syncActiveRepository(activeRepoId));
        }
        close();
      },
      'hotspots, ownership, rules',
    );
    add('actions', 'ai-panel', 'Toggle AI panel', 'codicon-comment-discussion', () => {
      toggleBottomPanel();
      close();
    });
    add('actions', 'inspector', 'Toggle inspector', 'codicon-layout-sidebar-right', () => {
      toggleInspector();
      close();
    });

    for (const v of VIEW_COMMANDS) {
      add('views', `view-${v.view}`, `Go to ${v.label}`, v.icon, () => {
        setSidebarView(v.view);
        close();
      });
    }

    return list;
    // eslint-disable-next-line react-hooks/exhaustive-deps -- recentVersion busts recent section
  }, [
    repositories,
    activeRepoId,
    selectedNodePath,
    bottomPanelOpen,
    setActiveRepo,
    setSidebarView,
    setFocusRepoInput,
    toggleBottomPanel,
    toggleInspector,
    recentVersion,
  ]);

  const allFiles = useMemo((): PaletteItem[] => {
    const list: PaletteItem[] = [];
    for (const f of clusterLayer?.files ?? []) {
      list.push({
        id: `file-${f.id}`,
        label: f.path,
        sub: 'Open in map',
        group: 'files',
        icon: 'codicon-file',
        searchText: f.path,
        action: () => {
          setSelectedNode(f.id, f.path);
          setSidebarView('map');
          close();
        },
      });
    }
    for (const h of hotspots) {
      list.push({
        id: `hot-${h.fileId}`,
        label: h.path,
        sub: `${h.riskLevel} hotspot`,
        group: 'files',
        icon: 'codicon-warning',
        searchText: `${h.path} hotspot ${h.riskLevel}`,
        action: () => {
          setSelectedNode(String(h.fileId), h.path);
          setSidebarView('map');
          close();
        },
      });
    }
    return list;
  }, [clusterLayer, hotspots, setSelectedNode, setSidebarView]);

  const unifiedItems = useMemo(() => {
    const query = q.trim();
    if (!query) {
      const recent = loadRecentCommands();
      const byId = new Map(allCommands.map((c) => [c.id, c]));
      const recentItems: PaletteItem[] = [];
      const recentIds = new Set<string>();
      for (const r of recent) {
        const cmd = byId.get(r.id);
        if (cmd) {
          recentItems.push({ ...cmd, group: 'recent' });
          recentIds.add(r.id);
        }
      }
      const rest = allCommands.filter((c) => !recentIds.has(c.id));
      const files = allFiles.slice(0, 12);
      return [...recentItems, ...rest, ...files].slice(0, 40);
    }
    const cmds = filterWithFuse(allCommands, query);
    const files = filterWithFuse(allFiles, query);
    return [...cmds, ...files].slice(0, 40);
  }, [q, allCommands, allFiles, recentVersion]);

  const fileItems = useMemo(() => filterWithFuse(allFiles, q).slice(0, 32), [q, allFiles]);

  const items = mode === 'files' ? fileItems : unifiedItems;

  useEffect(() => {
    if (!open) setQ('');
  }, [open]);

  if (!open) return null;

  const placeholder =
    mode === 'files'
      ? 'Search files by path…'
      : 'Search files, repositories, commands, AI…';

  return (
    <Command.Dialog
      open={open}
      onOpenChange={(v) => !v && close()}
      label={mode === 'files' ? 'Quick open' : 'Command palette'}
      shouldFilter={false}
      className="command-palette-overlay"
    >
      <div className="command-palette" onClick={(e) => e.stopPropagation()}>
        <div className="command-palette__input-row">
          <i className="codicon codicon-search command-palette__input-icon" aria-hidden />
          <Command.Input
            value={q}
            onValueChange={setQ}
            placeholder={placeholder}
            aria-label="Palette search"
            autoComplete="off"
          />
        </div>
        <Command.List className="command-palette__list">
          <Command.Empty className="command-palette__empty">No matches</Command.Empty>
          <PaletteGroups
            items={items}
            renderLabel={mode === 'files' ? (item) => basename(item.label) : undefined}
          />
        </Command.List>
        <footer className="command-palette__footer">
          <span>
            <kbd>↑↓</kbd> navigate
          </span>
          <span>
            <kbd>↵</kbd> run
          </span>
          <span>
            <kbd>Esc</kbd> close
          </span>
          <span className="command-palette__footer-hint">
            {modKey}K palette · {modKey}P files · {modKey}⇧P palette
          </span>
        </footer>
      </div>
    </Command.Dialog>
  );
}

/** Title bar entry — opens unified command palette. */
export function CommandPaletteTrigger({ onOpen }: { onOpen: () => void }) {
  const modKey = useModKey();

  return (
    <button
      type="button"
      className="command-trigger"
      onClick={onOpen}
      aria-label="Open command palette"
      title={`Command palette (${modKey}K)`}
    >
      <i className="codicon codicon-search" aria-hidden />
      <span className="command-trigger__label">Search commands…</span>
      <kbd className="command-trigger__kbd">{modKey}K</kbd>
    </button>
  );
}
