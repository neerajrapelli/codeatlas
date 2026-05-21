import type { Analyzer, AnalyzerResult, FrameworkId, RouteEntry } from '../types.js';

const EXPRESS_ROUTE =
  /\.(get|post|put|patch|delete|all)\s*\(\s*['"`]([^'"`]+)['"`]/gi;
const FASTIFY_ROUTE =
  /\.(get|post|put|patch|delete)\s*\(\s*['"`]([^'"`]+)['"`]/gi;
const CHI_ROUTE = /\.(Get|Post|Put|Patch|Delete)\s*\(\s*"([^"]+)"/g;
const MUX_ROUTE = /\.Handle(?:Func)?\s*\(\s*"([^"]+)"/g;
const GIN_ROUTE = /\.(GET|POST|PUT|PATCH|DELETE)\s*\(\s*"([^"]+)"/g;

const SCANNABLE_EXT = new Set([
  '.ts',
  '.tsx',
  '.js',
  '.jsx',
  '.go',
  '.py',
]);

function scanFileRoutes(
  relativePath: string,
  content: string,
  framework: FrameworkId | RouteEntry['framework'],
): RouteEntry[] {
  const routes: RouteEntry[] = [];
  const lines = content.split('\n');

  const addMatches = (
    regex: RegExp,
    methodIndex: number,
    pathIndex: number,
    defaultMethod = 'GET',
  ) => {
    for (let i = 0; i < lines.length; i++) {
      const line = lines[i] ?? '';
      regex.lastIndex = 0;
      let m: RegExpExecArray | null;
      while ((m = regex.exec(line)) !== null) {
        const method = (m[methodIndex] ?? defaultMethod).toUpperCase();
        const path = m[pathIndex] ?? '/';
        routes.push({
          method,
          path,
          sourceFile: relativePath,
          framework,
          line: i + 1,
        });
      }
    }
  };

  if (/\.(ts|js)x?$/.test(relativePath)) {
    addMatches(EXPRESS_ROUTE, 1, 2);
    addMatches(FASTIFY_ROUTE, 1, 2);
  }

  if (relativePath.endsWith('.go')) {
    addMatches(CHI_ROUTE, 1, 2);
    addMatches(MUX_ROUTE, 0, 1);
    addMatches(GIN_ROUTE, 1, 2);
  }

  return routes;
}

function detectNextAppRouter(ctx: import('../types.js').AnalysisContext): RouteEntry[] {
  const routes: RouteEntry[] = [];
  const appFiles = ctx
    .listFiles()
    .filter(
      (f) =>
        (f.includes('/app/') || f.startsWith('app/')) &&
        (f.endsWith('/page.tsx') ||
          f.endsWith('/page.ts') ||
          f.endsWith('/page.jsx') ||
          f.endsWith('/page.js') ||
          f.endsWith('/route.ts') ||
          f.endsWith('/route.js')),
    );

  for (const file of appFiles) {
    const routePath = file
      .replace(/\\/g, '/')
      .replace(/\/app\//, '/')
      .replace(/^(src\/)?app\//, '/')
      .replace(/\/page\.(tsx|ts|jsx|js)$/, '')
      .replace(/\/route\.(tsx|ts|jsx|js)$/, '')
      .replace(/\/\([^)]+\)/g, '')
      .replace(/\/\[[^\]]+\]/g, '/:param')
      || '/';

    const isApiRoute = file.includes('/route.');
    routes.push({
      method: isApiRoute ? 'HANDLER' : 'PAGE',
      path: routePath === '' ? '/' : routePath,
      sourceFile: file,
      framework: 'next-app-router',
    });
  }

  return routes;
}

export const routesAnalyzer: Analyzer = {
  id: 'routes',

  analyze(ctx): AnalyzerResult {
    const routes: RouteEntry[] = [];
    routes.push(...detectNextAppRouter(ctx));

    const codeFiles = ctx
      .listFiles()
      .filter((f) => SCANNABLE_EXT.has(f.slice(f.lastIndexOf('.'))));

    let framework: FrameworkId = 'unknown';
    const goMod = ctx.readText('go.mod') ?? '';
    if (/chi/.test(goMod)) framework = 'chi';
    else if (/gin-gonic/.test(goMod)) framework = 'gin';
    else if (/gorilla\/mux/.test(goMod)) framework = 'mux';

    for (const file of codeFiles.slice(0, 500)) {
      const text = ctx.readText(file);
      if (!text) continue;
      if (/from ['"]express['"]|require\(['"]express['"]\)/.test(text)) {
        framework = 'express';
      }
      if (/from ['"]fastify['"]|require\(['"]fastify['"]\)/.test(text)) {
        framework = 'fastify';
      }
      routes.push(...scanFileRoutes(file, text, framework));
    }

    const importEdges = buildImportEdges(ctx, codeFiles.slice(0, 300));

    return {
      partial: {
        routes,
        dependencyEdges: importEdges,
        dependencyNodes: importEdges.flatMap((e) => [
          { id: e.from, label: e.from, kind: 'import' as const },
          { id: e.to, label: e.to, kind: 'import' as const },
        ]),
      },
    };
  },
};

function buildImportEdges(
  ctx: import('../types.js').AnalysisContext,
  files: string[],
): import('../types.js').DependencyEdge[] {
  const importRe =
    /(?:import|export)\s+(?:[^'";]*\s+from\s+)?['"]([^'"]+)['"]|require\s*\(\s*['"]([^'"]+)['"]\s*\)/g;
  const edges: import('../types.js').DependencyEdge[] = [];

  for (const file of files) {
    if (!/\.(ts|tsx|js|jsx)$/.test(file)) continue;
    const text = ctx.readText(file);
    if (!text) continue;

    let m: RegExpExecArray | null;
    importRe.lastIndex = 0;
    while ((m = importRe.exec(text)) !== null) {
      const target = m[1] ?? m[2];
      if (!target || (!target.startsWith('.') && !target.startsWith('@/'))) continue;
      edges.push({
        from: file,
        to: target,
        kind: 'import',
      });
    }
  }

  return edges;
}
