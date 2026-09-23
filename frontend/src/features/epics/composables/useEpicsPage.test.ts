import { mount, flushPromises } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { defineComponent } from 'vue';
import { listChanges } from '@/features/changes/api/changeApi';
import { changeFixture, epicFixture } from '@/features/changes/model/change.fixtures';
import {
  createEpic,
  deleteEpic,
  getEpic,
  listEpics,
  updateEpic,
} from '@/features/epics/api/epicApi';
import { listProjects } from '@/features/projects/api/projectApi';
import type { Project } from '@/features/projects/model/project.types';
import { useEpicsPage } from './useEpicsPage';

vi.mock('@/features/changes/api/changeApi', () => ({
  listChanges: vi.fn(),
}));

vi.mock('@/features/epics/api/epicApi', () => ({
  createEpic: vi.fn(),
  deleteEpic: vi.fn(),
  getEpic: vi.fn(),
  listEpics: vi.fn(),
  updateEpic: vi.fn(),
}));

vi.mock('@/features/projects/api/projectApi', () => ({
  listProjects: vi.fn(),
}));

type EpicsPageState = ReturnType<typeof useEpicsPage>;

function projectFixture(project: Pick<Project, 'id' | 'name' | 'change_count'>): Project {
  return {
    created: '2026-06-23T00:00:00Z',
    modified: '2026-06-23T00:00:00Z',
    ...project,
  };
}

function mountEpicsPage() {
  let state: EpicsPageState | undefined;
  const wrapper = mount(
    defineComponent({
      setup() {
        state = useEpicsPage();
        return () => null;
      },
    }),
  );
  return { wrapper, state: state as EpicsPageState };
}

describe('useEpicsPage', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    localStorage.clear();
    vi.mocked(listProjects).mockResolvedValue([
      projectFixture({ id: 1, name: 'Project', change_count: 1 }),
    ]);
    vi.mocked(listChanges).mockResolvedValue([changeFixture({ id: 10, project_id: 1 })]);
    vi.mocked(listEpics).mockResolvedValue([epicFixture({ id: 7, name: 'Epic', change_count: 0 })]);
    vi.mocked(createEpic).mockResolvedValue(epicFixture({ id: 8, name: 'New epic', change_count: 0 }));
    vi.mocked(updateEpic).mockResolvedValue(epicFixture({ id: 7, name: 'Renamed', change_count: 0 }));
    vi.mocked(getEpic).mockResolvedValue(epicFixture({ id: 7, name: 'Loaded', change_count: 0 }));
    vi.mocked(deleteEpic).mockResolvedValue(undefined);
  });

  afterEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
  });

  it('loads selected project epics on mount', async () => {
    const { state } = mountEpicsPage();
    await flushPromises();

    expect(listEpics).toHaveBeenCalledWith(1);
    expect(state.epics.value).toEqual([epicFixture({ id: 7, name: 'Epic', change_count: 0 })]);
  });

  it('creates an epic for the selected project', async () => {
    const { state } = mountEpicsPage();
    await flushPromises();

    state.epicName.value = 'New epic';
    const epic = await state.createEpicFromForm();

    expect(createEpic).toHaveBeenCalledWith({ project_id: 1, name: 'New epic' });
    expect(epic).toEqual(epicFixture({ id: 8, name: 'New epic', change_count: 0 }));
  });

  it('loads and saves an epic', async () => {
    const { state } = mountEpicsPage();
    await flushPromises();

    await state.loadEpic(7);
    state.epicName.value = 'Renamed';
    const epic = await state.saveEpicFromForm();

    expect(getEpic).toHaveBeenCalledWith(7);
    expect(updateEpic).toHaveBeenCalledWith({ id: 7, name: 'Renamed' });
    expect(epic).toEqual(epicFixture({ id: 7, name: 'Renamed', change_count: 0 }));
  });

  it('confirms epic deletion before deleting it', async () => {
    const { state } = mountEpicsPage();
    await flushPromises();

    state.removeEpic(epicFixture({ id: 7, change_count: 0 }));

    expect(state.confirmationDialogOpen.value).toBe(true);
    expect(deleteEpic).not.toHaveBeenCalled();

    await state.confirm();

    expect(state.confirmationDialogOpen.value).toBe(false);
    expect(deleteEpic).toHaveBeenCalledWith(7);
  });
});
