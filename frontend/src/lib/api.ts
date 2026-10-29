export function getBackendHost() {
  const backendHost = import.meta.env.VITE_BACKEND_HOST?.trim();

  if (!backendHost) {
    throw new Error('Set VITE_BACKEND_HOST in envs/.env before continuing.');
  }

  return backendHost;
}
