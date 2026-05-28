# Dependency Analysis

CodeAtlas enforces lightweight architecture contracts in CI using static dependency analysis.

## Tooling

- `dependency-cruiser` for boundary rules and machine-readable graph output.
- `madge` for circular dependency checks.
- `scripts/architecture/dependency-report.mjs` to generate:
  - `artifacts/dependency-graph.json`
  - `artifacts/module-graph.json`
  - `artifacts/architecture.json`

## Local usage

```bash
pnpm arch:dep:json
pnpm arch:dep:cycles
pnpm arch:dep:validate
```

## CI behavior

The `architecture` job in `.github/workflows/ci.yml` runs on `push` and `pull_request`.

- Always uploads `artifacts/` as a build artifact.
- Fails when:
  - cycles are detected (`arch:dep:cycles`)
  - forbidden boundaries are violated (`arch:dep:validate`)

## Current boundary rules

Rules are defined in `.dependency-cruiser.cjs`:

- No circular dependencies.
- `apps/api` must not import from `apps/web`.
- App-layer imports into `apps/api/internal/httpserver` are warned.

Adjust rules incrementally to avoid noisy false positives.
