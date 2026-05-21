import type { Repository } from '../types';

/** Keep the newest row when the API returns duplicate IDs. */
export function dedupeRepositories(repos: Repository[]): Repository[] {
  const byId = new Map<number, Repository>();
  for (const repo of repos) {
    const prev = byId.get(repo.id);
    if (!prev || new Date(repo.updatedAt).getTime() >= new Date(prev.updatedAt).getTime()) {
      byId.set(repo.id, repo);
    }
  }
  return Array.from(byId.values()).sort(
    (a, b) => new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime(),
  );
}
