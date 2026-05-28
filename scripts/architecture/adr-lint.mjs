import { readFile } from 'node:fs/promises';
import path from 'node:path';
import { glob } from 'node:fs/promises';

const root = process.cwd();
const adrDir = path.join(root, 'docs', 'adr');
const requiredHeadings = ['#', '## Status', '## Context', '## Decision', '## Consequences'];

let hasError = false;

for await (const filePath of glob(path.join(adrDir, '*.md'))) {
  const content = await readFile(filePath, 'utf8');
  const rel = path.relative(root, filePath);
  for (const heading of requiredHeadings) {
    if (!content.includes(heading)) {
      console.error(`${rel}: missing section "${heading}"`);
      hasError = true;
    }
  }
}

if (hasError) {
  process.exit(1);
}

console.log('ADR lint passed');
