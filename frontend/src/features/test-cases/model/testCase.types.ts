import type { Change } from '@/features/changes/model/change.types';

export interface TestCase {
  id: number;
  version: number;
  change_id: number;
  scenario: string;
  done: boolean;
  created: string;
  modified: string;
}

export interface TestCaseMutation {
  test_case?: TestCase;
  change: Change;
  test_cases: TestCase[];
}

export interface TestCaseUpdateInput {
  id: number;
  scenario: string;
}
