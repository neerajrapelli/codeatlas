import type { IngestionProgress, IngestionStep } from '../../hooks/useIngestionProgress';

const STEP_LABELS: Record<string, string> = {
  clone_repository: 'Clone',
  extract_workspace: 'Extract',
  index_workspace: 'Index',
  parse_sources: 'Parse',
  build_dependency_graph: 'Graph',
  semantic_embeddings: 'Embed',
};

function stepClass(status: IngestionStep['status']): string {
  if (status === 'complete') return 'ingestion-bar__step-icon ingestion-bar__step-icon--done';
  if (status === 'running') return 'ingestion-bar__step-icon ingestion-bar__step-icon--active ingestion-bar__pulse';
  if (status === 'failed') return 'ingestion-bar__step-icon ingestion-bar__step-icon--failed';
  return 'ingestion-bar__step-icon ingestion-bar__step-icon--pending';
}

function StepIcon({ status }: { status: IngestionStep['status'] }) {
  if (status === 'complete') return <span className={stepClass(status)}>✓</span>;
  if (status === 'running') return <span className={stepClass(status)}>●</span>;
  if (status === 'failed') return <span className={stepClass(status)}>✗</span>;
  return <span className={stepClass(status)}>○</span>;
}

export function IngestionBar({
  progress,
  fading,
  repoName,
  repoReady,
}: {
  progress: IngestionProgress | null;
  fading?: boolean;
  repoName?: string;
  /** When the repository row is already `ready`, hide the bar even if SSE lagged at 95%. */
  repoReady?: boolean;
}) {
  if (repoReady) return null;
  if (!progress) return null;
  if (progress.status !== 'running' && progress.status !== 'queued') {
    if (!fading) return null;
  }

  const pct =
    progress.status === 'complete'
      ? 100
      : Math.min(100, Math.max(0, Math.round(progress.progress.percent)));

  return (
    <div
      className={`ingestion-bar ingestion-bar--graph ${fading ? 'ingestion-bar--fade' : ''}`}
      role="status"
      aria-live="polite"
    >
      <span className="ingestion-bar__lead ingestion-bar__pulse">●</span>
      <span className="ingestion-bar__label">Indexing {repoName ?? 'repository'}</span>
      <div className="ingestion-bar__steps">
        {progress.steps.map((step) => (
          <span key={step.name} className="ingestion-bar__step">
            <StepIcon status={step.status} />
            <span>{STEP_LABELS[step.name] ?? step.name}</span>
          </span>
        ))}
      </div>
      <span className="ingestion-bar__pct">{String(pct)}%</span>
    </div>
  );
}
