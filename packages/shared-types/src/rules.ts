export type ArchitectureRuleType = 'no_import' | 'must_import' | 'layer_order' | 'no_circular';
export type RuleSeverity = 'error' | 'warning' | 'info';

export interface ArchitectureRule {
  readonly id: string;
  readonly repositoryId: number;
  readonly name: string;
  readonly description?: string;
  readonly ruleType: ArchitectureRuleType;
  readonly sourcePattern: string;
  readonly targetPattern: string;
  readonly severity: RuleSeverity;
  readonly enabled: boolean;
  readonly createdAt: string;
}

export interface RuleViolation {
  readonly id?: string;
  readonly ruleId: string;
  readonly ruleName: string;
  readonly sourceFile: string;
  readonly targetFile: string;
  readonly severity: RuleSeverity;
  readonly message: string;
  readonly detectedAt?: string;
}
