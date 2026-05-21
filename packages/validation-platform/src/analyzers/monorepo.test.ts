import { describe, expect, it } from 'vitest';
import { createAnalysisContext } from '../core/context.js';
import { monorepoAnalyzer } from './monorepo.js';
import { SAMPLE_MONOREPO } from '../test/fixture.js';

describe('monorepoAnalyzer', () => {
  it('discovers workspace modules and edges', async () => {
    const ctx = createAnalysisContext(SAMPLE_MONOREPO, '/tmp/out');
    const result = await monorepoAnalyzer.analyze(ctx);
    const modules = result.partial.modules ?? [];
    const names = modules.map((m) => m.name);
    expect(names).toContain('@sample/web');
    expect(names).toContain('@sample/shared');

    const edges = result.partial.dependencyEdges ?? [];
    const workspaceEdge = edges.find(
      (e) => e.kind === 'workspace' && e.from.includes('web') && e.to.includes('shared'),
    );
    expect(workspaceEdge).toBeDefined();
  });
});
