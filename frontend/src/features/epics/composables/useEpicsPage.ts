import { onMounted, ref } from 'vue';
import { storeToRefs } from 'pinia';
import { createEpic, deleteEpic, getEpic, updateEpic } from '../api/epicApi';
import type { Epic } from '../model/epic.types';
import { useChangeCacheStore } from '@/features/changes/model/changeCache.store';
import { useProjectSelectionStore } from '@/features/projects/model/projectSelection.store';

function errorMessage(err: unknown, fallback: string) {
  return err instanceof Error ? err.message : fallback;
}

export function useEpicsPage() {
  const projectSelection = useProjectSelectionStore();
  const changeCache = useChangeCacheStore();
  const { currentProjectId } = storeToRefs(projectSelection);
  const { epics } = storeToRefs(changeCache);
  const loading = ref(false);
  const saving = ref(false);
  const error = ref('');
  const epicName = ref('');
  const loadedEpic = ref<Epic | null>(null);
  const confirmationDialogOpen = ref(false);
  let confirmationAction: (() => Promise<void>) | null = null;

  async function loadAll() {
    loading.value = true;
    error.value = '';

    try {
      if (!projectSelection.hasLoaded) await projectSelection.loadProjects();
      if (currentProjectId.value) {
        await changeCache.loadProjectChanges(currentProjectId.value);
      } else {
        changeCache.setChanges([], 0, []);
      }
    } catch (err) {
      error.value = errorMessage(err, 'Unable to load epics.');
    } finally {
      loading.value = false;
    }
  }

  async function loadEpic(id: number) {
    if (!id) {
      error.value = 'Invalid epic.';
      return;
    }

    loading.value = true;
    error.value = '';

    try {
      loadedEpic.value = await getEpic(id);
      epicName.value = loadedEpic.value.name;
    } catch (err) {
      loadedEpic.value = null;
      error.value = errorMessage(err, 'Unable to load epic.');
    } finally {
      loading.value = false;
    }
  }

  async function createEpicFromForm() {
    const name = epicName.value.trim();
    if (!name) return null;

    if (!currentProjectId.value) {
      error.value = 'Select a project before creating an epic.';
      return null;
    }

    saving.value = true;
    error.value = '';

    try {
      const epic = await createEpic({ project_id: currentProjectId.value, name });
      epicName.value = '';
      changeCache.upsertEpic(epic);
      await changeCache.loadProjectChanges(epic.project_id);
      return epic;
    } catch (err) {
      error.value = errorMessage(err, 'Unable to create epic.');
      return null;
    } finally {
      saving.value = false;
    }
  }

  async function saveEpicFromForm() {
    const name = epicName.value.trim();
    if (!loadedEpic.value || !name) return null;

    saving.value = true;
    error.value = '';

    try {
      const epic = await updateEpic({ id: loadedEpic.value.id, name });
      loadedEpic.value = epic;
      epicName.value = epic.name;
      changeCache.upsertEpic(epic);
      await changeCache.loadProjectChanges(epic.project_id);
      return epic;
    } catch (err) {
      error.value = errorMessage(err, 'Unable to update epic.');
      return null;
    } finally {
      saving.value = false;
    }
  }

  function requestConfirmation(action: () => Promise<void>) {
    confirmationAction = action;
    confirmationDialogOpen.value = true;
  }

  async function confirm() {
    const action = confirmationAction;
    if (!action) return;

    confirmationAction = null;
    confirmationDialogOpen.value = false;
    await action();
  }

  function removeEpic(epic: Epic) {
    if (epic.change_count > 0) {
      error.value = 'Delete all linked changes before deleting this epic.';
      return;
    }

    requestConfirmation(() => removeEpicConfirmed(epic));
  }

  async function removeEpicConfirmed(epic: Epic) {
    try {
      await deleteEpic(epic.id);
      changeCache.removeEpic(epic.id);
      if (currentProjectId.value) await changeCache.loadProjectChanges(currentProjectId.value);
    } catch (err) {
      error.value = errorMessage(err, 'Unable to delete epic.');
    }
  }

  onMounted(() => {
    void loadAll();
  });

  return {
    epics,
    currentProjectId,
    loading,
    saving,
    error,
    epicName,
    loadedEpic,
    confirmationDialogOpen,
    loadAll,
    loadEpic,
    createEpicFromForm,
    saveEpicFromForm,
    removeEpic,
    confirm,
  };
}
