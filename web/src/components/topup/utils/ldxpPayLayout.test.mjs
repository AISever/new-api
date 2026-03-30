import test from 'node:test';
import assert from 'node:assert/strict';

import { getLdxpPayLayout } from './ldxpPayLayout.js';

test('desktop ldxp pay layout keeps a dedicated summary column and larger iframe', () => {
  const layout = getLdxpPayLayout(false);

  assert.match(layout.contentGridClassName, /2xl:grid-cols-\[320px_minmax\(0,1fr\)\]/);
  assert.equal(layout.summaryActionsStacked, true);
  assert.equal(layout.iframeHeight, 860);
  assert.equal(
    layout.summaryGridClassName,
    'grid gap-2 [grid-template-columns:repeat(auto-fit,minmax(220px,1fr))]',
  );
});

test('mobile ldxp pay layout collapses to a single column with a shorter iframe', () => {
  const layout = getLdxpPayLayout(true);

  assert.equal(layout.contentGridClassName, 'grid grid-cols-1 gap-3');
  assert.equal(layout.summaryActionsStacked, false);
  assert.equal(layout.iframeHeight, 620);
  assert.equal(layout.mobileUsesExternalPayFlow, true);
  assert.equal(layout.mobileKeepsCurrentPage, true);
  assert.equal(layout.summaryGridClassName, 'grid grid-cols-1 gap-2');
});
