# Troubleshooting

Quick fixes for local development. Full guide: [docs/operations/troubleshooting.md](docs/operations/troubleshooting.md).

## Cannot connect to API from browser

- Start API: `cd apps/api && go run ./cmd/server`
- Use http://localhost:5173 (Vite proxy), not raw 8080, unless `VITE_API_URL` is set correctly
- Check `CORS_ALLOWED_ORIGINS` includes your UI origin

## Database connection failed

```bash
make docker-up
# Verify
docker compose ps
```

`DATABASE_URL` default: `postgresql://codeatlas:codeatlas@localhost:5432/codeatlas`

## Ingestion failed immediately

- Validate URL is `https://` for git sources
- ZIP: check size limits (`ZIP_MAX_BYTES`)
- Read `errorDetails` on repository card or `GET /repositories/{id}/progress`

## Map empty after ingest

- Repo must contain **TypeScript** files for MVP indexer
- Wait until `filesIndexed > 0` or `status=ready`

## No ownership / hotspots

- Set `GITHUB_TOKEN` in `apps/api/.env`
- Repository must be `sourceType: github`
- Reindex after adding token

## AI gives generic / empty answers

- Set `OPENAI_API_KEY` and reindex for real embeddings
- Ensure `repositoryId` matches indexed repo
- Try `provider: "openai"` with valid key

## Port already in use

- API: change `HTTP_ADDR` (e.g. `:8081`) and `VITE_API_PROXY_TARGET`
- Web: Vite `strictPort: true` on 5173—stop other Vite instances

## Windows: make not found

Run commands from `Makefile` manually, e.g. `pnpm dev`, `docker compose up -d`.
