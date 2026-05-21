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
  const rawPct = ingestionStatus?.codeIndex.progressPercent ?? repo?.progressPercent ?? 0;
  const pct = repo?.status === 'ready' ? 100 : Math.round(rawPct);
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

  const apiDotStatus =
    apiStatus === 'online' ? 'ready' : apiStatus === 'offline' ? 'failed' : 'running';

  const apiLabel =
    apiStatus === 'online'
      ? 'API online'
      : apiStatus === 'degraded'
        ? 'API degraded'
        : apiStatus === 'checking'
          ? 'API…'
          : 'API offline';

  const repoLabel =
    repo?.status === 'ready'
      ? `Ready — ${files != null ? String(files) : '—'} files · ${edges != null ? String(edges) : '—'} edges`
      : indexing
        ? `Indexing (${String(pct)}%)`
        : repo
          ? repo.status.replaceAll('_', ' ')
          : 'No repo';

  return (
    <footer className="status-bar" onClick={onClick} role="status">
      <div className="status-bar__left">
        <span className="status-bar__item" title={apiLabel}>
          <StatusDot status={apiDotStatus} />
          <span className="status-bar__text">{apiLabel}</span>
        </span>
        <span className="status-bar__item" title={repo ? `${repo.name} — ${repoLabel}` : repoLabel}>
          <StatusDot status={statusKind} />
          <span className="status-bar__text">{repoLabel}</span>
        </span>
        <span className="status-bar__item" title={`${String(hotspots.length)} hotspots`}>
          <span className="status-bar__text">⚠ {String(hotspots.length)} hotspots</span>
        </span>
        <span className="status-bar__item" title={nodeLabel}>
          <span className="status-bar__text">⬡ {nodeLabel}</span>
        </span>
      </div>
      <div className="status-bar__right">
        <span className="status-bar__hints">⌘K commands · ⌘P files · ⌘⇧P palette</span>
      </div>
    </footer>
  );
}
