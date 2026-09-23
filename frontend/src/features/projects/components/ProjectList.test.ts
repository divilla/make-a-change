import { mount } from '@vue/test-utils';
import type { Project } from '../model/project.types';
import ProjectList from './ProjectList.vue';

const quasarStubs = {
  QBadge: { template: '<span><slot /></span>' },
  QBtn: {
    emits: ['click'],
    props: ['icon', 'disable'],
    template:
      '<button :data-icon="icon" :disabled="disable" @click="!disable && $emit(\'click\', $event)"><slot /></button>',
  },
  QIcon: { template: '<span><slot /></span>' },
  QItem: {
    emits: ['click'],
    props: ['active'],
    template: '<li :data-active="active" @click="$emit(\'click\', $event)"><slot /></li>',
  },
  QItemLabel: { template: '<span><slot /></span>' },
  QItemSection: { template: '<div><slot /></div>' },
  QList: { template: '<ul><slot /></ul>' },
  QTooltip: { template: '<span><slot /></span>' },
};

function projectFixture(project: Pick<Project, 'id' | 'name' | 'change_count'>): Project {
  return {
    created: '2026-06-23T10:15:00Z',
    modified: '2026-06-24T11:30:00Z',
    ...project,
  };
}

function formattedDate(value: string) {
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(new Date(value));
}

describe('ProjectList', () => {
  it('emits select, rename, and delete events with the expected project', async () => {
    const project = projectFixture({ id: 1, name: 'Project', change_count: 0 });
    const wrapper = mount(ProjectList, {
      props: {
        projects: [project],
        selectedProjectId: 1,
        loading: false,
      },
      global: {
        stubs: quasarStubs,
      },
    });

    await wrapper.find('li').trigger('click');
    await wrapper.find('[data-icon="edit"]').trigger('click');
    await wrapper.find('[data-icon="delete"]').trigger('click');

    expect(wrapper.emitted('select')).toEqual([[1]]);
    expect(wrapper.emitted('rename')).toEqual([[project]]);
    expect(wrapper.emitted('delete')).toEqual([[project]]);
  });

  it('renders created and modified timestamps', () => {
    const project = projectFixture({ id: 1, name: 'Project', change_count: 0 });
    const wrapper = mount(ProjectList, {
      props: {
        projects: [project],
        selectedProjectId: 1,
        loading: false,
      },
      global: {
        stubs: quasarStubs,
      },
    });

    expect(wrapper.text()).toContain('Created');
    expect(wrapper.text()).toContain('Modified');
    expect(wrapper.text()).toContain(formattedDate(project.created));
    expect(wrapper.text()).toContain(formattedDate(project.modified));
  });

  it('disables delete for projects that still have changes', async () => {
    const project = projectFixture({ id: 1, name: 'Project', change_count: 2 });
    const wrapper = mount(ProjectList, {
      props: {
        projects: [project],
        selectedProjectId: 1,
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
