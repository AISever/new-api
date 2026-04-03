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

const t = (value, options = {}) =>
  value.replace(/\{\{(\w+)\}\}/g, (_, key) => options[key] ?? '');

const loadRowValidators = async () => import('./rowValidators.js').catch(() => ({}));

test('validateUserUsableGroupRows rejects rows with label but missing group key', async () => {
  const { validateUserUsableGroupRows } = await loadRowValidators();

  assert.equal(typeof validateUserUsableGroupRows, 'function');
  assert.equal(
    validateUserUsableGroupRows([{ key: '', value: 'VIP 用户' }], t),
    '用户可选分组中存在缺少分组名称的行',
  );
});

test('validateUserUsableGroupRows allows blank label when key exists', async () => {
  const { validateUserUsableGroupRows } = await loadRowValidators();

  assert.equal(
    validateUserUsableGroupRows([{ key: 'vip', value: '' }], t),
    '',
  );
});

test('validateSpecialUsableGroupRows rejects rows missing target group', async () => {
  const { validateSpecialUsableGroupRows } = await loadRowValidators();

  assert.equal(typeof validateSpecialUsableGroupRows, 'function');
  assert.equal(
    validateSpecialUsableGroupRows(
      [{ group: 'vip', action: 'add', targetGroup: '', description: '描述' }],
      t,
    ),
    '分组特殊可用分组中存在缺少目标分组的行',
  );
});

test('validateSpecialUsableGroupRows ignores untouched blank rows', async () => {
  const { validateSpecialUsableGroupRows } = await loadRowValidators();

  assert.equal(
    validateSpecialUsableGroupRows(
      [{ group: '', action: 'add', targetGroup: '', description: '' }],
      t,
    ),
    '',
  );
});
