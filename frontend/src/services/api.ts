import { get } from '@/shared/api/httpClient';

export interface HealthResponse {
  status: string;
  api: string;
  database: string;
  error?: string;
}

export function getHealth(): Promise<HealthResponse> {
  return get<HealthResponse>('/api/v1/health');
}
