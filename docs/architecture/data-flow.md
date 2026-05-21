# Data Flow

## 1. Repository onboarding (code graph)

```mermaid
sequenceDiagram
  participant U as User / Web
  participant API as httpserver
  participant RI as repoingest.Service
  participant FS as workspace/
  participant IDX as indexer.Service
  participant DB as PostgreSQL

  U->>API: POST /repositories
  API->>RI: Enqueue(CreateRequest)
  RI->>DB: INSERT repositories (queued)
  RI-->>API: 202 Accepted
  API-->>U: repository id + status

  Note over RI: Background goroutine
  RI->>FS: clone or extract ZIP
  RI->>DB: status cloning/extracting
  RI->>IDX: Run(repositoryPath)
  IDX->>FS: scan .ts files
  IDX->>DB: files, symbols, imports, deps, embeddings
  RI->>DB: status ready, progress 100%
  RI->>RI: runSocioEnrichment (async)
```

**Files:** `internal/repoingest/service.go`, `internal/indexer/service.go`, `internal/indexer/postgres_store.go`.

## 2. Graph visualization

```mermaid
flowchart LR
  U[User drills prefix] --> Web[HierarchyGraph]
  Web --> API[GET /graph/clusters]
  API --> GH[graphhierarchy.BuildLayer]
  GH --> DB[(files + file_dependencies)]
  API --> SQ[socio.QueryService.GetFileOverlays]
  SQ --> DB
  API --> Web[clusters + files + edges + socioOverlay]
  Web --> ELK[ELK layout]
  ELK --> RF[React Flow render]
```

**Why overlays on clusters response:** Keeps one round-trip for map paint; overlays keyed by `fileId` string.

## 3. AI architecture chat (Graph RAG)

```mermaid
flowchart TB
  Q[User query] --> Prep[ai.Service.PrepareChat]
  Prep --> Emb[ProviderManager.Embed]
  Emb --> Ret[Retriever.RetrieveContext]
  Ret --> Sem[pgvector semantic seeds]
  Ret --> Exp[BFS file_dependencies]
  Ret --> Load[loadContextItems + SocioContextForFiles]
  Load --> Prompt[buildUserPrompt]
  Prompt --> LLM[ProviderManager.Chat / Stream]
  LLM --> Ans[SSE tokens or JSON answer]
```

**Context budget:** `AI_CONTEXT_TOKEN_BUDGET` (default 7000) trims prompt blocks in `internal/ai/prompt.go`.

**Socio in prompt:** Owner, bus factor, churn, risk, hotspot flags per file when `file_metrics` exist.

## 4. Socio-technical Phase 1 (GitHub)

```mermaid
flowchart TB
  Trigger[repo ready / reindex] --> Ing[ingestion.RunPhase1GitHubHistory]
  Ing --> Run[socio_ingestion_runs]
  Ing --> GH[github.Client ListCommits / PRs]
  GH --> Store[socio.Store upserts]
  Store --> CF[commit_files / pr_files]
  Store --> Met[socio.ComputeFileMetrics]
  Met --> FM[file_metrics + contributor_file_ownership]
```

**Skip conditions (implemented):**

- `source_type != github` → run marked `skipped`
- `GITHUB_TOKEN` empty → `skipped`
- Invalid GitHub URL → `failed`

## 5. Frontend state flow

| State | Storage | Purpose |
|-------|---------|---------|
| `repositories` | React `useState` | Repo list from `GET /repositories` |
| `activeRepoId` | React `useState` | Selected workspace scope |
| `selectedFileId` | React `useState` | Inspector + AI grounding |
| `favorites` | `localStorage` | `codeatlas:fav-repos` |
| `chatMessages` | React `useState` | In-memory session |
| `socioIngestion` | React `useState` | From `SocioPanels` callback |

No Redux/Zustand—local component state with polling intervals (4s repos, 1.5s progress, 5s socio).

## 6. Python `apps/ai` service

**Current data flow:** None into the Go stack. `apps/ai` exposes `GET /health` only. **Assumption:** Reserved for future Python-specific ML workloads or separated embedding service.
