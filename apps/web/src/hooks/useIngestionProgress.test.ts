import { describe, expect, it } from 'vitest';

describe('useIngestionProgress', () => {
  it('exports the hook function', async () => {
    const mod = await import('./useIngestionProgress');
    expect(typeof mod.useIngestionProgress).toBe('function');
  });
});
