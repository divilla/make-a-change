import type { TestCase } from '@/features/test-cases/model/testCase.types';

export type { Epic } from '@/features/epics/model/epic.types';

export interface ChangePhase {
  slug: string;
  priority: number;
  color?: string;
}

export interface ChangeType {
  slug: string;
  priority: number;
}

export interface ChangeListItem {
  id: number;
  ref_uuid: string;
  ref: number | null;
  slug: string | null;
  project_id: number;
  epic_id?: number | null;
  epic_name?: string | null;
  change_phase: string;
  change_types: string[];
  title: string;
  agent_edit: boolean;
  open: boolean;
  done_tc: number;
  total_tc: number;
  completed: number;
  modified: string;
}

export interface Change extends ChangeListItem {
  version: number;
  def: string;
  spec: string;
  spec_html: string;
  pr: string;
  pr_html: string;
  pr_url: string;
  created: string;
}

export interface ChangeDetail {
  change: Change;
  test_cases: TestCase[];
}

export interface ChangeRenderedArtifact {
  id: number;
  spec_html: string;
  pr_html: string;
}

export interface ChangeRenderedArtifactsResponse {
  artifacts: ChangeRenderedArtifact[];
}

export interface ChangeCreateInput {
  project_id: number;
  title: string;
  def: string;
}

export interface SelectOption {
  label: string;
  value: string;
}
