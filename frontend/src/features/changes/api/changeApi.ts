import { post } from '@/shared/api/httpClient';
import type {
  Change,
  ChangeCreateInput,
  ChangeDetail,
  ChangeListItem,
  ChangePhase,
  ChangeRenderedArtifactsResponse,
  ChangeType,
} from '../model/change.types';

export function getChangePhases(): Promise<ChangePhase[]> {
  return post<ChangePhase[]>('/api/v1/options/change-phases-list');
}

export function getChangeTypes(): Promise<ChangeType[]> {
  return post<ChangeType[]>('/api/v1/options/change-types-list');
}

export function listChanges(projectId: number): Promise<ChangeListItem[]> {
  return post<ChangeListItem[]>('/api/v1/change/list', { project_id: projectId });
}

export function getChange(id: number): Promise<ChangeDetail> {
  return post<ChangeDetail>('/api/v1/change/get', { id });
}

export function getRenderedChangeArtifacts(ids: number[]): Promise<ChangeRenderedArtifactsResponse> {
  return post<ChangeRenderedArtifactsResponse>('/api/v1/change/rendered-artifacts', { ids });
}

export function createChange(input: ChangeCreateInput): Promise<Change> {
  return post<Change>('/api/v1/change/create', input);
}

export function updateChangeEpic(id: number, epicId: number | null): Promise<Change> {
  return post<Change>('/api/v1/change/update-epic', { id, epic_id: epicId });
}

export function updateChangeTitle(id: number, title: string): Promise<Change> {
  return post<Change>('/api/v1/change/update-title', { id, title });
}

export function updateChangeDefinition(id: number, def: string, agentEdit = false): Promise<Change> {
  return post<Change>('/api/v1/change/update-def', { id, def, agent_edit: agentEdit });
}

export function updateChangeSpec(id: number, spec: string, agentEdit = false): Promise<Change> {
  return post<Change>('/api/v1/change/update-spec', { id, spec, agent_edit: agentEdit });
}

export function updateChangeTypes(id: number, changeTypes: string[]): Promise<Change> {
  return post<Change>('/api/v1/change/update-change-types', { id, change_types: changeTypes });
}

export function updateChangePR(id: number, pr: string, agentEdit = false): Promise<Change> {
  return post<Change>('/api/v1/change/update-pr', { id, pr, agent_edit: agentEdit });
}

export function updateChangePRUrl(id: number, prUrl: string): Promise<Change> {
  return post<Change>('/api/v1/change/update-pr-url', { id, pr_url: prUrl });
}

export function updateChangePhase(id: number, changePhase: string): Promise<Change> {
  return post<Change>('/api/v1/change/update-phase', { id, change_phase: changePhase });
}

export function updateChangeOpen(id: number, open: boolean): Promise<Change> {
  return post<Change>('/api/v1/change/update-open', { id, open });
}

export function deleteChange(id: number): Promise<void> {
  return post<void>('/api/v1/change/delete', { id });
}
