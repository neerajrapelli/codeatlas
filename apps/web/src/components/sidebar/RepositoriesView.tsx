import { useState } from 'react';

import { api } from '../../lib/api';
import { useStore } from '../../store';
import { StatusDot } from '../ui/StatusDot';

export function RepositoriesView() {
  const repositories = useStore((s) => s.repositories);
  const activeRepoId = useStore((s) => s.activeRepoId);
  const setActiveRepo = useStore((s) => s.setActiveRepo);
  const setRepositories = useStore((s) => s.setRepositories);
  const [url, setUrl] = useState('');
  const [adding, setAdding] = useState(false);

  const statusKind = (s: string): 'queued' | 'running' | 'ready' | 'failed' => {
    if (s === 'ready') return 'ready';
    if (s === 'failed') return 'failed';
    if (s === 'queued') return 'queued';
    return 'running';
  };

  const addRepo = async () => {
    if (!url.trim()) return;
    setAdding(true);
    try {
      await api.addRepository({ sourceType: 'github', sourceUrl: url.trim(), branch: 'main' });
      const repos = await api.listRepositories();
      setRepositories(repos);
      if (repos[0]) setActiveRepo(repos[0].id);
      setUrl('');
    } catch (e) {
      alert(e instanceof Error ? e.message : 'Failed to add');
    } finally {
      setAdding(false);
    }
  };

  return (
    <div className="sidebar-view">
      <h3 className="sidebar-section-title">
        REPOSITORIES
        <button type="button" className="btn-icon" title="Add" onClick={() => void addRepo()} disabled={adding}>
          <i className="codicon codicon-add" />
        </button>
      </h3>
      <input
        className="ai-input"
        style={{ marginBottom: 8, width: '100%' }}
        placeholder="GitHub URL…"
        value={url}
        onChange={(e) => setUrl(e.target.value)}
        onKeyDown={(e) => e.key === 'Enter' && void addRepo()}
      />
      {repositories.length === 0 ? (
        <p className="empty-state">No repositories. Add a GitHub URL above.</p>
      ) : null}
      {repositories.map((repo) => (
        <div
          key={repo.id}
          className={`repo-row ${activeRepoId === repo.id ? 'repo-row--active' : ''}`}
          onClick={() => setActiveRepo(repo.id)}
          onKeyDown={(e) => e.key === 'Enter' && setActiveRepo(repo.id)}
          role="button"
          tabIndex={0}
        >
          <div className="repo-row__name">
            <StatusDot status={statusKind(repo.status)} /> {repo.name}
          </div>
          <div className="repo-row__meta">
            {repo.sourceType.toUpperCase()} ·{' '}
            {repo.status === 'ready'
              ? `${String(repo.filesIndexed ?? '—')} files`
              : repo.status.replaceAll('_', ' ')}
          </div>
        </div>
      ))}
    </div>
  );
}
