import { useEffect, useRef, useState } from 'react';

import { api } from '../../lib/api';
import { useStore } from '../../store';
import { EmptyState } from '../ui/EmptyState';
import { StatusDot } from '../ui/StatusDot';

export function RepositoriesView() {
  const repositories = useStore((s) => s.repositories);
  const activeRepoId = useStore((s) => s.activeRepoId);
  const setActiveRepo = useStore((s) => s.setActiveRepo);
  const setRepositories = useStore((s) => s.setRepositories);
  const pushToast = useStore((s) => s.pushToast);
  const focusRepoInput = useStore((s) => s.focusRepoInput);
  const setFocusRepoInput = useStore((s) => s.setFocusRepoInput);
  const urlRef = useRef<HTMLInputElement>(null);
  const [url, setUrl] = useState('');
  const [branch, setBranch] = useState('main');
  const [adding, setAdding] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (focusRepoInput) {
      urlRef.current?.focus();
      setFocusRepoInput(false);
    }
  }, [focusRepoInput, setFocusRepoInput]);

  const statusKind = (s: string): 'queued' | 'running' | 'ready' | 'failed' => {
    if (s === 'ready') return 'ready';
    if (s === 'failed') return 'failed';
    if (s === 'queued') return 'queued';
    return 'running';
  };

  const addRepo = async () => {
    const trimmed = url.trim();
    if (!trimmed) {
      setError('Enter a Git repository URL.');
      return;
    }
    setAdding(true);
    setError(null);
    try {
      const repo = await api.addRepository({
        sourceType: 'github',
        sourceUrl: trimmed,
        branch: branch.trim() || 'main',
      });
      const repos = await api.listRepositories();
      setRepositories(repos);
      setActiveRepo(repo.id);
      setUrl('');
      pushToast(`Ingestion queued for ${repo.name}`, 'success');
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Failed to add repository';
      setError(msg);
      pushToast(msg, 'error');
    } finally {
      setAdding(false);
    }
  };

  const refreshList = async () => {
    try {
      const repos = await api.listRepositories();
      setRepositories(repos);
      pushToast('Repository list updated', 'info');
    } catch (e) {
      pushToast(e instanceof Error ? e.message : 'Refresh failed', 'error');
    }
  };

  return (
    <div className="sidebar-view">
      <h3 className="sidebar-section-title">
        REPOSITORIES
        <button type="button" className="btn-icon" title="Refresh list" onClick={() => void refreshList()}>
          <i className="codicon codicon-refresh" />
        </button>
      </h3>

      <form
        className="repo-add-form"
        onSubmit={(e) => {
          e.preventDefault();
          void addRepo();
        }}
      >
        <label className="field-label" htmlFor="repo-url">
          Git URL
        </label>
        <input
          id="repo-url"
          ref={urlRef}
          className="field-input"
          placeholder="https://github.com/org/repo"
          value={url}
          onChange={(e) => setUrl(e.target.value)}
          disabled={adding}
          autoComplete="off"
        />
        <label className="field-label" htmlFor="repo-branch">
          Branch
        </label>
        <input
          id="repo-branch"
          className="field-input"
          placeholder="main"
          value={branch}
          onChange={(e) => setBranch(e.target.value)}
          disabled={adding}
        />
        {error ? <p className="field-error">{error}</p> : null}
        <button type="submit" className="btn-primary btn-primary--block" disabled={adding}>
          {adding ? 'Adding…' : 'Add repository'}
        </button>
      </form>

      {repositories.length === 0 ? (
        <EmptyState
          icon="codicon-folder-opened"
          title="No repositories yet"
          description="Paste a public GitHub URL above. Indexing usually takes one to three minutes."
        />
      ) : (
        <ul className="repo-list" role="list">
          {repositories.map((repo) => (
            <li key={repo.id}>
              <button
                type="button"
                className={`repo-row ${activeRepoId === repo.id ? 'repo-row--active' : ''}`}
                onClick={() => setActiveRepo(repo.id)}
              >
                <div className="repo-row__name">
                  <StatusDot status={statusKind(repo.status)} />
                  <span className="repo-row__title">{repo.name}</span>
                </div>
                <div className="repo-row__meta">
                  {repo.sourceType.toUpperCase()} ·{' '}
                  {repo.status === 'ready'
                    ? `${String(repo.filesIndexed ?? '—')} files`
                    : repo.status.replaceAll('_', ' ')}
                </div>
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
