import { describe, expect, it } from 'vitest';
import { createAnalysisContext } from '../core/context.js';
import { stackDetectorAnalyzer } from './stackDetector.js';
import { SAMPLE_MONOREPO } from '../test/fixture.js';

describe('stackDetectorAnalyzer', () => {
  it('detects node, go, react, vite, chi', async () => {
    const ctx = createAnalysisContext(SAMPLE_MONOREPO, '/tmp/out');
    const result = await stackDetectorAnalyzer.analyze(ctx);
    const stack = result.partial.stack!;
    expect(stack.runtimes).toContain('node');
    expect(stack.runtimes).toContain('go');
    expect(stack.frameworks).toContain('react');
    expect(stack.frameworks).toContain('vite');
    expect(stack.frameworks).toContain('chi');
  });

  it('detects postgres and redis from docker-compose', async () => {
    const ctx = createAnalysisContext(SAMPLE_MONOREPO, '/tmp/out');
    const result = await stackDetectorAnalyzer.analyze(ctx);
    expect(result.partial.stack?.databaseHints).toContain('postgresql');
    expect(result.partial.stack?.databaseHints).toContain('redis');
  });
});
