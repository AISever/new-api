import test from 'node:test';
import assert from 'node:assert/strict';
import { lazyWithRetry, maybeReloadForChunkError } from './lazyWithRetry.js';

function installWindowMocks() {
  const reloadCalls = [];
  const storage = new Map();

  global.window = {
    location: {
      reload: () => {
        reloadCalls.push('reload');
      },
    },
  };
  global.sessionStorage = {
    getItem: (key) => (storage.has(key) ? storage.get(key) : null),
    setItem: (key, value) => {
      storage.set(key, String(value));
    },
    removeItem: (key) => {
      storage.delete(key);
    },
  };

  return { reloadCalls, storage };
}

function cleanupWindowMocks() {
  delete global.window;
  delete global.sessionStorage;
}

test('maybeReloadForChunkError reloads once for stale lazy chunk failures', () => {
  const { reloadCalls, storage } = installWindowMocks();
  assert.equal(reloadCalls.length, 0);

  const pending = maybeReloadForChunkError(
    new Error('Failed to fetch dynamically imported module'),
    'subscription',
  );

  assert.equal(reloadCalls.length, 1);
  assert.equal(storage.get('lazy-retry:subscription'), '1');
  assert.equal(typeof pending.then, 'function');
  cleanupWindowMocks();
});

test('maybeReloadForChunkError rethrows after retry was already used', () => {
  const { storage } = installWindowMocks();
  storage.set('lazy-retry:subscription', '1');

  assert.throws(
    () =>
      maybeReloadForChunkError(
        new Error('Failed to fetch dynamically imported module'),
        'subscription',
      ),
    /Failed to fetch dynamically imported module/,
  );
  assert.equal(storage.has('lazy-retry:subscription'), false);
  cleanupWindowMocks();
});

test('lazyWithRetry passes through successful imports', async () => {
  installWindowMocks();
  const load = lazyWithRetry(
    async () => ({ default: 'ok' }),
    'subscription',
  );

  const module = await load();
  assert.deepEqual(module, { default: 'ok' });
  cleanupWindowMocks();
});
