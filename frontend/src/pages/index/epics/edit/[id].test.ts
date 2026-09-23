import { mount, flushPromises } from '@vue/test-utils';
import { ref } from 'vue';
import { epicFixture } from '@/features/changes/model/change.fixtures';
import EpicEditPage from './[id].vue';

const routerMock = vi.hoisted(() => ({
  push: vi.fn(),
}));

const routeMock = vi.hoisted(() => ({
  params: { id: '7' },
}));

const loadEpic = vi.hoisted(() => vi.fn());
const saveEpicFromForm = vi.hoisted(() => vi.fn());

vi.mock('vue-router', () => ({
  useRoute: () => routeMock,
  useRouter: () => routerMock,
}));

vi.mock('@/features/epics/composables/useEpicsPage', () => ({
  useEpicsPage: () => ({
    epicName: ref('Renamed epic'),
    loading: ref(false),
    saving: ref(false),
    error: ref(''),
    loadEpic,
    saveEpicFromForm,
  }),
}));

describe('Epic edit page', () => {
  beforeEach(() => {
    routerMock.push.mockClear();
    loadEpic.mockResolvedValue(undefined);
    saveEpicFromForm.mockResolvedValue(epicFixture({ id: 7, name: 'Renamed epic' }));
  });

  it('loads the route epic and redirects to epics after save', async () => {
    const wrapper = mount(EpicEditPage, {
      global: {
        stubs: {
          QBanner: { template: '<div><slot /></div>' },
          QBtn: { template: '<button><slot /></button>' },
          QForm: { emits: ['submit'], template: '<form @submit.prevent="$emit(\'submit\', $event)"><slot /></form>' },
          QIcon: { template: '<span><slot /></span>' },
          QInput: { template: '<input />' },
          QPage: { template: '<main><slot /></main>' },
        },
      },
    });
    await flushPromises();

    expect(loadEpic).toHaveBeenCalledWith(7);

    await wrapper.find('form').trigger('submit');

    expect(saveEpicFromForm).toHaveBeenCalled();
    expect(routerMock.push).toHaveBeenCalledWith('/epics');
  });
});
