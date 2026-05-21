import { useStore } from '../../store';

const STEPS = [
  { key: 'cloning', label: 'Clone repository' },
  { key: 'extracting', label: 'Extract workspace' },
  { key: 'parsing', label: 'Parse sources' },
  { key: 'building_graph', label: 'Build dependency graph' },
  { key: 'generating_embeddings', label: 'Semantic embeddings' },
];

/** Map SSE step ids and legacy status strings to STEPS keys. */
function normalizeStage(stage: string): string {
  switch (stage) {
    case 'clone_repository':
      return 'cloning';
    case 'extract_workspace':
      return 'extracting';
    case 'index_workspace':
      return 'indexing';
    case 'parse_sources':
      return 'parsing';
    case 'build_dependency_graph':
      return 'building_graph';
    case 'semantic_embeddings':
      return 'generating_embeddings';
    default:
      return stage;
  }
}

export function ProgressPopover() {
  const open = useStore((s) => s.progressPopoverOpen);
  const setOpen = useStore((s) => s.setProgressPopoverOpen);
  const ingestionStatus = useStore((s) => s.ingestionStatus);
  const repositories = useStore((s) => s.repositories);
  const activeRepoId = useStore((s) => s.activeRepoId);

  if (!open) return null;

  const repo = repositories.find((r) => r.id === activeRepoId);
  const stage = normalizeStage(ingestionStatus?.codeIndex.stage ?? repo?.status ?? '');
  const pct = Math.round(
    repo?.status === 'ready'
      ? 100
      : (ingestionStatus?.codeIndex.progressPercent ?? repo?.progressPercent ?? 0),
  );

  return (
    <>
      <div
        role="presentation"
        style={{ position: 'fixed', inset: 0, zIndex: 49 }}
        onClick={() => setOpen(false)}
      />
      <div className="progress-popover">
        <strong>INDEXING: {repo?.name ?? '—'}</strong>
        <hr style={{ border: 'none', borderTop: '1px solid var(--border-subtle)', margin: '8px 0' }} />
        {STEPS.map((step) => {
          const done =
            stage === 'ready' ||
            (step.key !== stage &&
              STEPS.findIndex((s) => s.key === stage) > STEPS.findIndex((s) => s.key === step.key));
          const active = stage === step.key;
          return (
            <div key={step.key} style={{ marginBottom: 4 }}>
              {done ? '✓' : active ? '▶' : '○'} {step.label}
              {active ? ` ${String(pct)}%` : ''}
            </div>
          );
        })}
        {ingestionStatus?.socioTechnical.status === 'running' ? (
          <div style={{ marginTop: 8, color: 'var(--text-secondary)' }}>
            Socio sync: {Math.round(ingestionStatus.socioTechnical.completionPercent)}%
          </div>
        ) : null}
      </div>
    </>
  );
}
