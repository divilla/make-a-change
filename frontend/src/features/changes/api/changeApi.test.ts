import { createChange, updateChangeDefinition } from './changeApi';
import { post } from '@/shared/api/httpClient';

vi.mock('@/shared/api/httpClient', () => ({ post: vi.fn() }));

describe('change definition API', () => {
  beforeEach(() => {
    vi.mocked(post).mockReset();
    vi.mocked(post).mockResolvedValue({});
  });

  it('creates changes with the def contract', async () => {
    await createChange({ project_id: 7, title: 'Change', def: '# Change\n\nDefinition' });

    expect(post).toHaveBeenCalledWith('/api/v1/change/create', {
      project_id: 7,
      title: 'Change',
      def: '# Change\n\nDefinition',
    });
  });

  it('updates definitions with provenance through update-def', async () => {
    await updateChangeDefinition(12, '# Change\n\nRewritten definition', true);

    expect(post).toHaveBeenCalledWith('/api/v1/change/update-def', {
      id: 12,
      def: '# Change\n\nRewritten definition',
      agent_edit: true,
    });
  });
});
