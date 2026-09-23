<template>
  <q-select
    :model-value="currentProjectId || null"
    :options="projectOptions"
    :dark="dark"
    :disable="!projects.length || isSwitchingProject"
    :loading="loading || isSwitchingProject"
    dense
    outlined
    emit-value
    map-options
    options-dense
    class="project-selector"
    label="Project"
    :hint="showHint && !projects.length ? 'Create a project on the Projects page to enable selection' : undefined"
    @update:model-value="(value) => selectProject(Number(value))"
  />
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia';
import { useProjectSelectionStore } from '../model/projectSelection.store';

withDefaults(
  defineProps<{
    dark?: boolean;
    showHint?: boolean;
  }>(),
  {
    dark: false,
    showHint: true,
  },
);

const projectSelection = useProjectSelectionStore();
const { projects, currentProjectId, projectOptions, loading, isSwitchingProject } =
  storeToRefs(projectSelection);
const { selectProject } = projectSelection;
</script>
