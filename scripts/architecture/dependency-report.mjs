import { mkdir, readFile, writeFile } from 'node:fs/promises';
import { spawn } from 'node:child_process';
import path from 'node:path';

const ROOT = process.cwd();
const ARTIFACT_DIR = path.join(ROOT, 'artifacts');
const depCruiseOutput = path.join(ARTIFACT_DIR, 'dependency-graph.json');
const moduleGraphOutput = path.join(ARTIFACT_DIR, 'module-graph.json');
const architectureOutput = path.join(ARTIFACT_DIR, 'architecture.json');

function run(command, args) {
  return new Promise((resolve, reject) => {
    const child = spawn(command, args, { stdio: ['ignore', 'pipe', 'pipe'], shell: true });
    let stdout = '';
    let stderr = '';
    child.stdout.on('data', (d) => {
      stdout += d.toString();
      process.stdout.write(d);
    });
    child.stderr.on('data', (d) => {
      stderr += d.toString();
      process.stderr.write(d);
    });
    child.on('error', reject);
    child.on('close', (code) => {
      if (code === 0) resolve(stdout);
      else reject(new Error(`${command} failed (${code}): ${stderr}`));
    });
  });
}

function moduleName(filePath) {
  const normalized = filePath.replace(/\\/g, '/');
  const parts = normalized.split('/');
  if (parts.length < 2) return normalized;
  if (parts[0] === 'apps' || parts[0] === 'packages') return `${parts[0]}/${parts[1]}`;
  return parts.slice(0, 2).join('/');
}

function buildModuleGraph(depCruiseJson) {
  const nodes = new Map();
  const edgeMap = new Map();
  for (const mod of depCruiseJson.modules ?? []) {
    const fromModule = moduleName(mod.source);
    nodes.set(fromModule, { id: fromModule });
    for (const dep of mod.dependencies ?? []) {
      if (!dep.resolved) continue;
      const toModule = moduleName(dep.resolved);
      nodes.set(toModule, { id: toModule });
      if (fromModule === toModule) continue;
      const key = `${fromModule}->${toModule}`;
      edgeMap.set(key, {
        from: fromModule,
        to: toModule,
        count: (edgeMap.get(key)?.count ?? 0) + 1,
      });
    }
  }
  return {
    generatedAt: new Date().toISOString(),
    nodeCount: nodes.size,
    edgeCount: edgeMap.size,
    nodes: [...nodes.values()].sort((a, b) => a.id.localeCompare(b.id)),
    edges: [...edgeMap.values()].sort((a, b) => `${a.from}|${a.to}`.localeCompare(`${b.from}|${b.to}`)),
  };
}

await mkdir(ARTIFACT_DIR, { recursive: true });

await run('depcruise', [
  '--config',
  '.dependency-cruiser.cjs',
  '--output-type',
  'json',
  '--output-to',
  depCruiseOutput,
  '--include-only',
  '^(apps|packages)',
  '.',
]);

const depCruiseJson = JSON.parse(await readFile(depCruiseOutput, 'utf8'));
const moduleGraph = buildModuleGraph(depCruiseJson);

await writeFile(moduleGraphOutput, JSON.stringify(moduleGraph, null, 2));
await writeFile(
  architectureOutput,
  JSON.stringify(
    {
      generatedAt: new Date().toISOString(),
      source: 'dependency-cruiser',
      dependencyGraphFile: path.relative(ROOT, depCruiseOutput),
      moduleGraphFile: path.relative(ROOT, moduleGraphOutput),
      summary: {
        modules: moduleGraph.nodeCount,
        edges: moduleGraph.edgeCount,
        forbiddenViolations: depCruiseJson.summary?.violations?.length ?? 0,
      },
    },
    null,
    2,
  ),
);

console.log(`Wrote ${path.relative(ROOT, depCruiseOutput)}`);
console.log(`Wrote ${path.relative(ROOT, moduleGraphOutput)}`);
console.log(`Wrote ${path.relative(ROOT, architectureOutput)}`);
