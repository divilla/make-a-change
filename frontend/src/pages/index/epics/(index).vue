<template>
  <q-page class="app-page">
    <q-banner v-if="error" class="status-banner status-banner--error" rounded>
      <template #avatar>
        <q-icon name="warning" />
      </template>
      {{ error }}
    </q-banner>

    <EpicList
      :epics="epics"
      :loading="loading"
      @create="openEpicCreate"
      @edit="openEpicEdit"
      @delete="removeEpic"
    />

    <DeleteConfirmationDialog v-model:open="confirmationDialogOpen" @confirm="confirm" />
  </q-page>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router';
import EpicList from '@/features/epics/components/EpicList.vue';
import { useEpicsPage } from '@/features/epics/composables/useEpicsPage';
import type { Epic } from '@/features/epics/model/epic.types';
import DeleteConfirmationDialog from '@/shared/ui/DeleteConfirmationDialog.vue';

const router = useRouter();
const {
  epics,
  loading,
  error,
  confirmationDialogOpen,
  removeEpic,
  confirm,
} = useEpicsPage();

function openEpicCreate() {
  void router.push('/epics/create');
}

function openEpicEdit(epic: Epic) {
  void router.push(`/epics/edit/${epic.id}`);
}
</script>
