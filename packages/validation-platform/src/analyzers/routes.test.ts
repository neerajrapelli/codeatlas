import { describe, expect, it } from 'vitest';
import { createAnalysisContext } from '../core/context.js';
import { routesAnalyzer } from './routes.js';
import { SAMPLE_MONOREPO } from '../test/fixture.js';

describe('routesAnalyzer', () => {
  it('extracts chi routes from Go API', async () => {
    const ctx = createAnalysisContext(SAMPLE_MONOREPO, '/tmp/out');
    const result = await routesAnalyzer.analyze(ctx);
    const paths = (result.partial.routes ?? []).map((r) => r.path);
    expect(paths).toContain('/health');
    expect(paths).toContain('/users');
  });
});
