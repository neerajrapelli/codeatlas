import { readFile, writeFile } from 'node:fs/promises';
import path from 'node:path';

const cwd = process.cwd();
const beforePath = process.argv[2] ?? 'artifacts/base-module-graph.json';
const afterPath = process.argv[3] ?? 'artifacts/module-graph.json';
const outputPath = process.argv[4] ?? 'artifacts/module-graph-diff.json';

function edgeKey(edge) {
  return `${edge.from}->${edge.to}`;
}

const before = JSON.parse(await readFile(path.resolve(cwd, beforePath), 'utf8'));
const after = JSON.parse(await readFile(path.resolve(cwd, afterPath), 'utf8'));

const beforeEdges = new Map((before.edges ?? []).map((e) => [edgeKey(e), e]));
const afterEdges = new Map((after.edges ?? []).map((e) => [edgeKey(e), e]));

const added = [];
const removed = [];
const changed = [];

for (const [key, edge] of afterEdges) {
  if (!beforeEdges.has(key)) {
    added.push(edge);
    continue;
  }
  const prev = beforeEdges.get(key);
  if ((prev.count ?? 0) !== (edge.count ?? 0)) {
    changed.push({ from: edge.from, to: edge.to, before: prev.count ?? 0, after: edge.count ?? 0 });
  }
}

for (const [key, edge] of beforeEdges) {
  if (!afterEdges.has(key)) removed.push(edge);
}

const diff = {
  generatedAt: new Date().toISOString(),
  before: beforePath,
  after: afterPath,
  summary: {
    addedEdges: added.length,
    removedEdges: removed.length,
    changedWeights: changed.length,
  },
  added,
  removed,
  changed,
};

await writeFile(path.resolve(cwd, outputPath), JSON.stringify(diff, null, 2));
console.log(`Wrote ${outputPath}`);
