import { mkdtempSync, readFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { describe, expect, it } from 'vitest';
import { analyzeAndWrite } from './orchestrator.js';
import { SAMPLE_MONOREPO } from '../test/fixture.js';

describe('analyzeAndWrite', () => {
  it('writes all artifact files for fixture monorepo', async () => {
    const out = mkdtempSync(join(tmpdir(), 'codeatlas-validate-'));
    const artifacts = await analyzeAndWrite({
      repoPath: SAMPLE_MONOREPO,
      outputDir: out,
    });

    expect(artifacts.architecture.modules.length).toBeGreaterThanOrEqual(2);
    expect(artifacts.stackSummary.stack.packageManagers).toContain('pnpm');

    for (const file of [
      'architecture.json',
      'dependency-graph.json',
      'module-graph.json',
      'stack-summary.json',
    ]) {
      const raw = readFileSync(join(out, file), 'utf8');
      expect(() => JSON.parse(raw)).not.toThrow();
    }
  });
});
