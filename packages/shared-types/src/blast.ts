/** Blast radius analysis (inbound dependency fan-in). */

export interface BlastRadiusTarget {
  readonly file_path: string;
  readonly symbol?: string;
  readonly owner?: string;
  readonly bus_factor_score: number;
}

export interface BlastRadiusSummary {
  readonly direct_dependents: number;
  readonly transitive_dependents: number;
  readonly total_files_affected: number;
  readonly risk_score: number;
  readonly teams_affected: readonly string[];
}

export interface BlastRadiusAffectedFile {
  readonly file_path: string;
  readonly depth: number;
  readonly relationship: 'direct_import' | 'transitive' | string;
  readonly owner?: string;
  readonly has_tests: boolean;
  readonly risk_level: string;
}

export interface BlastRadiusResult {
  readonly target: BlastRadiusTarget;
  readonly blast_radius: BlastRadiusSummary;
  readonly files: readonly BlastRadiusAffectedFile[];
  readonly warnings: readonly string[];
}
