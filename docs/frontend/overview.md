# Frontend Architecture

## Stack

| Technology | Version (approx.) | Role |
|------------|-------------------|------|
| React | 19 | UI |
| TypeScript | 5.7 | Types |
| Vite | 6 | Dev server + build |
| React Flow | 11 | Graph canvas |
| ELK.js | 0.9 | Layered layout |

**Package:** `@codeatlas/web` (`apps/web`)

## Entry

```
index.html → src/main.tsx → App.tsx
```

`App` wraps the workspace in `ReactFlowProvider` (required for `HierarchyGraph` hooks).

## Layout model

`GraphWorkspace` in `App.tsx` uses a **three-column workspace**:

| Region | Class | Content |
|--------|-------|---------|
| Left | `sidebar-left` | Add repo, repo list, workflow hints |
| Center | `main-canvas` | Architecture map, indexing banners |
| Right | `sidebar-right` | Socio panels, file inspector, AI chat |

**Responsive:** ≤1100px switches to **drawers** (`leftDrawerOpen`, `rightDrawerOpen`).

**Viewport:** `100vh` shell—panels scroll independently (see `styles.css`).

## Key components

### `HierarchyGraph.tsx`

- Fetches `GET /graph/clusters?repositoryId=&prefix=`
- Normalizes response including `socioOverlay`
- ELK layout → React Flow nodes (`cluster`, `file` custom nodes)
- File node badges: hotspot pulse, bus-factor warning, architecture signal count, owner initials
- Architecture search filters visible file nodes
- Edge hover highlights dependency strength

### `SocioPanels.tsx`

- Polls every 5s:
  - `GET /repositories/{id}/ingestion/status`
  - `GET /repositories/{id}/ownership` (optional `fileId`)
  - `GET /repositories/{id}/hotspots?limit=8`
- Partial-data banner when `graphCompleteness.partialDataWarning`

### `App.tsx` (orchestration)

- Repository CRUD UI (ingest, delete, reindex, undo snackbar)
- Progress polling (1.5s while not ready)
- File inspector (`GET /graph/file`)
- AI chat with SSE streaming
- Favorites in `localStorage` (`codeatlas:fav-repos`)
- Workspace flow strip: Repo → Map → Inspector

## API base URL

`src/apiBase.ts`:

- Dev default: `/api` (Vite proxy strips prefix → backend)
- Override: `VITE_API_URL` baked as `__CODEATLAS_API_URL__`

**Why proxy:** Avoids CORS and hostname mismatch (`localhost` vs `127.0.0.1`).

## State management

**No global store** (Redux/Zustand/Context beyond React Flow).

| Concern | Mechanism |
|---------|-----------|
| Repo list | `useState` + 4s poll `GET /repositories` |
| Active repo | `useState(activeRepoId)` |
| Index progress | `useState(activeProgress)` + 1.5s poll |
| Map prefix | `useState(mapPrefix)` + callback from graph |
| Selection | `selectedFileId`, `fileDetail` |
| Chat | `chatMessages`, `chatInput`, `chatLoading` |
| Highlights | `highlightedIds` from AI related files |
| Socio status | `socioIngestion` from `SocioPanels` callback |

## Shared packages

- `@codeatlas/shared-types` — health, symbol kinds (contracts)
- `@codeatlas/graph-core` — graph helpers (minimal usage in MVP UI)

## Styling

Single `styles.css` — design tokens via CSS variables (`--accent`, `--text-muted`, …).

Component-specific: graph nodes (`.file-node--hotspot`), socio panels (`.socio-panel`), indexing banners.

## Build

```bash
pnpm --filter @codeatlas/web build
# tsc --noEmit + vite build → dist/
```

## Browser assumptions

- Fetch API + EventSource-style SSE parsing (manual read of `response.body` stream in `submitChat`)
- `localStorage` for favorites
- Modern evergreen browser (Chrome, Firefox, Edge, Safari)
