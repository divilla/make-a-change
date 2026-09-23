import { mount } from '@vue/test-utils';
import { ref } from 'vue';
import { epicFixture } from '@/features/changes/model/change.fixtures';
import EpicCreatePage from './create.vue';

const routerMock = vi.hoisted(() => ({
  push: vi.fn(),
}));

const createEpicFromForm = vi.hoisted(() => vi.fn());

vi.mock('vue-router', () => ({
  useRouter: () => routerMock,
}));

vi.mock('@/features/epics/composables/useEpicsPage', () => ({
  useEpicsPage: () => ({
    epicName: ref('New epic'),
    loading: ref(false),
    saving: ref(false),
    error: ref(''),
    createEpicFromForm,
  }),
}));

describe('Epic create page', () => {
  beforeEach(() => {
    routerMock.push.mockClear();
    createEpicFromForm.mockResolvedValue(epicFixture({ id: 9, name: 'New epic' }));
  });

  it('redirects to epics after create', async () => {
    const wrapper = mount(EpicCreatePage, {
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

    await wrapper.find('form').trigger('submit');

    expect(createEpicFromForm).toHaveBeenCalled();
    expect(routerMock.push).toHaveBeenCalledWith('/epics');
  });
});
