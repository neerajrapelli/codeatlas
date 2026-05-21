export type {
  AnalysisArtifacts,
  AnalysisContext,
  AnalyzeOptions,
  Analyzer,
  AnalyzerResult,
  ArchitectureSnapshot,
  DatabaseHint,
  DependencyEdge,
  DependencyGraph,
  DependencyNode,
  DetectedStack,
  FrameworkId,
  ModuleGraph,
  PackageManagerId,
  RouteEntry,
  RuntimeId,
  StackSummary,
  WorkspaceModule,
} from './types.js';

export { createAnalysisContext } from './core/context.js';
export { analyzeAndWrite, runAnalysis } from './core/orchestrator.js';
export { AnalyzerRegistry, defaultRegistry } from './plugins/registry.js';
export { writeAnalysisArtifacts } from './writers/jsonWriter.js';
export { filesystemAnalyzer } from './analyzers/filesystem.js';
export { packageManagerAnalyzer } from './analyzers/packageManager.js';
export { stackDetectorAnalyzer } from './analyzers/stackDetector.js';
export { monorepoAnalyzer } from './analyzers/monorepo.js';
export { routesAnalyzer } from './analyzers/routes.js';
