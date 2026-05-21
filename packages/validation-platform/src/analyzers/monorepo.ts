import type {
  Analyzer,
  AnalyzerResult,
  DependencyEdge,
  DependencyNode,
  PackageManagerId,
  RuntimeId,
  WorkspaceModule,
} from '../types.js';

function inferKind(path: string): WorkspaceModule['kind'] {
  if (path.startsWith('apps/')) return 'app';
  if (path.startsWith('packages/')) return 'package';
  if (path.includes('services/')) return 'service';
  return 'unknown';
}

function parsePnpmWorkspace(text: string): string[] {
  const paths: string[] = [];
  let inPackages = false;
  for (const line of text.split('\n')) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith('#')) continue;
    if (trimmed === 'packages:') {
      inPackages = true;
      continue;
    }
    if (!inPackages) continue;
    if (!trimmed.startsWith('-')) break;
    const match = trimmed.match(/^-\s*['"]?([^'"]+)['"]?/);
    if (match?.[1]) paths.push(match[1]);
  }
  return paths;
}

function resolveWorkspacePackage(
  ctx: import('../types.js').AnalysisContext,
  basePath: string,
): WorkspaceModule | null {
  const pkgPath = basePath.endsWith('package.json')
    ? basePath
    : `${basePath.replace(/\/$/, '')}/package.json`;

  if (!ctx.fileExists(pkgPath)) {
    const goMod = basePath.endsWith('go.mod') ? basePath : `${basePath}/go.mod`;
    if (ctx.fileExists(goMod)) {
      const goText = ctx.readText(goMod);
      const nameMatch = goText?.match(/^module\s+(\S+)/m);
      return {
        id: basePath,
        name: nameMatch?.[1] ?? basePath,
        path: basePath,
        kind: inferKind(basePath),
        runtime: 'go',
        packageManager: 'go-mod',
      };
    }
    return null;
  }

  const text = ctx.readText(pkgPath);
  if (!text) return null;

  let name = basePath;
  let runtime: RuntimeId = 'node';
  let packageManager: PackageManagerId | undefined;

  try {
    const parsed = JSON.parse(text) as { name?: string };
    if (parsed.name) name = parsed.name;
  } catch {
    /* keep defaults */
  }

  if (ctx.fileExists(`${basePath}/pnpm-lock.yaml`) || ctx.fileExists('pnpm-lock.yaml')) {
    packageManager = 'pnpm';
  }

  return {
    id: basePath,
    name,
    path: basePath,
    kind: inferKind(basePath),
    runtime,
    packageManager,
  };
}

function expandGlobRoots(ctx: import('../types.js').AnalysisContext, pattern: string): string[] {
  const base = pattern.replace(/\/\*$/, '').replace(/\*$/, '');
  const files = ctx.listFiles();
  const dirs = new Set<string>();

  for (const f of files) {
    if (f.startsWith(`${base}/`)) {
      const parts = f.split('/');
      if (parts.length >= 2) dirs.add(`${parts[0]}/${parts[1]}`);
    }
  }

  if (ctx.fileExists(`${base}/package.json`) || ctx.fileExists(`${base}/go.mod`)) {
    dirs.add(base);
  }

  return [...dirs];
}

export const monorepoAnalyzer: Analyzer = {
  id: 'monorepo',

  analyze(ctx): AnalyzerResult {
    const modules: WorkspaceModule[] = [];
    const dependencyNodes: DependencyNode[] = [];
    const dependencyEdges: DependencyEdge[] = [];
    const services: string[] = [];

    const wsText = ctx.readText('pnpm-workspace.yaml');
    if (wsText) {
      const roots = parsePnpmWorkspace(wsText);
      for (const root of roots) {
        if (root.endsWith('/*')) {
          for (const dir of expandGlobRoots(ctx, root)) {
            const mod = resolveWorkspacePackage(ctx, dir);
            if (mod) modules.push(mod);
          }
        } else {
          const mod = resolveWorkspacePackage(ctx, root);
          if (mod) modules.push(mod);
        }
      }
    }

    for (const dir of ['apps', 'packages', 'services']) {
      if (!ctx.fileExists(dir)) continue;
      for (const sub of expandGlobRoots(ctx, dir)) {
        const mod = resolveWorkspacePackage(ctx, sub);
        if (mod && !modules.some((m) => m.path === mod.path)) {
          modules.push(mod);
        }
      }
    }

    for (const mod of modules) {
      dependencyNodes.push({
        id: mod.id,
        label: mod.name,
        kind: 'workspace',
        path: mod.path,
      });

      const pkgJson = ctx.readText(`${mod.path}/package.json`);
      if (!pkgJson) continue;

      try {
        const parsed = JSON.parse(pkgJson) as {
          dependencies?: Record<string, string>;
          devDependencies?: Record<string, string>;
        };
        const deps = { ...parsed.dependencies, ...parsed.devDependencies };
        for (const [dep, version] of Object.entries(deps ?? {})) {
          if (!version.startsWith('workspace:')) continue;
          const target = modules.find((m) => m.name === dep);
          if (target) {
            dependencyEdges.push({
              from: mod.id,
              to: target.id,
              kind: 'workspace',
            });
          }
        }
      } catch {
        /* skip malformed */
      }
    }

    if (ctx.fileExists('docker-compose.yml') || ctx.fileExists('docker-compose.yaml')) {
      services.push('docker-compose');
    }

    return {
      partial: {
        modules,
        dependencyNodes,
        dependencyEdges,
        services,
        metadata: {
          monorepo: modules.length > 1,
          turbo: ctx.fileExists('turbo.json'),
        },
      },
    };
  },
};
