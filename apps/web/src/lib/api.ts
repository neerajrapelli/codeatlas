import type {
  ArchitectureSearchHit,
  ArchitectureRule,
  BlastRadiusResult,
  BoundaryViolationRow,
  ClusterLayer,
  DecisionRecord,
  GraphFileDetail,
  HotspotEntry,
  IngestionStatusPayload,
  MaintainerInfluence,
  ModuleIntelligence,
  OnboardingPlan,
  OwnershipSummary,
  PRInsight,
  Repository,
  RuleViolation,
  TeamRow,
  TimelineEntry,
} from '../types';
import { getApiBase } from '../apiBase';
import { apiFetch, apiJson, apiPostJson } from './apiFetch';
import { authQueryString, jsonHeaders } from './authToken';

export const api = {
  health: async (): Promise<{ status: string; service?: string }> => {
    const res = await apiFetch('/health');
    if (!res.ok) throw new Error(`health ${res.status}`);
    return res.json() as Promise<{ status: string; service?: string }>;
  },

  issueToken: async (body: { subject: string; tenant_id?: string }, bootstrapSecret: string) => {
    const res = await apiFetch('/auth/token', {
      method: 'POST',
      headers: { ...jsonHeaders(), 'X-Bootstrap-Secret': bootstrapSecret },
      body: JSON.stringify(body),
    });
    if (!res.ok) {
      const err = (await res.json().catch(() => ({}))) as { error?: string };
      throw new Error(err.error ?? `auth ${String(res.status)}`);
    }
    return res.json() as Promise<{ token: string; expires_in: number }>;
  },

  listRepositories: async (): Promise<Repository[]> => {
    const json = await apiJson<{ repositories?: Repository[] }>('/repositories');
    return Array.isArray(json.repositories) ? json.repositories : [];
  },

  getAuthProviders: async (): Promise<
    { provider: string; connected: boolean; tokenType?: string }[]
  > => {
    const json = await apiJson<{ providers?: { provider: string; connected: boolean }[] }>(
      '/auth/providers',
    );
    return Array.isArray(json.providers) ? json.providers : [];
  },

  connectProviderOAuth: async (provider: 'github' | 'gitlab' | 'bitbucket') => {
    const json = await apiPostJson<{ authorizeUrl: string }>(`/auth/${provider}/login`, {});
    if (json.authorizeUrl) {
      window.open(json.authorizeUrl, '_blank', 'noopener,noreferrer');
    }
    return json;
  },

  saveProviderToken: async (
    provider: 'github' | 'gitlab' | 'bitbucket',
    token: string,
    tokenType?: 'pat' | 'app_password',
  ) => {
    return apiPostJson(`/auth/${provider}/token`, { token, tokenType });
  },

  listRemoteRepositories: async (
    provider: 'github' | 'gitlab' | 'bitbucket',
    page = 1,
  ): Promise<
    {
      id: string;
      fullName: string;
      cloneUrl: string;
      defaultBranch: string;
      private: boolean;
    }[]
  > => {
    const json = await apiJson<{
      repositories?: {
        id: string;
        fullName: string;
        cloneUrl: string;
        defaultBranch: string;
        private: boolean;
      }[];
    }>(`/auth/${provider}/repositories?page=${String(page)}`);
    return Array.isArray(json.repositories) ? json.repositories : [];
  },

  uploadZipRepository: async (file: File, displayName?: string): Promise<Repository> => {
    const form = new FormData();
    form.append('file', file);
    form.append('sourceType', 'zip');
    if (displayName) form.append('displayName', displayName);
    const res = await apiFetch('/repos/upload-zip', { method: 'POST', body: form });
    if (!res.ok) {
      const err = (await res.json().catch(() => ({}))) as { error?: string };
      throw new Error(err.error ?? `upload ${String(res.status)}`);
    }
    const json = (await res.json()) as { repository?: Repository } | Repository;
    if (json && typeof json === 'object' && 'repository' in json && json.repository) {
      return json.repository;
    }
    return json as Repository;
  },

  getIngestionJob: async (jobId: string) => {
    return apiJson<{
      jobId: string;
      repositoryId: string;
      status: string;
      progress?: unknown;
      error?: string;
    }>(`/ingestion/jobs/${jobId}`);
  },

  syncRepository: async (repositoryId: number) => {
    return apiPostJson('/repos/sync', { repositoryId });
  },

  addRepository: async (body: {
    sourceType: string;
    sourceUrl?: string;
    branch?: string;
    displayName?: string;
    providerTokenId?: string;
    externalRepoId?: string;
    externalRepoFullName?: string;
  }): Promise<Repository> => {
    const res = await apiFetch('/repositories', {
      method: 'POST',
      headers: jsonHeaders(),
      body: JSON.stringify(body),
    });
    if (!res.ok) {
      const err = (await res.json().catch(() => ({}))) as { error?: string };
      throw new Error(err.error ?? `ingest ${String(res.status)}`);
    }
    const json = (await res.json()) as { repository?: Repository } | Repository;
    if (json && typeof json === 'object' && 'repository' in json && json.repository) {
      return json.repository;
    }
    return json as Repository;
  },

  deleteRepository: async (id: number): Promise<void> => {
    const res = await apiFetch(`/repositories/${String(id)}`, { method: 'DELETE' });
    if (!res.ok) {
      const err = (await res.json().catch(() => ({}))) as { error?: string };
      throw new Error(err.error ?? `delete repository ${String(res.status)}`);
    }
  },

  reindexRepository: async (id: number) => {
    return apiJson(`/repositories/${String(id)}/reindex`, { method: 'POST' });
  },

  getClusters: async (repositoryId: number, prefix: string): Promise<ClusterLayer> => {
    const q = new URLSearchParams({
      repositoryId: String(repositoryId),
      prefix,
    });
    return apiJson<ClusterLayer>(`/graph/clusters?${q.toString()}`);
  },

  getGraphFile: async (repositoryId: number, fileId: string): Promise<GraphFileDetail> => {
    const q = new URLSearchParams({
      repositoryId: String(repositoryId),
      fileId,
    });
    return apiJson<GraphFileDetail>(`/graph/file?${q.toString()}`);
  },

  getIngestionStatus: async (repositoryId: number): Promise<IngestionStatusPayload> => {
    return apiJson<IngestionStatusPayload>(`/repositories/${String(repositoryId)}/ingestion/status`);
  },

  ingestionStreamUrl: (repositoryId: number) => {
    const q = authQueryString();
    const suffix = q ? `?${q}` : '';
    return `${getApiBase()}/repositories/${String(repositoryId)}/ingestion/stream${suffix}`;
  },

  getHotspots: async (repositoryId: number, limit = 30): Promise<HotspotEntry[]> => {
    try {
      const json = await apiJson<{ hotspots?: HotspotEntry[] }>(
        `/repositories/${String(repositoryId)}/hotspots?limit=${String(limit)}`,
      );
      return Array.isArray(json.hotspots) ? json.hotspots : [];
    } catch {
      return [];
    }
  },

  getBlastRadius: async (
    repositoryId: number,
    filePath: string,
    opts?: { symbol?: string; depth?: number },
  ): Promise<BlastRadiusResult> => {
    const params = new URLSearchParams({ file_path: filePath });
    if (opts?.symbol) params.set('symbol', opts.symbol);
    if (opts?.depth != null) params.set('depth', String(opts.depth));
    return apiJson<BlastRadiusResult>(
      `/repositories/${String(repositoryId)}/blast-radius?${params.toString()}`,
    );
  },

  getOwnership: async (repositoryId: number, fileId?: string): Promise<OwnershipSummary[]> => {
    const q = fileId ? `?fileId=${encodeURIComponent(fileId)}` : '';
    try {
      const json = await apiJson<{ ownership?: OwnershipSummary[] }>(
        `/repositories/${String(repositoryId)}/ownership${q}`,
      );
      return Array.isArray(json.ownership) ? json.ownership : [];
    } catch {
      return [];
    }
  },

  listRules: async (repositoryId: number): Promise<ArchitectureRule[]> => {
    try {
      const json = await apiJson<{ rules?: ArchitectureRule[] }>(
        `/repositories/${String(repositoryId)}/rules`,
      );
      return Array.isArray(json.rules) ? json.rules : [];
    } catch {
      return [];
    }
  },

  createRule: async (
    repositoryId: number,
    body: {
      name: string;
      ruleType: string;
      sourcePattern: string;
      targetPattern: string;
      description?: string;
      severity?: string;
      enabled?: boolean;
    },
  ): Promise<ArchitectureRule> => {
    const json = await apiPostJson<{ rule: ArchitectureRule }>(
      `/repositories/${String(repositoryId)}/rules`,
      body,
    );
    return json.rule;
  },

  deleteRule: async (repositoryId: number, ruleId: string) => {
    await apiFetch(`/repositories/${String(repositoryId)}/rules/${encodeURIComponent(ruleId)}`, {
      method: 'DELETE',
    });
  },

  getViolations: async (repositoryId: number): Promise<RuleViolation[]> => {
    try {
      const json = await apiJson<{ violations?: RuleViolation[] }>(
        `/repositories/${String(repositoryId)}/violations`,
      );
      return Array.isArray(json.violations) ? json.violations : [];
    } catch {
      return [];
    }
  },

  validateRules: async (repositoryId: number): Promise<RuleViolation[]> => {
    const json = await apiPostJson<{ violations?: RuleViolation[] }>(
      `/repositories/${String(repositoryId)}/rules/validate`,
      {},
    );
    return Array.isArray(json.violations) ? json.violations : [];
  },

  listTeams: async (repositoryId: number): Promise<TeamRow[]> => {
    try {
      const json = await apiJson<{ teams?: TeamRow[] }>(`/repositories/${String(repositoryId)}/teams`);
      return Array.isArray(json.teams) ? json.teams : [];
    } catch {
      return [];
    }
  },

  listTeamFiles: async (repositoryId: number, teamId: string): Promise<string[]> => {
    try {
      const json = await apiJson<{ files?: string[] }>(
        `/repositories/${String(repositoryId)}/teams/${encodeURIComponent(teamId)}/files`,
      );
      return Array.isArray(json.files) ? json.files : [];
    } catch {
      return [];
    }
  },

  getBoundaryViolations: async (repositoryId: number): Promise<BoundaryViolationRow[]> => {
    try {
      const json = await apiJson<{ violations?: BoundaryViolationRow[] }>(
        `/repositories/${String(repositoryId)}/boundary-violations`,
      );
      return Array.isArray(json.violations) ? json.violations : [];
    } catch {
      return [];
    }
  },

  getOwnershipGaps: async (repositoryId: number) => {
    try {
      const json = await apiJson<{ gaps?: { filePath: string; message?: string }[] }>(
        `/repositories/${String(repositoryId)}/ownership-gaps`,
      );
      return Array.isArray(json.gaps) ? json.gaps : [];
    } catch {
      return [];
    }
  },

  generateOnboardingPlan: async (
    repositoryId: number,
    body: { role: string; team?: string; experience_level: string },
  ): Promise<OnboardingPlan> => {
    return apiPostJson<OnboardingPlan>(
      `/repositories/${String(repositoryId)}/onboarding-plan`,
      {
        role: body.role,
        experience_level: body.experience_level,
        focus_area: body.team,
      },
    );
  },

  getC4Diagram: async (repositoryId: number, level: string, scope?: string) => {
    const params = new URLSearchParams({ level });
    if (scope) params.set('scope', scope);
    return apiJson<{ mermaid: string; level: string }>(
      `/repositories/${String(repositoryId)}/docs/c4-diagram?${params.toString()}`,
    );
  },

  getArchADRs: async (repositoryId: number) => {
    try {
      const json = await apiJson<{ adrs?: { title: string; body: string }[] }>(
        `/repositories/${String(repositoryId)}/docs/adrs`,
      );
      return Array.isArray(json.adrs) ? json.adrs : [];
    } catch {
      return [];
    }
  },

  getArchitectureTimeline: async (repositoryId: number): Promise<TimelineEntry[]> => {
    const json = await apiJson<{ items?: TimelineEntry[] }>(
      `/repositories/${String(repositoryId)}/architecture/timeline`,
    );
    return Array.isArray(json.items) ? json.items : [];
  },

  getArchitectureDecisions: async (repositoryId: number): Promise<DecisionRecord[]> => {
    const json = await apiJson<{ items?: DecisionRecord[] }>(
      `/repositories/${String(repositoryId)}/architecture/decisions`,
    );
    return Array.isArray(json.items) ? json.items : [];
  },

  getModuleIntelligence: async (
    repositoryId: number,
    modulePath: string,
  ): Promise<ModuleIntelligence> => {
    const q = new URLSearchParams({ path: modulePath });
    return apiJson<ModuleIntelligence>(
      `/repositories/${String(repositoryId)}/architecture/module-intel?${q.toString()}`,
    );
  },

  getPRInsights: async (repositoryId: number): Promise<PRInsight[]> => {
    const json = await apiJson<{ items?: PRInsight[] }>(
      `/repositories/${String(repositoryId)}/architecture/pr-insights`,
    );
    return Array.isArray(json.items) ? json.items : [];
  },

  getMaintainerInfluence: async (repositoryId: number): Promise<MaintainerInfluence[]> => {
    const json = await apiJson<{ items?: MaintainerInfluence[] }>(
      `/repositories/${String(repositoryId)}/architecture/maintainer-influence`,
    );
    return Array.isArray(json.items) ? json.items : [];
  },

  searchArchitecture: async (
    repositoryId: number,
    query: string,
  ): Promise<ArchitectureSearchHit[]> => {
    const q = new URLSearchParams({ q: query });
    const json = await apiJson<{ items?: ArchitectureSearchHit[] }>(
      `/repositories/${String(repositoryId)}/architecture/search?${q.toString()}`,
    );
    return Array.isArray(json.items) ? json.items : [];
  },

  exportDocs: async (repositoryId: number) => {
    const res = await apiFetch(
      `/repositories/${String(repositoryId)}/docs/export?format=markdown`,
    );
    if (!res.ok) throw new Error(`export ${String(res.status)}`);
    return res.text();
  },

  chatStream: async (
    repositoryId: number,
    body: { query: string; provider?: string; model?: string },
    onEvent: (ev: {
      type: string;
      token?: string;
      content?: string;
      relatedFiles?: Array<{ fileId: number; path: string; reason: string }>;
      validation?: { paths?: Record<string, boolean>; rules?: Record<string, boolean> };
      error?: string;
    }) => void,
  ) => {
    const res = await apiFetch('/ai/chat', {
      method: 'POST',
      headers: jsonHeaders(),
      body: JSON.stringify({ repositoryId, ...body, stream: true }),
    });
    if (!res.ok && !(res.headers.get('content-type') ?? '').includes('text/event-stream')) {
      const err = (await res.json().catch(() => ({}))) as { error?: string };
      throw new Error(err.error ?? `chat ${String(res.status)}`);
    }
    if (!res.body) throw new Error('no stream body');
    const reader = res.body.getReader();
    const decoder = new TextDecoder();
    let buffer = '';
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      buffer += decoder.decode(value, { stream: true });
      let sep: number;
      while ((sep = buffer.indexOf('\n\n')) >= 0) {
        const block = buffer.slice(0, sep);
        buffer = buffer.slice(sep + 2);
        const line = block.split('\n').find((l) => l.startsWith('data: '));
        if (!line) continue;
        try {
          onEvent(JSON.parse(line.slice(6)) as Parameters<typeof onEvent>[0]);
        } catch {
          /* skip */
        }
      }
    }
  },
};
