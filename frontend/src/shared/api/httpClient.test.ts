import { get, post } from './httpClient';

describe('httpClient', () => {
  const fetchMock = vi.fn();

  beforeEach(() => {
    fetchMock.mockReset();
    vi.stubGlobal('fetch', fetchMock);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('sends JSON POST requests and returns parsed payloads', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: () => Promise.resolve({ id: 1 }),
    });

    await expect(post('/api/v1/project/create', { name: 'Project' })).resolves.toEqual({ id: 1 });
    expect(fetchMock).toHaveBeenCalledWith('http://localhost:8080/api/v1/project/create', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ name: 'Project' }),
    });
  });

  it('handles no-content responses', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      status: 204,
    });

    await expect(post('/api/v1/change/delete', { id: 1 })).resolves.toBeUndefined();
  });

  it('throws backend error messages', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: false,
      status: 400,
      json: () => Promise.resolve({ message: 'invalid payload' }),
    });

    await expect(post('/api/v1/change/create', {})).rejects.toThrow('invalid payload');
  });

  it('throws fallback errors when backend response has no message', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: false,
      status: 500,
      json: () => Promise.resolve({}),
    });

    await expect(get('/api/v1/health')).rejects.toThrow('Backend request failed.');
  });
});
