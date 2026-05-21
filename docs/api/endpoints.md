# HTTP API Reference

**Base URL (local):** `http://localhost:8080`  
**Web dev proxy:** `http://localhost:5173/api` → same routes

**Authentication:** None (see [security](../security/overview.md)).

**CORS:** `CORS_ALLOWED_ORIGINS` (default includes `http://localhost:5173`).

---

## `GET /health`

**Response 200:**

```json
{
  "service": "codeatlas-api",
  "status": "ok",
  "version": "0.1.0"
}
```

---

## Graph

### `GET /graph/files?repositoryId={id}`

Returns flat file list + dependencies for legacy/full graph views.

**Query:** `repositoryId` (optional, defaults to `1` in code).

### `GET /graph/clusters?repositoryId={id}&prefix={path}`

**Response 200:**

```json
{
  "prefix": "src/auth",
  "clusters": [
    {
      "id": "cluster:src/auth/services",
      "label": "services",
      "pathPrefix": "src/auth/services",
      "level": 3,
      "fileCount": 4,
      "internalEdges": 12,
      "density": 0.42,
      "hasChildren": true
    }
  ],
  "files": [
    { "id": "42", "path": "src/auth/index.ts", "symbolCount": 8 }
  ],
  "edges": [
    { "from": "cluster:src/auth", "to": "f:42", "count": 3 }
  ],
  "socioOverlay": {
    "fileOverlays": {
      "42": {
        "fileId": "42",
        "isHotspot": true,
        "hasBusFactorRisk": false,
        "riskLevel": "high",
        "architectureSignalCount": 0,
        "dominantOwnerLogin": "alice"
      }
    }
  }
}
```

*`socioOverlay` omitted if socio query unavailable.*

### `GET /graph/file?repositoryId={id}&fileId={id}`

**Response 200:**

```json
{
  "id": "42",
  "path": "src/auth/service.ts",
  "imports": ["./types", "../db"],
  "exports": ["AuthService"],
  "symbols": [{ "name": "AuthService", "kind": "class" }]
}
```

### `GET /graph/symbols?repositoryId={id}&fileId={id}`

Symbol list for a file (same shape as embedded symbols in `/graph/file`).

---

## Repositories

### `GET /repositories`

**Response 200:**

```json
{
  "repositories": [
    {
      "id": 1,
      "name": "my-app",
      "sourceType": "github",
      "sourceUrl": "https://github.com/org/repo",
      "branch": "main",
      "workspacePath": "/.../workspace/github",
      "status": "ready",
      "progressPercent": 100,
      "filesIndexed": 120,
      "symbolsIndexed": 890,
      "edgesIndexed": 340,
      "embeddingsIndexed": 120,
      "createdAt": "2026-05-20T12:00:00Z",
      "updatedAt": "2026-05-20T12:05:00Z"
    }
  ]
}
```

### `POST /repositories`

**JSON body (git sources):**

```json
{
  "sourceType": "github",
  "sourceUrl": "https://github.com/org/repo",
  "branch": "main",
  "displayName": "My App"
}
```

**Multipart (zip):** fields `sourceType=zip`, `displayName`, `file=@archive.zip`

**Response 202:** `Repository` object (status `queued`).

### `GET /repositories/{id}/progress`

```json
{
  "repositoryId": 1,
  "stage": "building_graph",
  "status": "building_graph",
  "progressPercent": 65,
  "metrics": {
    "filesIndexed": 80,
    "symbolsIndexed": 400,
    "edgesIndexed": 120,
    "embeddingsIndexed": 0
  }
}
```

### `GET /repositories/{id}/ingestion/status`

Combined code + socio status:

```json
{
  "repositoryId": 1,
  "codeIndex": {
    "status": "ready",
    "stage": "ready",
    "progressPercent": 100,
    "filesIndexed": 120
  },
  "socioTechnical": {
    "phase": "github_history",
    "status": "completed",
    "completionPercent": 100,
    "staleness": "fresh",
    "steps": [
      { "step": "sync_commits", "status": "completed", "itemsProcessed": 240 }
    ],
    "availablePhases": ["github_history", "engineering_memory", "operational_intel"]
  },
  "graphCompleteness": {
    "codeGraphReady": true,
    "socioHistoryReady": true,
    "engineeringReady": false,
    "operationalReady": false,
    "partialDataWarning": false
  }
}
```

### `GET /repositories/{id}/ownership?fileId={id}`

```json
{
  "ownership": [
    {
      "fileId": 42,
      "path": "src/auth/service.ts",
      "contributorCount": 3,
      "busFactor": 2,
      "riskLevel": "medium",
      "dominantOwnerShare": 0.62,
      "dominantOwner": { "id": "…", "login": "alice" }
    }
  ]
}
```

Without `fileId`, returns top files by risk/churn (limit 50).

### `GET /repositories/{id}/hotspots?limit=25`

```json
{
  "hotspots": [
    {
      "fileId": 42,
      "path": "src/auth/service.ts",
      "hotspotScore": 0.91,
      "churnScore": 4200,
      "riskLevel": "high",
      "busFactor": 1,
      "commitCount90d": 18
    }
  ]
}
```

### `DELETE /repositories/{id}`

**Response 200:**

```json
{
  "deleted": true,
  "undo": {
    "sourceType": "github",
    "sourceUrl": "https://github.com/org/repo",
    "branch": "main",
    "displayName": "My App",
    "canRestore": true
  }
}
```

### `POST /repositories/{id}/reindex`

**Response 202:** `{ "status": "reindex_started" }`

---

## AI

### `POST /ai/chat`

**Request:**

```json
{
  "repositoryId": 1,
  "query": "What breaks if we change auth?",
  "provider": "openai",
  "model": "gpt-4o-mini",
  "stream": false
}
```

**Response 200 (non-stream):**

```json
{
  "answer": "…",
  "relatedFiles": [
    { "fileId": 42, "path": "src/auth/service.ts", "reason": "semantic+graph" }
  ],
  "provider": "openai",
  "model": "gpt-4o-mini"
}
```

**Stream (`stream: true`):** `Content-Type: text/event-stream`

```
data: {"type":"meta","relatedFiles":[...],"provider":"openai","model":"gpt-4o-mini"}

data: {"type":"token","token":"Based"}

data: {"type":"done"}
```

---

## Error shape

```json
{ "error": "human-readable message" }
```

| Status | Typical cause |
|--------|----------------|
| 400 | Invalid id, missing query param |
| 404 | Repository not found |
| 502 | AI provider failure |
| 503 | Service nil (misconfiguration) |
