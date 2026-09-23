<template>
  <q-page class="app-page">
    <div class="change-form-heading">
      <q-btn flat round icon="arrow_back" aria-label="Back to epics" @click="goBack" />
      <div class="text-subtitle1">Edit Epic</div>
    </div>

    <q-banner v-if="error" class="status-banner status-banner--error" rounded>
      <template #avatar>
        <q-icon name="warning" />
      </template>
      {{ error }}
    </q-banner>

    <q-form class="change-create-form" @submit.prevent="saveEpicFromPage">
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
        <q-btn color="primary" icon="save" label="Save" type="submit" :loading="saving" />
      </div>
    </q-form>
  </q-page>
</template>

<script setup lang="ts">
import { computed, onMounted, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useEpicsPage } from '@/features/epics/composables/useEpicsPage';

const route = useRoute();
const router = useRouter();
const { epicName, loading, saving, error, loadEpic, saveEpicFromForm } = useEpicsPage();
const requiredRules = [
  (value: unknown) => {
    if (typeof value === 'string') return Boolean(value.trim()) || 'Required';
    return value != null || 'Required';
  },
];

const epicId = computed(() => {
  const params = route.params as { id?: string | string[] };
  const rawID = Array.isArray(params.id) ? params.id[0] : params.id;
  const value = Number(rawID);
  return Number.isInteger(value) && value > 0 ? value : 0;
});

async function loadEditPage() {
  await loadEpic(epicId.value);
}

async function saveEpicFromPage() {
  const epic = await saveEpicFromForm();
  if (epic) void router.push('/epics');
}

function goBack() {
  void router.push('/epics');
}

onMounted(() => {
  void loadEditPage();
});

watch(epicId, () => {
  void loadEditPage();
});
</script>
