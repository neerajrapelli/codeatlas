import { useQuery } from '@tanstack/react-query';

import { api } from '../../lib/api';
import { queryKeys } from '../../lib/queryKeys';

export function useRepositoriesQuery(enabled = true) {
  return useQuery({
    queryKey: queryKeys.repositories,
    queryFn: () => api.listRepositories(),
    enabled,
    refetchInterval: 8_000,
  });
}

export function useHealthQuery() {
  return useQuery({
    queryKey: queryKeys.health,
    queryFn: () => api.health(),
    refetchInterval: 30_000,
    retry: false,
  });
}
