import { useQuery } from '@tanstack/react-query';

import { api } from '../../lib/api';
import { queryKeys } from '../../lib/queryKeys';
import { useStore } from '../../store';

async function fetchAndApplyRepoSync(repoId: number) {
  useStore.getState().setSocioLoading(true);
  const [hotspots, ownership, rules, violations, ingestion] = await Promise.all([
    api.getHotspots(repoId, 30),
    api.getOwnership(repoId),
    api.listRules(repoId),
    api.getViolations(repoId),
    api.getIngestionStatus(repoId).catch(() => null),
  ]);
  const store = useStore.getState();
  store.setHotspots(hotspots);
  store.setOwnershipRows(ownership);
  store.setArchitectureRules(rules);
  store.setRuleViolations(violations);
  if (ingestion) store.setIngestionStatus(ingestion);
  store.setSocioLoading(false);
  return { hotspots, ownership, rules, violations, ingestion };
}

export function useRepositorySyncQuery(repositoryId: number | null) {
  return useQuery({
    queryKey: repositoryId != null ? queryKeys.repoSync(repositoryId) : ['repoSync', 'none'],
    queryFn: () => {
      if (repositoryId == null) throw new Error('no repository');
      return fetchAndApplyRepoSync(repositoryId);
    },
    enabled: repositoryId != null,
    refetchInterval: 15_000,
  });
}
