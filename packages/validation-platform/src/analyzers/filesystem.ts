import type { Analyzer, AnalyzerResult } from '../types.js';
import { buildFileTree } from '../core/utils.js';

export const filesystemAnalyzer: Analyzer = {
  id: 'filesystem',

  analyze(ctx): AnalyzerResult {
    const files = ctx.listFiles();
    const fileTree = buildFileTree(files);

    return {
      partial: {
        fileTree,
        metadata: {
          fileCount: files.length,
          topLevelEntries: fileTree.map((e) => e.path),
        },
      },
    };
  },
};
