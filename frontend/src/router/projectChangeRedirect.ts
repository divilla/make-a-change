const PROJECT_SCOPED_TOPICS = new Set(['changes', 'projects', 'planning']);

export const PROJECT_CHANGE_LOADING_PATH = '/loading';

export function projectChangeTargetPath(path: string) {
  if (path === PROJECT_CHANGE_LOADING_PATH) return '/';

  const topic = path.replace(/^\/+/, '').split('/')[0];
  if (!topic) return '/';
  if (!PROJECT_SCOPED_TOPICS.has(topic)) return path;

  return `/${topic}`;
}
