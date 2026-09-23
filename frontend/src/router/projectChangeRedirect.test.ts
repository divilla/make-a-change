import { describe, expect, it } from 'vitest';
import { PROJECT_CHANGE_LOADING_PATH, projectChangeTargetPath } from './projectChangeRedirect';

describe('projectChangeTargetPath', () => {
  it('targets project-scoped topic indexes', () => {
    expect(projectChangeTargetPath('/changes')).toBe('/changes');
    expect(projectChangeTargetPath('/changes/12')).toBe('/changes');
    expect(projectChangeTargetPath('/changes/create')).toBe('/changes');
    expect(projectChangeTargetPath('/changes/edit/12')).toBe('/changes');
    expect(projectChangeTargetPath('/projects/12')).toBe('/projects');
    expect(projectChangeTargetPath('/planning/session/12')).toBe('/planning');
  });

  it('keeps non-project-scoped pages on their current path', () => {
    expect(projectChangeTargetPath('/help')).toBe('/help');
    expect(projectChangeTargetPath('/')).toBe('/');
    expect(projectChangeTargetPath(PROJECT_CHANGE_LOADING_PATH)).toBe('/');
  });
});
