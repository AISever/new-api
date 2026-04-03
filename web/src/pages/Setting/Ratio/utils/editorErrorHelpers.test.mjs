import test from 'node:test';
import assert from 'node:assert/strict';

const loadEditorErrorHelpers = async () =>
  import('./editorErrorHelpers.js').catch(() => ({}));

test('getSubmitBlockingError returns the first non-empty card error', async () => {
  const { getSubmitBlockingError } = await loadEditorErrorHelpers();

  assert.equal(typeof getSubmitBlockingError, 'function');
  assert.equal(
    getSubmitBlockingError({
      a: { error: '' },
      b: { error: 'bad json' },
      c: { error: 'later error' },
    }),
    'bad json',
  );
});
