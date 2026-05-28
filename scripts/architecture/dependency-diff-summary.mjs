import { readFile, writeFile } from 'node:fs/promises';

const diffPath = process.argv[2] ?? 'artifacts/module-graph-diff.json';
const outPath = process.argv[3] ?? 'artifacts/architecture-pr-summary.md';

const diff = JSON.parse(await readFile(diffPath, 'utf8'));

const lines = [
  '## Architecture Dependency Diff',
  '',
  `- Added edges: **${diff.summary?.addedEdges ?? 0}**`,
  `- Removed edges: **${diff.summary?.removedEdges ?? 0}**`,
  `- Changed edge weights: **${diff.summary?.changedWeights ?? 0}**`,
  '',
];

if ((diff.added ?? []).length > 0) {
  lines.push('### Added (top 10)', '');
  for (const edge of diff.added.slice(0, 10)) {
    lines.push(`- \`${edge.from}\` -> \`${edge.to}\` (${edge.count ?? 0})`);
  }
  lines.push('');
}

if ((diff.removed ?? []).length > 0) {
  lines.push('### Removed (top 10)', '');
  for (const edge of diff.removed.slice(0, 10)) {
    lines.push(`- \`${edge.from}\` -> \`${edge.to}\` (${edge.count ?? 0})`);
  }
  lines.push('');
}

await writeFile(outPath, lines.join('\n'));
console.log(`Wrote ${outPath}`);
