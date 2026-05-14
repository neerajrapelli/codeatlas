/**
 * Base URL for the CodeAtlas HTTP API (no trailing slash).
 * In development, defaults to `/api` so Vite can proxy to the Go server and avoid CORS issues.
 * Set `VITE_API_URL` to override (e.g. point at a remote API).
 */
export function getApiBase(): string {
  const fromEnv = import.meta.env.VITE_API_URL;
  if (fromEnv !== undefined && String(fromEnv).trim() !== '') {
    return String(fromEnv).replace(/\/$/, '');
  }
  if (import.meta.env.DEV) {
    return '/api';
  }
  return String(__CODEATLAS_API_URL__).replace(/\/$/, '');
}
