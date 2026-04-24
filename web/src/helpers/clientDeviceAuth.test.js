import test from 'node:test';
import assert from 'node:assert/strict';

import { normalizeClientDeviceUserCode } from './clientDeviceAuth.js';

test('normalizeClientDeviceUserCode uppercases and inserts dash', () => {
  assert.equal(normalizeClientDeviceUserCode(' abcd ef12 '), 'ABCD-EF12');
});
