# @codeatlas/validation-platform

Autonomous testing and validation for repositories CodeAtlas ingests. Phase 1 is a **read-only architecture analysis engine**: it scans a repo on disk and emits structured JSON artifacts teams can diff in CI, feed to later test generators, or compare against CodeAtlas’s own graph.

## Why this exists

CodeAtlas already indexes symbols and dependencies server-side. A separate validation platform lets us:

- **Validate before ingest** — catch stack mismatches, missing workspace packages, or route surface drift on a PR branch.
- **Run locally and in CI** without the full API/Postgres stack.
- **Grow in phases** toward sandboxed test execution, contract checks, and autonomous regression — without bolting one-off scripts into `apps/api`.

Phase 1 deliberately avoids Docker, network calls, and cloning third-party repos in CI. Heuristics trade perfect accuracy for speed and zero credentials.

## Folder structure

```
packages/validation-platform/
  README.md                 # This file — vision, roadmap, usage
  package.json
  tsconfig.json
  vitest.config.ts
  fixtures/
    sample-monorepo/        # Lightweight fixture for unit tests
  src/
    index.ts                # Public API
    cli.ts                  # `analyze` command
    types.ts                # Shared types
    core/
      context.ts            # Repo-scoped file access
      orchestrator.ts       # Runs analyzers, merges, writes artifacts
      utils.ts              # Walk, ignore rules, merges
    analyzers/              # One concern per analyzer (plugin-shaped)
      filesystem.ts
      packageManager.ts
      stackDetector.ts
      monorepo.ts
      routes.ts
    writers/
      jsonWriter.ts
    plugins/
      registry.ts           # Register built-in + custom analyzers
    test/
      fixture.ts
```

Future phases will add `runners/`, `sandboxes/`, `policies/`, and `reporters/` alongside `analyzers/`.

## Phase roadmap (1–9)

| Phase | Focus | Adds |
|-------|--------|------|
| **1** | Architecture analysis engine | Stack detection, monorepo modules, heuristic routes, JSON artifacts, CLI (this package) |
| **2** | Sandbox runner | Ephemeral Docker (or devcontainer) to install deps and run lint/typecheck/test with timeouts |
| **3** | Contract validation | OpenAPI/GraphQL/schema diff vs detected routes; breaking-change hints |
| **4** | Test discovery | Map `*_test.go`, `*.test.ts`, Playwright/Cypress configs to modules |
| **5** | Coverage & gates | Parse coverage output; policy thresholds per package |
| **6** | Synthetic test stubs | Generate skipped test skeletons from routes and public APIs |
| **7** | Live API probing | Optional smoke HTTP against declared routes (dev only) |
| **8** | CodeAtlas integration | Compare `architecture.json` to ingested graph; PR annotations |
| **9** | Autonomous loop | Schedule re-runs, flake detection, fix suggestions via agent |

## Phase 1 architecture decisions

| Decision | Rationale | Tradeoff |
|----------|-----------|----------|
| TypeScript CLI package in pnpm workspace | Matches `apps/web` and shared packages; easy CI on Node job | Go repos analyzed by file heuristics only, not `go list` |
| Plugin registry + `Analyzer` interface | New detectors ship without editing orchestrator | Slight ceremony vs one big script |
| Filesystem walk with ignore set | Predictable, no git required | May scan more than `git ls-files` would |
| Regex / manifest heuristics for routes | Fast, no AST parser dependency | Misses dynamic routes and framework magic |
| Four JSON artifacts | Stable contract for Phase 8 diff | Not a single mega-document |
| Fixture monorepo in-repo | CI stays fast; no Payload clone | Fixture is smaller than real monorepos |

## Output artifacts

Written to `--out` (default `./validation-output`):

| File | Contents |
|------|----------|
| `architecture.json` | Stack, modules, routes, services, directory layout summary |
| `dependency-graph.json` | Workspace, declared, and import edges (best-effort) |
| `module-graph.json` | Workspace modules and inter-package edges |
| `stack-summary.json` | Human-readable headline and bullets |

## CLI usage

Build once, then analyze:

```bash
pnpm install
pnpm --filter @codeatlas/validation-platform build

# Analyze the CodeAtlas monorepo root (pnpm runs the script with cwd = this package)
pnpm --filter @codeatlas/validation-platform analyze -- --repo ../.. --out ./validation-output

# Analyze the bundled fixture
pnpm --filter @codeatlas/validation-platform analyze -- --repo ./fixtures/sample-monorepo --out ./validation-output

# Or use the bin after build
pnpm exec codeatlas-validate analyze --repo /path/to/repo --out ./out
```

Scripts:

- `pnpm --filter @codeatlas/validation-platform test` — vitest on fixture
- `pnpm --filter @codeatlas/validation-platform typecheck`
- `pnpm --filter @codeatlas/validation-platform analyze:payload` — manual run (see below)

## Testing against Payload CMS

CI does **not** clone [payloadcms/payload](https://github.com/payloadcms/payload). For manual exploration:

```bash
git clone https://github.com/payloadcms/payload.git ../payload
pnpm --filter @codeatlas/validation-platform build
pnpm --filter @codeatlas/validation-platform analyze:payload
# artifacts under packages/validation-platform/out/payload/
```

Or:

```bash
pnpm --filter @codeatlas/validation-platform analyze -- --repo ../payload --out ./out/payload
```

Review `stack-summary.json` and `architecture.json` for detected Next.js, pnpm/turbo monorepo layout, and route counts.

## Programmatic API

```ts
import { analyzeAndWrite, AnalyzerRegistry } from '@codeatlas/validation-platform';

await analyzeAndWrite({
  repoPath: '/path/to/repo',
  outputDir: '/path/to/out',
});
```

Register custom analyzers via `AnalyzerRegistry` (see `src/plugins/registry.ts`).

## Phase 2 preview

Next step: a **sandbox runner** that takes `architecture.json` as input, picks the right package manager, runs install + lint + typecheck + test inside an isolated container with resource limits, and emits `validation-report.json` with pass/fail and logs. Reuse Phase 1 stack detection to choose Node vs Go vs Python entrypoints.
