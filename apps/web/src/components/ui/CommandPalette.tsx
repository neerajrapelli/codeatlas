import Fuse, { type FuseResult, type FuseOptionKey } from 'fuse.js';
import { useEffect, useMemo, useRef, useState, type ReactNode } from 'react';
import { createPortal } from 'react-dom';

import { basename } from '../../lib/fileType';
import { loadRecentCommands, pushRecentCommand } from '../../lib/recentCommands';
import type { SidebarView } from '../../types';
import { useStore } from '../../store';

type CommandGroup = 'recent' | 'actions' | 'views' | 'files';

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
  actions: 'Actions',
  views: 'Views',
  files: 'Files & hotspots',
};

const VIEW_COMMANDS: Array<{ view: SidebarView; label: string; icon: string }> = [
  { view: 'repos', label: 'Repositories', icon: 'codicon-folder-opened' },
  { view: 'map', label: 'Architecture map', icon: 'codicon-type-hierarchy-sub' },
  { view: 'hotspots', label: 'Hotspots', icon: 'codicon-warning' },
  { view: 'ownership', label: 'Ownership', icon: 'codicon-person' },
  { view: 'teams', label: 'Teams', icon: 'codicon-group-by-ref-type' },
  { view: 'docs', label: 'Living docs', icon: 'codicon-book' },
  { view: 'onboarding', label: 'Onboarding', icon: 'codicon-mortar-board' },
  { view: 'drift', label: 'Architecture drift', icon: 'codicon-shield' },
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

function PaletteShell({
  ariaLabel,
  placeholder,
  footerHints,
  q,
  setQ,
  items,
  idx,
  setIdx,
  onClose,
  renderLabel,
}: {
  ariaLabel: string;
  placeholder: string;
  footerHints: ReactNode;
  q: string;
  setQ: (v: string) => void;
  items: PaletteItem[];
  idx: number;
  setIdx: (fn: (i: number) => number) => void;
  onClose: () => void;
  renderLabel?: (item: PaletteItem) => string;
}) {
  const inputRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const t = window.setTimeout(() => inputRef.current?.focus(), 0);
    return () => window.clearTimeout(t);
  }, []);

  useEffect(() => {
    const el = listRef.current?.querySelector('.command-palette__item--active');
    el?.scrollIntoView({ block: 'nearest' });
  }, [idx]);

  const select = (i: number) => {
    const item = items[i];
    if (item) item.action();
  };

  let lastGroup: CommandGroup | null = null;
  const rows: ReactNode[] = [];
  items.forEach((item, i) => {
    if (item.group !== lastGroup) {
      lastGroup = item.group;
      rows.push(
        <div key={`g-${item.group}-${i}`} className="command-palette__group">
          {GROUP_LABELS[item.group]}
        </div>,
      );
    }
    const label = renderLabel ? renderLabel(item) : item.label;
    rows.push(
      <button
        key={item.id}
        type="button"
        className={`command-palette__item ${i === idx ? 'command-palette__item--active' : ''}`}
        onClick={() => select(i)}
        onMouseEnter={() => setIdx(() => i)}
      >
        <span className="command-palette__item-main">
          <i className={`codicon ${item.icon}`} aria-hidden />
          <span className="command-palette__label" title={item.label}>
            {label}
          </span>
        </span>
        {item.sub ? <span className="command-palette__sub">{item.sub}</span> : null}
      </button>,
    );
  });

  return (
    <div className="command-palette-overlay" onClick={onClose} role="presentation">
      <div
        className="command-palette"
        onClick={(e) => e.stopPropagation()}
        role="dialog"
        aria-modal="true"
        aria-label={ariaLabel}
        onKeyDown={(e) => {
          if (e.key === 'Escape') {
            e.stopPropagation();
            onClose();
          }
          if (e.key === 'ArrowDown') {
            e.preventDefault();
            setIdx((i) => Math.min(items.length - 1, i + 1));
          }
          if (e.key === 'ArrowUp') {
            e.preventDefault();
            setIdx((i) => Math.max(0, i - 1));
          }
          if (e.key === 'Enter') {
            e.preventDefault();
            select(idx);
          }
        }}
      >
        <div className="command-palette__input-row">
          <i className="codicon codicon-search command-palette__input-icon" aria-hidden />
          <input
            ref={inputRef}
            value={q}
            onChange={(e) => setQ(e.target.value)}
            placeholder={placeholder}
            aria-label="Palette search"
            autoComplete="off"
            spellCheck={false}
          />
        </div>
        <div className="command-palette__list" ref={listRef}>
          {items.length === 0 ? (
            <p className="command-palette__empty">No matches</p>
          ) : (
            rows
          )}
        </div>
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
          {footerHints}
        </footer>
      </div>
    </div>
  );
}

export function CommandPalette() {
  const open = useStore((s) => s.commandPaletteOpen);
  const mode = useStore((s) => s.paletteMode);
  const setOpen = useStore((s) => s.setCommandPaletteOpen);
  const clusterLayer = useStore((s) => s.clusterLayer);
  const hotspots = useStore((s) => s.hotspots);
  const setSelectedNode = useStore((s) => s.setSelectedNode);
  const setSidebarView = useStore((s) => s.setSidebarView);
  const setFocusRepoInput = useStore((s) => s.setFocusRepoInput);
  const toggleBottomPanel = useStore((s) => s.toggleBottomPanel);
  const toggleInspector = useStore((s) => s.toggleInspector);
  const activeRepoId = useStore((s) => s.activeRepoId);
  const [q, setQ] = useState('');
  const [idx, setIdx] = useState(0);
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
    activeRepoId,
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

  const commandItems = useMemo(() => {
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
      return [...recentItems, ...rest].slice(0, 24);
    }
    return filterWithFuse(allCommands, query).slice(0, 24);
  }, [q, allCommands, recentVersion]);

  const fileItems = useMemo(() => {
    const filtered = filterWithFuse(allFiles, q);
    return filtered.slice(0, 24);
  }, [q, allFiles]);

  useEffect(() => {
    if (!open) {
      setQ('');
      setIdx(0);
    }
  }, [open]);

  useEffect(() => {
    setIdx(0);
  }, [q, mode]);

  if (!open) return null;

  if (mode === 'files') {
    return createPortal(
      <PaletteShell
        ariaLabel="Quick open"
        placeholder="Search files by path…"
        footerHints={
          <span className="command-palette__footer-hint">
            {modKey}P files · {modKey}⇧P commands
          </span>
        }
        q={q}
        setQ={setQ}
        items={fileItems}
        idx={idx}
        setIdx={setIdx}
        onClose={close}
        renderLabel={(item) => basename(item.label)}
      />,
      document.body,
    );
  }

  return createPortal(
    <PaletteShell
      ariaLabel="Command palette"
      placeholder="Search commands and views…"
      footerHints={
        <span className="command-palette__footer-hint">
          {modKey}K · {modKey}P files · {modKey}⇧P commands
        </span>
      }
      q={q}
      setQ={setQ}
      items={commandItems}
      idx={idx}
      setIdx={setIdx}
      onClose={close}
    />,
    document.body,
  );
}

/** Title bar entry — opens command palette (not quick open). */
export function CommandPaletteTrigger({ onOpen }: { onOpen: () => void }) {
  const modKey = useModKey();

  return (
    <button
      type="button"
      className="command-trigger"
      onClick={onOpen}
      aria-label="Open command palette"
      title={`Command palette (${modKey}K, ${modKey}⇧P)`}
    >
      <i className="codicon codicon-search" aria-hidden />
      <span className="command-trigger__label">Search commands…</span>
      <kbd className="command-trigger__kbd">{modKey}K</kbd>
    </button>
  );
}
