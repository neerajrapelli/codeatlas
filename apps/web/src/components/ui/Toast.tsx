import { useEffect } from 'react';

import { useStore } from '../../store';

export function Toast() {
  const toast = useStore((s) => s.toast);
  const clearToast = useStore((s) => s.clearToast);

  useEffect(() => {
    if (!toast) return;
    const id = window.setTimeout(() => clearToast(), 5000);
    return () => clearTimeout(id);
  }, [toast, clearToast]);

  if (!toast) return null;

  return (
    <div className={`toast toast--${toast.variant}`} role="status" aria-live="polite">
      <span>{toast.message}</span>
      <button type="button" className="toast__dismiss" onClick={clearToast} aria-label="Dismiss">
        <i className="codicon codicon-close" />
      </button>
    </div>
  );
}
