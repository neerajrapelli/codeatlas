# ADR Index

Architecture Decision Records are stored in this folder and should be append-only.

## Naming

- Use: `NNNN-short-title.md` (example: `0001-ingestion-stage-model.md`)
- Keep `NNNN` zero-padded and monotonically increasing

## Required sections

Every ADR must contain:

- Top-level title (`# ...`)
- `## Status`
- `## Context`
- `## Decision`
- `## Consequences`

Run `pnpm arch:adr:lint` to validate shape.

## Status vocabulary

- `proposed`
- `accepted`
- `superseded`
- `deprecated`
