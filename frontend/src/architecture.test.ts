import { ESLint } from 'eslint';
import { describe, expect, it } from 'vitest';

const frontendRoot = process.cwd();

async function ruleIds(filePath: string, source: string): Promise<(string | null)[]> {
  const eslint = new ESLint({ cwd: frontendRoot });
  const [result] = await eslint.lintText(source, { filePath: `${frontendRoot}/${filePath}` });
  return result?.messages.map((message) => message.ruleId) ?? [];
}

describe('frontend architecture boundaries', () => {
  it('rejects shared imports from features and pages', async () => {
    await expect(
      ruleIds('src/shared/fixture.js', "import '@/features/projects/api/projectApi';\n"),
    ).resolves.toContain('no-restricted-imports');
    await expect(
      ruleIds('src/shared/fixture.js', "import '../../pages/index.vue';\n"),
    ).resolves.toContain('no-restricted-imports');
  });

  it('allows features and pages to import their supported dependencies', async () => {
    await expect(
      ruleIds('src/features/projects/fixture.js', "import '@/shared/api/httpClient';\n"),
    ).resolves.not.toContain('no-restricted-imports');
    await expect(
      ruleIds('src/pages/fixture.js', "import '@/features/projects/api/projectApi';\n"),
    ).resolves.not.toContain('no-restricted-imports');
  });

  it('restricts direct network calls to API and shared infrastructure', async () => {
    await expect(
      ruleIds('src/features/projects/components/fixture.js', "fetch('/api/projects');\n"),
    ).resolves.toContain('no-restricted-globals');
    await expect(ruleIds('src/pages/fixture.js', "fetch('/api/projects');\n")).resolves.toContain(
      'no-restricted-globals',
    );
    await expect(
      ruleIds('src/features/projects/api/fixture.js', "fetch('/api/projects');\n"),
    ).resolves.not.toContain('no-restricted-globals');
    await expect(
      ruleIds('src/shared/api/fixture.js', "fetch('/api/projects');\n"),
    ).resolves.not.toContain('no-restricted-globals');
  });
});
