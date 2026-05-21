/** API and domain types shared across web, graph helpers, and contract tests. */

export type {
  BlastRadiusAffectedFile,
  BlastRadiusResult,
  BlastRadiusSummary,
  BlastRadiusTarget,
} from './blast';
export type {
  ArchitectureRule,
  ArchitectureRuleType,
  RuleSeverity,
  RuleViolation,
} from './rules';
export type { McpManifest, McpToolCallLog } from './mcp';

export type HealthStatus = 'ok' | 'degraded';

export interface HealthResponse {
  readonly service: string;
  readonly status: HealthStatus;
  readonly version: string;
}

/** Symbol kinds aligned with Tree-sitter / TypeScript concepts (MVP subset). */
export type SymbolKind =
  | 'file'
  | 'module'
  | 'namespace'
  | 'class'
  | 'interface'
  | 'type_alias'
  | 'enum'
  | 'function'
  | 'method'
  | 'property'
  | 'variable'
  | 'parameter'
  | 'import'
  | 'export';

export interface SourceLocation {
  readonly path: string;
  readonly startLine: number;
  readonly startColumn: number;
  readonly endLine: number;
  readonly endColumn: number;
}

export interface CodeSymbol {
  readonly id: string;
  readonly name: string;
  readonly kind: SymbolKind;
  readonly location: SourceLocation;
}

export type DependencyEdgeKind = 'imports' | 'extends' | 'implements' | 'references';

export interface DependencyEdge {
  readonly id: string;
  readonly fromSymbolId: string;
  readonly toSymbolId: string;
  readonly kind: DependencyEdgeKind;
}
