import type { Change, ChangeDetail, ChangePhase, ChangeType, Epic } from './change.types';

export function changeFixture(overrides: Partial<Change> = {}): Change {
  const change: Change = {
    id: 1,
    version: 0,
    ref_uuid: '00000000-0000-0000-0000-000000000001',
    ref: 1,
    slug: '000001-change',
    project_id: 1,
    epic_id: null,
    epic_name: null,
    change_phase: 'backlog',
    change_types: ['feature'],
    title: 'Change',
    def: '',
    spec: '',
    spec_html: '',
    pr: '',
    pr_html: '',
    pr_url: '',
    agent_edit: false,
    open: true,
    done_tc: 0,
    total_tc: 0,
    completed: 0,
    created: '2026-01-01T00:00:00Z',
    modified: '2026-01-01T00:00:00Z',
  };
  return { ...change, ...overrides };
}

export function changeDetailFixture(overrides: Partial<ChangeDetail> = {}): ChangeDetail {
  return {
    change: changeFixture(),
    test_cases: [],
    ...overrides,
  };
}

export function epicFixture(overrides: Partial<Epic> = {}): Epic {
  const epic: Epic = {
    id: 1,
    version: 0,
    project_id: 1,
    name: 'Epic',
    done_tc: 0,
    total_tc: 0,
    completed: 0,
    change_count: 0,
    created: '2026-01-01T00:00:00Z',
    modified: '2026-01-01T00:00:00Z',
  };
  return { ...epic, ...overrides };
}

export function changePhasesFixture(overrides: ChangePhase[] = []): ChangePhase[] {
  return overrides.length
    ? overrides
    : [
        { slug: 'backlog', priority: 1 },
        { slug: 'review', priority: 2 },
      ];
}

export function changeTypesFixture(overrides: ChangeType[] = []): ChangeType[] {
  return overrides.length
    ? overrides
    : [
        { slug: 'change', priority: 1 },
        { slug: 'feature', priority: 2 },
      ];
}
