import type { FileType } from '../../types';

const COLORS: Record<FileType, string> = {
  ts: 'var(--node-ts)',
  go: 'var(--node-go)',
  css: 'var(--node-css)',
  test: 'var(--node-test)',
  config: 'var(--node-config)',
  other: 'var(--node-config)',
};

export function FileTypeIcon({ type, label }: { type: FileType; label: string }) {
  return (
    <span className="graph-file-node__type" style={{ background: COLORS[type] }}>
      {label}
    </span>
  );
}
