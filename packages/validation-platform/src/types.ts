/** Shared types for the validation platform analysis engine. */

export type PackageManagerId =
  | 'pnpm'
  | 'npm'
  | 'yarn'
  | 'bun'
  | 'go-mod'
  | 'pip'
  | 'poetry'
  | 'uv'
  | 'cargo'
  | 'unknown';

export type RuntimeId = 'node' | 'go' | 'python' | 'rust' | 'java' | 'dotnet' | 'unknown';

export type FrameworkId =
  | 'react'
  | 'next'
  | 'vite'
  | 'vue'
  | 'angular'
  | 'express'
  | 'fastify'
  | 'nestjs'
  | 'gin'
  | 'echo'
  | 'chi'
  | 'mux'
  | 'fiber'
  | 'django'
  | 'flask'
  | 'fastapi'
  | 'unknown';

export type DatabaseHint =
  | 'postgresql'
  | 'mysql'
  | 'sqlite'
  | 'mongodb'
  | 'redis'
  | 'prisma'
  | 'drizzle'
  | 'typeorm'
  | 'unknown';

export interface DetectedStack {
  runtimes: RuntimeId[];
  frameworks: FrameworkId[];
  packageManagers: PackageManagerId[];
  databaseHints: DatabaseHint[];
}

export interface FileTreeEntry {
  path: string;
  type: 'file' | 'directory';
  children?: FileTreeEntry[];
}

export interface WorkspaceModule {
  id: string;
  name: string;
  path: string;
  kind: 'app' | 'package' | 'service' | 'unknown';
  packageManager?: PackageManagerId;
  runtime?: RuntimeId;
}

export interface RouteEntry {
  method: string;
  path: string;
  sourceFile: string;
  framework: FrameworkId | 'next-app-router' | 'unknown';
  line?: number;
}

export interface DependencyNode {
  id: string;
  label: string;
  kind: 'workspace' | 'npm' | 'go' | 'python' | 'import';
  path?: string;
}

export interface DependencyEdge {
  from: string;
  to: string;
  kind: 'workspace' | 'declared' | 'import';
}

export interface ArchitectureSnapshot {
  repoPath: string;
  analyzedAt: string;
  rootLayout: FileTreeEntry[];
  stack: DetectedStack;
  modules: WorkspaceModule[];
  routes: RouteEntry[];
  services: string[];
  metadata: Record<string, unknown>;
}

export interface DependencyGraph {
  nodes: DependencyNode[];
  edges: DependencyEdge[];
}

export interface ModuleGraph {
  modules: WorkspaceModule[];
  edges: DependencyEdge[];
}

export interface StackSummary {
  headline: string;
  bullets: string[];
  stack: DetectedStack;
  moduleCount: number;
  routeCount: number;
}

export interface AnalysisArtifacts {
  architecture: ArchitectureSnapshot;
  dependencyGraph: DependencyGraph;
  moduleGraph: ModuleGraph;
  stackSummary: StackSummary;
}

export interface AnalyzerResult {
  partial: Partial<AnalysisArtifacts> & {
    fileTree?: FileTreeEntry[];
    stack?: Partial<DetectedStack>;
    modules?: WorkspaceModule[];
    routes?: RouteEntry[];
    dependencyNodes?: DependencyNode[];
    dependencyEdges?: DependencyEdge[];
    services?: string[];
    metadata?: Record<string, unknown>;
  };
}

export interface Analyzer {
  readonly id: string;
  analyze(ctx: AnalysisContext): Promise<AnalyzerResult> | AnalyzerResult;
}

export interface AnalysisContext {
  readonly repoPath: string;
  readonly outputDir: string;
  readonly ignoreDirs: ReadonlySet<string>;
  listFiles(relativeDir?: string): string[];
  readText(relativePath: string): string | null;
  fileExists(relativePath: string): boolean;
  resolve(relativePath: string): string;
}

export interface AnalyzeOptions {
  repoPath: string;
  outputDir: string;
  extraAnalyzers?: Analyzer[];
}
