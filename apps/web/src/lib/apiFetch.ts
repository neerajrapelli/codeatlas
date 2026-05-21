import { getApiBase } from '../apiBase';
import { authHeaders, jsonHeaders } from './authToken';
import { useStore } from '../store';

export { jsonHeaders, authHeaders };

function handleAuthFailure(status: number): void {
  if (status !== 401 && status !== 403) return;
  const store = useStore.getState();
  store.pushToast('API unauthorized — add or refresh your token in Settings', 'error');
  store.setSidebarView('settings');
}

export async function apiFetch(
  path: string,
  init: RequestInit = {},
): Promise<Response> {
  const headers = new Headers(init.headers);
  const auth = authHeaders();
  for (const [k, v] of Object.entries(auth)) {
    headers.set(k, v);
  }
  if (init.body != null && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json');
  }
  const res = await fetch(`${getApiBase()}${path}`, { ...init, headers });
  if (res.status === 401 || res.status === 403) {
    handleAuthFailure(res.status);
  }
  return res;
}

export async function apiJson<T>(path: string, init: RequestInit = {}): Promise<T> {
  const res = await apiFetch(path, init);
  if (!res.ok) {
    const err = (await res.json().catch(() => ({}))) as { error?: string };
    const msg = err.error ?? `${path} ${String(res.status)}`;
    if (res.status >= 500) {
      useStore.getState().pushToast(msg, 'error');
    }
    throw new Error(msg);
  }
  return res.json() as Promise<T>;
}

export async function apiPostJson<T>(path: string, body: unknown): Promise<T> {
  return apiJson<T>(path, { method: 'POST', headers: jsonHeaders(), body: JSON.stringify(body) });
}
