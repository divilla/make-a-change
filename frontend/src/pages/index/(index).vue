<template>
  <q-page class="app-page">
    <section class="page-heading">
      <div>
        <h1>Home</h1>
        <p>Completeness overview and local system status.</p>
      </div>
      <q-btn
        color="primary"
        icon="refresh"
        label="Refresh"
        :loading="loading"
        no-caps
        @click="loadHome"
      />
    </section>

    <q-banner v-if="error" class="status-banner status-banner--error" rounded>
      <template #avatar>
        <q-icon name="warning" />
      </template>
      {{ error }}
    </q-banner>

    <q-banner v-else-if="health" class="status-banner" rounded>
      <template #avatar>
        <q-icon :name="health.status === 'ok' ? 'check_circle' : 'error'" />
      </template>
      API: {{ health.api }} · Database: {{ health.database }}
    </q-banner>

    <div class="content-grid">
      <q-card flat bordered>
        <q-card-section>
          <div class="text-subtitle1">Project Completeness</div>
          <div v-if="currentProject" class="empty-state">
            <q-icon name="donut_large" size="40px" />
            <span>{{ currentProject.name }} has {{ currentProject.change_count }} tracked changes.</span>
          </div>
          <div v-else class="empty-state">
            <q-icon name="folder_open" size="40px" />
            <span>Create a project to enable dashboard metrics.</span>
          </div>
        </q-card-section>
      </q-card>

      <q-card flat bordered>
        <q-card-section>
          <div class="text-subtitle1">Phase Summary</div>
          <div v-if="currentProjectId" class="empty-state">
            <q-icon name="view_column" size="40px" />
            <span
              >Phase metrics will load for project #{{ currentProjectId }} when dashboard APIs are
              available.</span
            >
          </div>
          <div v-else class="empty-state">
            <q-icon name="view_column" size="40px" />
            <span>No active project selected.</span>
          </div>
        </q-card-section>
      </q-card>
    </div>
  </q-page>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { storeToRefs } from 'pinia';
import { useProjectSelectionStore } from '@/features/projects/model/projectSelection.store';
import { getHealth, type HealthResponse } from '@/services/api';

const projectSelection = useProjectSelectionStore();
const { currentProject, currentProjectId } = storeToRefs(projectSelection);
const health = ref<HealthResponse | null>(null);
const loading = ref(false);
const error = ref('');

async function loadHome() {
  loading.value = true;
  error.value = '';

  try {
    const [response] = await Promise.all([getHealth(), projectSelection.loadProjects()]);
    health.value = response;
  } catch (err) {
    health.value = null;
    error.value = err instanceof Error ? err.message : 'Unable to reach the backend.';
  } finally {
    loading.value = false;
  }
}

onMounted(() => {
  void loadHome();
});
</script>
