import { useStore } from '../../store';
import { StatusDot } from '../ui/StatusDot';

export function StatusBar({ onClick }: { onClick: () => void }) {
  const activeRepoId = useStore((s) => s.activeRepoId);
  const repositories = useStore((s) => s.repositories);
  const ingestionStatus = useStore((s) => s.ingestionStatus);
  const hotspots = useStore((s) => s.hotspots);
  const clusterLayer = useStore((s) => s.clusterLayer);
  const apiStatus = useStore((s) => s.apiStatus);

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

  const apiLabel =
    apiStatus === 'online'
      ? 'API online'
      : apiStatus === 'degraded'
        ? 'API degraded'
        : apiStatus === 'checking'
          ? 'API…'
          : 'API offline';

  return (
    <footer className="status-bar" onClick={onClick} role="status">
      <div className="status-bar__left">
        <span className="status-bar__item">
          <StatusDot status={apiStatus === 'online' ? 'ready' : apiStatus === 'offline' ? 'failed' : 'running'} />
          {apiLabel}
        </span>
        <span className="status-bar__item">
          <StatusDot status={statusKind} />
          {repo?.status === 'ready'
            ? `Ready — ${files != null ? String(files) : '—'} files · ${edges != null ? String(edges) : '—'} edges`
            : indexing
              ? `Indexing (${String(pct)}%)`
              : repo
                ? repo.status
                : 'No repo'}
        </span>
        <span className="status-bar__item">⚠ {String(hotspots.length)} hotspots</span>
        <span className="status-bar__item">⬡ {nodeLabel}</span>
      </div>
      <div className="status-bar__right">
        <span>⌘K commands · ⌘P files · ⌘⇧P palette</span>
      </div>
    </footer>
  );
}
