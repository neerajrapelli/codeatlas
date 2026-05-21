import { getApiBase } from '../apiBase';
import type {
  ArchitectureRule,
  BlastRadiusResult,
  ClusterLayer,
  GraphFileDetail,
  HotspotEntry,
  IngestionStatusPayload,
  OwnershipSummary,
  Repository,
  RuleViolation,
} from '../types';

const base = () => getApiBase();

export const api = {
  listRepositories: async (): Promise<Repository[]> => {
    const res = await fetch(`${base()}/repositories`);
    if (!res.ok) throw new Error(`repositories ${res.status}`);
    const json = (await res.json()) as { repositories?: Repository[] };
    return Array.isArray(json.repositories) ? json.repositories : [];
  },

  addRepository: async (body: {
    sourceType: string;
    sourceUrl?: string;
    branch?: string;
    displayName?: string;
  }): Promise<Repository> => {
    const res = await fetch(`${base()}/repositories`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    if (!res.ok) {
      const err = (await res.json().catch(() => ({}))) as { error?: string };
      throw new Error(err.error ?? `ingest ${res.status}`);
    }
    const json = (await res.json()) as { repository?: Repository } | Repository;
    if (json && typeof json === 'object' && 'repository' in json && json.repository) {
      return json.repository;
    }
    return json as Repository;
  },

  deleteRepository: async (id: number) => {
    const res = await fetch(`${base()}/repositories/${String(id)}`, { method: 'DELETE' });
    if (!res.ok) throw new Error(`delete ${res.status}`);
    return res.json();
  },

  reindexRepository: async (id: number) => {
    const res = await fetch(`${base()}/repositories/${String(id)}/reindex`, { method: 'POST' });
    if (!res.ok) throw new Error(`reindex ${res.status}`);
    return res.json();
  },

  getClusters: async (repositoryId: number, prefix: string): Promise<ClusterLayer> => {
    const url = `${base()}/graph/clusters?repositoryId=${String(repositoryId)}&prefix=${encodeURIComponent(prefix)}`;
    const res = await fetch(url);
    if (!res.ok) throw new Error(`clusters ${res.status}`);
    return res.json() as Promise<ClusterLayer>;
  },

  getGraphFile: async (repositoryId: number, fileId: string): Promise<GraphFileDetail> => {
    const res = await fetch(
      `${base()}/graph/file?repositoryId=${String(repositoryId)}&fileId=${encodeURIComponent(fileId)}`,
    );
    if (!res.ok) throw new Error(`file ${res.status}`);
    const raw = (await res.json()) as GraphFileDetail;
    return raw;
  },

  getIngestionStatus: async (repositoryId: number): Promise<IngestionStatusPayload> => {
    const res = await fetch(`${base()}/repositories/${String(repositoryId)}/ingestion/status`);
    if (!res.ok) throw new Error(`ingestion status ${res.status}`);
    return res.json() as Promise<IngestionStatusPayload>;
  },

  ingestionStreamUrl: (repositoryId: number) =>
    `${base()}/repositories/${String(repositoryId)}/ingestion/stream`,

  getHotspots: async (repositoryId: number, limit = 30): Promise<HotspotEntry[]> => {
    const res = await fetch(`${base()}/repositories/${String(repositoryId)}/hotspots?limit=${String(limit)}`);
    if (!res.ok) return [];
    const json = (await res.json()) as { hotspots?: HotspotEntry[] };
    return Array.isArray(json.hotspots) ? json.hotspots : [];
  },

  getBlastRadius: async (
    repositoryId: number,
    filePath: string,
    opts?: { symbol?: string; depth?: number },
  ): Promise<BlastRadiusResult> => {
    const params = new URLSearchParams({ file_path: filePath });
    if (opts?.symbol) params.set('symbol', opts.symbol);
    if (opts?.depth != null) params.set('depth', String(opts.depth));
    const res = await fetch(
      `${base()}/repositories/${String(repositoryId)}/blast-radius?${params.toString()}`,
    );
    if (!res.ok) {
      const err = (await res.json().catch(() => ({}))) as { error?: string };
      throw new Error(err.error ?? `blast-radius ${res.status}`);
    }
    return res.json() as Promise<BlastRadiusResult>;
  },

  getOwnership: async (repositoryId: number, fileId?: string): Promise<OwnershipSummary[]> => {
    const q = fileId ? `?fileId=${encodeURIComponent(fileId)}` : '';
    const res = await fetch(`${base()}/repositories/${String(repositoryId)}/ownership${q}`);
    if (!res.ok) return [];
    const json = (await res.json()) as { ownership?: OwnershipSummary[] };
    return Array.isArray(json.ownership) ? json.ownership : [];
  },

  listRules: async (repositoryId: number): Promise<ArchitectureRule[]> => {
    const res = await fetch(`${base()}/repositories/${String(repositoryId)}/rules`);
    if (!res.ok) return [];
    const json = (await res.json()) as { rules?: ArchitectureRule[] };
    return Array.isArray(json.rules) ? json.rules : [];
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
    const res = await fetch(`${base()}/repositories/${String(repositoryId)}/rules`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    if (!res.ok) throw new Error(`create rule ${res.status}`);
    const json = (await res.json()) as { rule: ArchitectureRule };
    return json.rule;
  },

  deleteRule: async (repositoryId: number, ruleId: string) => {
    const res = await fetch(
      `${base()}/repositories/${String(repositoryId)}/rules/${encodeURIComponent(ruleId)}`,
      { method: 'DELETE' },
    );
    if (!res.ok) throw new Error(`delete rule ${res.status}`);
  },

  getViolations: async (repositoryId: number): Promise<RuleViolation[]> => {
    const res = await fetch(`${base()}/repositories/${String(repositoryId)}/violations`);
    if (!res.ok) return [];
    const json = (await res.json()) as { violations?: RuleViolation[] };
    return Array.isArray(json.violations) ? json.violations : [];
  },

  validateRules: async (repositoryId: number): Promise<RuleViolation[]> => {
    const res = await fetch(`${base()}/repositories/${String(repositoryId)}/rules/validate`, {
      method: 'POST',
    });
    if (!res.ok) throw new Error(`validate ${res.status}`);
    const json = (await res.json()) as { violations?: RuleViolation[] };
    return Array.isArray(json.violations) ? json.violations : [];
  },

  chatStream: async (
    repositoryId: number,
    body: { query: string; provider?: string; model?: string },
    onEvent: (ev: {
      type: string;
      token?: string;
      relatedFiles?: Array<{ fileId: number; path: string; reason: string }>;
      error?: string;
    }) => void,
  ) => {
    const res = await fetch(`${base()}/ai/chat`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ repositoryId, ...body, stream: true }),
    });
    if (!res.ok && !(res.headers.get('content-type') ?? '').includes('text/event-stream')) {
      const err = (await res.json().catch(() => ({}))) as { error?: string };
      throw new Error(err.error ?? `chat ${res.status}`);
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
