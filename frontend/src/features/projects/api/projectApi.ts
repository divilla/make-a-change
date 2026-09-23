import { post } from '@/shared/api/httpClient';
import type { Project } from '../model/project.types';

export function listProjects(): Promise<Project[]> {
  return post<Project[]>('/api/v1/project/list');
}

export function createProject(name: string): Promise<Project> {
  return post<Project>('/api/v1/project/create', { name });
}

export function updateProject(id: number, name: string): Promise<Project> {
  return post<Project>('/api/v1/project/update', { id, name });
}

export function deleteProject(id: number): Promise<void> {
  return post<void>('/api/v1/project/delete', { id });
}
