import test from 'node:test';
import assert from 'node:assert/strict';

import {
  isGroupVisibleInUserUsableRows,
  renameLinkedUserUsableGroup,
  setGroupVisibilityInUserUsableRows,
} from './groupVisibilityLinkage.js';

test('detects whether a group is visible in user usable groups', () => {
  const rows = [{ key: 'default', value: '默认分组' }];

  assert.equal(isGroupVisibleInUserUsableRows(rows, 'default'), true);
  assert.equal(isGroupVisibleInUserUsableRows(rows, 'vip'), false);
});

test('enabling visibility appends a missing user usable group row', () => {
  const rows = [{ key: 'default', value: '默认分组' }];

  assert.deepEqual(setGroupVisibilityInUserUsableRows(rows, 'vip', true), [
    { key: 'default', value: '默认分组' },
    { key: 'vip', value: 'vip' },
  ]);
});

test('disabling visibility removes the linked user usable group row', () => {
  const rows = [
    { key: 'default', value: '默认分组' },
    { key: 'vip', value: 'VIP 用户' },
  ];

  assert.deepEqual(setGroupVisibilityInUserUsableRows(rows, 'vip', false), [
    { key: 'default', value: '默认分组' },
  ]);
});

test('renaming a visible group preserves its display name', () => {
  const rows = [
    { key: 'default', value: '默认分组' },
    { key: 'vip', value: 'VIP 用户' },
  ];

  assert.deepEqual(renameLinkedUserUsableGroup(rows, 'vip', 'vip-new'), [
    { key: 'default', value: '默认分组' },
    { key: 'vip-new', value: 'VIP 用户' },
  ]);
});

test('renaming to an existing target keeps the existing row and removes the old one', () => {
  const rows = [
    { key: 'default', value: '默认分组' },
    { key: 'vip', value: 'VIP 用户' },
    { key: 'vip-new', value: '自定义名称' },
  ];

  assert.deepEqual(renameLinkedUserUsableGroup(rows, 'vip', 'vip-new'), [
    { key: 'default', value: '默认分组' },
    { key: 'vip-new', value: '自定义名称' },
  ]);
});
