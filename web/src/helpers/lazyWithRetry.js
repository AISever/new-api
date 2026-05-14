const RELOAD_FLAG_PREFIX = 'lazy-retry:';

function isRetryableChunkError(error) {
  const message = String(error?.message || error || '').toLowerCase();
  return (
    message.includes('failed to fetch dynamically imported module') ||
    message.includes('error loading dynamically imported module') ||
    message.includes('importing a module script failed') ||
    message.includes('loading chunk') ||
    message.includes('chunkloaderror')
  );
}

function getReloadFlagKey(key) {
  return `${RELOAD_FLAG_PREFIX}${key}`;
}

export function maybeReloadForChunkError(error, key = 'global') {
  if (typeof window === 'undefined' || typeof sessionStorage === 'undefined') {
    throw error;
  }
  if (!isRetryableChunkError(error)) {
    throw error;
  }

  const flagKey = getReloadFlagKey(key);
  const alreadyRetried = sessionStorage.getItem(flagKey) === '1';
  if (alreadyRetried) {
    sessionStorage.removeItem(flagKey);
    throw error;
  }

  sessionStorage.setItem(flagKey, '1');
  window.location.reload();
  return new Promise(() => {});
}

export function resetChunkRetryFlag(key = 'global') {
  if (typeof sessionStorage === 'undefined') {
    return;
  }
  sessionStorage.removeItem(getReloadFlagKey(key));
}

export function lazyWithRetry(importer, key) {
  return () =>
    importer()
      .then((module) => {
        resetChunkRetryFlag(key);
        return module;
      })
      .catch((error) => maybeReloadForChunkError(error, key));
}
