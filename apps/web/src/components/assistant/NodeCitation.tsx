import { basename } from '../../lib/fileType';
import { useStore } from '../../store';

export function NodeCitation({
  path,
  fileId,
  verified = true,
  onSelect,
}: {
  path: string;
  fileId?: string;
  verified?: boolean;
  onSelect: (fileId: string, path: string) => void;
}) {
  const setGraphHoverFileId = useStore((s) => s.setGraphHoverFileId);
  const setSidebarView = useStore((s) => s.setSidebarView);

  const hoverIn = () => {
    if (fileId) setGraphHoverFileId(fileId);
    setSidebarView('map');
  };
  const hoverOut = () => setGraphHoverFileId(null);

  return (
    <button
      type="button"
      className={`node-citation ${verified ? '' : 'node-citation--unverified'}`}
      onClick={() => fileId && onSelect(fileId, path)}
      onMouseEnter={hoverIn}
      onMouseLeave={hoverOut}
      onFocus={hoverIn}
      onBlur={hoverOut}
      title={verified ? path : `${path} (not found in index)`}
    >
      <span className="node-citation__label">{basename(path)}</span>
      <i className="codicon codicon-link-external node-citation__icon" aria-hidden />
    </button>
  );
}
