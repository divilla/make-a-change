import { mount } from '@vue/test-utils';
import { epicFixture } from '@/features/changes/model/change.fixtures';
import EpicList from './EpicList.vue';

const quasarStubs = {
  QBtn: {
    emits: ['click'],
    props: ['icon', 'disable'],
    template:
      '<button :data-icon="icon" :disabled="disable" @click="!disable && $emit(\'click\', $event)"><slot /></button>',
  },
  QIcon: { template: '<span><slot /></span>' },
  QMarkupTable: { template: '<table><slot /></table>' },
  QTooltip: { template: '<span><slot /></span>' },
};

describe('EpicList', () => {
  it('emits create, edit, and delete events', async () => {
    const epic = epicFixture({ id: 3, name: 'API work', change_count: 0 });
    const wrapper = mount(EpicList, {
      props: {
        epics: [epic],
        loading: false,
      },
      global: {
        stubs: quasarStubs,
      },
    });

    await wrapper.find('[data-icon="add"]').trigger('click');
    await wrapper.find('[data-icon="edit"]').trigger('click');
    await wrapper.find('[data-icon="delete"]').trigger('click');

    expect(wrapper.emitted('create')).toEqual([[]]);
    expect(wrapper.emitted('edit')).toEqual([[epic]]);
    expect(wrapper.emitted('delete')).toEqual([[epic]]);
  });

  it('disables delete for epics with linked changes', async () => {
    const epic = epicFixture({ id: 3, name: 'API work', change_count: 2 });
    const wrapper = mount(EpicList, {
      props: {
        epics: [epic],
        loading: false,
      },
      global: {
        stubs: quasarStubs,
      },
    });

    await wrapper.find('[data-icon="delete"]').trigger('click');

    expect(wrapper.find('[data-icon="delete"]').attributes('disabled')).toBeDefined();
    expect(wrapper.emitted('delete')).toBeUndefined();
  });
});
