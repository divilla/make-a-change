import { post } from '@/shared/api/httpClient';
import type {
  TestCase,
  TestCaseMutation,
  TestCaseUpdateInput,
} from '../model/testCase.types';

export function listTestCases(changeId: number): Promise<TestCase[]> {
  return post<TestCase[]>('/api/v1/test-case/list', { change_id: changeId });
}

export function createTestCase(
  changeId: number,
  scenario: string,
): Promise<TestCaseMutation> {
  return post<TestCaseMutation>('/api/v1/test-case/create', { change_id: changeId, scenario });
}

export function updateTestCase(input: TestCaseUpdateInput): Promise<TestCaseMutation> {
  return post<TestCaseMutation>('/api/v1/test-case/update', input);
}

export function updateTestCaseDone(id: number, done: boolean): Promise<TestCaseMutation> {
  return post<TestCaseMutation>('/api/v1/test-case/update-done', { id, done });
}

export function updateTestCaseChange(id: number, changeId: number): Promise<TestCaseMutation> {
  return post<TestCaseMutation>('/api/v1/test-case/update-change', { id, change_id: changeId });
}

export function deleteTestCase(id: number): Promise<TestCaseMutation> {
  return post<TestCaseMutation>('/api/v1/test-case/delete', { id });
}
