/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import test from 'node:test';
import assert from 'node:assert/strict';

import { getSubmitBlockingError } from './editorErrorHelpers.js';

test('getSubmitBlockingError returns empty string when there are no card errors', () => {
  assert.equal(
    getSubmitBlockingError({
      groupRatio: { error: '' },
      userUsableGroups: { error: '' },
    }),
    '',
  );
});

test('getSubmitBlockingError returns the first table validation error verbatim', () => {
  assert.equal(
    getSubmitBlockingError({
      groupRatio: { error: '表格中存在非法数值：abc' },
      userUsableGroups: { error: '' },
    }),
    '表格中存在非法数值：abc',
  );
});

test('getSubmitBlockingError returns the first JSON validation error verbatim', () => {
  assert.equal(
    getSubmitBlockingError({
      groupRatio: { error: '' },
      userUsableGroups: { error: 'JSON 格式错误：Unexpected token }' },
    }),
    'JSON 格式错误：Unexpected token }',
  );
});
