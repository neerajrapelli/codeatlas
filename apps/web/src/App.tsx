import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { ReactFlowProvider } from 'reactflow';

import { getApiBase } from './apiBase';
import { HierarchyGraph } from './components/HierarchyGraph';

interface GraphSymbol {
  name: string;
  kind: string;
}

interface GraphFile {
  id: string;
  path: string;
  imports: string[];
  exports: string[];
  symbols: GraphSymbol[];
}

type SourceType = 'github' | 'gitlab' | 'bitbucket' | 'zip';

interface RepositoryRecord {
  id: number;
  name: string;
  sourceType: SourceType;
  sourceUrl: string;
  branch: string;
  workspacePath: string;
  status:
    | 'queued'
    | 'cloning'
    | 'extracting'
    | 'indexing'
    | 'parsing'
    | 'building_graph'
    | 'generating_embeddings'
    | 'ready'
    | 'failed';
  progressPercent?: number;
  filesIndexed?: number;
  symbolsIndexed?: number;
  edgesIndexed?: number;
  embeddingsIndexed?: number;
  errorDetails?: string;
  createdAt: string;
}

interface RepositoryProgress {
  repositoryId: number;
  stage: RepositoryRecord['status'];
  status: RepositoryRecord['status'];
  progressPercent: number;
  metrics: {
    filesIndexed: number;
    symbolsIndexed: number;
    edgesIndexed: number;
    embeddingsIndexed: number;
  };
  errorDetails?: string;
}

interface RelatedFile {
  fileId: number;
  path: string;
  reason: string;
}

interface AIChatResponse {
  answer: string;
  relatedFiles: RelatedFile[];
  provider?: string;
  model?: string;
  error?: string;
}

interface ChatMessage {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  relatedFiles: RelatedFile[];
}

const DEFAULT_MODEL_BY_PROVIDER: Record<string, string> = {
  local: 'local-default',
  openai: 'gpt-4o-mini',
  anthropic: 'claude-3-5-sonnet',
  gemini: 'gemini-1.5-flash',
  huggingface: 'mistral-7b-instruct',
  openrouter: 'openai/gpt-4o-mini',
};
const ENABLED_PROVIDERS = ['local', 'openai'] as const;

const FAV_STORAGE_KEY = 'codeatlas:fav-repos';

function loadFavoriteRepoIds(): Set<number> {
  try {
    const raw = localStorage.getItem(FAV_STORAGE_KEY);
    if (!raw) return new Set();
    const arr = JSON.parse(raw) as unknown;
    if (!Array.isArray(arr)) return new Set();
    return new Set(
      arr.filter((x): x is number => typeof x === 'number' && Number.isFinite(x) && x > 0),
    );
  } catch {
    return new Set();
  }
}

function persistFavoriteRepoIds(ids: Set<number>): void {
  try {
    localStorage.setItem(FAV_STORAGE_KEY, JSON.stringify([...ids]));
  } catch {
    // ignore quota / private mode
  }
}

function formatRepoDate(iso: string): string {
  try {
    return new Intl.DateTimeFormat(undefined, { dateStyle: 'medium' }).format(new Date(iso));
  } catch {
    return '';
  }
}

function pathBasename(p: string): string {
  const i = Math.max(p.lastIndexOf('/'), p.lastIndexOf('\\'));
  return i >= 0 ? p.slice(i + 1) : p;
}

const INDEX_PIPELINE: Array<{ stage: RepositoryRecord['status']; title: string; hint: string }> = [
  { stage: 'queued', title: 'Queued', hint: 'Waiting to start ingestion' },
  { stage: 'cloning', title: 'Clone repository', hint: 'Fetching files from your provider' },
  { stage: 'extracting', title: 'Extract workspace', hint: 'Preparing sources on disk' },
  { stage: 'indexing', title: 'Index workspace', hint: 'Walking files and building trees' },
  { stage: 'parsing', title: 'Parse sources', hint: 'Understanding modules and symbols' },
  { stage: 'building_graph', title: 'Build dependency graph', hint: 'Connecting imports and references' },
  { stage: 'generating_embeddings', title: 'Semantic embeddings', hint: 'Enhancing retrieval for AI' },
  { stage: 'failed', title: 'Ingestion failed', hint: 'Check logs or repository access' },
  { stage: 'ready', title: 'Ready', hint: 'Architecture map is available' },
];

async function fetchGraphFile(repositoryId: number, fileId: string): Promise<GraphFile> {
  const base = getApiBase();
  const res = await fetch(
    `${base}/graph/file?repositoryId=${String(repositoryId)}&fileId=${encodeURIComponent(fileId)}`,
  );
  if (!res.ok) {
    throw new Error(`Could not load file (${String(res.status)})`);
  }
  const raw = (await res.json()) as {
    id?: string;
    path?: string;
    imports?: unknown[];
    exports?: unknown[];
    symbols?: Array<{ name?: string; kind?: string }>;
  };
  const symbols: GraphSymbol[] = Array.isArray(raw.symbols)
    ? raw.symbols.map((s) => ({
        name: String(s.name ?? ''),
        kind: String(s.kind ?? 'symbol'),
      }))
    : [];
  return {
    id: String(raw.id ?? fileId),
    path: String(raw.path ?? ''),
    imports: Array.isArray(raw.imports) ? raw.imports.map(String) : [],
    exports: Array.isArray(raw.exports) ? raw.exports.map(String) : [],
    symbols,
  };
}

function sourceIcon(sourceType: SourceType): string {
  if (sourceType === 'github') return 'GH';
  if (sourceType === 'gitlab') return 'GL';
  if (sourceType === 'bitbucket') return 'BB';
  return 'ZIP';
}

function prettyStage(stage: RepositoryRecord['status']): string {
  return stage.replaceAll('_', ' ');
}

function GraphWorkspace() {
  const zipInputRef = useRef<HTMLInputElement | null>(null);
  const [activeRepoId, setActiveRepoId] = useState<number>(1);
  const [repositories, setRepositories] = useState<RepositoryRecord[]>([]);
  const [repoLoading, setRepoLoading] = useState(false);
  const [tab, setTab] = useState<SourceType>('github');
  const [sourceUrl, setSourceUrl] = useState('');
  const [branch, setBranch] = useState('');
  const [displayName, setDisplayName] = useState('');
  const [zipFile, setZipFile] = useState<File | null>(null);
  const [repoSubmitError, setRepoSubmitError] = useState<string | null>(null);
  const [repoSubmitting, setRepoSubmitting] = useState(false);
  const [selectedFileId, setSelectedFileId] = useState<string | null>(null);
  const [fileDetail, setFileDetail] = useState<GraphFile | null>(null);
  const [fileDetailLoading, setFileDetailLoading] = useState(false);
  const [fileDetailError, setFileDetailError] = useState<string | null>(null);
  const [focusGeneration, setFocusGeneration] = useState(0);
  const [focusPaths, setFocusPaths] = useState<string[]>([]);
  const [chatInput, setChatInput] = useState('');
  const [chatLoading, setChatLoading] = useState(false);
  const [chatMessages, setChatMessages] = useState<ChatMessage[]>([]);
  const [highlightedIds, setHighlightedIds] = useState<Set<string>>(new Set());
  const [activeProgress, setActiveProgress] = useState<RepositoryProgress | null>(null);
  const [aiProvider, setAIProvider] = useState('local');
  const [aiModel, setAIModel] = useState('local-default');
  const [workspaceQuery, setWorkspaceQuery] = useState('');
  const [favorites, setFavorites] = useState<Set<number>>(() => loadFavoriteRepoIds());
  const [pendingUndo, setPendingUndo] = useState<{
    sourceType: SourceType
    sourceUrl: string
    branch: string
    displayName: string
  } | null>(null);
  const [compactLayout, setCompactLayout] = useState(false);
  const [leftDrawerOpen, setLeftDrawerOpen] = useState(false);
  const [rightDrawerOpen, setRightDrawerOpen] = useState(false);
  /** Current folder prefix in the architecture map (synced from HierarchyGraph). */
  const [mapPrefix, setMapPrefix] = useState('');
  const workspaceSearchRef = useRef<HTMLInputElement | null>(null);

  const refreshRepositories = useCallback(async () => {
    try {
      const res = await fetch(`${getApiBase()}/repositories`);
      if (!res.ok) return;
      const json = (await res.json()) as { repositories: RepositoryRecord[] };
      if (!Array.isArray(json.repositories)) return;
      setRepositories(json.repositories);
      const active = json.repositories.find((repo) => repo.id === activeRepoId);
      const firstReady = json.repositories.find((repo) => repo.status === 'ready');
      if (!active) {
        if (firstReady) setActiveRepoId(firstReady.id);
        return;
      }
      // Keep the user's selected repository even while it is indexing—do not snap back to "first ready".
    } catch {
      // noop; keep last known list
    }
  }, [activeRepoId]);

  useEffect(() => {
    setRepoLoading(true);
    void refreshRepositories().finally(() => {
      setRepoLoading(false);
    });
    const timer = setInterval(() => {
      void refreshRepositories();
    }, 4000);
    return () => {
      clearInterval(timer);
    };
  }, [refreshRepositories]);

  useEffect(() => {
    const active = repositories.find((repo) => repo.id === activeRepoId);
    if (!active || active.status === 'ready') {
      setActiveProgress(null);
      return;
    }
    const run = async () => {
      try {
        const res = await fetch(`${getApiBase()}/repositories/${String(activeRepoId)}/progress`);
        if (!res.ok) return;
        const data = (await res.json()) as RepositoryProgress;
        setActiveProgress(data);
      } catch {
        // noop
      }
    };
    void run();
    const timer = setInterval(() => {
      void run();
    }, 1500);
    return () => clearInterval(timer);
  }, [activeRepoId, repositories]);

  useEffect(() => {
    setAIModel(DEFAULT_MODEL_BY_PROVIDER[aiProvider] ?? 'local-default');
  }, [aiProvider]);

  useEffect(() => {
    const mq = window.matchMedia('(max-width: 1100px)');
    const sync = () => {
      setCompactLayout(mq.matches);
      if (!mq.matches) {
        setLeftDrawerOpen(false);
        setRightDrawerOpen(false);
      }
    };
    sync();
    mq.addEventListener('change', sync);
    return () => mq.removeEventListener('change', sync);
  }, []);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        setLeftDrawerOpen(false);
        setRightDrawerOpen(false);
      }
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, []);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && (e.key === 'k' || e.key === 'K')) {
        e.preventDefault();
        workspaceSearchRef.current?.focus();
      }
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, []);

  const filteredRepositories = useMemo(() => {
    const q = workspaceQuery.trim().toLowerCase();
    if (!q) return repositories;
    return repositories.filter((r) => r.name.toLowerCase().includes(q));
  }, [repositories, workspaceQuery]);

  const sortedRepositories = useMemo(() => {
    const list = [...filteredRepositories];
    list.sort((a, b) => {
      const fa = favorites.has(a.id) ? 1 : 0;
      const fb = favorites.has(b.id) ? 1 : 0;
      if (fb !== fa) return fb - fa;
      return b.id - a.id;
    });
    return list;
  }, [filteredRepositories, favorites]);

  const toggleFavorite = useCallback((repoId: number, event: React.MouseEvent) => {
    event.stopPropagation();
    setFavorites((prev) => {
      const next = new Set(prev);
      if (next.has(repoId)) next.delete(repoId);
      else next.add(repoId);
      persistFavoriteRepoIds(next);
      return next;
    });
  }, []);

  const activeRepository = repositories.find((repo) => repo.id === activeRepoId) ?? null;
  const isGraphReady = activeRepository?.status === 'ready';
  const canShowGraph =
    !!activeRepository &&
    activeRepository.status !== 'failed' &&
    (isGraphReady || (activeRepository.filesIndexed ?? 0) > 0);
  const showIndexingAsidePanel =
    !!activeRepository &&
    !isGraphReady &&
    (activeRepository.status === 'failed' || (activeRepository.filesIndexed ?? 0) === 0);
  const showIndexingBanner =
    !!activeRepository &&
    !isGraphReady &&
    activeRepository.status !== 'failed' &&
    (activeRepository.filesIndexed ?? 0) > 0;

  useEffect(() => {
    setSelectedFileId(null);
    setFileDetail(null);
    setFileDetailError(null);
    setFocusPaths([]);
    setFocusGeneration(0);
    setHighlightedIds(new Set());
    setMapPrefix('');
  }, [activeRepoId]);

  useEffect(() => {
    if (!selectedFileId || !canShowGraph) {
      setFileDetail(null);
      setFileDetailLoading(false);
      setFileDetailError(null);
      return;
    }
    const ac = new AbortController();
    setFileDetailLoading(true);
    setFileDetailError(null);
    void (async () => {
      try {
        const detail = await fetchGraphFile(activeRepoId, selectedFileId);
        if (ac.signal.aborted) return;
        setFileDetail(detail);
      } catch (e) {
        if (ac.signal.aborted) return;
        setFileDetail(null);
        setFileDetailError(e instanceof Error ? e.message : 'Failed to load file');
      } finally {
        if (!ac.signal.aborted) setFileDetailLoading(false);
      }
    })();
    return () => ac.abort();
  }, [selectedFileId, activeRepoId, canShowGraph]);
  const suggestedPrompts = [
    'What breaks if we change auth? Highlight impacted modules.',
    'Where are the highest coupling hotspots in this map scope?',
    'Trace downstream effects from the selected file.',
    'Suggest clearer architectural boundaries for this repo.',
    'Which clusters own API vs UI responsibilities?',
  ];

  const buildGroundedQuery = useCallback(
    (userLine: string) => {
      const ctx: string[] = [];
      if (activeRepository) {
        ctx.push(`Repository: "${activeRepository.name}"`);
      }
      if (mapPrefix) {
        ctx.push(`Architecture map folder: "${mapPrefix}"`);
      }
      if (fileDetail?.path) {
        ctx.push(`Inspector file: "${fileDetail.path}"`);
      }
      if (ctx.length === 0) return userLine.trim();
      return `${userLine.trim()}\n\n---\n${ctx.join('\n')}`;
    },
    [activeRepository, mapPrefix, fileDetail?.path],
  );

  const submitChat = async () => {
    const query = chatInput.trim();
    if (query.length === 0 || chatLoading) return;
    const groundedQuery = buildGroundedQuery(query);

    const userMessage: ChatMessage = {
      id: `u-${String(Date.now())}`,
      role: 'user',
      content: query,
      relatedFiles: [],
    };
    setChatMessages((prev) => [...prev, userMessage]);
    setChatInput('');
    setChatLoading(true);

    const assistantId = `a-${String(Date.now())}`;

    const applyRelatedHighlights = (relatedFiles: RelatedFile[]) => {
      const nextHighlights = new Set(
        relatedFiles.map((file) => String(file.fileId)).filter((id) => id.length > 0),
      );
      setHighlightedIds(nextHighlights);
      const pathsFromRelated = relatedFiles
        .map((file) => file.path.trim())
        .filter((path) => path.length > 0);
      if (pathsFromRelated.length > 0) {
        setFocusPaths(pathsFromRelated);
        setFocusGeneration((g) => g + 1);
      }
    };

    try {
      if (!ENABLED_PROVIDERS.includes(aiProvider as (typeof ENABLED_PROVIDERS)[number])) {
        throw new Error(`${aiProvider} is not enabled yet in this build. Use local or openai.`);
      }
      const res = await fetch(`${getApiBase()}/ai/chat`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          repositoryId: activeRepoId,
          query: groundedQuery,
          provider: aiProvider,
          model: aiModel,
          stream: true,
        }),
      });

      const contentType = res.headers.get('content-type') ?? '';

      if (!res.ok && !contentType.includes('text/event-stream')) {
        let msg = `AI request failed (${String(res.status)})`;
        try {
          const errBody = (await res.json()) as { error?: string };
          if (errBody.error) msg = errBody.error;
        } catch {
          // ignore
        }
        throw new Error(msg);
      }

      if (contentType.includes('text/event-stream') && res.body) {
        const assistantMessage: ChatMessage = {
          id: assistantId,
          role: 'assistant',
          content: '',
          relatedFiles: [],
        };
        setChatMessages((prev) => [...prev, assistantMessage]);

        const reader = res.body.getReader();
        const decoder = new TextDecoder();
        let buffer = '';
        let related: RelatedFile[] = [];

        const pump = async (): Promise<void> => {
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
              let payload: {
                type?: string
                token?: string
                relatedFiles?: RelatedFile[]
                error?: string
              };
              try {
                payload = JSON.parse(line.slice(6)) as typeof payload;
              } catch {
                continue;
              }
              if (payload.type === 'meta' && Array.isArray(payload.relatedFiles)) {
                related = payload.relatedFiles;
                setChatMessages((prev) =>
                  prev.map((msg) =>
                    msg.id === assistantId ? { ...msg, relatedFiles: related } : msg,
                  ),
                );
              }
              if (payload.type === 'token' && typeof payload.token === 'string') {
                const piece = payload.token;
                setChatMessages((prev) =>
                  prev.map((msg) =>
                    msg.id === assistantId ? { ...msg, content: msg.content + piece } : msg,
                  ),
                );
              }
              if (payload.type === 'error') {
                throw new Error(payload.error ?? 'Stream error');
              }
              if (payload.type === 'done') {
                return;
              }
            }
          }
        };

        await pump();
        applyRelatedHighlights(related);
        return;
      }

      const payloadResponse = (await res.json()) as AIChatResponse & { error?: string };
      if (!res.ok) {
        throw new Error(payloadResponse.error ?? `AI request failed (${String(res.status)})`);
      }
      const relatedFiles = Array.isArray(payloadResponse.relatedFiles)
        ? payloadResponse.relatedFiles
        : [];
      const assistantMessage: ChatMessage = {
        id: assistantId,
        role: 'assistant',
        content: payloadResponse.answer ?? '',
        relatedFiles,
      };
      setChatMessages((prev) => [...prev, assistantMessage]);
      applyRelatedHighlights(relatedFiles);
    } catch (error) {
      const msg =
        error instanceof Error
          ? `Unable to answer right now: ${error.message}`
          : 'Unable to answer right now.';
      setChatMessages((prev) => {
        const hasPlaceholder = prev.some((m) => m.id === assistantId);
        if (hasPlaceholder) {
          return prev.map((m) => (m.id === assistantId ? { ...m, content: msg, relatedFiles: [] } : m));
        }
        return [...prev, { id: assistantId, role: 'assistant', content: msg, relatedFiles: [] }];
      });
    } finally {
      setChatLoading(false);
    }
  };

  const handleMapPrefixChange = useCallback((prefix: string) => {
    setMapPrefix(prefix);
  }, []);

  const focusFile = useCallback(
    (fileId: string, path?: string) => {
      setSelectedFileId(fileId);
      const trimmed = path?.trim() ?? '';
      if (trimmed.length > 0) {
        setFocusPaths([trimmed]);
        setFocusGeneration((g) => g + 1);
      }
      if (compactLayout) {
        setLeftDrawerOpen(false);
        setRightDrawerOpen(true);
      }
    },
    [compactLayout],
  );

  const switchToLatestReadySnapshot = () => {
    const active = repositories.find((repo) => repo.id === activeRepoId);
    const newestReadyByName = active
      ? repositories.find((repo) => repo.status === 'ready' && repo.name === active.name)
      : undefined;
    const newestReady = repositories.find((repo) => repo.status === 'ready');
    const target = newestReadyByName ?? newestReady;
    if (target) {
      setActiveRepoId(target.id);
    }
  };

  const submitRepository = async () => {
    setRepoSubmitting(true);
    setRepoSubmitError(null);
    try {
      let res: Response;
      const controller = new AbortController();
      const timeout = setTimeout(() => controller.abort(), 120_000);
      try {
        if (tab === 'zip') {
          if (!zipFile) throw new Error('Please select a ZIP file');
          const form = new FormData();
          form.append('sourceType', 'zip');
          form.append('displayName', displayName);
          form.append('file', zipFile);
          res = await fetch(`${getApiBase()}/repositories`, {
            method: 'POST',
            body: form,
            signal: controller.signal,
          });
        } else {
          if (sourceUrl.trim() === '') throw new Error('Repository URL is required');
          res = await fetch(`${getApiBase()}/repositories`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            signal: controller.signal,
            body: JSON.stringify({
              sourceType: tab,
              sourceUrl: sourceUrl.trim(),
              branch: branch.trim(),
              displayName: displayName.trim(),
            }),
          });
        }
      } finally {
        clearTimeout(timeout);
      }
      let body: Partial<RepositoryRecord> & { error?: string } = {};
      try {
        body = (await res.json()) as Partial<RepositoryRecord> & { error?: string };
      } catch {
        body = {};
      }
      if (!res.ok) throw new Error(body.error ?? 'Ingestion failed');
      const id = Number(body.id);
      if (Number.isFinite(id) && id > 0) setActiveRepoId(id);
      setSourceUrl('');
      setBranch('');
      setDisplayName('');
      setZipFile(null);
      await refreshRepositories();
    } catch (error) {
      const message =
        error instanceof Error && error.name === 'AbortError'
          ? 'Upload timed out. Try a smaller ZIP or verify API is running.'
          : error instanceof Error && error.message.toLowerCase().includes('failed to fetch')
            ? 'Network request failed. Verify API is running at the configured URL and CORS allows this origin.'
            : error instanceof Error
              ? error.message
              : 'Failed to submit repository';
      setRepoSubmitError(message);
    } finally {
      setRepoSubmitting(false);
    }
  };

  const restoreRepository = async () => {
    if (!pendingUndo) return;
    setRepoSubmitting(true);
    setRepoSubmitError(null);
    try {
      const res = await fetch(`${getApiBase()}/repositories`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          sourceType: pendingUndo.sourceType,
          sourceUrl: pendingUndo.sourceUrl,
          branch: pendingUndo.branch,
          displayName: pendingUndo.displayName,
        }),
      });
      let body: { id?: number; error?: string } = {};
      try {
        body = (await res.json()) as { id?: number; error?: string };
      } catch {
        body = {};
      }
      if (!res.ok) throw new Error(body.error ?? 'Could not restore repository');
      const id = Number(body.id);
      if (Number.isFinite(id) && id > 0) setActiveRepoId(id);
      setPendingUndo(null);
      await refreshRepositories();
    } catch (error) {
      setRepoSubmitError(error instanceof Error ? error.message : 'Could not restore repository');
    } finally {
      setRepoSubmitting(false);
    }
  };

  const deleteRepository = async (repoId: number, event: React.MouseEvent) => {
    event.stopPropagation();
    if (!window.confirm('Delete this repository and all indexed data for it?')) return;
    setRepoSubmitError(null);
    try {
      const res = await fetch(`${getApiBase()}/repositories/${String(repoId)}`, { method: 'DELETE' });
      let data: {
        deleted?: boolean
        undo?: { canRestore?: boolean; sourceType?: string; sourceUrl?: string; branch?: string; displayName?: string }
        error?: string
      } = {};
      try {
        data = (await res.json()) as typeof data;
      } catch {
        data = {};
      }
      if (!res.ok) throw new Error(data.error ?? 'Delete failed');
      const u = data.undo;
      if (u?.canRestore && u.sourceType && u.sourceType !== 'zip') {
        setPendingUndo({
          sourceType: u.sourceType as SourceType,
          sourceUrl: String(u.sourceUrl ?? ''),
          branch: String(u.branch ?? ''),
          displayName: String(u.displayName ?? ''),
        });
      }
      if (activeRepoId === repoId) {
        setSelectedFileId(null);
      }
      await refreshRepositories();
    } catch (error) {
      setRepoSubmitError(error instanceof Error ? error.message : 'Delete failed');
    }
  };

  const reindexRepository = async (repoId: number, event: React.MouseEvent) => {
    event.stopPropagation();
    setRepoSubmitError(null);
    try {
      const res = await fetch(`${getApiBase()}/repositories/${String(repoId)}/reindex`, { method: 'POST' });
      let data: { error?: string } = {};
      try {
        data = (await res.json()) as { error?: string };
      } catch {
        data = {};
      }
      if (!res.ok) throw new Error(data.error ?? 'Re-index failed');
      await refreshRepositories();
    } catch (error) {
      setRepoSubmitError(error instanceof Error ? error.message : 'Re-index failed');
    }
  };

  const pipelineActive = INDEX_PIPELINE.filter((s) => s.stage !== 'ready' && s.stage !== 'failed');
  const liveStage = activeProgress?.stage ?? activeRepository?.status;
  let liveStepIdx = pipelineActive.findIndex((s) => s.stage === liveStage);
  if (liveStage === 'failed') liveStepIdx = pipelineActive.length;
  if (liveStepIdx < 0) liveStepIdx = 0;

  return (
    <div className="app-shell">
      <header className="top-nav">
        <div className="top-nav__brand">
          <div className="top-nav__logo" aria-hidden />
          <div className="top-nav__titles">
            <strong>CodeAtlas</strong>
            <span>Architecture intelligence</span>
          </div>
        </div>

        <div className="top-nav__drawers" aria-label="Panel toggles">
          <button
            type="button"
            className="icon-btn"
            title="Repositories & workspace"
            aria-label="Open repositories and workspace panel"
            aria-expanded={leftDrawerOpen}
            onClick={() => {
              setRightDrawerOpen(false);
              setLeftDrawerOpen((o) => !o);
            }}
          >
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <rect x="3" y="3" width="7" height="18" rx="1" />
              <rect x="14" y="3" width="7" height="8" rx="1" />
              <rect x="14" y="13" width="7" height="8" rx="1" />
            </svg>
          </button>
          <button
            type="button"
            className="icon-btn"
            title="Inspector & assistant"
            aria-label="Open inspector and assistant panel"
            aria-expanded={rightDrawerOpen}
            onClick={() => {
              setLeftDrawerOpen(false);
              setRightDrawerOpen((o) => !o);
            }}
          >
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M21 15a4 4 0 01-4 4H7l-4 4V7a4 4 0 014-4h10a4 4 0 014 4v8z" />
            </svg>
          </button>
        </div>

        <div className="top-nav__repo">
          <select
            value={repositories.some((r) => r.id === activeRepoId) ? String(activeRepoId) : ''}
            onChange={(e) => {
              const v = Number(e.target.value);
              if (Number.isFinite(v) && v > 0) {
                setActiveRepoId(v);
                if (compactLayout) setLeftDrawerOpen(false);
              }
            }}
            aria-label="Active repository"
          >
            {repositories.length === 0 ? (
              <option value="">No repositories</option>
            ) : (
              [
                !repositories.some((r) => r.id === activeRepoId) ? (
                  <option key="select" value="">
                    Select…
                  </option>
                ) : null,
                ...repositories.map((r) => (
                  <option key={r.id} value={String(r.id)}>
                    {r.name} ({prettyStage(r.status)})
                  </option>
                )),
              ]
            )}
          </select>
        </div>

        <div className="top-nav__search">
          <div className="top-nav__search-wrap">
            <input
              ref={workspaceSearchRef}
              type="search"
              placeholder="Search repositories in workspace…"
              value={workspaceQuery}
              onChange={(e) => setWorkspaceQuery(e.target.value)}
              aria-label="Workspace search"
            />
          </div>
        </div>

        <div className="top-nav__command">
          <button type="button" className="cmd-trigger" onClick={() => workspaceSearchRef.current?.focus()}>
            Command <kbd>⌘</kbd>
            <kbd>K</kbd>
          </button>
        </div>

        <div className="top-nav__actions">
          <button type="button" className="icon-btn" title="Settings (soon)" aria-label="Settings">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <circle cx="12" cy="12" r="3" />
              <path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42" />
            </svg>
          </button>
        </div>
      </header>

      {compactLayout && (leftDrawerOpen || rightDrawerOpen) ? (
        <button
          type="button"
          className="workspace-backdrop"
          aria-label="Close side panels"
          onClick={() => {
            setLeftDrawerOpen(false);
            setRightDrawerOpen(false);
          }}
        />
      ) : null}

      <div className="app-grid">
        <aside className={`sidebar-left ${leftDrawerOpen ? 'sidebar-left--open' : ''}`}>
          <div className="sidebar-left__body">
          <div className="sidebar-section">
            <div className="sidebar-section__title">Add repository</div>
            <div className="ingest-compact">
              <div className="tabs">
                {(['github', 'gitlab', 'bitbucket', 'zip'] as SourceType[]).map((t) => (
                  <button
                    key={t}
                    type="button"
                    className={`tab ${tab === t ? 'active' : ''}`}
                    onClick={() => setTab(t)}
                  >
                    {t === 'zip' ? 'ZIP' : t.charAt(0).toUpperCase() + t.slice(1)}
                  </button>
                ))}
              </div>
              {tab === 'zip' ? (
                <div className="zip-drop-light">
                  <p>{zipFile ? zipFile.name : 'ZIP archive'}</p>
                  <input
                    ref={zipInputRef}
                    type="file"
                    accept=".zip"
                    style={{ display: 'none' }}
                    onChange={(event) => setZipFile(event.target.files?.[0] ?? null)}
                  />
                  <button type="button" className="btn btn-ghost-sm" onClick={() => zipInputRef.current?.click()}>
                    Browse
                  </button>
                </div>
              ) : (
                <>
                  <div className="field">
                    <input
                      type="text"
                      placeholder={`${tab} repository URL`}
                      value={sourceUrl}
                      onChange={(e) => setSourceUrl(e.target.value)}
                    />
                  </div>
                  <div className="field">
                    <input
                      type="text"
                      placeholder="Branch (optional)"
                      value={branch}
                      onChange={(e) => setBranch(e.target.value)}
                    />
                  </div>
                </>
              )}
              <div className="field">
                <input
                  type="text"
                  placeholder="Display name (optional)"
                  value={displayName}
                  onChange={(e) => setDisplayName(e.target.value)}
                />
              </div>
              <button type="button" className="btn btn-primary" disabled={repoSubmitting} onClick={() => void submitRepository()}>
                {repoSubmitting ? 'Starting…' : 'Index repository'}
              </button>
              {repoSubmitError ? <p className="warning">{repoSubmitError}</p> : null}
              {pendingUndo ? (
                <div className="undo-snackbar">
                  <span>Repository removed.</span>
                  <button type="button" className="btn btn-primary" onClick={() => void restoreRepository()}>
                    Restore
                  </button>
                  <button type="button" className="btn btn-ghost-sm" onClick={() => setPendingUndo(null)}>
                    Dismiss
                  </button>
                </div>
              ) : null}
            </div>
          </div>

          <div className="sidebar-section sidebar-section--grow">
            <div className="sidebar-section__title">Repositories</div>
            {repoLoading ? <div className="skeleton" style={{ height: 72 }} /> : null}
            {!repoLoading && sortedRepositories.length === 0 ? (
              <p className="empty-sidebar-msg">
                {repositories.length === 0 ? 'No repositories yet. Add a URL on the left.' : 'No matches for your search.'}
              </p>
            ) : null}
            <div className="repo-list-scroll">
              {sortedRepositories.map((repo) => (
                <div
                  key={repo.id}
                  role="button"
                  tabIndex={0}
                  className={`repo-card-v2 ${activeRepoId === repo.id ? 'repo-card-v2--active' : ''}`}
                  onClick={() => {
                    setActiveRepoId(repo.id);
                    if (compactLayout) setLeftDrawerOpen(false);
                  }}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter' || e.key === ' ') {
                      e.preventDefault();
                      setActiveRepoId(repo.id);
                      if (compactLayout) setLeftDrawerOpen(false);
                    }
                  }}
                >
                  <div className="repo-card-v2__top">
                    <div className="repo-card-v2__icon">{sourceIcon(repo.sourceType || 'github')}</div>
                    <div className="repo-card-v2__body">
                      <p className="repo-card-v2__name">{repo.name}</p>
                      <p className="repo-card-v2__meta">
                        <span
                          className={`status-pill status-pill--${
                            repo.status === 'ready' ? 'ready' : repo.status === 'failed' ? 'failed' : 'progress'
                          }`}
                        >
                          {prettyStage(repo.status)}
                        </span>
                        {repo.createdAt ? (
                          <span style={{ marginLeft: 8 }}>Updated {formatRepoDate(repo.createdAt)}</span>
                        ) : null}
                      </p>
                      <div className="repo-card-v2__metrics">
                        <span>{repo.filesIndexed ?? 0} files</span>
                        <span>{repo.symbolsIndexed ?? 0} symbols</span>
                        <span>{repo.edgesIndexed ?? 0} edges</span>
                      </div>
                      {repo.status !== 'ready' && repo.status !== 'failed' ? (
                        <div className="repo-progress-v2">
                          <div style={{ width: `${Math.max(3, Math.round(repo.progressPercent ?? 0))}%` }} />
                        </div>
                      ) : null}
                    </div>
                    <div className="repo-card-v2__actions">
                      <button
                        type="button"
                        title={favorites.has(repo.id) ? 'Remove favorite' : 'Favorite'}
                        className={favorites.has(repo.id) ? 'favorite--on' : ''}
                        onClick={(e) => toggleFavorite(repo.id, e)}
                      >
                        ★
                      </button>
                      <button
                        type="button"
                        title="Re-index"
                        onClick={(e) => void reindexRepository(repo.id, e)}
                      >
                        ↻
                      </button>
                      {repo.status === 'failed' ? (
                        <button
                          type="button"
                          className="repo-card-retry"
                          title="Retry ingestion"
                          onClick={(e) => void reindexRepository(repo.id, e)}
                        >
                          Retry
                        </button>
                      ) : null}
                      <details className="repo-more" onClick={(e) => e.stopPropagation()}>
                        <summary className="repo-more__hit" title="More actions" aria-label="More actions">
                          ···
                        </summary>
                        <div className="repo-more__menu">
                          <button
                            type="button"
                            className="repo-more__danger"
                            onClick={(e) => void deleteRepository(repo.id, e)}
                          >
                            Delete repository…
                          </button>
                        </div>
                      </details>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>

          <div className="sidebar-section sidebar-section--hints">
            <div className="sidebar-section__title">Exploration workflow</div>
            <ol className="workflow-hint-list">
              <li>
                <strong>Pick a repo</strong> — the map and assistant stay scoped to it.
              </li>
              <li>
                <strong>Drill clusters</strong> — click packages; breadcrumbs jump back up.
              </li>
              <li>
                <strong>Select a file</strong> — inspector + AI use it as focus context.
              </li>
              <li>
                <strong>Ask in assistant</strong> — answers highlight related nodes on the map.
              </li>
            </ol>
          </div>
          </div>
        </aside>

        <main className="main-canvas">
          {!repoLoading && repositories.length === 0 ? (
            <div className="onboarding-hero">
              <div className="onboarding-hero__visual" aria-hidden />
              <h1>Map any codebase like a product</h1>
              <p className="lead">
                Paste a Git URL or upload a ZIP. CodeAtlas builds a live architecture map—clusters first, then modules,
                files, and symbols—as ingestion completes.
              </p>
              <div className="example-repos">
                <button type="button" onClick={() => { setTab('github'); setSourceUrl('https://github.com/stemmlerjs/simple-typescript-starter'); }}>
                  TypeScript starter
                </button>
                <button type="button" onClick={() => { setTab('github'); setSourceUrl('https://github.com/vercel/next.js'); }}>
                  Next.js
                </button>
                <button type="button" onClick={() => { setTab('github'); setSourceUrl('https://github.com/facebook/react'); }}>
                  React
                </button>
              </div>
            </div>
          ) : null}

          {repositories.length > 0 && canShowGraph ? (
            <div className="canvas-toolbar workspace-flow">
              <div className="workspace-flow__chain" aria-label="Workspace context">
                <span className="workspace-flow__step workspace-flow__step--repo">
                  <span className="workspace-flow__label">Repo</span>
                  <strong>{activeRepository?.name ?? '—'}</strong>
                </span>
                <span className="workspace-flow__arrow" aria-hidden>
                  →
                </span>
                <span className="workspace-flow__step">
                  <span className="workspace-flow__label">Map</span>
                  <strong>{mapPrefix ? mapPrefix : 'root'}</strong>
                </span>
                <span className="workspace-flow__arrow" aria-hidden>
                  →
                </span>
                <span className="workspace-flow__step">
                  <span className="workspace-flow__label">Inspector</span>
                  <strong>
                    {fileDetail?.path
                      ? pathBasename(fileDetail.path)
                      : selectedFileId
                        ? 'Loading…'
                        : 'pick a file'}
                  </strong>
                </span>
              </div>
              <p className="canvas-toolbar__hint workspace-flow__hint">
                Click clusters to drill in · hover edges to trace deps · search finds paths · assistant uses map + selection.
              </p>
              <div className="canvas-stats">
                <div>
                  <span>Indexed files · </span>
                  {activeRepository?.filesIndexed ?? 0}
                </div>
                <div>
                  <span>Edges · </span>
                  {activeRepository?.edgesIndexed ?? 0}
                </div>
              </div>
            </div>
          ) : null}

          {repositories.length > 0 ? (
          <div className="canvas">
            {showIndexingBanner ? (
              <div className="indexing-live-banner" role="status">
                <span className="indexing-live-banner__dot" aria-hidden />
                <span>
                  Indexing in motion — {activeProgress?.stage?.replaceAll('_', ' ') ?? activeRepository?.status ?? ''} ·{' '}
                  {Math.round(activeProgress?.progressPercent ?? activeRepository?.progressPercent ?? 0)}% ·{' '}
                  {activeProgress?.metrics.filesIndexed ?? activeRepository?.filesIndexed ?? 0} files in graph
                </span>
              </div>
            ) : null}
            {canShowGraph ? (
              <HierarchyGraph
                key={activeRepoId}
                repositoryId={activeRepoId}
                highlightedFileIds={highlightedIds}
                selectedFileId={selectedFileId}
                focusGeneration={focusGeneration}
                focusPaths={focusPaths}
                onPrefixChange={handleMapPrefixChange}
                onSelectFile={(id) => {
                  setSelectedFileId(id);
                  if (compactLayout) {
                    setLeftDrawerOpen(false);
                    setRightDrawerOpen(true);
                  }
                }}
              />
            ) : null}

            {repositories.length > 0 && showIndexingAsidePanel ? (
              <div className="indexing-workspace">
                <div className="indexing-feed">
                  <h2>Building architecture map</h2>
                  <p className="sub">
                    We&apos;re cloning, parsing, and wiring dependencies. The cluster map appears as soon as files land in
                    the index; full readiness still tracks below.
                  </p>
                  <div className="activity-list">
                    {pipelineActive.map((step, i) => {
                      const done = i < liveStepIdx;
                      const running = i === liveStepIdx;
                      return (
                        <div
                          key={step.stage}
                          className={`activity-step ${done ? 'activity-step--done' : running ? 'activity-step--run' : 'activity-step--pending'}`}
                        >
                          <div className="activity-step__mark">{done ? '✓' : running ? '●' : '○'}</div>
                          <div className="activity-step__body">
                            <strong>{step.title}</strong>
                            <span>{step.hint}</span>
                          </div>
                        </div>
                      );
                    })}
                  </div>
                  <div className="indexing-metrics">
                    <div className="metric-tile">
                      <div className="metric-tile__value">{Math.round(activeProgress?.progressPercent ?? activeRepository.progressPercent ?? 0)}%</div>
                      <div className="metric-tile__label">Overall</div>
                    </div>
                    <div className="metric-tile">
                      <div className="metric-tile__value">{activeProgress?.metrics.filesIndexed ?? activeRepository.filesIndexed ?? 0}</div>
                      <div className="metric-tile__label">Files</div>
                    </div>
                    <div className="metric-tile">
                      <div className="metric-tile__value">{activeProgress?.metrics.symbolsIndexed ?? activeRepository.symbolsIndexed ?? 0}</div>
                      <div className="metric-tile__label">Symbols</div>
                    </div>
                    <div className="metric-tile">
                      <div className="metric-tile__value">{activeProgress?.metrics.edgesIndexed ?? activeRepository.edgesIndexed ?? 0}</div>
                      <div className="metric-tile__label">Edges</div>
                    </div>
                  </div>
                  {activeRepository.status === 'failed' && activeRepository.errorDetails ? (
                    <p className="warning">{activeRepository.errorDetails}</p>
                  ) : null}
                  {activeRepository.status === 'failed' ? (
                    <div className="indexing-recovery">
                      <button
                        type="button"
                        className="btn btn-primary"
                        onClick={(e) => void reindexRepository(activeRepoId, e)}
                      >
                        Retry ingestion
                      </button>
                      <p className="meta indexing-recovery__help">
                        Retries clone/index with the same source URL. ZIP uploads need files still on disk—otherwise add the
                        archive again.
                      </p>
                    </div>
                  ) : null}
                </div>
                <div className="indexing-preview">
                  <div className="indexing-preview__canvas">
                    <div className="indexing-preview__pulse" />
                  </div>
                  <p className="indexing-preview__label">Architecture preview</p>
                  <p className="indexing-preview__sub">
                    Full interactive map opens automatically when indexing completes. You can switch repos or keep working
                    in the assistant.
                  </p>
                  <div className="indexing-actions">
                    <button type="button" className="btn btn-ghost-sm" onClick={switchToLatestReadySnapshot}>
                      Jump to latest ready copy
                    </button>
                  </div>
                </div>
              </div>
            ) : null}

            {repositories.length > 0 && !canShowGraph && !showIndexingAsidePanel && !activeRepository ? (
              <div className="onboarding-hero onboarding-hero--compact">
                <h1>Select a workspace</h1>
                <p className="lead">
                  Pick a repository from the sidebar or use the header dropdown. Everything — map, inspector, and assistant —
                  activates together once you choose.
                </p>
              </div>
            ) : null}
          </div>
          ) : null}
        </main>

        <aside className={`sidebar-right ${rightDrawerOpen ? 'sidebar-right--open' : ''}`}>
          <div className="panel-detail">
        {fileDetail ? (
          <>
            <div className="inspector-head">
              <h2>{pathBasename(fileDetail.path)}</h2>
              <p className="inspector-path">{fileDetail.path}</p>
              <dl className="inspector-metrics">
                <div>
                  <dt>Imports</dt>
                  <dd>{fileDetail.imports.length}</dd>
                </div>
                <div>
                  <dt>Exports</dt>
                  <dd>{fileDetail.exports.length}</dd>
                </div>
                <div>
                  <dt>Symbols</dt>
                  <dd>{fileDetail.symbols.length}</dd>
                </div>
              </dl>
            </div>
            <div className="inspector-actions" role="group" aria-label="File actions">
              <button
                type="button"
                className="inspector-action inspector-action--primary"
                onClick={() =>
                  setChatInput(
                    `Explain how "${fileDetail.path}" fits into the architecture, what depends on it, and key risks if we change it.`,
                  )
                }
              >
                Ask AI about this file
              </button>
              <button
                type="button"
                className="inspector-action"
                onClick={() =>
                  setChatInput(
                    `Impact analysis: if we change "${fileDetail.path}", what breaks downstream? List modules and files most affected.`,
                  )
                }
              >
                Downstream impact
              </button>
              <button
                type="button"
                className="inspector-action"
                onClick={() => {
                  const pathLike = fileDetail.imports.filter(
                    (s) => s.includes('/') || /\.[a-z]+$/i.test(s),
                  );
                  const paths = (pathLike.length > 0 ? pathLike : fileDetail.imports).slice(0, 12);
                  if (paths.length > 0) {
                    setFocusPaths(paths);
                    setFocusGeneration((g) => g + 1);
                  }
                }}
              >
                Focus import paths on map
              </button>
            </div>
            <section>
              <h3>Imports</h3>
              <ul>
                {fileDetail.imports.length === 0 ? <li>None</li> : null}
                {fileDetail.imports.map((imp) => (
                  <li key={imp}>{imp}</li>
                ))}
              </ul>
            </section>
            <section>
              <h3>Exports</h3>
              <ul>
                {fileDetail.exports.length === 0 ? <li>None</li> : null}
                {fileDetail.exports.map((exp) => (
                  <li key={exp}>{exp}</li>
                ))}
              </ul>
            </section>
            <section>
              <h3>Symbols</h3>
              <ul>
                {fileDetail.symbols.length === 0 ? <li>None</li> : null}
                {fileDetail.symbols.map((symbol) => (
                  <li key={`${symbol.kind}:${symbol.name}`}>
                    <span>{symbol.name}</span>
                    <code>{symbol.kind}</code>
                  </li>
                ))}
              </ul>
            </section>
          </>
        ) : selectedFileId ? (
          fileDetailError ? (
            <div className="empty">
              <h2>Could not load file</h2>
              <p className="warning">{fileDetailError}</p>
            </div>
          ) : fileDetailLoading ? (
            <div className="skeleton" style={{ height: 140 }} />
          ) : (
            <div className="empty">
              <h2>Loading…</h2>
              <p className="meta">Fetching file metadata from the API.</p>
            </div>
          )
        ) : (
          <div className="empty-state-sm empty-state-sm--guided">
            <h2>Inspector</h2>
            <p className="meta">
              Click a <strong>file node</strong> on the map to load imports, exports, and symbols here. Selection drives AI
              context and dependency highlighting.
            </p>
            <ol className="hint-steps">
              <li>Open the architecture map (center).</li>
              <li>Drill folders with clusters or use &quot;Find file in map&quot;.</li>
              <li>Choose a file — this panel and the assistant sync automatically.</li>
            </ol>
          </div>
        )}
          </div>

        <section className="panel-ai">
          <div className="ai-context-card" role="status">
            <div className="ai-context-card__row">
              <span className="ai-context-card__label">Grounded in</span>
              <span className="ai-context-card__value">{activeRepository?.name ?? 'repository'}</span>
            </div>
            <div className="ai-context-card__row">
              <span className="ai-context-card__label">Map scope</span>
              <span className="ai-context-card__value">{mapPrefix || 'repository root'}</span>
            </div>
            <div className="ai-context-card__row">
              <span className="ai-context-card__label">Selection</span>
              <span className="ai-context-card__value">
                {fileDetail?.path ? pathBasename(fileDetail.path) : selectedFileId ? 'loading file…' : 'none — pick a file'}
              </span>
            </div>
          </div>
          <div className="panel-ai__header">
            <h3>Architecture assistant</h3>
            <p>Answers use retrieval over your index and include graph references you can jump to.</p>
          </div>
          <div className="ai-controls-row">
            <select value={aiProvider} onChange={(event) => setAIProvider(event.target.value)}>
              <option value="local">Local</option>
              <option value="openai">OpenAI</option>
              <option value="anthropic">Anthropic (soon)</option>
              <option value="gemini">Gemini (soon)</option>
              <option value="huggingface">HuggingFace (soon)</option>
              <option value="openrouter">OpenRouter (soon)</option>
            </select>
            <input
              type="text"
              value={aiModel}
              placeholder="Model"
              onChange={(event) => setAIModel(event.target.value)}
            />
          </div>
          <div className="prompt-grid">
            {suggestedPrompts.map((prompt) => (
              <button key={prompt} type="button" className="prompt-chip" onClick={() => setChatInput(prompt)}>
                {prompt}
              </button>
            ))}
          </div>
          <div className="shortcut-grid">
            <button type="button" className="btn-ghost-sm" onClick={() => setChatInput('Give me architecture insights and hotspots.')}>
              Insights
            </button>
            <button type="button" className="btn-ghost-sm" onClick={() => setChatInput('Run an impact analysis for auth changes.')}>
              Impact
            </button>
          </div>
          <div className="chat-list-v2">
            {chatMessages.length === 0 ? (
              <div className="chat-empty-guide">
                <p className="meta">
                  Questions are <strong>scoped</strong> to the active repo, map folder, and inspector file (see card above).
                  Related files in replies highlight on the graph.
                </p>
              </div>
            ) : null}
            {chatMessages.map((message) => (
              <article key={message.id} className={`bubble-v2 ${message.role}`}>
                <p>{message.content}</p>
                {message.relatedFiles.length > 0 ? (
                  <div className="ref-row">
                    {message.relatedFiles.map((file) => (
                      <button
                        key={`${message.id}-${String(file.fileId)}`}
                        type="button"
                        className="ref-chip"
                        onClick={() => {
                          focusFile(String(file.fileId), file.path);
                        }}
                      >
                        {file.path}
                      </button>
                    ))}
                  </div>
                ) : null}
              </article>
            ))}
          </div>
          <div className="chat-compose-v2">
            <textarea
              value={chatInput}
              placeholder="Ask about architecture impact…"
              onChange={(event) => {
                setChatInput(event.target.value);
              }}
              onKeyDown={(event) => {
                if (event.key === 'Enter' && !event.shiftKey) {
                  event.preventDefault();
                  void submitChat();
                }
              }}
            />
            <button type="button" className="btn-primary-full" onClick={() => void submitChat()} disabled={chatLoading}>
              {chatLoading ? 'Thinking…' : 'Send'}
            </button>
          </div>
        </section>
      </aside>
      </div>
    </div>
  );
}

export function App() {
  return (
    <ReactFlowProvider>
      <GraphWorkspace />
    </ReactFlowProvider>
  );
}
