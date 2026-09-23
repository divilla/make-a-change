<template>
  <q-page class="app-page">
    <section class="changes-page-actions">
      <ChangeSearchForm
        :visible="Boolean(currentProject)"
        v-model:title="changeTitle"
        v-model:change-type="changeType"
        v-model:change-phase="changePhase"
        :type-options="typeOptions"
        :phase-options="phaseOptions"
        :loading="loading"
        @new-change="openChangeCreate"
        @search="searchChanges"
        @clear="clearChangeSearch"
      />
    </section>

    <q-banner v-if="error" class="status-banner status-banner--error" rounded>
      <template #avatar>
        <q-icon name="warning" />
      </template>
      {{ error }}
    </q-banner>

    <main class="change-board">
      <ChangeBoard
        :has-selected-project="Boolean(currentProject)"
        :board-phases="boardPhases"
        :changes-by-phase="changesByPhase"
        :phase-options="phaseOptions"
        @open-change="openChangePage"
        @move-change="moveChange"
        @delete-change="removeChange"
      />
    </main>

    <DeleteConfirmationDialog v-model:open="confirmationDialogOpen" @confirm="confirm" />
  </q-page>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router';
import { useProjectsPage } from '@/features/projects/composables/useProjectsPage';
import ChangeBoard from '@/features/changes/components/ChangeBoard.vue';
import ChangeSearchForm from '@/features/changes/components/ChangeSearchForm.vue';
import type { ChangeListItem } from '@/features/changes/model/change.types';
import DeleteConfirmationDialog from '@/shared/ui/DeleteConfirmationDialog.vue';

const router = useRouter();

const {
  changeTitle,
  changeType,
  changePhase,
  loading,
  error,
  currentProject,
  phaseOptions,
  typeOptions,
  boardPhases,
  changesByPhase,
  confirmationDialogOpen,
  searchChanges,
  clearChangeSearch,
  moveChange,
  removeChange,
  confirm,
} = useProjectsPage();

function openChangePage(change: ChangeListItem) {
  void router.push(`/changes/${change.id}`);
}

function openChangeCreate() {
  void router.push('/changes/create');
}
</script>
