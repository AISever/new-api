import test from 'node:test';
import assert from 'node:assert/strict';

test('vite config resolves plugins for build mode under Node 22', async () => {
  const configModule = await import('./vite.config.js');
  const configFactory = configModule.default;
  const resolved =
    typeof configFactory === 'function'
      ? await configFactory({ command: 'build', mode: 'production' })
      : configFactory;

  assert.ok(resolved);
  assert.ok(Array.isArray(resolved.plugins));
  assert.ok(resolved.plugins.length > 0);
});
