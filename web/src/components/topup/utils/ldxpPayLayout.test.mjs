import test from 'node:test';
import assert from 'node:assert/strict';

import { getLdxpPayLayout } from './ldxpPayLayout.js';

test('desktop layout keeps a balanced split and moderate placeholder height', () => {
  const layout = getLdxpPayLayout(false);

  assert.equal(layout.mobileUsesExternalPayFlow, false);
  assert.equal(layout.sidebarWidth, 380);
  assert.equal(layout.placeholderHeight, 260);
  assert.equal(layout.iframeHeight, 780);
  assert.match(layout.contentGridClassName, /xl:grid-cols-\[minmax\(360px,380px\)_minmax\(0,1fr\)\]/);
});

test('mobile layout still uses external pay flow', () => {
  const layout = getLdxpPayLayout(true);

  assert.equal(layout.mobileUsesExternalPayFlow, true);
  assert.equal(layout.placeholderHeight, 220);
  assert.equal(layout.iframeHeight, 620);
});
