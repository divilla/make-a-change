import { useProjectSelectionStore } from '@/features/projects/model/projectSelection.store';
import { useChangeCacheStore } from '@/features/changes/model/changeCache.store';

export async function refreshProjectScope() {
  const projectSelection = useProjectSelectionStore();
  const changeCache = useChangeCacheStore();

  await projectSelection.loadProjects();

  if (projectSelection.currentProjectId) {
    await changeCache.loadProjectChanges(projectSelection.currentProjectId);
    return;
  }

  changeCache.setChanges([], 0);
}
