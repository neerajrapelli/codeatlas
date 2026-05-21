import { describe, expect, it } from 'vitest';
import { createAnalysisContext } from '../core/context.js';
import { packageManagerAnalyzer } from './packageManager.js';
import { SAMPLE_MONOREPO } from '../test/fixture.js';

describe('packageManagerAnalyzer', () => {
  it('detects pnpm from lockfile', async () => {
    const ctx = createAnalysisContext(SAMPLE_MONOREPO, '/tmp/out');
    const result = await packageManagerAnalyzer.analyze(ctx);
    expect(result.partial.stack?.packageManagers).toContain('pnpm');
  });
});
