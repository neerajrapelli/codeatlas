import { useEffect, useMemo, useState } from 'react';

import { basename } from '../../lib/fileType';
import { useStore } from '../../store';

export function CommandPalette() {
  const open = useStore((s) => s.commandPaletteOpen);
  const setOpen = useStore((s) => s.setCommandPaletteOpen);
  const clusterLayer = useStore((s) => s.clusterLayer);
  const hotspots = useStore((s) => s.hotspots);
  const setSelectedNode = useStore((s) => s.setSelectedNode);
  const setSidebarView = useStore((s) => s.setSidebarView);
  const [q, setQ] = useState('');
  const [idx, setIdx] = useState(0);

  const items = useMemo(() => {
    const list: Array<{ label: string; sub?: string; action: () => void }> = [];
    const query = q.trim().toLowerCase();
    for (const f of clusterLayer?.files ?? []) {
      if (!query || f.path.toLowerCase().includes(query)) {
        list.push({
          label: f.path,
          action: () => {
            setSelectedNode(f.id, f.path);
            setSidebarView('map');
            setOpen(false);
          },
        });
      }
    }
    for (const h of hotspots) {
      if (!query || h.path.toLowerCase().includes(query)) {
        list.push({
          label: h.path,
          sub: `${h.riskLevel} risk`,
          action: () => {
            setSelectedNode(String(h.fileId), h.path);
            setOpen(false);
          },
        });
      }
    }
    if (!query) {
      list.push({
        label: 'Show all hotspots',
        action: () => {
          setSidebarView('hotspots');
          setOpen(false);
        },
      });
    }
    return list.slice(0, 12);
  }, [q, clusterLayer, hotspots, setSelectedNode, setSidebarView, setOpen]);

  useEffect(() => {
    if (!open) {
      setQ('');
      setIdx(0);
    }
  }, [open]);

  useEffect(() => {
    setIdx(0);
  }, [q]);

  if (!open) return null;

  const select = (i: number) => {
    const item = items[i];
    if (item) item.action();
  };

  return (
    <div
      className="command-palette-overlay"
      onClick={() => setOpen(false)}
      onKeyDown={(e) => {
        if (e.key === 'Escape') setOpen(false);
        if (e.key === 'ArrowDown') setIdx((i) => Math.min(items.length - 1, i + 1));
        if (e.key === 'ArrowUp') setIdx((i) => Math.max(0, i - 1));
        if (e.key === 'Enter') select(idx);
      }}
    >
      <div className="command-palette" onClick={(e) => e.stopPropagation()} role="dialog">
        <input
          autoFocus
          value={q}
          onChange={(e) => setQ(e.target.value)}
          placeholder="> Find file in architecture map…"
        />
        {items.map((item, i) => (
          <div
            key={`${item.label}-${String(i)}`}
            className={`command-palette__item ${i === idx ? 'command-palette__item--active' : ''}`}
            onClick={() => select(i)}
          >
            <span>{basename(item.label)}</span>
            {item.sub ? <span style={{ color: 'var(--text-muted)' }}>{item.sub}</span> : null}
          </div>
        ))}
      </div>
    </div>
  );
}
