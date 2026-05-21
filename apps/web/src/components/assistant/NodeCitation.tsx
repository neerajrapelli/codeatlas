export function NodeCitation({
  path,
  fileId,
  onSelect,
}: {
  path: string;
  fileId?: string;
  onSelect: (fileId: string, path: string) => void;
}) {
  return (
    <button
      type="button"
      className="node-citation"
      onClick={() => fileId && onSelect(fileId, path)}
    >
      {path} ↗
    </button>
  );
}
