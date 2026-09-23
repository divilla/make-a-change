<template>
  <q-page class="app-page">
    <div class="change-form-heading">
      <q-btn flat round icon="arrow_back" aria-label="Back to change" @click="goBack" />
      <div>
        <div class="text-caption text-grey-7">Change</div>
        <div v-if="loadedChange" class="text-subtitle1">
          #{{ loadedChange.id }} {{ loadedChange.title }}
        </div>
        <div v-else class="text-subtitle1">Loading</div>
      </div>
    </div>

    <q-banner v-if="error" class="status-banner status-banner--error" rounded>
      <template #avatar>
        <q-icon name="warning" />
      </template>
      {{ error }}
    </q-banner>

    <q-form class="change-create-form" @submit.prevent="saveChangeFromPage">
      <q-input
        v-model="title"
        outlined
        label="Change title"
        :disable="loading || saving"
        :rules="requiredRules"
        autofocus
      />

      <q-select
        v-model="changeTypes"
        outlined
        emit-value
        map-options
        multiple
        label="Types"
        :options="typeOptions"
        :disable="loading || saving"
      />

      <q-select
        v-model="changePhase"
        outlined
        emit-value
        map-options
        label="Phase"
        :options="phaseOptions"
        :disable="loading || saving"
      />

      <q-select
        v-model="epicId"
        outlined
        emit-value
        map-options
        clearable
        label="Epic"
        :options="epicOptions"
        :disable="loading || saving"
      />

      <q-toggle v-model="open" label="Open" :disable="loading || saving" />

      <q-input
        v-model="def"
        outlined
        type="textarea"
        label="Definition"
        class="change-artifact-input"
        input-style="min-height: 240px"
        :rules="requiredRules"
        :disable="loading || saving"
      />

      <q-input
        v-model="spec"
        outlined
        type="textarea"
        label="Spec"
        class="change-artifact-input"
        input-style="min-height: 600px"
        :disable="loading || saving"
      />

      <q-input
        v-model="pr"
        outlined
        type="textarea"
        label="PR body"
        class="change-artifact-input"
        input-style="min-height: 240px"
        :disable="loading || saving"
      />

      <q-input
        v-model="prUrl"
        outlined
        label="PR URL"
        :disable="loading || saving"
      />

      <div class="change-create-actions">
        <q-btn flat icon="close" label="Cancel" :disable="saving" @click="goBack" />
        <q-btn color="primary" icon="save" label="Save" type="submit" :loading="saving" />
      </div>
    </q-form>
  </q-page>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { storeToRefs } from 'pinia';
import {
  getChange,
  getChangePhases,
  getChangeTypes,
  updateChangeEpic,
  updateChangeOpen,
  updateChangePhase,
  updateChangeDefinition,
  updateChangeSpec,
  updateChangePR,
  updateChangePRUrl,
  updateChangeTitle,
  updateChangeTypes,
} from '@/features/changes/api/changeApi';
import { useChangeCacheStore } from '@/features/changes/model/changeCache.store';
import type { Change, SelectOption } from '@/features/changes/model/change.types';

const route = useRoute();
const router = useRouter();
const changeCache = useChangeCacheStore();
const { epics } = storeToRefs(changeCache);

const loadedChange = ref<Change | null>(null);
const title = ref('');
const def = ref('');
const spec = ref('');
const pr = ref('');
const prUrl = ref('');
const changeTypes = ref<string[]>([]);
const changePhase = ref('');
const epicId = ref<number | null>(null);
const open = ref(true);
const typeOptions = ref<SelectOption[]>([]);
const phaseOptions = ref<SelectOption[]>([]);
const loading = ref(false);
const saving = ref(false);
const error = ref('');
const requiredRules = [
  (value: unknown) => {
    if (typeof value === 'string') return Boolean(value.trim()) || 'Required';
    return value != null || 'Required';
  },
];

const changeId = computed(() => {
  const value = Number(route.params.changeId);
  return Number.isInteger(value) && value > 0 ? value : 0;
});

const epicOptions = computed(() =>
  epics.value.map((epic) => ({ label: epic.name, value: epic.id })),
);

async function loadEditContext() {
  if (!changeId.value) {
    error.value = 'Invalid change.';
    return;
  }

  loading.value = true;
  error.value = '';

  try {
    const [phases, types, detail] = await Promise.all([
      getChangePhases(),
      getChangeTypes(),
      getChange(changeId.value),
    ]);
    typeOptions.value = types.map((type) => ({ label: type.slug, value: type.slug }));
    phaseOptions.value = phases.map((phase) => ({ label: phase.slug, value: phase.slug }));
    loadedChange.value = detail.change;
    title.value = detail.change.title;
    def.value = detail.change.def;
    spec.value = detail.change.spec || '';
    pr.value = detail.change.pr || '';
    prUrl.value = detail.change.pr_url || '';
    changeTypes.value = [...detail.change.change_types];
    changePhase.value = detail.change.change_phase;
    epicId.value = detail.change.epic_id || null;
    open.value = detail.change.open;
    await changeCache.loadProjectChanges(detail.change.project_id);
    changeCache.upsertChange(detail.change);
  } catch (err) {
    loadedChange.value = null;
    error.value = err instanceof Error ? err.message : 'Unable to load change.';
  } finally {
    loading.value = false;
  }
}

async function saveChangeFromPage() {
  if (saving.value || !loadedChange.value) return;

  const changeTitle = title.value.trim();
  const changeDefinition = def.value.trim();
  if (!changeTitle || !changeDefinition) return;
  const validationError = changedArtifactValidationError(loadedChange.value);
  if (validationError) {
    error.value = validationError;
    return;
  }

  saving.value = true;
  error.value = '';

  try {
    let change = loadedChange.value;
    if (changeTitle !== change.title) {
      change = await updateChangeTitle(change.id, changeTitle);
    }
    if (changeDefinition !== change.def) {
      change = await updateChangeDefinition(change.id, changeDefinition);
    }
    const nextSpec = spec.value.trim();
    if (nextSpec !== change.spec) {
      change = await updateChangeSpec(change.id, nextSpec, false);
    }
    const nextPR = pr.value.trim();
    if (nextPR !== change.pr) {
      change = await updateChangePR(change.id, nextPR, false);
    }
    const nextPRUrl = prUrl.value.trim();
    if (nextPRUrl !== change.pr_url) {
      change = await updateChangePRUrl(change.id, nextPRUrl);
    }
    if (!sameStringList(changeTypes.value, change.change_types)) {
      change = await updateChangeTypes(change.id, changeTypes.value);
    }
    if (changePhase.value && changePhase.value !== change.change_phase) {
      change = await updateChangePhase(change.id, changePhase.value);
    }
    if ((epicId.value || null) !== (change.epic_id || null)) {
      change = await updateChangeEpic(change.id, epicId.value || null);
    }
    if (open.value !== change.open) {
      change = await updateChangeOpen(change.id, open.value);
    }

    await changeCache.loadProjectChanges(change.project_id);
    changeCache.upsertChange(change);
    void router.push(`/changes/${change.id}`);
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Unable to update change.';
  } finally {
    saving.value = false;
  }
}

function goBack() {
  if (loadedChange.value) {
    void router.push(`/changes/${loadedChange.value.id}`);
    return;
  }

  void router.push('/changes');
}

function changedArtifactValidationError(change: Change) {
  const nextSpec = spec.value.trim();
  if (nextSpec !== change.spec && !nextSpec) {
    return 'Spec is required.';
  }
  const nextPR = pr.value.trim();
  if (nextPR !== change.pr && !nextPR) {
    return 'PR is required.';
  }
  const nextPRUrl = prUrl.value.trim();
  if (nextPRUrl !== change.pr_url && !isHTTPURL(nextPRUrl)) {
    return 'PR URL must be an absolute http or https URL.';
  }
  return '';
}

function isHTTPURL(value: string) {
  if (!value) return false;
  try {
    const parsed = new URL(value);
    return Boolean(parsed.hostname) && (parsed.protocol === 'http:' || parsed.protocol === 'https:');
  } catch {
    return false;
  }
}

function sameStringList(left: string[], right: string[]) {
  return left.length === right.length && left.every((value, index) => value === right[index]);
}

onMounted(() => {
  void loadEditContext();
});

watch(changeId, () => {
  void loadEditContext();
});
</script>
