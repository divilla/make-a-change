<template>
  <q-page class="app-page">
    <div class="change-form-heading">
      <q-btn flat round icon="arrow_back" aria-label="Back to epics" @click="goBack" />
      <div class="text-subtitle1">Create Epic</div>
    </div>

    <q-banner v-if="error" class="status-banner status-banner--error" rounded>
      <template #avatar>
        <q-icon name="warning" />
      </template>
      {{ error }}
    </q-banner>

    <q-form class="change-create-form" @submit.prevent="createEpicFromPage">
      <q-input
        v-model="epicName"
        outlined
        label="Epic name"
        :disable="loading || saving"
        :rules="requiredRules"
        autofocus
      />

      <div class="change-create-actions">
        <q-btn flat icon="close" label="Cancel" :disable="saving" @click="goBack" />
        <q-btn color="primary" icon="save" label="Create" type="submit" :loading="saving" />
      </div>
    </q-form>
  </q-page>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router';
import { useEpicsPage } from '@/features/epics/composables/useEpicsPage';

const router = useRouter();
const { epicName, loading, saving, error, createEpicFromForm } = useEpicsPage();
const requiredRules = [
  (value: unknown) => {
    if (typeof value === 'string') return Boolean(value.trim()) || 'Required';
    return value != null || 'Required';
  },
];

async function createEpicFromPage() {
  const epic = await createEpicFromForm();
  if (epic) void router.push('/epics');
}

function goBack() {
  void router.push('/epics');
}
</script>
