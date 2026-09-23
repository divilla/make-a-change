<template>
  <q-list bordered separator class="project-list project-table">
    <div class="project-table-header" aria-hidden="true">
      <span>Name</span>
      <span>Created</span>
      <span>Modified</span>
      <span>Changes</span>
      <span></span>
    </div>

    <q-item
      v-for="project in projects"
      :key="project.id"
      clickable
      :active="project.id === selectedProjectId"
      active-class="selected-project"
      class="project-table-row"
      @click="$emit('select', project.id)"
    >
      <q-item-section class="project-name-cell">
        <q-item-label class="project-name">{{ project.name }}</q-item-label>
      </q-item-section>
      <q-item-section class="project-date-cell">
        <q-item-label caption>{{ formatProjectDate(project.created) }}</q-item-label>
      </q-item-section>
      <q-item-section class="project-date-cell">
        <q-item-label caption>{{ formatProjectDate(project.modified) }}</q-item-label>
      </q-item-section>
      <q-item-section class="project-count-cell">
        <q-item-label caption>
          {{ project.change_count }}
        </q-item-label>
      </q-item-section>
      <q-item-section side class="project-actions-cell">
        <div class="item-actions">
          <q-btn dense flat round icon="edit" @click.stop="$emit('rename', project)">
            <q-tooltip>Rename project</q-tooltip>
          </q-btn>
          <q-btn
            dense
            flat
            round
            icon="delete"
            color="negative"
            :disable="project.change_count > 0"
            @click.stop="$emit('delete', project)"
          >
            <q-tooltip>
              {{
                project.change_count > 0
                  ? 'Delete all project changes before deleting this project'
                  : 'Delete project'
              }}
            </q-tooltip>
          </q-btn>
        </div>
      </q-item-section>
    </q-item>
  </q-list>

  <div v-if="!projects.length && !loading" class="empty-state compact-empty">
    <q-icon name="folder_open" size="32px" />
    <span>Create a project to start tracking changes.</span>
  </div>
</template>

<script setup lang="ts">
import type { Project } from '../model/project.types';

function formatProjectDate(value: string) {
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(new Date(value));
}

defineProps<{
  projects: Project[];
  selectedProjectId: number;
  loading: boolean;
}>();

defineEmits<{
  select: [projectId: number];
  rename: [project: Project];
  delete: [project: Project];
}>();
</script>
