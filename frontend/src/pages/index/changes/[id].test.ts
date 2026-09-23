import { flushPromises, mount } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { listProjects } from '@/features/projects/api/projectApi';
import type { Project } from '@/features/projects/model/project.types';
import { useProjectSelectionStore } from '@/features/projects/model/projectSelection.store';
import {
  deleteTestCase,
  updateTestCaseChange,
} from '@/features/test-cases/api/testCaseApi';
import {
  testCaseFixture,
  testCaseMutationFixture,
} from '@/features/test-cases/model/testCase.fixtures';
import {
  deleteChange,
  getChange,
  listChanges,
} from '@/features/changes/api/changeApi';
import { listEpics } from '@/features/epics/api/epicApi';
import { changeDetailFixture, changeFixture, epicFixture } from '@/features/changes/model/change.fixtures';
import ChangeDetailPage from './[id].vue';

const routerMock = vi.hoisted(() => ({
  back: vi.fn(),
  push: vi.fn(),
  replace: vi.fn(),
  route: {
    fullPath: '/changes/2',
    params: {
      id: '2',
    },
  },
}));

vi.mock('vue-router', () => ({
  useRoute: () => routerMock.route,
  useRouter: () => ({
    back: routerMock.back,
    push: routerMock.push,
    replace: routerMock.replace,
  }),
}));

vi.mock('@/features/projects/api/projectApi', () => ({
  listProjects: vi.fn(),
}));

vi.mock('@/features/changes/api/changeApi', () => ({
  deleteChange: vi.fn(),
  getChange: vi.fn(),
  listChanges: vi.fn(),
}));

vi.mock('@/features/epics/api/epicApi', () => ({
  listEpics: vi.fn(),
}));

vi.mock('@/features/test-cases/api/testCaseApi', () => ({
  createTestCase: vi.fn(),
  deleteTestCase: vi.fn(),
  updateTestCase: vi.fn(),
  updateTestCaseChange: vi.fn(),
  updateTestCaseDone: vi.fn(),
}));

function projectFixture(project: Pick<Project, 'id' | 'name' | 'change_count'>): Project {
  return {
    created: '2026-06-23T00:00:00Z',
    modified: '2026-06-23T00:00:00Z',
    ...project,
  };
}

const quasarStubs = {
  QBanner: { template: '<div><slot name="avatar" /><slot /></div>' },
  QBtn: {
    emits: ['click'],
    props: ['color', 'disable', 'flat', 'icon', 'label', 'loading'],
    template: `
      <button
        :data-color="color"
        :data-icon="icon"
        :disabled="disable || loading"
        type="button"
        @click="$emit('click', $event)"
      >
        {{ label }}<slot />
      </button>
    `,
  },
  QBtnDropdown: {
    emits: ['click'],
    props: ['dropdownIcon'],
    template: `
      <div>
        <button :data-icon="dropdownIcon" type="button" @click="$emit('click', $event)" />
        <slot />
      </div>
    `,
  },
  QCard: { template: '<section><slot /></section>' },
  QCardActions: { template: '<footer><slot /></footer>' },
  QCardSection: { template: '<section><slot /></section>' },
  QCheckbox: {
    emits: ['update:modelValue'],
    props: ['modelValue'],
    template: '<input type="checkbox" :checked="modelValue" @change="$emit(\'update:modelValue\', true)" />',
  },
  QDialog: {
    props: ['modelValue', 'persistent'],
    template: `
      <div
        v-if="modelValue"
        :data-persistent="persistent !== false && persistent !== undefined ? 'true' : undefined"
      >
        <slot />
      </div>
    `,
  },
  QForm: { template: '<form @submit.prevent="$emit(\'submit\', $event)"><slot /></form>' },
  QIcon: { props: ['name'], template: '<span :data-icon="name"><slot /></span>' },
  QInput: { template: '<input />' },
  QItem: {
    emits: ['click'],
    props: ['disable'],
    template: '<button type="button" :disabled="disable" @click="$emit(\'click\', $event)"><slot /></button>',
  },
  QItemLabel: { template: '<span><slot /></span>' },
  QItemSection: { template: '<span><slot /></span>' },
  QList: { template: '<div><slot /></div>' },
  QMarkupTable: { template: '<table><slot /></table>' },
  QPage: { template: '<main><slot /></main>' },
  QSelect: {
    emits: ['update:modelValue'],
    props: ['modelValue', 'options'],
    template: '<select :value="modelValue" @change="$emit(\'update:modelValue\', Number($event.target.value))"><option v-for="option in options" :key="option.value" :value="option.value">{{ option.label }}</option></select>',
  },
  QSpinner: { template: '<span />' },
};

describe('ChangeDetailPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setActivePinia(createPinia());
    localStorage.clear();
    routerMock.route.params.id = '2';
    routerMock.route.fullPath = '/changes/2';
    vi.mocked(listProjects).mockResolvedValue([
      projectFixture({ id: 1, name: 'Project', change_count: 3 }),
    ]);
    vi.mocked(getChange).mockResolvedValue(
      changeDetailFixture({
        change: changeFixture({
          id: 2,
          title: 'Current change',
          project_id: 1,
          epic_id: 1,
          epic_name: 'Backend Epic',
          def: '# Current change\n\nInitial definition',
          spec_html: '<p>Current spec</p>',
          pr_html: '<p>Pull request body</p>',
          pr_url: 'https://example.test/pr/2',
          agent_edit: true,
        }),
        test_cases: [],
      }),
    );
    vi.mocked(listChanges).mockResolvedValue([
      changeFixture({ id: 2, title: 'Current change', project_id: 1, epic_id: 1 }),
      changeFixture({ id: 3, title: 'Other change', project_id: 1 }),
    ]);
    vi.mocked(listEpics).mockResolvedValue([epicFixture({ id: 1, name: 'Project Epic' })]);
    vi.mocked(deleteChange).mockResolvedValue(undefined);
    vi.mocked(deleteTestCase).mockResolvedValue(testCaseMutationFixture());
    vi.mocked(updateTestCaseChange).mockResolvedValue(testCaseMutationFixture());
  });

  afterEach(() => {
    vi.restoreAllMocks();
    localStorage.clear();
  });

  function mountPage() {
    return mount(ChangeDetailPage, {
      global: {
        directives: {
          ClosePopup: {},
        },
        stubs: quasarStubs,
      },
    });
  }

  it('renders current change actions and change metadata', async () => {
    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.text()).toContain('Current change');
    expect(wrapper.text()).toContain('#1');
    expect(wrapper.text()).toContain('000001-change');
    expect(wrapper.text()).toContain('Backend Epic');
    expect(wrapper.text()).toContain('Agent Edit');
    expect(wrapper.text()).toContain('Yes');
    expect(wrapper.text()).toContain('https://example.test/pr/2');
    expect(wrapper.find('a').attributes('href')).toBe('https://example.test/pr/2');
    expect(wrapper.text()).toContain('Initial definition');
    expect(wrapper.html()).toContain('<p>Current spec</p>');
    expect(wrapper.html()).toContain('<p>Pull request body</p>');
    expect(wrapper.findAll('[data-icon="more_vert"]')).toHaveLength(1);
    expect(wrapper.findAll('[data-action="edit-change"]')).toHaveLength(1);
    expect(wrapper.findAll('[data-action="delete-change"]')).toHaveLength(1);
  });

  it('renders unsafe stored PR URLs as plain text instead of executable links', async () => {
    vi.mocked(getChange).mockResolvedValue(
      changeDetailFixture({
        change: changeFixture({
          id: 2,
          title: 'Current change',
          project_id: 1,
          pr_url: 'javascript:alert(1)',
        }),
        test_cases: [],
      }),
    );
    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.text()).toContain('javascript:alert(1)');
    expect(wrapper.find('a').exists()).toBe(false);
  });

  it('asks before switching project context for a pasted change URL', async () => {
    vi.mocked(listProjects).mockResolvedValue([
      projectFixture({ id: 1, name: 'Project A', change_count: 1 }),
      projectFixture({ id: 2, name: 'Project B', change_count: 1 }),
    ]);
    vi.mocked(getChange).mockResolvedValue(
      changeDetailFixture({
        change: changeFixture({ id: 2, title: 'Current change', project_id: 2 }),
        test_cases: [],
      }),
    );
    vi.mocked(listChanges).mockResolvedValue([
      changeFixture({ id: 2, title: 'Current change', project_id: 2 }),
    ]);
    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.text()).toContain('Switch project?');
    expect(wrapper.text()).toContain('Project A');
    expect(wrapper.text()).toContain('Project B');

    const switchButton = wrapper.findAll('button').find((button) => button.text() === 'Switch');
    await switchButton?.trigger('click');

    const projectSelection = useProjectSelectionStore();
    expect(projectSelection.currentProjectId).toBe(2);
    expect(projectSelection.routeDrivenTargetPath).toBe('/changes/2');
  });

  it('opens the change edit route from the menu action', async () => {
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.find('[data-action="edit-change"]').trigger('click');

    expect(routerMock.push).toHaveBeenCalledWith('/changes/edit/2');
  });

  it('confirms change deletion', async () => {
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.find('[data-action="delete-change"]').trigger('click');
    await flushPromises();

    expect(wrapper.text()).toContain('Are you sure?');
    expect(deleteChange).not.toHaveBeenCalled();

    const confirmButton = wrapper.findAll('button').find((button) => button.text() === 'OK');
    await confirmButton?.trigger('click');
    await flushPromises();

    expect(deleteChange).toHaveBeenCalledWith(2);
    expect(listChanges).toHaveBeenCalledWith(1);
  });

  it('renders test case actions and opens the edit dialog', async () => {
    vi.mocked(getChange).mockResolvedValue(
      changeDetailFixture({
        change: changeFixture({ id: 2, title: 'Current change', project_id: 1 }),
        test_cases: [testCaseFixture({ id: 9, change_id: 2, scenario: 'Existing test case' })],
      }),
    );
    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.findAll('[data-action="edit-test-case"]')).toHaveLength(1);
    expect(wrapper.findAll('[data-action="delete-test-case"]')).toHaveLength(1);

    await wrapper.find('[data-action="edit-test-case"]').trigger('click');

    expect(wrapper.text()).toContain('Edit Test Case');
  });

  it('confirms test case deletion with the shared delete dialog', async () => {
    const testCase = testCaseFixture({ id: 9, change_id: 2, scenario: 'Delete me' });
    vi.mocked(getChange).mockResolvedValue(
      changeDetailFixture({
        change: changeFixture({ id: 2, title: 'Current change', project_id: 1 }),
        test_cases: [testCase],
      }),
    );
    vi.mocked(deleteTestCase).mockResolvedValue(
      testCaseMutationFixture({
        change: changeFixture({ id: 2, project_id: 1 }),
        test_cases: [],
      }),
    );
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.find('[data-action="delete-test-case"]').trigger('click');
    await flushPromises();

    expect(wrapper.text()).toContain('Are you sure?');
    expect(deleteTestCase).not.toHaveBeenCalled();

    const okButton = wrapper.findAll('button').find((button) => button.text() === 'OK');
    expect(okButton?.attributes('data-color')).toBe('negative');
    await okButton?.trigger('click');
    await flushPromises();

    expect(deleteTestCase).toHaveBeenCalledWith(9);
    expect(listChanges).toHaveBeenCalledWith(1);
  });
});
