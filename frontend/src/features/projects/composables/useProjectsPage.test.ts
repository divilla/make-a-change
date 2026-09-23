import { mount, flushPromises } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { defineComponent } from 'vue';
import {
  createProject,
  deleteProject,
  listProjects,
  updateProject,
} from '@/features/projects/api/projectApi';
import { useProjectSelectionStore } from '@/features/projects/model/projectSelection.store';
import type { Project } from '@/features/projects/model/project.types';
import {
  deleteChange,
  getChangePhases,
  getChangeTypes,
  listChanges,
  updateChangePhase,
} from '@/features/changes/api/changeApi';
import { listEpics } from '@/features/epics/api/epicApi';
import {
  changeFixture,
  changePhasesFixture,
  changeTypesFixture,
} from '@/features/changes/model/change.fixtures';
import { useProjectsPage } from './useProjectsPage';

vi.mock('@/features/projects/api/projectApi', () => ({
  createProject: vi.fn(),
  deleteProject: vi.fn(),
  listProjects: vi.fn(),
  updateProject: vi.fn(),
}));

vi.mock('@/features/changes/api/changeApi', () => ({
  deleteChange: vi.fn(),
  getChangePhases: vi.fn(),
  getChangeTypes: vi.fn(),
  listChanges: vi.fn(),
  updateChangePhase: vi.fn(),
}));

vi.mock('@/features/epics/api/epicApi', () => ({
  listEpics: vi.fn(),
}));

type ProjectsPageState = ReturnType<typeof useProjectsPage>;

function projectFixture(project: Pick<Project, 'id' | 'name' | 'change_count'>): Project {
  return {
    created: '2026-06-23T00:00:00Z',
    modified: '2026-06-23T00:00:00Z',
    ...project,
  };
}

function mountProjectsPage(options?: Parameters<typeof useProjectsPage>[0]) {
  let state: ProjectsPageState | undefined;
  const wrapper = mount(
    defineComponent({
      setup() {
        state = useProjectsPage(options);
        return () => null;
      },
    }),
  );
  return { wrapper, state: state as ProjectsPageState };
}

describe('useProjectsPage', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    localStorage.clear();
    vi.mocked(getChangePhases).mockResolvedValue(changePhasesFixture());
    vi.mocked(getChangeTypes).mockResolvedValue(changeTypesFixture());
    vi.mocked(listProjects).mockResolvedValue([
      projectFixture({ id: 1, name: 'Project', change_count: 1 }),
    ]);
    vi.mocked(listChanges).mockResolvedValue([changeFixture({ id: 10, project_id: 1 })]);
    vi.mocked(listEpics).mockResolvedValue([]);
    vi.mocked(createProject).mockResolvedValue(
      projectFixture({ id: 2, name: 'New project', change_count: 0 }),
    );
    vi.mocked(updateProject).mockResolvedValue(
      projectFixture({ id: 1, name: 'Renamed', change_count: 1 }),
    );
    vi.mocked(deleteProject).mockResolvedValue(undefined);
    vi.mocked(updateChangePhase).mockResolvedValue(
      changeFixture({ id: 10, project_id: 1, change_phase: 'review' }),
    );
    vi.mocked(deleteChange).mockResolvedValue(undefined);
  });

  afterEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
  });

  it('loads phase and type options, projects, empty search fields, and changes on mount', async () => {
    const { state } = mountProjectsPage();
    await flushPromises();

    expect(state.projects.value).toEqual([
      projectFixture({ id: 1, name: 'Project', change_count: 1 }),
    ]);
    expect(state.currentProjectId.value).toBe(1);
    expect(state.changeType.value).toBe('');
    expect(state.changePhase.value).toBe('');
    expect(state.changes.value).toEqual([changeFixture({ id: 10, project_id: 1 })]);
    expect(listChanges).toHaveBeenCalledWith(1);
  });

  it('filters visible change board results by search fields', async () => {
    vi.mocked(listChanges).mockResolvedValueOnce([
      changeFixture({
        id: 10,
        title: 'Build change search',
        change_types: ['feature'],
        change_phase: 'backlog',
      }),
      changeFixture({
        id: 11,
        title: 'Review filters',
        change_types: ['fix'],
        change_phase: 'review',
      }),
      changeFixture({
        id: 12,
        title: 'Build refresh',
        change_types: ['feature'],
        change_phase: 'review',
      }),
    ]);
    const { state } = mountProjectsPage();
    await flushPromises();

    state.changeTitle.value = 'build';
    state.changeType.value = 'feature';
    state.changePhase.value = 'review';

    expect(state.changesByPhase.value.backlog).toEqual([]);
    expect(state.changesByPhase.value.review).toEqual([
      changeFixture({
        id: 12,
        title: 'Build refresh',
        change_types: ['feature'],
        change_phase: 'review',
      }),
    ]);
  });

  it('refreshes changes when search is submitted', async () => {
    const { state } = mountProjectsPage();
    await flushPromises();
    vi.mocked(listChanges).mockClear();
    vi.mocked(listChanges).mockResolvedValueOnce([
      changeFixture({ id: 13, title: 'Fresh result', project_id: 1 }),
    ]);

    await state.searchChanges();

    expect(listChanges).toHaveBeenCalledWith(1);
    expect(state.changes.value).toEqual([
      changeFixture({ id: 13, title: 'Fresh result', project_id: 1 }),
    ]);
  });

  it('clears search fields and refreshes changes', async () => {
    const { state } = mountProjectsPage();
    await flushPromises();
    vi.mocked(listChanges).mockClear();
    vi.mocked(listChanges).mockResolvedValueOnce([
      changeFixture({ id: 14, title: 'Cleared result', project_id: 1 }),
    ]);
    state.changeTitle.value = 'build';
    state.changeType.value = 'feature';
    state.changePhase.value = 'review';

    await state.clearChangeSearch();

    expect(state.changeTitle.value).toBe('');
    expect(state.changeType.value).toBe('');
    expect(state.changePhase.value).toBe('');
    expect(listChanges).toHaveBeenCalledWith(1);
    expect(state.changes.value).toEqual([
      changeFixture({ id: 14, title: 'Cleared result', project_id: 1 }),
    ]);
  });

  it('loads only projects when change-board behavior is disabled', async () => {
    const { state } = mountProjectsPage({ changesEnabled: false });
    await flushPromises();

    expect(state.projects.value).toEqual([
      projectFixture({ id: 1, name: 'Project', change_count: 1 }),
    ]);
    expect(state.currentProjectId.value).toBe(1);
    expect(getChangePhases).not.toHaveBeenCalled();
    expect(getChangeTypes).not.toHaveBeenCalled();
    expect(listChanges).not.toHaveBeenCalled();

    await state.selectProject(1);

    expect(listChanges).not.toHaveBeenCalled();
  });

  it('does not load changes when the shared selector changes on the projects CRUD page', async () => {
    vi.mocked(listProjects).mockResolvedValue([
      projectFixture({ id: 1, name: 'Project', change_count: 1 }),
      projectFixture({ id: 2, name: 'Other project', change_count: 0 }),
    ]);
    const { state } = mountProjectsPage({ changesEnabled: false });
    await flushPromises();
    vi.mocked(listChanges).mockClear();

    useProjectSelectionStore().selectProject(2);
    await flushPromises();

    expect(state.currentProjectId.value).toBe(2);
    expect(listChanges).not.toHaveBeenCalled();
  });

  it('selects a created project and loads its change list', async () => {
    const { state } = mountProjectsPage();
    await flushPromises();
    vi.mocked(listChanges).mockClear();

    state.projectName.value = 'New project';
    await state.createProjectFromForm();

    expect(createProject).toHaveBeenCalledWith('New project');
    expect(state.currentProjectId.value).toBe(2);
    expect(listChanges).toHaveBeenCalledWith(2);
  });

  it('confirms project deletion before removing it', async () => {
    const { state } = mountProjectsPage({ changesEnabled: false });
    await flushPromises();

    state.removeProject(projectFixture({ id: 2, name: 'Empty project', change_count: 0 }));

    expect(state.confirmationDialogOpen.value).toBe(true);
    expect(deleteProject).not.toHaveBeenCalled();

    await state.confirm();

    expect(state.confirmationDialogOpen.value).toBe(false);
    expect(deleteProject).toHaveBeenCalledWith(2);
  });

  it('refreshes project change counts after change deletion', async () => {
    const { state } = mountProjectsPage();
    await flushPromises();
    vi.mocked(listProjects).mockClear();
    vi.mocked(listChanges).mockClear();
    vi.mocked(listProjects).mockResolvedValueOnce([
      projectFixture({ id: 1, name: 'Project', change_count: 0 }),
    ]);

    state.removeChange(changeFixture({ id: 10, project_id: 1 }));

    expect(state.confirmationDialogOpen.value).toBe(true);
    expect(deleteChange).not.toHaveBeenCalled();

    await state.confirm();

    expect(state.confirmationDialogOpen.value).toBe(false);
    expect(deleteChange).toHaveBeenCalledWith(10);
    expect(listChanges).toHaveBeenCalledWith(1);
    expect(listProjects).toHaveBeenCalledTimes(1);
    expect(state.projects.value).toEqual([
      projectFixture({ id: 1, name: 'Project', change_count: 0 }),
    ]);
  });

  it('refreshes selected project changes after change phase changes', async () => {
    const { state } = mountProjectsPage();
    await flushPromises();
    vi.mocked(listChanges).mockClear();
    vi.mocked(listChanges).mockResolvedValueOnce([
      changeFixture({ id: 10, change_phase: 'review' }),
    ]);

    await state.moveChange(changeFixture({ id: 10 }), 'review');

    expect(updateChangePhase).toHaveBeenCalledWith(10, 'review');
    expect(listChanges).toHaveBeenCalledWith(1);
    expect(state.changes.value[0]?.change_phase).toBe('review');
  });

  it('sets a user-facing error when change loading fails', async () => {
    const { state } = mountProjectsPage();
    await flushPromises();
    vi.mocked(listChanges).mockRejectedValueOnce(new Error('network down'));

    await state.selectProject(1);

    expect(state.error.value).toBe('network down');
  });
});
