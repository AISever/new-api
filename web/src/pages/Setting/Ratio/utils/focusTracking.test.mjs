import test from 'node:test';
import assert from 'node:assert/strict';

import { resolveAppendedRowFocus } from './focusTracking.js';

test('returns null when add-row focus was not requested', () => {
  assert.equal(
    resolveAppendedRowFocus({
      shouldFocusNewRow: false,
      previousLength: 1,
      rows: [{ id: 'row-1' }, { id: 'row-2' }],
    }),
    null,
  );
});

test('returns the appended row id when add-row focus was requested', () => {
  assert.equal(
    resolveAppendedRowFocus({
      shouldFocusNewRow: true,
      previousLength: 1,
      rows: [{ id: 'row-1' }, { id: 'row-2' }],
    }),
    'row-2',
  );
});

test('does not focus when row count did not increase', () => {
  assert.equal(
    resolveAppendedRowFocus({
      shouldFocusNewRow: true,
      previousLength: 2,
      rows: [{ id: 'row-1' }, { id: 'row-2' }],
    }),
    null,
  );
});
