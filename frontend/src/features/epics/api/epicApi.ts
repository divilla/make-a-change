import { post } from '@/shared/api/httpClient';
import type { Epic, EpicCreateInput, EpicUpdateInput } from '../model/epic.types';

export function listEpics(projectId: number): Promise<Epic[]> {
  return post<Epic[]>('/api/v1/epic/list', { project_id: projectId });
}

export function getEpic(id: number): Promise<Epic> {
  return post<Epic>('/api/v1/epic/get', { id });
}

export function createEpic(input: EpicCreateInput): Promise<Epic> {
  return post<Epic>('/api/v1/epic/create', input);
}

export function updateEpic(input: EpicUpdateInput): Promise<Epic> {
  return post<Epic>('/api/v1/epic/update', input);
}

export function deleteEpic(id: number): Promise<void> {
  return post<void>('/api/v1/epic/delete', { id });
}
