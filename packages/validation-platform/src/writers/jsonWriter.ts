import { mkdirSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import type { AnalysisArtifacts } from '../types.js';

export function writeAnalysisArtifacts(
  outputDir: string,
  artifacts: AnalysisArtifacts,
): void {
  mkdirSync(outputDir, { recursive: true });

  const files: Array<[string, unknown]> = [
    ['architecture.json', artifacts.architecture],
    ['dependency-graph.json', artifacts.dependencyGraph],
    ['module-graph.json', artifacts.moduleGraph],
    ['stack-summary.json', artifacts.stackSummary],
  ];

  for (const [name, data] of files) {
    const path = join(outputDir, name);
    writeFileSync(path, `${JSON.stringify(data, null, 2)}\n`, 'utf8');
  }
}
