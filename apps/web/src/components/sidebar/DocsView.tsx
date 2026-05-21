import { useEffect, useRef, useState } from 'react';

import { api } from '../../lib/api';
import { useStore } from '../../store';

type Tab = 'context' | 'container' | 'component' | 'adrs';

export function DocsView() {
  const activeRepoId = useStore((s) => s.activeRepoId);
  const [tab, setTab] = useState<Tab>('container');
  const [mermaid, setMermaid] = useState('');
  const [adrs, setAdrs] = useState<{ title: string; body: string }[]>([]);
  const diagramRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (activeRepoId == null) return;
    if (tab === 'adrs') {
      void api.getArchADRs(activeRepoId).then(setAdrs).catch(() => setAdrs([]));
      return;
    }
    const level = tab === 'context' ? 'context' : tab === 'component' ? 'component' : 'container';
    void api
      .getC4Diagram(activeRepoId, level)
      .then(async (res) => {
        setMermaid(res.mermaid);
        if (diagramRef.current) {
          diagramRef.current.textContent = res.mermaid;
          try {
            const m = await import('mermaid');
            m.default.initialize({ startOnLoad: false, theme: 'dark' });
            await m.default.run({ nodes: [diagramRef.current] });
          } catch {
            diagramRef.current.textContent = res.mermaid;
          }
        }
      })
      .catch(() => setMermaid(''));
  }, [activeRepoId, tab]);

  const exportMd = async () => {
    if (activeRepoId == null) return;
    const md = await api.exportDocs(activeRepoId);
    const blob = new Blob([md], { type: 'text/markdown' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'codeatlas-architecture.md';
    a.click();
    URL.revokeObjectURL(url);
  };

  if (activeRepoId == null) {
    return <p className="empty-state">Select a repository.</p>;
  }

  return (
    <div className="sidebar-view">
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h3 className="sidebar-section-title">ARCHITECTURE DOCS</h3>
        <button type="button" onClick={() => void exportMd()}>
          Export ↓
        </button>
      </div>
      <div style={{ display: 'flex', gap: 6, marginBottom: 8 }}>
        {(['context', 'container', 'component', 'adrs'] as Tab[]).map((t) => (
          <button
            key={t}
            type="button"
            className={tab === t ? 'active' : ''}
            onClick={() => setTab(t)}
          >
            {t}
          </button>
        ))}
      </div>
      {tab === 'adrs' ? (
        adrs.length === 0 ? (
          <p className="empty-state">No architectural decision signals yet.</p>
        ) : (
          adrs.map((a, i) => (
            <div key={i} className="sidebar-card">
              <strong>{a.title}</strong>
              <pre style={{ whiteSpace: 'pre-wrap', fontSize: 11 }}>{a.body}</pre>
            </div>
          ))
        )
      ) : (
        <div ref={diagramRef} className="mermaid" style={{ fontSize: 11, overflow: 'auto' }} />
      )}
    </div>
  );
}
