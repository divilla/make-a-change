import { expect, test } from 'bun:test';
import { mkdtempSync, mkdirSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { dirname, join, resolve } from 'node:path';

const repo = resolve(import.meta.dir, '../..');
const root = join(repo, '.archon/workflows');
const paths = [...new Bun.Glob('*/*/*.yaml').scanSync(root)];
const workflows = paths.map(path => ({
  path,
  doc: Bun.YAML.parse(readFileSync(join(root, path), 'utf8')) as any,
}));
const byName = new Map(workflows.map(w => [w.doc.name, w]));

function walk(nodes: any[]): any[] {
  return nodes.flatMap(node => [node, ...walk(node.loop_group?.nodes ?? [])]);
}

function condition(script: string, cwd: string): number {
  return Bun.spawnSync(['bash', '-c', script], {
    cwd, env: { ...process.env, ARTIFACTS_DIR: cwd }, stdout: 'pipe', stderr: 'pipe',
  }).exitCode;
}

test('all ship dependencies, commands and named scripts are owned by this repo', () => {
  const seen = new Set<string>();
  function visit(name: string) {
    if (seen.has(name)) return;
    seen.add(name);
    const workflow = byName.get(name);
    expect(workflow).toBeDefined();
    for (const node of walk(workflow!.doc.nodes)) {
      if (node.include || node.workflow) visit(node.include ?? node.workflow);
      const command = node.command ?? node.loop?.command;
      if (command) {
        expect(readFileSync(join(root, dirname(workflow!.path), 'commands', `${command}.md`), 'utf8').length).toBeGreaterThan(0);
      }
      if (node.script && /^[\w-]+$/.test(node.script)) {
        expect(readFileSync(join(root, dirname(workflow!.path), 'scripts', `${node.script}.ts`), 'utf8').length).toBeGreaterThan(0);
      }
    }
  }
  visit('archon-ship');
  expect([...seen].sort()).toEqual([
    'archon-deliver', 'archon-implement', 'archon-investigate', 'archon-plan',
    'archon-pr', 'archon-review', 'archon-ship', 'archon-triage', 'archon-validate',
  ]);
  for (const name of byName.keys()) visit(name);
});

test('no local loop uses deprecated completion or interactive fields', () => {
  for (const { doc } of workflows) {
    for (const node of walk(doc.nodes)) {
      const loop = node.loop ?? node.loop_group;
      if (!loop) continue;
      expect(loop.until).toBeUndefined();
      expect(loop.interactive).toBeUndefined();
      expect(Boolean(loop.until_bash || loop.until_field)).toBe(true);
      if (loop.until_field) {
        expect(node.output_format.properties[loop.until_field].type).toBe('boolean');
        expect(node.output_format.required).toContain(loop.until_field);
      }
    }
  }
});

test('adversarial completion requires a terminal status in valid state JSON', () => {
  const node = byName.get('archon-adversarial-dev')!.doc.nodes.find((n: any) => n.loop);
  const temp = mkdtempSync(join(tmpdir(), 'archon-state-'));
  try {
    expect(condition(node.loop.until_bash, temp)).not.toBe(0);
    for (const [state, completes] of [
      ['{"status":"complete"}', true], ['{"status":"failed"}', true],
      ['{"status":"running"}', false], ['{"other":{"status":"complete"}}', false],
      ['{}', false], ['not json', false],
    ] as const) {
      writeFileSync(join(temp, 'state.json'), state);
      expect(condition(node.loop.until_bash, temp) === 0).toBe(completes);
    }
  } finally { rmSync(temp, { recursive: true, force: true }); }
});

test('counter completion checks the actual file and rejects missing or invalid counts', () => {
  const node = byName.get('archon-test-loop-dag')!.doc.nodes.find((n: any) => n.loop);
  const temp = mkdtempSync(join(tmpdir(), 'archon-counter-'));
  try {
    mkdirSync(join(temp, '.archon'));
    expect(condition(node.loop.until_bash, temp)).not.toBe(0);
    for (const [value, completes] of [['0', false], ['2', false], ['3', true], ['4', true], ['bad', false], ['', false]] as const) {
      writeFileSync(join(temp, '.archon/test-loop-dag-counter.txt'), value);
      expect(condition(node.loop.until_bash, temp) === 0).toBe(completes);
    }
  } finally { rmSync(temp, { recursive: true, force: true }); }
});

test('all three PIV gates require explicit approval and carry revision feedback forward', () => {
  const { doc, path } = byName.get('archon-piv-loop')!;
  expect(doc.interactive).toBe(true);
  for (const id of ['explore', 'refine-plan', 'fix-feedback']) {
    const group = doc.nodes.find((n: any) => n.id === id).loop_group;
    const [work, gate] = group.nodes;
    expect(gate.depends_on).toEqual([work.id]);
    expect(gate.approval.decisions.map((d: any) => d.id)).toEqual(['approve', 'revise']);
    expect(gate.approval.message).toContain(`$${work.id}.output`);
    const prompt = readFileSync(join(root, dirname(path), 'commands', `${work.command}.md`), 'utf8');
    expect(prompt).toContain(`$LOOP_PREV.${gate.id}.output.text`);
    for (const decision of ['approve', 'revise', '']) {
      const script = group.until_bash.replace(`$${gate.id}.output.decision`, decision);
      expect(condition(script, repo) === 0).toBe(decision === 'approve');
    }
  }
  const commands = join(root, dirname(path), 'commands');
  expect(readFileSync(join(commands, 'explore-work.md'), 'utf8')).toContain('$ARTIFACTS_DIR/exploration.md');
  expect(readFileSync(join(commands, 'create-plan.md'), 'utf8')).toContain('$ARTIFACTS_DIR/exploration.md');
});

test('Archon discovers every override without warnings', () => {
  const result = Bun.spawnSync(['archon', 'workflow', 'list', '--json'], { cwd: repo, stdout: 'pipe', stderr: 'pipe' });
  expect(result.exitCode).toBe(0);
  const catalog = JSON.parse(result.stdout.toString());
  expect(catalog.errors).toEqual([]);
  for (const name of byName.keys()) {
    const matches = catalog.workflows.filter((w: any) => w.name === name);
    expect(matches.length).toBe(1);
    expect(matches[0].parseWarnings ?? []).toEqual([]);
  }
});

test('forge defaults and bindings cover standalone workflows and every composed call', () => {
  const names = ['archon-ship', 'archon-upkeep', 'archon-deliver', 'archon-pr', 'archon-review'];
  let consumers = 0;
  let calls = 0;
  for (const { doc, path } of workflows) {
    if (names.includes(doc.name)) expect(doc.inputs.forge.default).toBe('gh');
    for (const node of walk(doc.nodes)) {
      if (names.includes(node.include)) {
        expect(node.with.forge).toBe('$INPUTS.forge');
        calls++;
      }
      if (!node.script || !/^[\w-]+$/.test(node.script)) continue;
      const source = readFileSync(join(root, dirname(path), 'scripts', `${node.script}.ts`), 'utf8');
      expect(source).not.toContain('process.env.ARCHON_SDLC_FORGE');
      if (source.includes('process.env.INPUTS_FORGE')) {
        expect(node.with.forge).toBe('$INPUTS.forge');
        consumers++;
      }
    }
  }
  expect(calls).toBe(6);
  expect(consumers).toBe(10);
});

test('every affected script reads the bound forge input even when the old env variable is set', () => {
  const scripts = [
    'deliver/scripts/ci-note.ts', 'deliver/scripts/check-ci.ts',
    'deliver/scripts/read-pr-body.ts', 'deliver/scripts/publish-pr-body.ts',
    'deliver/scripts/flip-ready.ts', 'pr/scripts/publish-pr.ts',
    'review/scripts/publish-review.ts',
  ];
  const temp = mkdtempSync(join(tmpdir(), 'archon-forge-'));
  try {
    // No credentialed commands can run, even if selection regresses.
    for (const script of scripts) {
      const result = Bun.spawnSync([process.execPath, join(root, 'sdlc', script)], {
        cwd: temp,
        env: {
          PATH: temp,
          INPUTS_FORGE: 'invalid-backend',
          ARCHON_SDLC_FORGE: 'gh',
          INPUTS_PR: JSON.stringify({ repo: { host: 'github.com', path: 'example/repo' }, number: 1 }),
        },
        stdout: 'pipe', stderr: 'pipe',
      });
      expect(result.stderr.toString()).toContain('forge input must be "gh" (the default) or "forge"');
      // ci-note deliberately treats a failed read as advisory; the others refuse.
      expect(result.exitCode).toBe(script.endsWith('/ci-note.ts') ? 0 : 1);
    }
  } finally { rmSync(temp, { recursive: true, force: true }); }
});

test('PIV pauses at its first real human gate before creating a plan', () => {
  const temp = mkdtempSync(join(tmpdir(), 'archon-piv-'));
  try {
    const result = Bun.spawnSync([
      'archon', 'workflow', 'run', 'archon-piv-loop', '--dry-run', '--default-stubs', '--pause-at-gates', '--json',
    ], { cwd: repo, env: { ...process.env, ARCHON_HOME: temp }, stdout: 'pipe', stderr: 'pipe' });
    expect(result.exitCode).toBe(0);
    const run = JSON.parse(result.stdout.toString());
    expect(run.outcome).toBe('paused');
    expect(run.trace.some((entry: any) => entry.nodeId === 'explore-review' && entry.state === 'paused')).toBe(true);
    expect(run.trace.some((entry: any) => entry.nodeId === 'create-plan')).toBe(false);
  } finally { rmSync(temp, { recursive: true, force: true }); }
});
