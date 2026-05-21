# @codeatlas/shared-types

## Owns

- API request/response TypeScript interfaces (what the Go API sends and receives)
- Graph primitive types (GraphNode, GraphEdge, GraphResponse)
- Ingestion status types (IngestionProgress, IngestionStep)
- Shared enums (FileType, RiskLevel, SignalType, EdgeType)

## Does NOT own

- Graph algorithms or computation logic → that belongs in graph-core
- React component props → define those locally in apps/web
- Database schemas → those live in apps/api/migrations/

## Rule

If it's a type that crosses the API boundary (Go → TypeScript), it belongs here.

If it's a type only used inside the frontend, define it locally.
