const STORAGE_KEY = 'codeatlas-recent-commands';
const MAX_RECENT = 8;

export type RecentCommand = { id: string; label: string };

export function loadRecentCommands(): RecentCommand[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw) as unknown;
    if (!Array.isArray(parsed)) return [];
    return parsed
      .filter(
        (x): x is RecentCommand =>
          typeof x === 'object' &&
          x != null &&
          typeof (x as RecentCommand).id === 'string' &&
          typeof (x as RecentCommand).label === 'string',
      )
      .slice(0, MAX_RECENT);
  } catch {
    return [];
  }
}

export function pushRecentCommand(entry: RecentCommand): void {
  const prev = loadRecentCommands().filter((c) => c.id !== entry.id);
  const next = [{ id: entry.id, label: entry.label }, ...prev].slice(0, MAX_RECENT);
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(next));
  } catch {
    /* quota / private mode */
  }
}
