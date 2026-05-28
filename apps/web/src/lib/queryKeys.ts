export const queryKeys = {
  health: ['health'] as const,
  repositories: ['repositories'] as const,
  clusterLayer: (repoId: number, prefix: string) => ['graph', 'clusters', repoId, prefix] as const,
  violations: (repoId: number) => ['violations', repoId] as const,
  repoSync: (repoId: number) => ['repoSync', repoId] as const,
  architectureTimeline: (repoId: number) => ['architecture', 'timeline', repoId] as const,
  architectureDecisions: (repoId: number) => ['architecture', 'decisions', repoId] as const,
  architecturePRInsights: (repoId: number) => ['architecture', 'pr-insights', repoId] as const,
  architectureMaintainerInfluence: (repoId: number) =>
    ['architecture', 'maintainer-influence', repoId] as const,
  architectureModuleIntel: (repoId: number, modulePath: string) =>
    ['architecture', 'module-intel', repoId, modulePath] as const,
  architectureSearch: (repoId: number, query: string) =>
    ['architecture', 'search', repoId, query] as const,
  mcpManifest: ['mcp', 'manifest'] as const,
  mcpLogs: (limit: number) => ['mcp', 'logs', limit] as const,
};
