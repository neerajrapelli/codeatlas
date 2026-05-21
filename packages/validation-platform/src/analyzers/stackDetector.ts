import type {
  Analyzer,
  AnalyzerResult,
  DatabaseHint,
  FrameworkId,
  RuntimeId,
} from '../types.js';

const DB_DEP_PATTERNS: Array<{ pattern: RegExp; hint: DatabaseHint }> = [
  { pattern: /\bpg\b|postgres|postgresql|@prisma\/client|prisma/i, hint: 'postgresql' },
  { pattern: /\bmysql2?\b|mariadb/i, hint: 'mysql' },
  { pattern: /\bsqlite3?\b|better-sqlite3/i, hint: 'sqlite' },
  { pattern: /\bmongodb\b|mongoose/i, hint: 'mongodb' },
  { pattern: /\bredis\b|ioredis/i, hint: 'redis' },
  { pattern: /\bprisma\b/i, hint: 'prisma' },
  { pattern: /\bdrizzle-orm\b/i, hint: 'drizzle' },
  { pattern: /\btypeorm\b/i, hint: 'typeorm' },
];

function detectFromPackageJson(text: string): {
  frameworks: FrameworkId[];
  databaseHints: DatabaseHint[];
} {
  const frameworks: FrameworkId[] = [];
  const databaseHints: DatabaseHint[] = [];

  let deps: Record<string, string> = {};
  try {
    const parsed = JSON.parse(text) as {
      dependencies?: Record<string, string>;
      devDependencies?: Record<string, string>;
    };
    deps = { ...parsed.dependencies, ...parsed.devDependencies };
  } catch {
    return { frameworks, databaseHints };
  }

  const names = Object.keys(deps).join(' ');

  if (/\bnext\b/.test(names)) frameworks.push('next');
  if (/\breact\b/.test(names)) frameworks.push('react');
  if (/\bvite\b/.test(names)) frameworks.push('vite');
  if (/\bvue\b/.test(names)) frameworks.push('vue');
  if (/\b@angular\b/.test(names)) frameworks.push('angular');
  if (/\bexpress\b/.test(names)) frameworks.push('express');
  if (/\bfastify\b/.test(names)) frameworks.push('fastify');
  if (/\b@nestjs\b/.test(names)) frameworks.push('nestjs');

  const depBlob = `${names} ${Object.values(deps).join(' ')}`;
  for (const { pattern, hint } of DB_DEP_PATTERNS) {
    if (pattern.test(depBlob) && !databaseHints.includes(hint)) {
      databaseHints.push(hint);
    }
  }

  return { frameworks, databaseHints };
}

function detectGoFrameworks(text: string): FrameworkId[] {
  const frameworks: FrameworkId[] = [];
  if (/github\.com\/gin-gonic\/gin/.test(text)) frameworks.push('gin');
  if (/github\.com\/labstack\/echo/.test(text)) frameworks.push('echo');
  if (/github\.com\/go-chi\/chi/.test(text)) frameworks.push('chi');
  if (/github\.com\/gorilla\/mux/.test(text)) frameworks.push('mux');
  if (/github\.com\/gofiber\/fiber/.test(text)) frameworks.push('fiber');
  return frameworks;
}

function detectPythonFrameworks(text: string): FrameworkId[] {
  const frameworks: FrameworkId[] = [];
  if (/\bdjango\b/i.test(text)) frameworks.push('django');
  if (/\bflask\b/i.test(text)) frameworks.push('flask');
  if (/\bfastapi\b/i.test(text)) frameworks.push('fastapi');
  return frameworks;
}

export const stackDetectorAnalyzer: Analyzer = {
  id: 'stack-detector',

  analyze(ctx): AnalyzerResult {
    const runtimes: RuntimeId[] = [];
    const frameworks: FrameworkId[] = [];
    const databaseHints: DatabaseHint[] = [];

    const files = ctx.listFiles();

    if (files.some((f) => f.endsWith('package.json'))) runtimes.push('node');
    if (files.some((f) => f.endsWith('go.mod')) || ctx.fileExists('go.mod')) {
      runtimes.push('go');
    }
    if (
      ctx.fileExists('requirements.txt') ||
      ctx.fileExists('pyproject.toml') ||
      files.some((f) => f.endsWith('.py'))
    ) {
      runtimes.push('python');
    }
    if (ctx.fileExists('Cargo.toml')) runtimes.push('rust');

    for (const file of files.filter((f) => f.endsWith('package.json'))) {
      const text = ctx.readText(file);
      if (!text) continue;
      const d = detectFromPackageJson(text);
      frameworks.push(...d.frameworks);
      databaseHints.push(...d.databaseHints);
    }

    const goModPath = files.find((f) => f.endsWith('go.mod'));
    const goMod = goModPath ? ctx.readText(goModPath) : ctx.readText('go.mod');
    if (goMod) frameworks.push(...detectGoFrameworks(goMod));

    for (const f of ['requirements.txt', 'pyproject.toml']) {
      const t = ctx.readText(f);
      if (t) frameworks.push(...detectPythonFrameworks(t));
    }

    const dockerCompose = ctx.readText('docker-compose.yml') ?? ctx.readText('docker-compose.yaml');
    if (dockerCompose) {
      if (/postgres/i.test(dockerCompose)) databaseHints.push('postgresql');
      if (/mysql/i.test(dockerCompose)) databaseHints.push('mysql');
      if (/mongo/i.test(dockerCompose)) databaseHints.push('mongodb');
      if (/redis/i.test(dockerCompose)) databaseHints.push('redis');
    }

    return {
      partial: {
        stack: { runtimes, frameworks, databaseHints },
      },
    };
  },
};
