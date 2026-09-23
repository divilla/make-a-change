import { flushPromises, mount } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { createChange, listChanges } from '@/features/changes/api/changeApi';
import { changeFixture } from '@/features/changes/model/change.fixtures';
import { useProjectSelectionStore } from '@/features/projects/model/projectSelection.store';
import { listProjects } from '@/features/projects/api/projectApi';
import { listEpics } from '@/features/epics/api/epicApi';
import ChangeCreatePage from './create.vue';

const routerMock = vi.hoisted(() => ({ push: vi.fn() }));

vi.mock('vue-router', () => ({ useRouter: () => routerMock }));
vi.mock('@/features/changes/api/changeApi', () => ({
  createChange: vi.fn(),
  listChanges: vi.fn(),
}));
vi.mock('@/features/projects/api/projectApi', () => ({ listProjects: vi.fn() }));
vi.mock('@/features/epics/api/epicApi', () => ({ listEpics: vi.fn() }));

const quasarStubs = {
  QBanner: { template: '<div><slot /></div>' },
  QBtn: { props: ['label'], template: '<button>{{ label }}</button>' },
  QForm: { template: '<form @submit.prevent="$emit(\'submit\', $event)"><slot /></form>' },
  QIcon: { template: '<span />' },
  QInput: {
    emits: ['update:modelValue'],
    props: ['label', 'modelValue', 'type'],
    template: '<label>{{ label }}<textarea v-if="type === \'textarea\'" :aria-label="label" :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" /><input v-else :aria-label="label" :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" /></label>',
  },
  QPage: { template: '<main><slot /></main>' },
};

describe('ChangeCreatePage', () => {
  beforeEach(async () => {
    vi.clearAllMocks();
    setActivePinia(createPinia());
    const projects = [{ id: 7, name: 'Project', created: '', modified: '', change_count: 0 }];
    const selection = useProjectSelectionStore();
    vi.mocked(listProjects).mockResolvedValue(projects);
    await selection.loadProjects();
    selection.selectProject(7);
    vi.mocked(listChanges).mockResolvedValue([]);
    vi.mocked(listEpics).mockResolvedValue([]);
    vi.mocked(createChange).mockResolvedValue(
      changeFixture({ id: 12, project_id: 7, title: 'Trimmed change', def: 'Definition body' }),
    );
  });

  it('labels, trims, and creates through the definition contract', async () => {
    const wrapper = mount(ChangeCreatePage, { global: { stubs: quasarStubs } });
    await flushPromises();
    await wrapper.get('[aria-label="Change title"]').setValue('  Trimmed change  ');
    await wrapper.get('[aria-label="Definition"]').setValue('  Definition body  ');
    await wrapper.get('form').trigger('submit');
    await flushPromises();

    expect(createChange).toHaveBeenCalledWith({
      project_id: 7,
      title: 'Trimmed change',
      def: 'Definition body',
    });
    expect(routerMock.push).toHaveBeenCalledWith('/changes/12');
  });
});
