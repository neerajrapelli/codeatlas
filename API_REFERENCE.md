# API Reference (Quick)

Full documentation: [docs/api/endpoints.md](docs/api/endpoints.md)

**Base:** `http://localhost:8080` · **Dev proxy:** `/api` on port 5173

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Liveness |
| GET | `/graph/files` | Full file + dependency list |
| GET | `/graph/clusters` | Hierarchy layer + `socioOverlay` |
| GET | `/graph/file` | Single file detail |
| GET | `/graph/symbols` | Symbols for file |
| GET | `/repositories` | List repositories |
| POST | `/repositories` | Start ingest (JSON or ZIP) |
| GET | `/repositories/{id}/progress` | Code indexing progress |
| GET | `/repositories/{id}/ingestion/status` | Code + socio completeness |
| GET | `/repositories/{id}/ownership` | Ownership summaries |
| GET | `/repositories/{id}/hotspots` | Hotspot ranking |
| DELETE | `/repositories/{id}` | Delete repo + graph |
| POST | `/repositories/{id}/reindex` | Re-run index + socio |
| POST | `/ai/chat` | Architecture Q&A (optional SSE) |

**Auth:** None.

**Example — ingest:**

```bash
curl -X POST http://localhost:8080/repositories \
  -H "Content-Type: application/json" \
  -d '{"sourceType":"github","sourceUrl":"https://github.com/org/repo","branch":"main"}'
```

**Example — chat:**

```bash
curl -X POST http://localhost:8080/ai/chat \
  -H "Content-Type: application/json" \
  -d '{"repositoryId":1,"query":"What depends on auth?","stream":false}'
```
