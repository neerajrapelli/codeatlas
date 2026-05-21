const KEY = 'codeatlas_jwt';

export function getAuthToken(): string | null {
  try {
    return localStorage.getItem(KEY);
  } catch {
    return null;
  }
}

export function setAuthToken(token: string) {
  localStorage.setItem(KEY, token);
}

export function clearAuthToken() {
  localStorage.removeItem(KEY);
}

export function authHeaders(): Record<string, string> {
  const token = getAuthToken();
  if (!token) return {};
  return { Authorization: `Bearer ${token}` };
}

export function jsonHeaders(): Record<string, string> {
  return { 'Content-Type': 'application/json', ...authHeaders() };
}

/** EventSource cannot set Authorization; pass token as query param when auth is enabled. */
export function authQueryString(): string {
  const token = getAuthToken();
  if (!token) return '';
  return `access_token=${encodeURIComponent(token)}`;
}
