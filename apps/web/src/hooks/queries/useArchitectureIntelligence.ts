import { useQuery } from '@tanstack/react-query';

import { api } from '../../lib/api';
import { queryKeys } from '../../lib/queryKeys';

export function useArchitectureTimelineQuery(repositoryId: number | null) {
  return useQuery({
    queryKey:
      repositoryId != null ? queryKeys.architectureTimeline(repositoryId) : ['architecture', 'timeline', 'none'],
    queryFn: () => {
      if (repositoryId == null) throw new Error('no repository');
      return api.getArchitectureTimeline(repositoryId);
    },
    enabled: repositoryId != null,
  });
}

export function useArchitectureDecisionsQuery(repositoryId: number | null) {
  return useQuery({
    queryKey:
      repositoryId != null
        ? queryKeys.architectureDecisions(repositoryId)
        : ['architecture', 'decisions', 'none'],
    queryFn: () => {
      if (repositoryId == null) throw new Error('no repository');
      return api.getArchitectureDecisions(repositoryId);
    },
    enabled: repositoryId != null,
  });
}

export function useArchitecturePRInsightsQuery(repositoryId: number | null) {
  return useQuery({
    queryKey:
      repositoryId != null
        ? queryKeys.architecturePRInsights(repositoryId)
        : ['architecture', 'pr-insights', 'none'],
    queryFn: () => {
      if (repositoryId == null) throw new Error('no repository');
      return api.getPRInsights(repositoryId);
    },
    enabled: repositoryId != null,
  });
}

export function useMaintainerInfluenceQuery(repositoryId: number | null) {
  return useQuery({
    queryKey:
      repositoryId != null
        ? queryKeys.architectureMaintainerInfluence(repositoryId)
        : ['architecture', 'maintainer-influence', 'none'],
    queryFn: () => {
      if (repositoryId == null) throw new Error('no repository');
      return api.getMaintainerInfluence(repositoryId);
    },
    enabled: repositoryId != null,
  });
}

export function useArchitectureModuleIntelQuery(repositoryId: number | null, modulePath: string) {
  return useQuery({
    queryKey:
      repositoryId != null
        ? queryKeys.architectureModuleIntel(repositoryId, modulePath)
        : ['architecture', 'module-intel', 'none'],
    queryFn: () => {
      if (repositoryId == null) throw new Error('no repository');
      return api.getModuleIntelligence(repositoryId, modulePath);
    },
    enabled: repositoryId != null && modulePath.trim().length > 0,
  });
}

export function useArchitectureSearchQuery(repositoryId: number | null, query: string) {
  const trimmed = query.trim();
  return useQuery({
    queryKey:
      repositoryId != null
        ? queryKeys.architectureSearch(repositoryId, trimmed)
        : ['architecture', 'search', 'none'],
    queryFn: () => {
      if (repositoryId == null) throw new Error('no repository');
      return api.searchArchitecture(repositoryId, trimmed);
    },
    enabled: repositoryId != null && trimmed.length > 1,
  });
}
