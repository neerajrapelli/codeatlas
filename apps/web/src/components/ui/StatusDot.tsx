export function StatusDot({ status }: { status: 'queued' | 'running' | 'ready' | 'failed' }) {
  return <span className={`status-dot status-dot--${status}`} aria-hidden />;
}
