import test from 'node:test';
import assert from 'node:assert/strict';

import { DEFAULT_OPTION_EDITOR_MODE } from './editorMode.js';

test('ratio editors default to json mode', () => {
  assert.equal(DEFAULT_OPTION_EDITOR_MODE, 'json');
});
