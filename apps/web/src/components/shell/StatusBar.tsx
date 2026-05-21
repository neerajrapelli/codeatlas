import { useStore } from '../../store';
import { StatusDot } from '../ui/StatusDot';

export function StatusBar({ onClick }: { onClick: () => void }) {
  const activeRepoId = useStore((s) => s.activeRepoId);
  const repositories = useStore((s) => s.repositories);
  const ingestionStatus = useStore((s) => s.ingestionStatus);
  const hotspots = useStore((s) => s.hotspots);
  const clusterLayer = useStore((s) => s.clusterLayer);

  const repo = repositories.find((r) => r.id === activeRepoId);
  const indexing = repo && repo.status !== 'ready' && repo.status !== 'failed';
  const pct = Math.round(
    ingestionStatus?.codeIndex.progressPercent ?? repo?.progressPercent ?? 0,
  );
  const files = indexing ? null : (ingestionStatus?.codeIndex.filesIndexed ?? repo?.filesIndexed);
  const edges = indexing ? null : repo?.edgesIndexed;
  const fileCount = clusterLayer?.files.length;
  const nodeLabel =
    indexing
      ? 'Indexing…'
      : fileCount != null && fileCount > 0
        ? `${String(fileCount)} in view`
        : files != null && files > 0
          ? `${String(files)} files`
          : '—';

  const statusKind =
    repo?.status === 'ready'
      ? 'ready'
      : repo?.status === 'failed'
        ? 'failed'
        : indexing
          ? 'running'
          : 'queued';

  return (
    <footer className="status-bar" onClick={onClick} role="status">
      <div className="status-bar__left">
        <span className="status-bar__item">
          <StatusDot status={statusKind} />
          {repo?.status === 'ready'
            ? `Ready — ${files != null ? String(files) : '—'} files · ${edges != null ? String(edges) : '—'} edges`
            : indexing
              ? `Indexing (${String(pct)}%)`
              : '—'}
        </span>
        <span className="status-bar__item">⚠ {String(hotspots.length)} hotspots</span>
        <span className="status-bar__item">⬡ {nodeLabel}</span>
      </div>
      <div className="status-bar__right">
        <span>Go API ✓</span>
        <span>pgvector ✓</span>
        <span>TypeScript</span>
      </div>
    </footer>
  );
}
