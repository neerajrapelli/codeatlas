import { useEffect, useMemo, useRef, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';

import { api } from '../../lib/api';
import { queryKeys } from '../../lib/queryKeys';
import { findExistingRepository } from '../../lib/repoMatch';
import { dedupeRepositories } from '../../lib/repositories';
import { useStore } from '../../store';
import type { Repository } from '../../types';
import { EmptyState } from '../ui/EmptyState';
import { StatusDot } from '../ui/StatusDot';

type VCSProvider = 'github' | 'gitlab' | 'bitbucket';

const PROVIDERS: { id: VCSProvider; label: string }[] = [
  { id: 'github', label: 'GitHub' },
  { id: 'gitlab', label: 'GitLab' },
  { id: 'bitbucket', label: 'Bitbucket' },
];

export function RepositoriesView() {
  const repositories = useStore((s) => s.repositories);
  const activeRepoId = useStore((s) => s.activeRepoId);
  const setActiveRepo = useStore((s) => s.setActiveRepo);
  const setRepositories = useStore((s) => s.setRepositories);
  const pushToast = useStore((s) => s.pushToast);
  const focusRepoInput = useStore((s) => s.focusRepoInput);
  const setFocusRepoInput = useStore((s) => s.setFocusRepoInput);
  const queryClient = useQueryClient();
  const urlRef = useRef<HTMLInputElement>(null);
  const zipRef = useRef<HTMLInputElement>(null);
  const [url, setUrl] = useState('');
  const [branch, setBranch] = useState('main');
  const [sourceType, setSourceType] = useState<VCSProvider>('github');
  const [adding, setAdding] = useState(false);
  const [uploadingZip, setUploadingZip] = useState(false);
  const [deletingId, setDeletingId] = useState<number | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [connections, setConnections] = useState<Record<string, boolean>>({});
  const [patProvider, setPatProvider] = useState<VCSProvider | ''>('');
  const [patValue, setPatValue] = useState('');
  const [remoteRepos, setRemoteRepos] = useState<
    { id: string; fullName: string; cloneUrl: string; defaultBranch: string }[]
  >([]);
  const [loadingRemote, setLoadingRemote] = useState(false);

  const repoList = useMemo(() => dedupeRepositories(repositories), [repositories]);

  useEffect(() => {
    if (focusRepoInput) {
      urlRef.current?.focus();
      setFocusRepoInput(false);
    }
  }, [focusRepoInput, setFocusRepoInput]);

  useEffect(() => {
    void (async () => {
      try {
        const rows = await api.getAuthProviders();
        const map: Record<string, boolean> = {};
        for (const r of rows) {
          map[r.provider] = r.connected;
        }
        setConnections(map);
      } catch {
        /* auth optional in dev */
      }
    })();
  }, []);

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const connected = params.get('vcs_connected');
    const vcsError = params.get('vcs_error');
    if (connected) {
      pushToast(`Connected ${connected}`, 'success');
      params.delete('vcs_connected');
      window.history.replaceState({}, '', `${window.location.pathname}?${params}`);
      void refreshConnections();
    }
    if (vcsError) {
      pushToast(`Provider connection failed: ${vcsError}`, 'error');
      params.delete('vcs_error');
      window.history.replaceState({}, '', `${window.location.pathname}?${params}`);
    }
  }, [pushToast]);

  const refreshConnections = async () => {
    const rows = await api.getAuthProviders();
    const map: Record<string, boolean> = {};
    for (const r of rows) {
      map[r.provider] = r.connected;
    }
    setConnections(map);
  };

  const statusKind = (s: string): 'queued' | 'running' | 'ready' | 'failed' => {
    if (s === 'ready') return 'ready';
    if (s === 'failed') return 'failed';
    if (s === 'queued') return 'queued';
    return 'running';
  };

  const syncRepositories = async () => {
    const repos = dedupeRepositories(await api.listRepositories());
    setRepositories(repos);
    return repos;
  };

  const confirmDuplicate = (existing: Repository, action: string): boolean => {
    return window.confirm(
      `"${existing.name}" is already in CodeAtlas (status: ${existing.status}).\n\nDo you want to ${action} another copy?`,
    );
  };

  const addRepo = async () => {
    const trimmed = url.trim();
    if (!trimmed) {
      setError('Enter a Git repository URL.');
      return;
    }
    const branchName = branch.trim() || 'main';
    const existing = findExistingRepository(repoList, sourceType, trimmed, branchName);
    if (existing && !confirmDuplicate(existing, 'add')) {
      return;
    }
    setAdding(true);
    setError(null);
    try {
      const repo = await api.addRepository({
        sourceType,
        sourceUrl: trimmed,
        branch: branchName,
      });
      await syncRepositories();
      setActiveRepo(repo.id);
      setUrl('');
      void queryClient.invalidateQueries({ queryKey: queryKeys.repositories });
      pushToast(`Ingestion queued for ${repo.name}`, 'success');
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Failed to add repository';
      setError(msg);
      pushToast(msg, 'error');
    } finally {
      setAdding(false);
    }
  };

  const importRemote = async (remote: {
    id: string;
    fullName: string;
    cloneUrl: string;
    defaultBranch: string;
  }) => {
    const branchName = remote.defaultBranch || 'main';
    const existing = findExistingRepository(repoList, sourceType, remote.cloneUrl, branchName);
    if (existing && !confirmDuplicate(existing, 'import')) {
      return;
    }
    setAdding(true);
    setError(null);
    try {
      const repo = await api.addRepository({
        sourceType,
        sourceUrl: remote.cloneUrl,
        branch: branchName,
        displayName: remote.fullName,
        externalRepoId: remote.id,
        externalRepoFullName: remote.fullName,
      });
      await syncRepositories();
      setActiveRepo(repo.id);
      pushToast(`Queued ${repo.name}`, 'success');
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Import failed';
      pushToast(msg, 'error');
    } finally {
      setAdding(false);
    }
  };

  const loadRemoteRepos = async (provider: VCSProvider) => {
    setLoadingRemote(true);
    setSourceType(provider);
    try {
      const repos = await api.listRemoteRepositories(provider);
      setRemoteRepos(repos);
    } catch (e) {
      pushToast(e instanceof Error ? e.message : 'Failed to list repositories', 'error');
      setRemoteRepos([]);
    } finally {
      setLoadingRemote(false);
    }
  };

  const savePat = async () => {
    if (!patProvider || !patValue.trim()) return;
    try {
      await api.saveProviderToken(
        patProvider,
        patValue.trim(),
        patProvider === 'bitbucket' ? 'app_password' : 'pat',
      );
      setPatValue('');
      await refreshConnections();
      pushToast(`Saved ${patProvider} token`, 'success');
    } catch (e) {
      pushToast(e instanceof Error ? e.message : 'Token save failed', 'error');
    }
  };

  const onZipSelected = async (file: File | undefined) => {
    if (!file) return;
    if (!file.name.endsWith('.zip')) {
      pushToast('Only .zip archives are supported', 'error');
      return;
    }
    setUploadingZip(true);
    try {
      const repo = await api.uploadZipRepository(file, file.name.replace(/\.zip$/i, ''));
      await syncRepositories();
      setActiveRepo(repo.id);
      pushToast(`ZIP upload queued: ${repo.name}`, 'success');
    } catch (e) {
      pushToast(e instanceof Error ? e.message : 'ZIP upload failed', 'error');
    } finally {
      setUploadingZip(false);
      if (zipRef.current) zipRef.current.value = '';
    }
  };

  const refreshList = async () => {
    try {
      await syncRepositories();
      void queryClient.invalidateQueries({ queryKey: queryKeys.repositories });
      pushToast('Repository list updated', 'info');
    } catch (e) {
      pushToast(e instanceof Error ? e.message : 'Refresh failed', 'error');
    }
  };

  const deleteRepo = async (repoId: number, name: string) => {
    const ok = window.confirm(
      'Are you sure you want to delete this repository? This action cannot be undone.',
    );
    if (!ok) return;
    setDeletingId(repoId);
    try {
      await api.deleteRepository(repoId);
      const remaining = dedupeRepositories(
        useStore.getState().repositories.filter((r) => r.id !== repoId),
      );
      setRepositories(remaining);
      queryClient.setQueryData<Repository[]>(queryKeys.repositories, (prev) =>
        dedupeRepositories((prev ?? useStore.getState().repositories).filter((r) => r.id !== repoId)),
      );
      await queryClient.refetchQueries({ queryKey: queryKeys.repositories });
      void queryClient.removeQueries({ queryKey: queryKeys.repoSync(repoId) });
      void queryClient.removeQueries({ queryKey: ['graph', 'clusters', repoId] });
      if (activeRepoId === repoId) {
        const next = remaining.find((r) => r.status === 'ready') ?? remaining[0];
        setActiveRepo(next?.id ?? null);
        useStore.getState().setGraphPrefix('');
        useStore.getState().setSelectedNode(null, null);
        useStore.getState().setClusterLayer(null);
      }
      pushToast(`Deleted ${name}`, 'success');
    } catch (e) {
      pushToast(e instanceof Error ? e.message : 'Delete failed', 'error');
    } finally {
      setDeletingId(null);
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

      <div className="repo-providers">
        <p className="field-label">Connect provider</p>
        <div className="repo-providers__row">
          {PROVIDERS.map((p) => (
            <button
              key={p.id}
              type="button"
              className={`btn-secondary btn-secondary--compact ${connections[p.id] ? 'repo-providers__connected' : ''}`}
              title={connections[p.id] ? `${p.label} connected` : `Connect ${p.label}`}
              onClick={() => void api.connectProviderOAuth(p.id).then(() => refreshConnections())}
            >
              {p.label}
              {connections[p.id] ? ' ✓' : ''}
            </button>
          ))}
        </div>
        <div className="repo-providers__row">
          <select
            className="field-input"
            value={patProvider}
            onChange={(e) => setPatProvider(e.target.value as VCSProvider | '')}
            aria-label="Token provider"
          >
            <option value="">PAT / app password…</option>
            {PROVIDERS.map((p) => (
              <option key={p.id} value={p.id}>
                {p.label}
              </option>
            ))}
          </select>
          <input
            className="field-input"
            type="password"
            placeholder="Token"
            value={patValue}
            onChange={(e) => setPatValue(e.target.value)}
            disabled={!patProvider}
          />
          <button type="button" className="btn-secondary" disabled={!patProvider || !patValue} onClick={() => void savePat()}>
            Save
          </button>
        </div>
        <div className="repo-providers__row">
          {PROVIDERS.map((p) => (
            <button
              key={`browse-${p.id}`}
              type="button"
              className="btn-secondary btn-secondary--compact"
              disabled={!connections[p.id] || loadingRemote}
              onClick={() => void loadRemoteRepos(p.id)}
            >
              Browse {p.label}
            </button>
          ))}
        </div>
      </div>

      {remoteRepos.length > 0 ? (
        <ul className="repo-remote-list">
          {remoteRepos.slice(0, 12).map((r) => (
            <li key={r.id}>
              <button
                type="button"
                className="repo-remote-list__item"
                disabled={adding}
                onClick={() => void importRemote(r)}
              >
                {r.fullName}
              </button>
            </li>
          ))}
        </ul>
      ) : null}

      <form
        className="repo-add-form"
        onSubmit={(e) => {
          e.preventDefault();
          void addRepo();
        }}
      >
        <label className="field-label" htmlFor="repo-source">
          Source
        </label>
        <select
          id="repo-source"
          className="field-input"
          value={sourceType}
          onChange={(e) => setSourceType(e.target.value as VCSProvider)}
          disabled={adding}
        >
          {PROVIDERS.map((p) => (
            <option key={p.id} value={p.id}>
              {p.label}
            </option>
          ))}
        </select>
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

      <div className="repo-zip-upload">
        <label className="field-label" htmlFor="repo-zip">
          Upload ZIP repository
        </label>
        <input
          id="repo-zip"
          ref={zipRef}
          type="file"
          accept=".zip,application/zip"
          disabled={uploadingZip}
          onChange={(e) => void onZipSelected(e.target.files?.[0])}
        />
        {uploadingZip ? <p className="field-hint">Uploading and queuing…</p> : null}
      </div>

      {repoList.length === 0 ? (
        <EmptyState
          icon="codicon-folder-opened"
          title="No repositories yet"
          description="Connect a provider, paste a Git URL, or upload a ZIP. Indexing usually takes one to three minutes."
        />
      ) : (
        <ul className="repo-list" role="list">
          {repoList.map((repo) => (
            <li key={repo.id} className="repo-list__item">
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
              <button
                type="button"
                className="repo-row__delete btn-icon"
                title={`Delete ${repo.name}`}
                aria-label={`Delete ${repo.name}`}
                disabled={deletingId === repo.id}
                onMouseDown={(e) => e.stopPropagation()}
                onClick={(e) => {
                  e.preventDefault();
                  e.stopPropagation();
                  void deleteRepo(repo.id, repo.name);
                }}
              >
                <i className="codicon codicon-trash" />
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
