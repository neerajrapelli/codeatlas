import type {
  AnalysisArtifacts,
  AnalysisContext,
  AnalyzeOptions,
  AnalyzerResult,
  ArchitectureSnapshot,
  DependencyEdge,
  DependencyGraph,
  DependencyNode,
  DetectedStack,
  ModuleGraph,
  StackSummary,
  WorkspaceModule,
} from '../types.js';
import { createAnalysisContext } from './context.js';
import { buildFileTree, mergeStack, unique } from './utils.js';
import { AnalyzerRegistry, defaultRegistry } from '../plugins/registry.js';
import { writeAnalysisArtifacts } from '../writers/jsonWriter.js';

function emptyStack(): DetectedStack {
  return {
    runtimes: [],
    frameworks: [],
    packageManagers: [],
    databaseHints: [],
  };
}

function mergeResults(results: AnalyzerResult[]): {
  fileTree: ReturnType<typeof buildFileTree>;
  stack: DetectedStack;
  modules: WorkspaceModule[];
  routes: ArchitectureSnapshot['routes'];
  dependencyNodes: DependencyNode[];
  dependencyEdges: DependencyEdge[];
  services: string[];
  metadata: Record<string, unknown>;
} {
  let stack = emptyStack();
  const modules: WorkspaceModule[] = [];
  const routes: ArchitectureSnapshot['routes'] = [];
  const dependencyNodes: DependencyNode[] = [];
  const dependencyEdges: DependencyEdge[] = [];
  const services: string[] = [];
  const metadata: Record<string, unknown> = {};
  let fileTree = buildFileTree([]);

  for (const { partial } of results) {
    if (partial.stack) {
      stack = mergeStack(stack, partial.stack);
    }
    if (partial.modules) modules.push(...partial.modules);
    if (partial.routes) routes.push(...partial.routes);
    if (partial.dependencyNodes) dependencyNodes.push(...partial.dependencyNodes);
    if (partial.dependencyEdges) dependencyEdges.push(...partial.dependencyEdges);
    if (partial.services) services.push(...partial.services);
    if (partial.metadata) Object.assign(metadata, partial.metadata);
    if (partial.fileTree) fileTree = partial.fileTree;
  }

  return {
    fileTree,
    stack,
    modules: dedupeModules(modules),
    routes,
    dependencyNodes: dedupeNodes(dependencyNodes),
    dependencyEdges: dedupeEdges(dependencyEdges),
    services: unique(services),
    metadata,
  };
}

function dedupeModules(modules: WorkspaceModule[]): WorkspaceModule[] {
  const seen = new Map<string, WorkspaceModule>();
  for (const m of modules) {
    seen.set(m.path, m);
  }
  return [...seen.values()];
}

function dedupeNodes(nodes: DependencyNode[]): DependencyNode[] {
  const seen = new Map<string, DependencyNode>();
  for (const n of nodes) {
    seen.set(n.id, n);
  }
  return [...seen.values()];
}

function dedupeEdges(edges: DependencyEdge[]): DependencyEdge[] {
  const seen = new Set<string>();
  const out: DependencyEdge[] = [];
  for (const e of edges) {
    const key = `${e.from}|${e.to}|${e.kind}`;
    if (seen.has(key)) continue;
    seen.add(key);
    out.push(e);
  }
  return out;
}

function buildStackSummary(
  stack: DetectedStack,
  modules: WorkspaceModule[],
  routes: ArchitectureSnapshot['routes'],
): StackSummary {
  const parts: string[] = [];
  if (stack.runtimes.length) parts.push(`Runtimes: ${stack.runtimes.join(', ')}`);
  if (stack.frameworks.length) parts.push(`Frameworks: ${stack.frameworks.join(', ')}`);
  if (stack.packageManagers.length) {
    parts.push(`Package managers: ${stack.packageManagers.join(', ')}`);
  }
  if (stack.databaseHints.length) {
    parts.push(`Database hints: ${stack.databaseHints.join(', ')}`);
  }

  const headline =
    parts.length > 0 ? parts[0]! : 'No dominant stack detected';

  const bullets = [
    ...parts.slice(1),
    `${modules.length} workspace module(s)`,
    `${routes.length} route(s) detected (heuristic)`,
  ];

  return {
    headline,
    bullets,
    stack,
    moduleCount: modules.length,
    routeCount: routes.length,
  };
}

function resolveRegistry(options: AnalyzeOptions): AnalyzerRegistry {
  if (!options.extraAnalyzers?.length) return defaultRegistry;
  return new AnalyzerRegistry([...defaultRegistry.list(), ...options.extraAnalyzers]);
}

export async function runAnalysis(
  options: AnalyzeOptions,
  registry?: AnalyzerRegistry,
): Promise<AnalysisArtifacts> {
  const ctx = createAnalysisContext(options.repoPath, options.outputDir);
  const reg = registry ?? resolveRegistry(options);
  const analyzers = reg.list();

  const results: AnalyzerResult[] = [];
  for (const analyzer of analyzers) {
    const result = await analyzer.analyze(ctx);
    results.push(result);
  }

  const merged = mergeResults(results);
  const moduleEdges = merged.dependencyEdges.filter((e) => e.kind === 'workspace');

  const architecture: ArchitectureSnapshot = {
    repoPath: ctx.repoPath,
    analyzedAt: new Date().toISOString(),
    rootLayout: merged.fileTree,
    stack: merged.stack,
    modules: merged.modules,
    routes: merged.routes,
    services: merged.services,
    metadata: merged.metadata,
  };

  const dependencyGraph: DependencyGraph = {
    nodes: merged.dependencyNodes,
    edges: merged.dependencyEdges,
  };

  const moduleGraph: ModuleGraph = {
    modules: merged.modules,
    edges: moduleEdges,
  };

  const stackSummary = buildStackSummary(
    merged.stack,
    merged.modules,
    merged.routes,
  );

  return {
    architecture,
    dependencyGraph,
    moduleGraph,
    stackSummary,
  };
}

export async function analyzeAndWrite(
  options: AnalyzeOptions,
  registry?: AnalyzerRegistry,
): Promise<AnalysisArtifacts> {
  const artifacts = await runAnalysis(options, registry);
  const ctx = createAnalysisContext(options.repoPath, options.outputDir);
  writeAnalysisArtifacts(ctx.outputDir, artifacts);
  return artifacts;
}
