import { useEffect } from 'react';

import { applyTheme } from '../lib/theme';
import { checkApiHealth, syncActiveRepository } from '../lib/syncRepository';
import { api } from '../lib/api';
import { useStore } from '../store';

/** App-wide API health, theme, repositories poll, and per-repo data sync. */
export function useBackend() {
  const activeRepoId = useStore((s) => s.activeRepoId);
  const setRepositories = useStore((s) => s.setRepositories);
  const setActiveRepo = useStore((s) => s.setActiveRepo);
  const setSidebarView = useStore((s) => s.setSidebarView);
  const theme = useStore((s) => s.theme);
  const setTourStep = useStore((s) => s.setTourStep);

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
    void checkApiHealth();
    const t = setInterval(() => void checkApiHealth(), 30_000);
    return () => clearInterval(t);
  }, []);

  useEffect(() => {
    let cancelled = false;
    const load = async () => {
      try {
        const repos = await api.listRepositories();
        if (cancelled) return;
        setRepositories(repos);
        if (repos.length === 0) {
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
        if (current == null || !repos.some((r) => r.id === current)) {
          const ready = repos.find((r) => r.status === 'ready');
          setActiveRepo(ready?.id ?? repos[0]?.id ?? null);
        }
      } catch {
        if (!cancelled) useStore.getState().setApiStatus('offline');
      }
    };
    void load();
    const t = setInterval(() => {
      void api.listRepositories().then(setRepositories).catch(() => undefined);
    }, 8000);
    return () => {
      cancelled = true;
      clearInterval(t);
    };
  }, [setRepositories, setActiveRepo, setSidebarView, setTourStep]);

  useEffect(() => {
    if (activeRepoId == null) return;
    void syncActiveRepository(activeRepoId);
    const t = setInterval(() => void syncActiveRepository(activeRepoId), 15_000);
    return () => clearInterval(t);
  }, [activeRepoId]);
}
