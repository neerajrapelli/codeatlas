import { useEffect, useRef, useState } from 'react';

import { api } from '../../lib/api';
import { useStore } from '../../store';
import { EmptyState } from '../ui/EmptyState';
import { ViewSkeleton } from '../ui/ViewSkeleton';

type Tab = 'context' | 'container' | 'component' | 'adrs';

const TABS: Array<{ id: Tab; label: string }> = [
  { id: 'context', label: 'Context' },
  { id: 'container', label: 'Container' },
  { id: 'component', label: 'Component' },
  { id: 'adrs', label: 'ADRs' },
];

export function DocsView() {
  const activeRepoId = useStore((s) => s.activeRepoId);
  const [tab, setTab] = useState<Tab>('container');
  const [adrs, setAdrs] = useState<{ title: string; body: string }[]>([]);
  const [loading, setLoading] = useState(false);
  const [diagramError, setDiagramError] = useState<string | null>(null);
  const diagramRef = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (activeRepoId == null) return;

    if (tab === 'adrs') {
      setLoading(true);
      setDiagramError(null);
      void api
        .getArchADRs(activeRepoId)
        .then(setAdrs)
        .catch(() => setAdrs([]))
        .finally(() => setLoading(false));
      return;
    }

    const level = tab;
    setLoading(true);
    setDiagramError(null);
    void api
      .getC4Diagram(activeRepoId, level)
      .then(async (res) => {
        const el = diagramRef.current;
        if (!el) return;
        el.removeAttribute('data-processed');
        el.innerHTML = '';
        const src = (res.mermaid ?? '').trim();
        if (!src) {
          setDiagramError('No diagram data for this level yet.');
          return;
        }
        try {
          const m = await import('mermaid');
          m.default.initialize({
            startOnLoad: false,
            theme: 'dark',
            securityLevel: 'strict',
          });
          el.classList.add('mermaid');
          el.textContent = src;
          await m.default.run({ nodes: [el] });
        } catch {
          setDiagramError('Could not render diagram. Showing source.');
          el.classList.remove('mermaid');
          el.textContent = src;
        }
      })
      .catch(() => setDiagramError('Failed to load C4 diagram.'))
      .finally(() => setLoading(false));
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
    return (
      <div className="sidebar-view">
        <h3 className="sidebar-section-title">ARCHITECTURE DOCS</h3>
        <EmptyState
          icon="codicon-book"
          title="No repository"
          description="Select a repository to view C4 diagrams and ADRs."
        />
      </div>
    );
  }

  return (
    <div className="sidebar-view docs-view">
      <div className="docs-view__header">
        <h3 className="sidebar-section-title">ARCHITECTURE DOCS</h3>
        <button type="button" className="btn-ghost" onClick={() => void exportMd()}>
          Export ↓
        </button>
      </div>
      <div className="docs-view__tabs" role="tablist">
        {TABS.map((t) => (
          <button
            key={t.id}
            type="button"
            role="tab"
            aria-selected={tab === t.id}
            className={`docs-view__tab ${tab === t.id ? 'docs-view__tab--active' : ''}`}
            onClick={() => setTab(t.id)}
          >
            {t.label}
          </button>
        ))}
      </div>
      {loading ? <ViewSkeleton rows={6} /> : null}
      {!loading && tab === 'adrs' ? (
        adrs.length === 0 ? (
          <EmptyState
            icon="codicon-law"
            title="No ADRs yet"
            description="Architectural decision signals appear after indexing and drift analysis."
          />
        ) : (
          adrs.map((a, i) => (
            <article key={`${a.title}-${String(i)}`} className="sidebar-card adr-card">
              <strong>{a.title}</strong>
              <pre className="adr-card__body">{a.body}</pre>
            </article>
          ))
        )
      ) : null}
      {!loading && tab !== 'adrs' ? (
        <>
          {diagramError ? <p className="sidebar-hint">{diagramError}</p> : null}
          <div ref={diagramRef} className="docs-view__diagram" />
        </>
      ) : null}
    </div>
  );
}
