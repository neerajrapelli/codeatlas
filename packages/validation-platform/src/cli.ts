#!/usr/bin/env node
import { parseArgs } from 'node:util';
import { resolve } from 'node:path';
import { analyzeAndWrite } from './core/orchestrator.js';

async function main(): Promise<void> {
  const { values, positionals } = parseArgs({
    allowPositionals: true,
    options: {
      repo: { type: 'string', short: 'r' },
      out: { type: 'string', short: 'o' },
      help: { type: 'boolean', short: 'h' },
    },
  });

  if (values.help) {
    printHelp();
    process.exit(0);
  }

  const command = positionals[0];
  if (command !== 'analyze') {
    console.error(`Unknown command: ${command ?? '(none)'}`);
    printHelp();
    process.exit(1);
  }

  const repo = values.repo;
  if (!repo) {
    console.error('Missing required --repo <path>');
    printHelp();
    process.exit(1);
  }

  const repoPath = resolve(repo);
  const outputDir = resolve(values.out ?? './validation-output');

  console.error(`Analyzing ${repoPath} → ${outputDir}`);

  const artifacts = await analyzeAndWrite({ repoPath, outputDir });

  console.error(
    `Done. ${artifacts.stackSummary.moduleCount} modules, ${artifacts.stackSummary.routeCount} routes.`,
  );
  console.error(`Wrote: architecture.json, dependency-graph.json, module-graph.json, stack-summary.json`);
}

function printHelp(): void {
  console.log(`codeatlas-validate — CodeAtlas validation platform (Phase 1)

Usage:
  codeatlas-validate analyze --repo <path> [--out <dir>]

Options:
  --repo, -r   Repository root to analyze (required)
  --out,  -o   Output directory (default: ./validation-output)
  --help, -h   Show this help

Examples:
  pnpm --filter @codeatlas/validation-platform analyze -- --repo . --out ./out
  pnpm --filter @codeatlas/validation-platform analyze:payload
`);
}

main().catch((err: unknown) => {
  console.error(err instanceof Error ? err.message : err);
  process.exit(1);
});
