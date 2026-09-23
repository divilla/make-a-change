import { defineStore } from 'pinia';
import { ref } from 'vue';
import { listChanges } from '../api/changeApi';
import type { ChangeListItem, Epic } from './change.types';
import { listEpics } from '@/features/epics/api/epicApi';

export const useChangeCacheStore = defineStore('changeCache', () => {
  const changes = ref<ChangeListItem[]>([]);
  const epics = ref<Epic[]>([]);
  const projectId = ref(0);
  const loading = ref(false);

  async function loadProjectChanges(nextProjectId: number) {
    loading.value = true;
    try {
      const [nextChanges, nextEpics] = await Promise.all([
        listChanges(nextProjectId),
        listEpics(nextProjectId),
      ]);
      changes.value = nextChanges;
      epics.value = nextEpics;
      projectId.value = nextProjectId;
      return changes.value;
    } finally {
      loading.value = false;
    }
  }

  function setChanges(
    items: ChangeListItem[],
    nextProjectId = projectId.value,
    nextEpics: Epic[] = epics.value,
  ) {
    changes.value = items;
    epics.value = nextEpics;
    projectId.value = nextProjectId;
  }

  function upsertChange(change: ChangeListItem) {
    if (projectId.value && change.project_id !== projectId.value) return;

    const exists = changes.value.some((item) => item.id === change.id);
    changes.value = exists
      ? changes.value.map((item) => (item.id === change.id ? change : item))
      : [...changes.value, change];
    projectId.value = change.project_id;
  }

  function upsertEpic(epic: Epic) {
    if (projectId.value && epic.project_id !== projectId.value) return;

    const exists = epics.value.some((item) => item.id === epic.id);
    epics.value = exists
      ? epics.value.map((item) => (item.id === epic.id ? epic : item))
      : [...epics.value, epic];
    projectId.value = epic.project_id;
  }

  function removeEpic(id: number) {
    epics.value = epics.value.filter((item) => item.id !== id);
  }

  return {
    changes,
    epics,
    projectId,
    loading,
    loadProjectChanges,
    setChanges,
    upsertChange,
    upsertEpic,
    removeEpic,
  };
});
