import type { IngestionProgress, IngestionStep } from '../../hooks/useIngestionProgress';

const STEP_LABELS: Record<string, string> = {
  clone_repository: 'Clone',
  extract_workspace: 'Extract',
  index_workspace: 'Index',
  parse_sources: 'Parse',
  build_dependency_graph: 'Graph',
  semantic_embeddings: 'Embed',
};

function StepIcon({ status }: { status: IngestionStep['status'] }) {
  if (status === 'complete') return <span style={{ color: '#4ec9b0' }}>✓</span>;
  if (status === 'running') {
    return (
      <span className="ingestion-bar__pulse" style={{ color: '#007acc' }}>
        ●
      </span>
    );
  }
  if (status === 'failed') return <span style={{ color: '#f44747' }}>✗</span>;
  return <span style={{ color: '#555555' }}>○</span>;
}

export function IngestionBar({
  progress,
  fading,
  repoName,
}: {
  progress: IngestionProgress | null;
  fading?: boolean;
  repoName?: string;
}) {
  if (!progress) return null;
  if (progress.status !== 'running' && progress.status !== 'queued') {
    if (!fading) return null;
  }

  const pct = progress.progress.percent;

  return (
    <div
      className={`ingestion-bar ingestion-bar--shell ${fading ? 'ingestion-bar--fade' : ''}`}
      role="status"
      style={{
        height: 36,
        display: 'flex',
        alignItems: 'center',
        gap: 12,
        padding: '0 12px',
        background: '#1a1a1a',
        borderBottom: '1px solid #2a2a2a',
        fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace',
        fontSize: 12,
        flexShrink: 0,
      }}
    >
      <span className="ingestion-bar__pulse" style={{ color: '#007acc' }}>
        ●
      </span>
      <span>Indexing {repoName ?? 'repository'}</span>
      {progress.steps.map((step) => (
        <span key={step.name} style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}>
          <StepIcon status={step.status} />
          {STEP_LABELS[step.name] ?? step.name}
        </span>
      ))}
      <span style={{ marginLeft: 'auto' }}>{String(pct)}%</span>
    </div>
  );
}
