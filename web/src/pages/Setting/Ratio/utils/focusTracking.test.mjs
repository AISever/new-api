import test from 'node:test';
import assert from 'node:assert/strict';

import { resolveAppendedRowFocus } from './focusTracking.js';

test('resolveAppendedRowFocus returns appended row id when requested', () => {
  assert.equal(
    resolveAppendedRowFocus({
      shouldFocusNewRow: true,
      previousLength: 1,
      rows: [{ id: 'row-1' }, { id: 'row-2' }],
    }),
    'row-2',
  );
});

test('resolveAppendedRowFocus ignores unchanged row counts', () => {
  assert.equal(
    resolveAppendedRowFocus({
      shouldFocusNewRow: true,
      previousLength: 2,
      rows: [{ id: 'row-1' }, { id: 'row-2' }],
    }),
    null,
  );
});
