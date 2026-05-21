import { useQuery } from '@tanstack/react-query';

import { apiJson } from '../../lib/apiFetch';
import { queryKeys } from '../../lib/queryKeys';

export function useMcpManifestQuery() {
  return useQuery({
    queryKey: queryKeys.mcpManifest,
    queryFn: async () => {
      const json = await apiJson<{ tools?: Array<{ name: string }> }>('/mcp/manifest');
      return (json.tools ?? []).map((t) => t.name);
    },
    refetchInterval: 3_000,
  });
}

export function useMcpLogsQuery(limit = 10) {
  return useQuery({
    queryKey: queryKeys.mcpLogs(limit),
    queryFn: async () => {
      const json = await apiJson<{
        calls?: Array<{ tool: string; at: string; ok: boolean; error?: string }>;
      }>(`/mcp/logs?limit=${String(limit)}`);
      return json.calls ?? [];
    },
    refetchInterval: 3_000,
  });
}
