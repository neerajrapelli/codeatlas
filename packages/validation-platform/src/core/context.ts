import { existsSync } from 'node:fs';
import { join, resolve } from 'node:path';
import type { AnalysisContext } from '../types.js';
import { DEFAULT_IGNORE_DIRS, readTextFile, walkRepo } from './utils.js';

export function createAnalysisContext(
  repoPath: string,
  outputDir: string,
  ignoreDirs: ReadonlySet<string> = DEFAULT_IGNORE_DIRS,
): AnalysisContext {
  const root = resolve(repoPath);
  const out = resolve(outputDir);
  let cachedFiles: string[] | null = null;

  return {
    repoPath: root,
    outputDir: out,
    ignoreDirs,

    listFiles(relativeDir = ''): string[] {
      if (!cachedFiles) {
        cachedFiles = walkRepo(root, ignoreDirs);
      }
      const prefix = relativeDir.replace(/\\/g, '/').replace(/\/$/, '');
      if (!prefix) return cachedFiles;
      return cachedFiles.filter((f) => f === prefix || f.startsWith(`${prefix}/`));
    },

    readText(relativePath: string): string | null {
      const abs = join(root, relativePath);
      return readTextFile(abs);
    },

    fileExists(relativePath: string): boolean {
      return existsSync(join(root, relativePath));
    },

    resolve(relativePath: string): string {
      return resolve(root, relativePath);
    },
  };
}
