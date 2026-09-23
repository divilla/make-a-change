import { changeFixture } from '@/features/changes/model/change.fixtures';
import type { TestCase, TestCaseMutation } from './testCase.types';

export function testCaseFixture(overrides: Partial<TestCase> = {}): TestCase {
  return {
    id: 1,
    version: 0,
    change_id: 1,
    scenario: 'Test case',
    done: false,
    created: '2026-01-01T00:00:00Z',
    modified: '2026-01-01T00:00:00Z',
    ...overrides,
  };
}

export function testCaseMutationFixture(
  overrides: Partial<TestCaseMutation> = {},
): TestCaseMutation {
  const testCase = testCaseFixture(overrides.test_case ?? {});
  return {
    test_case: testCase,
    change: changeFixture({ id: testCase.change_id }),
    test_cases: [testCase],
    ...overrides,
  };
}
