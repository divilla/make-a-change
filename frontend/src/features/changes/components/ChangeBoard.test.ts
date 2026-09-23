import { mount } from '@vue/test-utils';
import { changeFixture } from '../model/change.fixtures';
import ChangeBoard from './ChangeBoard.vue';

const quasarStubs = {
  QBadge: { props: ['label'], template: '<span>{{ label }}<slot /></span>' },
  QBtn: {
    emits: ['click'],
    props: ['icon'],
    template: '<button :data-icon="icon" @click="$emit(\'click\', $event)"><slot /></button>',
  },
  QCard: {
    emits: ['click'],
    template: '<article class="change-card-stub" @click="$emit(\'click\', $event)"><slot /></article>',
  },
  QCardActions: { template: '<div><slot /></div>' },
  QCardSection: { template: '<section><slot /></section>' },
  QIcon: { props: ['name'], template: '<span :data-icon="name"><slot /></span>' },
  QLinearProgress: { template: '<div />' },
  QSelect: {
    emits: ['update:modelValue'],
    props: ['modelValue'],
    template:
      '<button class="phase-select" @click="$emit(\'update:modelValue\', \'review\')">{{ modelValue }}</button>',
  },
  QTooltip: { template: '<span><slot /></span>' },
};

describe('ChangeBoard', () => {
  it('renders changes by phase and emits change actions', async () => {
    const backlogChange = changeFixture({ id: 1, ref: 201, title: 'Backlog change', change_phase: 'backlog' });
    const reviewChange = changeFixture({ id: 2, ref: 202, title: 'Review change', change_phase: 'review' });
    const wrapper = mount(ChangeBoard, {
      props: {
        hasSelectedProject: true,
        boardPhases: [
          { slug: 'backlog', priority: 1 },
          { slug: 'review', priority: 2 },
        ],
        changesByPhase: {
          backlog: [backlogChange],
          review: [reviewChange],
        },
        phaseOptions: [
          { label: 'backlog', value: 'backlog' },
          { label: 'review', value: 'review' },
        ],
      },
      global: {
        stubs: quasarStubs,
      },
    });

    expect(wrapper.text()).toContain('Backlog change');
    expect(wrapper.text()).toContain('#201');
    expect(wrapper.text()).toContain('Review change');

    await wrapper.find('.change-card-stub').trigger('click');
    await wrapper.find('.phase-select').trigger('click');
    await wrapper.find('[data-icon="delete"]').trigger('click');

    expect(wrapper.emitted('open-change')).toEqual([[backlogChange]]);
    expect(wrapper.emitted('move-change')).toEqual([[backlogChange, 'review']]);
    expect(wrapper.emitted('delete-change')).toEqual([[backlogChange]]);
  });

  it('renders a blank ref for unreferenced changes', () => {
    const unreferencedChange = changeFixture({ id: 1, ref: null, title: 'Unreferenced change', change_phase: 'backlog' });
    const wrapper = mount(ChangeBoard, {
      props: {
        hasSelectedProject: true,
        boardPhases: [{ slug: 'backlog', priority: 1 }],
        changesByPhase: {
          backlog: [unreferencedChange],
        },
        phaseOptions: [{ label: 'backlog', value: 'backlog' }],
      },
      global: {
        stubs: quasarStubs,
      },
    });

    expect(wrapper.text()).toContain('Unreferenced change');
    expect(wrapper.text()).not.toContain('#null');
    expect(wrapper.text()).not.toContain('id:');
  });
});
