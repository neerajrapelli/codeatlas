export const queryKeys = {
  health: ['health'] as const,
  repositories: ['repositories'] as const,
  clusterLayer: (repoId: number, prefix: string) => ['graph', 'clusters', repoId, prefix] as const,
  violations: (repoId: number) => ['violations', repoId] as const,
  repoSync: (repoId: number) => ['repoSync', repoId] as const,
  mcpManifest: ['mcp', 'manifest'] as const,
  mcpLogs: (limit: number) => ['mcp', 'logs', limit] as const,
};
