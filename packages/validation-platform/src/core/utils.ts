import { readdirSync, readFileSync, statSync } from 'node:fs';
import { join, relative, resolve } from 'node:path';

export const DEFAULT_IGNORE_DIRS = new Set([
  '.git',
  '.svn',
  '.hg',
  'node_modules',
  '.pnpm',
  'dist',
  'build',
  '.next',
  '.turbo',
  'coverage',
  '.cache',
  'vendor',
  '__pycache__',
  '.venv',
  'venv',
]);

export function normalizeRepoPath(repoPath: string): string {
  return resolve(repoPath);
}

export function walkRepo(
  root: string,
  ignoreDirs: ReadonlySet<string> = DEFAULT_IGNORE_DIRS,
  maxDepth = 12,
): string[] {
  const files: string[] = [];

  function walk(dir: string, depth: number): void {
    if (depth > maxDepth) return;
    let entries: string[];
    try {
      entries = readdirSync(dir);
    } catch {
      return;
    }

    for (const name of entries) {
      const full = join(dir, name);
      const rel = relative(root, full).replace(/\\/g, '/');
      if (ignoreDirs.has(name)) continue;

      let stat;
      try {
        stat = statSync(full);
      } catch {
        continue;
      }

      if (stat.isDirectory()) {
        walk(full, depth + 1);
      } else if (stat.isFile()) {
        files.push(rel);
      }
    }
  }

  walk(root, 0);
  return files.sort();
}

export function readTextFile(absPath: string): string | null {
  try {
    return readFileSync(absPath, 'utf8');
  } catch {
    return null;
  }
}

export function buildFileTree(
  files: string[],
  maxEntries = 200,
): import('../types.js').FileTreeEntry[] {
  const root: Map<string, import('../types.js').FileTreeEntry> = new Map();

  for (const file of files.slice(0, maxEntries)) {
    const parts = file.split('/');
    let current = root;
    for (let i = 0; i < parts.length; i++) {
      const part = parts[i];
      if (!part) continue;
      const isLast = i === parts.length - 1;
      const key = parts.slice(0, i + 1).join('/');
      let entry = current.get(key);
      if (!entry) {
        entry = {
          path: key,
          type: isLast ? 'file' : 'directory',
          children: isLast ? undefined : [],
        };
        current.set(key, entry);
      }
      if (!isLast && entry.children) {
        const childMap = new Map<string, import('../types.js').FileTreeEntry>();
        for (const c of entry.children) childMap.set(c.path, c);
        current = childMap;
        entry.children = [...childMap.values()];
      }
    }
  }

  const topLevel = new Map<string, import('../types.js').FileTreeEntry>();
  for (const file of files.slice(0, maxEntries)) {
    const first = file.split('/')[0];
    if (!first) continue;
    if (!topLevel.has(first)) {
      const isDir = files.some((f) => f.startsWith(`${first}/`));
      topLevel.set(first, {
        path: first,
        type: isDir ? 'directory' : 'file',
      });
    }
  }

  return [...topLevel.values()].sort((a, b) => a.path.localeCompare(b.path));
}

export function unique<T>(items: T[]): T[] {
  return [...new Set(items)];
}

export function mergeStack(
  a: Partial<import('../types.js').DetectedStack>,
  b: Partial<import('../types.js').DetectedStack>,
): import('../types.js').DetectedStack {
  return {
    runtimes: unique([...(a.runtimes ?? []), ...(b.runtimes ?? [])]),
    frameworks: unique([...(a.frameworks ?? []), ...(b.frameworks ?? [])]),
    packageManagers: unique([...(a.packageManagers ?? []), ...(b.packageManagers ?? [])]),
    databaseHints: unique([...(a.databaseHints ?? []), ...(b.databaseHints ?? [])]),
  };
}
