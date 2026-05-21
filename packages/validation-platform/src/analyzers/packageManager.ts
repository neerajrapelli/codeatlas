import type { Analyzer, AnalyzerResult, PackageManagerId } from '../types.js';

const LOCKFILE_MAP: Array<{ file: string; id: PackageManagerId }> = [
  { file: 'pnpm-lock.yaml', id: 'pnpm' },
  { file: 'package-lock.json', id: 'npm' },
  { file: 'yarn.lock', id: 'yarn' },
  { file: 'bun.lockb', id: 'bun' },
  { file: 'bun.lock', id: 'bun' },
  { file: 'go.sum', id: 'go-mod' },
  { file: 'go.mod', id: 'go-mod' },
  { file: 'poetry.lock', id: 'poetry' },
  { file: 'Pipfile.lock', id: 'pip' },
  { file: 'requirements.txt', id: 'pip' },
  { file: 'uv.lock', id: 'uv' },
  { file: 'Cargo.lock', id: 'cargo' },
];

export const packageManagerAnalyzer: Analyzer = {
  id: 'package-manager',

  analyze(ctx): AnalyzerResult {
    const found: PackageManagerId[] = [];
    const files = ctx.listFiles();

    for (const { file, id } of LOCKFILE_MAP) {
      if (files.includes(file) || ctx.fileExists(file)) {
        if (!found.includes(id)) found.push(id);
      }
    }

    if (ctx.fileExists('pyproject.toml') && !found.includes('poetry')) {
      const text = ctx.readText('pyproject.toml');
      if (text?.includes('[tool.poetry]')) found.push('poetry');
      else if (text?.includes('[project]')) found.push('pip');
    }

    return {
      partial: {
        stack: { packageManagers: found },
        metadata: { detectedLockfiles: found },
      },
    };
  },
};
