<template>
  <q-page class="app-page">
    <section class="page-heading">
      <div>
        <h1>Planning</h1>
        <p>AI-assisted planning workspace for decomposing features into changes and test cases.</p>
      </div>
    </section>

    <q-card flat bordered>
      <q-card-section>
        <div v-if="currentProject" class="empty-state">
          <q-icon name="psychology" size="44px" />
          <span>Planning workspace is scoped to {{ currentProject.name }}.</span>
          <q-btn color="primary" label="Commit changes" no-caps disabled />
        </div>
        <div v-else class="empty-state">
          <q-icon name="folder_open" size="44px" />
          <span>Create a project before committing planned changes.</span>
          <q-btn color="primary" label="Commit changes" no-caps disabled />
        </div>
      </q-card-section>
    </q-card>
  </q-page>
</template>

<script setup lang="ts">
import { onMounted } from 'vue';
import { storeToRefs } from 'pinia';
import { useProjectSelectionStore } from '@/features/projects/model/projectSelection.store';

const projectSelection = useProjectSelectionStore();
const { currentProject } = storeToRefs(projectSelection);

onMounted(() => {
  if (!projectSelection.hasLoaded) {
    void projectSelection.loadProjects().catch(() => undefined);
  }
});
</script>
