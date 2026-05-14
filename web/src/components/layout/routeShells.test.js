import test from 'node:test';
import assert from 'node:assert/strict';

import {
  shouldHidePublicFooter,
  shouldUsePublicShell,
} from './routeShells.js';

test('shouldUsePublicShell keeps public marketing and docs routes in shared shell', () => {
  assert.equal(shouldUsePublicShell('/'), true);
  assert.equal(shouldUsePublicShell('/about'), true);
  assert.equal(shouldUsePublicShell('/pricing'), true);
  assert.equal(shouldUsePublicShell('/help'), true);
  assert.equal(shouldUsePublicShell('/docs'), true);
  assert.equal(shouldUsePublicShell('/docs/getting-started'), true);
});

test('shouldUsePublicShell excludes console and auth routes', () => {
  assert.equal(shouldUsePublicShell('/console'), false);
  assert.equal(shouldUsePublicShell('/console/token'), false);
  assert.equal(shouldUsePublicShell('/login'), false);
  assert.equal(shouldUsePublicShell('/register'), false);
});

test('shouldHidePublicFooter only hides footer on pricing page', () => {
  assert.equal(shouldHidePublicFooter('/pricing'), true);
  assert.equal(shouldHidePublicFooter('/'), false);
  assert.equal(shouldHidePublicFooter('/help'), false);
});
