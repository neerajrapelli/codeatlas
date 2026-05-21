import type { Repository } from '../types';

/** Normalize git remote URLs for duplicate detection. */
export function normalizeRepoUrl(url: string): string {
  let u = url.trim().toLowerCase();
  u = u.replace(/\.git$/i, '');
  u = u.replace(/\/+$/, '');
  return u;
}

export function findExistingRepository(
  repos: Repository[],
  sourceType: string,
  sourceUrl: string,
  branch: string,
): Repository | undefined {
  const norm = normalizeRepoUrl(sourceUrl);
  const br = (branch.trim() || 'main').toLowerCase();
  return repos.find((r) => {
    if (r.sourceType !== sourceType) return false;
    if (normalizeRepoUrl(r.sourceUrl ?? '') !== norm) return false;
    return (r.branch ?? 'main').toLowerCase() === br;
  });
}
