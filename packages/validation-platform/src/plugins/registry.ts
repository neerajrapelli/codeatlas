import type { Analyzer } from '../types.js';
import { filesystemAnalyzer } from '../analyzers/filesystem.js';
import { monorepoAnalyzer } from '../analyzers/monorepo.js';
import { packageManagerAnalyzer } from '../analyzers/packageManager.js';
import { routesAnalyzer } from '../analyzers/routes.js';
import { stackDetectorAnalyzer } from '../analyzers/stackDetector.js';

const builtInAnalyzers: Analyzer[] = [
  filesystemAnalyzer,
  packageManagerAnalyzer,
  stackDetectorAnalyzer,
  monorepoAnalyzer,
  routesAnalyzer,
];

export class AnalyzerRegistry {
  private readonly analyzers = new Map<string, Analyzer>();

  constructor(initial: Analyzer[] = builtInAnalyzers) {
    for (const a of initial) {
      this.register(a);
    }
  }

  register(analyzer: Analyzer): void {
    if (this.analyzers.has(analyzer.id)) {
      throw new Error(`Analyzer already registered: ${analyzer.id}`);
    }
    this.analyzers.set(analyzer.id, analyzer);
  }

  list(): Analyzer[] {
    return [...this.analyzers.values()];
  }

  get(id: string): Analyzer | undefined {
    return this.analyzers.get(id);
  }
}

export const defaultRegistry = new AnalyzerRegistry();
