import { api } from './api';
import { useStore } from '../store';

/** Loads socio + drift data for the active repository from the API. */
export async function syncActiveRepository(repoId: number): Promise<void> {
  const store = useStore.getState();
  store.setSocioLoading(true);
  try {
    const [hotspots, ownership, rules, violations, ingestion] = await Promise.all([
      api.getHotspots(repoId, 30),
      api.getOwnership(repoId),
      api.listRules(repoId),
      api.getViolations(repoId),
      api.getIngestionStatus(repoId).catch(() => null),
    ]);
    useStore.getState().setHotspots(hotspots);
    useStore.getState().setOwnershipRows(ownership);
    useStore.getState().setArchitectureRules(rules);
    useStore.getState().setRuleViolations(violations);
    if (ingestion) useStore.getState().setIngestionStatus(ingestion);
  } finally {
    useStore.getState().setSocioLoading(false);
  }
}

export async function checkApiHealth(): Promise<void> {
  useStore.getState().setApiStatus('checking');
  try {
    const h = await api.health();
    useStore.getState().setApiStatus(h.status === 'ok' ? 'online' : 'degraded');
  } catch {
    useStore.getState().setApiStatus('offline');
  }
}
