import { expect, test } from 'bun:test';
import { existsSync, mkdtempSync, mkdirSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';

const workflow = Bun.YAML.parse(readFileSync(
  join(import.meta.dir, '../workflows/sdlc/ship/archon-ship.yaml'), 'utf8',
)) as any;
const step = workflow.nodes.find((node: any) => node.id === 'prepare-target');

function run(source: string, artifacts: string, target = '') {
  return Bun.spawnSync(['bash', '-c', step.bash], {
    env: { ...process.env, INPUTS_SPEC_PATH: source, INPUTS_TARGET: target, ARTIFACTS_DIR: artifacts },
    stdout: 'pipe', stderr: 'pipe',
  });
}

test('copies the spec, deletes its source, and returns the artifact path instead of target', () => {
  const temp = mkdtempSync(join(tmpdir(), 'mch-spec-'));
  try {
    const source = join(temp, "123-slug with 'quotes' and $variables.md");
    const artifacts = join(temp, 'run artifacts');
    mkdirSync(artifacts);
    const spec = '# Specification\n\nAcceptance: preserve this exact content.\n';
    writeFileSync(source, spec);
    const result = run(source, artifacts, 'ignored target');
    expect(result.exitCode).toBe(0);
    expect(result.stdout.toString()).toBe(join(artifacts, 'spec.md'));
    expect(readFileSync(join(artifacts, 'spec.md'), 'utf8')).toBe(spec);
    expect(existsSync(source)).toBe(false);
  } finally { rmSync(temp, { recursive: true, force: true }); }
});

test('a failed copy stops the step and retains the source', () => {
  const temp = mkdtempSync(join(tmpdir(), 'mch-spec-'));
  try {
    const source = join(temp, '123-slug.md');
    writeFileSync(source, 'accepted spec');
    const result = run(source, join(temp, 'missing-artifacts-directory'));
    expect(result.exitCode).not.toBe(0);
    expect(result.stdout.toString()).toBe('');
    expect(readFileSync(source, 'utf8')).toBe('accepted spec');
  } finally { rmSync(temp, { recursive: true, force: true }); }
});

test('a missing source fails without handing a nonexistent artifact to triage', () => {
  const temp = mkdtempSync(join(tmpdir(), 'mch-spec-'));
  try {
    const result = run(join(temp, 'missing.md'), temp);
    expect(result.exitCode).not.toBe(0);
    expect(result.stdout.toString()).toBe('');
    expect(existsSync(join(temp, 'spec.md'))).toBe(false);
  } finally { rmSync(temp, { recursive: true, force: true }); }
});

test('without spec_path the original target or empty message fallback passes through', () => {
  for (const target of ['', 'Issue #123 with spaces and $variables']) {
    const result = run('', '/dev/null', target);
    expect(result.exitCode).toBe(0);
    expect(result.stdout.toString()).toBe(target);
  }
  const triage = workflow.nodes.find((node: any) => node.id === 'triage');
  expect(workflow.inputs.spec_path.default).toBe('');
  expect(triage.depends_on).toEqual(['prepare-target']);
  expect(triage.with.target).toBe('$prepare-target.output');
});
