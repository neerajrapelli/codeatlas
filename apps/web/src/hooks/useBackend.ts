import { useEffect } from 'react';
import { useQueryClient } from '@tanstack/react-query';

import { dedupeRepositories } from '../lib/repositories';
import { applyTheme } from '../lib/theme';
import { queryKeys } from '../lib/queryKeys';
import { useStore } from '../store';
import { useHealthQuery, useRepositoriesQuery } from './queries/useRepositories';
import { useRepositorySyncQuery } from './queries/useRepositorySync';

/** App-wide API health, theme, repositories poll, and per-repo data sync. */
export function useBackend() {
  const activeRepoId = useStore((s) => s.activeRepoId);
  const setRepositories = useStore((s) => s.setRepositories);
  const setActiveRepo = useStore((s) => s.setActiveRepo);
  const setSidebarView = useStore((s) => s.setSidebarView);
  const theme = useStore((s) => s.theme);
  const setTourStep = useStore((s) => s.setTourStep);
  const queryClient = useQueryClient();

  const health = useHealthQuery();
  const repos = useRepositoriesQuery();
  useRepositorySyncQuery(activeRepoId);

  useEffect(() => {
    applyTheme(theme);
  }, [theme]);

  useEffect(() => {
    const mq = window.matchMedia('(prefers-color-scheme: dark)');
    const onChange = () => {
      if (useStore.getState().theme === 'system') {
        applyTheme('system');
      }
    };
    mq.addEventListener('change', onChange);
    return () => mq.removeEventListener('change', onChange);
  }, []);

  useEffect(() => {
    if (health.isLoading) {
      useStore.getState().setApiStatus('checking');
      return;
    }
    if (health.isError) {
      useStore.getState().setApiStatus('offline');
      return;
    }
    const status = health.data?.status;
    useStore.getState().setApiStatus(status === 'ok' ? 'online' : 'degraded');
  }, [health.isLoading, health.isError, health.data?.status]);

  useEffect(() => {
    if (!repos.data) return;
    const list = dedupeRepositories(repos.data);
    setRepositories(list);
    if (list.length === 0) {
      setSidebarView('repos');
      try {
        if (!localStorage.getItem('codeatlas-tour-done')) {
          setTourStep(0);
        }
      } catch {
        setTourStep(0);
      }
      return;
    }
    const current = useStore.getState().activeRepoId;
    if (current == null || !list.some((r) => r.id === current)) {
      const ready = list.find((r) => r.status === 'ready');
      setActiveRepo(ready?.id ?? list[0]?.id ?? null);
    }
  }, [repos.data, setRepositories, setActiveRepo, setSidebarView, setTourStep]);

  useEffect(() => {
    if (activeRepoId == null) return;
    return () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.repoSync(activeRepoId) });
    };
  }, [activeRepoId, queryClient]);
}
