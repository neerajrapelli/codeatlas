# @codeatlas/graph-core

## Owns

- Pure TypeScript graph helper functions (no React, no API calls)
- Layout utilities (cluster grouping, node positioning helpers)
- Graph traversal helpers (find dependents, find ancestors, BFS/DFS)
- Risk score computation (churn + bus_factor composite scoring) — *planned; currently adjacency helpers only*

## Does NOT own

- API types → those belong in shared-types
- React components → those belong in apps/web
- Rendering logic → no DOM or React Flow dependencies allowed here

## Rule

Every function in this package must be a pure function.

No side effects. No network calls. No React imports.

Testable with plain `vitest` or `jest` — no jsdom needed.
