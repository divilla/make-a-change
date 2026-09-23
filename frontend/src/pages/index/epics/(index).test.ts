import { mount } from '@vue/test-utils';
import { ref } from 'vue';
import { epicFixture } from '@/features/changes/model/change.fixtures';
import EpicsPage from './(index).vue';

const routerMock = vi.hoisted(() => ({
  push: vi.fn(),
}));

const epicsPageMock = vi.hoisted(() => ({
  removeEpic: vi.fn(),
  confirm: vi.fn(),
}));

vi.mock('vue-router', () => ({
  useRouter: () => routerMock,
}));

vi.mock('@/features/epics/composables/useEpicsPage', () => ({
  useEpicsPage: () => ({
    epics: ref([epicFixture({ id: 4, name: 'Planning', change_count: 0 })]),
    loading: ref(false),
    error: ref(''),
    confirmationDialogOpen: ref(false),
    removeEpic: epicsPageMock.removeEpic,
    confirm: epicsPageMock.confirm,
  }),
}));

describe('Epics page', () => {
  beforeEach(() => {
    routerMock.push.mockClear();
    epicsPageMock.removeEpic.mockClear();
    epicsPageMock.confirm.mockClear();
  });

  it('routes to create and edit pages from table actions', async () => {
    const wrapper = mount(EpicsPage, {
      global: {
        stubs: {
          QBanner: { template: '<div><slot /></div>' },
          QIcon: { template: '<span><slot /></span>' },
          QPage: { template: '<main><slot /></main>' },
          DeleteConfirmationDialog: true,
          EpicList: {
            emits: ['create', 'edit', 'delete'],
            template:
              '<div><button class="create" @click="$emit(\'create\')"></button><button class="edit" @click="$emit(\'edit\', epic)"></button></div>',
            setup() {
              return { epic: epicFixture({ id: 4, name: 'Planning', change_count: 0 }) };
            },
          },
        },
      },
    });

    await wrapper.find('.create').trigger('click');
    await wrapper.find('.edit').trigger('click');

    expect(routerMock.push).toHaveBeenCalledWith('/epics/create');
    expect(routerMock.push).toHaveBeenCalledWith('/epics/edit/4');
  });
});
