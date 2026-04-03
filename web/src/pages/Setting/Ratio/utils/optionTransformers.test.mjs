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

import {
  parseGroupRelationOption,
  parseOrderedStringListOption,
  parseSimpleMapOption,
  parseSpecialUsableGroupOption,
  stringifyGroupRelationOption,
  stringifyOrderedStringListOption,
  stringifySimpleMapOption,
  stringifySpecialUsableGroupOption,
} from './optionTransformers.js';

test('parseSimpleMapOption parses numeric map into editable rows', () => {
  assert.deepEqual(parseSimpleMapOption('{"vip":0.5,"test":1}', 'number'), [
    { key: 'vip', value: '0.5' },
    { key: 'test', value: '1' },
  ]);
});

test('stringifySimpleMapOption serializes editable numeric rows into formatted json', () => {
  assert.equal(
    stringifySimpleMapOption(
      [
        { key: 'vip', value: '0.5' },
        { key: 'test', value: '1' },
      ],
      'number',
    ),
    '{\n  "vip": 0.5,\n  "test": 1\n}',
  );
});

test('stringifySimpleMapOption preserves wildcard model names', () => {
  assert.equal(
    stringifySimpleMapOption([{ key: 'gpt-4-gizmo-*', value: '0.1' }], 'number'),
    '{\n  "gpt-4-gizmo-*": 0.1\n}',
  );
});

test('parseGroupRelationOption flattens nested relation map into rows', () => {
  assert.deepEqual(
    parseGroupRelationOption('{"vip":{"default":0.5,"test":1}}'),
    [
      { group: 'vip', targetGroup: 'default', value: '0.5' },
      { group: 'vip', targetGroup: 'test', value: '1' },
    ],
  );
});

test('stringifyGroupRelationOption rebuilds nested relation map from rows', () => {
  assert.equal(
    stringifyGroupRelationOption([
      { group: 'vip', targetGroup: 'default', value: '0.5' },
      { group: 'vip', targetGroup: 'test', value: '1' },
    ]),
    '{\n  "vip": {\n    "default": 0.5,\n    "test": 1\n  }\n}',
  );
});

test('parseSpecialUsableGroupOption expands add remove and direct rules', () => {
  assert.deepEqual(
    parseSpecialUsableGroupOption(
      '{"vip":{"+:premium":"高级分组","-:default":"默认分组","special":"特殊分组"}}',
    ),
    [
      {
        group: 'vip',
        action: 'add',
        targetGroup: 'premium',
        description: '高级分组',
      },
      {
        group: 'vip',
        action: 'remove',
        targetGroup: 'default',
        description: '默认分组',
      },
      {
        group: 'vip',
        action: 'direct',
        targetGroup: 'special',
        description: '特殊分组',
      },
    ],
  );
});

test('stringifySpecialUsableGroupOption rebuilds prefixed rule keys', () => {
  assert.equal(
    stringifySpecialUsableGroupOption([
      {
        group: 'vip',
        action: 'add',
        targetGroup: 'premium',
        description: '高级分组',
      },
      {
        group: 'vip',
        action: 'remove',
        targetGroup: 'default',
        description: '默认分组',
      },
      {
        group: 'vip',
        action: 'direct',
        targetGroup: 'special',
        description: '特殊分组',
      },
    ]),
    '{\n  "vip": {\n    "+:premium": "高级分组",\n    "-:default": "默认分组",\n    "special": "特殊分组"\n  }\n}',
  );
});

test('parseOrderedStringListOption preserves array order', () => {
  assert.deepEqual(parseOrderedStringListOption('["default","vip","svip"]'), [
    { value: 'default' },
    { value: 'vip' },
    { value: 'svip' },
  ]);
});

test('stringifyOrderedStringListOption rebuilds formatted array json', () => {
  assert.equal(
    stringifyOrderedStringListOption([
      { value: 'default' },
      { value: 'vip' },
      { value: 'svip' },
    ]),
    '[\n  "default",\n  "vip",\n  "svip"\n]',
  );
});
