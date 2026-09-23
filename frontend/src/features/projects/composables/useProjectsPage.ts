import { computed, onMounted, ref } from 'vue';
import { storeToRefs } from 'pinia';
import type { Project } from '@/features/projects/model/project.types';
import { useProjectSelectionStore } from '@/features/projects/model/projectSelection.store';
import { useChangeCacheStore } from '@/features/changes/model/changeCache.store';
import {
  deleteChange,
  getChangePhases,
  getChangeTypes,
  updateChangePhase,
} from '@/features/changes/api/changeApi';
import type {
  ChangePhase,
  ChangeType,
  SelectOption,
  ChangeListItem,
} from '@/features/changes/model/change.types';

function errorMessage(err: unknown, fallback: string) {
  return err instanceof Error ? err.message : fallback;
}

interface UseProjectsPageOptions {
  changesEnabled?: boolean;
}

export function useProjectsPage(options: UseProjectsPageOptions = {}) {
  const changesEnabled = options.changesEnabled ?? true;
  const projectSelection = useProjectSelectionStore();
  const changeCache = useChangeCacheStore();
  const { projects, currentProjectId, currentProject } = storeToRefs(projectSelection);
  const { changes, epics } = storeToRefs(changeCache);
  const phases = ref<ChangePhase[]>([]);
  const types = ref<ChangeType[]>([]);
  const projectName = ref('');
  const changeTitle = ref('');
  const changeType = ref('');
  const changePhase = ref('');
  const loading = ref(false);
  const error = ref('');

  const projectDialogOpen = ref(false);
  const projectEditId = ref(0);
  const projectEditName = ref('');

  const confirmationDialogOpen = ref(false);
  let confirmationAction: (() => Promise<void>) | null = null;

  const phaseOptions = computed<SelectOption[]>(() =>
    phases.value.map((phase) => ({ label: phase.slug, value: phase.slug })),
  );
  const typeOptions = computed<SelectOption[]>(() =>
    types.value.map((type) => ({ label: type.slug, value: type.slug })),
  );
  const boardPhases = computed(() =>
    phases.value.length ? phases.value : uniqueChangePhases.value,
  );

  const uniqueChangePhases = computed<ChangePhase[]>(() =>
    [...new Set(changes.value.map((change) => change.change_phase))].map((slug, index) => ({
      slug,
      priority: index,
    })),
  );

  const filteredChanges = computed<ChangeListItem[]>(() => {
    const title = changeTitle.value.trim().toLowerCase();
    const type = changeType.value;
    const phase = changePhase.value;

    return changes.value.filter((change) => {
      if (title && !change.title.toLowerCase().includes(title)) return false;
      if (type && !change.change_types.includes(type)) return false;
      if (phase && change.change_phase !== phase) return false;
      return true;
    });
  });

  const changesByPhase = computed<Record<string, ChangeListItem[]>>(() => {
    const grouped: Record<string, ChangeListItem[]> = {};
    for (const phase of boardPhases.value) grouped[phase.slug] = [];
    for (const change of filteredChanges.value) {
      const group = grouped[change.change_phase] || [];
      group.push(change);
      grouped[change.change_phase] = group;
    }
    return grouped;
  });

  async function loadAll() {
    loading.value = true;
    error.value = '';

    try {
      if (changesEnabled) {
        const [loadedPhases, loadedTypes] = await Promise.all([
          getChangePhases(),
          getChangeTypes(),
          projectSelection.loadProjects(),
        ]);
        phases.value = loadedPhases;
        types.value = loadedTypes;

        if (currentProjectId.value) {
          await loadChanges(currentProjectId.value);
        } else {
          changeCache.setChanges([]);
        }
      } else {
        await projectSelection.loadProjects();
        changeCache.setChanges([]);
      }
    } catch (err) {
      error.value = errorMessage(err, 'Unable to load projects.');
    } finally {
      loading.value = false;
    }
  }

  async function loadChanges(projectId: number) {
    await changeCache.loadProjectChanges(projectId);
  }

  async function selectProject(projectId: number) {
    projectSelection.selectProject(projectId);
    error.value = '';
    try {
      if (!changesEnabled) {
        changeCache.setChanges([]);
      } else if (currentProjectId.value) {
        await loadChanges(currentProjectId.value);
      } else {
        changeCache.setChanges([]);
      }
    } catch (err) {
      error.value = errorMessage(err, 'Unable to load changes.');
    }
  }

  async function createProjectFromForm() {
    const name = projectName.value.trim();
    if (!name) return;

    try {
      const project = await projectSelection.createProject(name);
      projectName.value = '';
      if (changesEnabled) await loadChanges(project.id);
    } catch (err) {
      error.value = errorMessage(err, 'Unable to create project.');
    }
  }

  function startProjectRename(project: Project) {
    projectEditId.value = project.id;
    projectEditName.value = project.name;
    projectDialogOpen.value = true;
  }

  async function saveProjectName() {
    const name = projectEditName.value.trim();
    if (!projectEditId.value || !name) return;

    try {
      await projectSelection.renameProject(projectEditId.value, name);
      projectDialogOpen.value = false;
    } catch (err) {
      error.value = errorMessage(err, 'Unable to update project.');
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

  function removeProject(project: Project) {
    if (project.change_count > 0) {
      error.value = 'Delete all project changes before deleting this project.';
      return;
    }

    requestConfirmation(() => removeProjectConfirmed(project));
  }

  async function removeProjectConfirmed(project: Project) {
    try {
      const wasSelected = currentProjectId.value === project.id;
      await projectSelection.removeProject(project);
      if (changesEnabled && wasSelected) {
        changeCache.setChanges([]);
        if (currentProjectId.value) await loadChanges(currentProjectId.value);
      }
    } catch (err) {
      error.value = errorMessage(err, 'Unable to delete project.');
    }
  }

  async function searchChanges() {
    await loadAll();
  }

  async function clearChangeSearch() {
    changeTitle.value = '';
    changeType.value = '';
    changePhase.value = '';
    await loadAll();
  }

  async function moveChange(change: ChangeListItem, phase: string) {
    try {
      const moved = await updateChangePhase(change.id, phase);
      changeCache.upsertChange(moved);
      await refreshCurrentProjectChanges();
    } catch (err) {
      error.value = errorMessage(err, 'Unable to move change.');
    }
  }

  async function refreshCurrentProjectChanges() {
    if (currentProjectId.value) {
      await loadChanges(currentProjectId.value);
    }
  }

  function removeChange(change: ChangeListItem) {
    requestConfirmation(() => removeChangeConfirmed(change));
  }

  async function removeChangeConfirmed(change: ChangeListItem) {
    try {
      await deleteChange(change.id);
      await refreshCurrentProjectChanges();
      await projectSelection.loadProjects();
    } catch (err) {
      error.value = errorMessage(err, 'Unable to delete change.');
    }
  }

  onMounted(() => {
    void loadAll();
  });

  return {
    projects,
    changes,
    epics,
    phases,
    types,
    currentProjectId,
    projectName,
    changeTitle,
    changeType,
    changePhase,
    loading,
    error,
    projectDialogOpen,
    projectEditId,
    projectEditName,
    confirmationDialogOpen,
    currentProject,
    phaseOptions,
    typeOptions,
    boardPhases,
    changesByPhase,
    loadAll,
    selectProject,
    createProjectFromForm,
    startProjectRename,
    saveProjectName,
    removeProject,
    searchChanges,
    clearChangeSearch,
    moveChange,
    removeChange,
    confirm,
  };
}
