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
  normalizeAnnouncementPublishDate,
  ensureOptionUpdateSucceeded,
} from './announcements.js';

test('normalizeAnnouncementPublishDate serializes Date values to RFC3339', () => {
  const publishDate = new Date('2026-04-19T08:30:45.000Z');

  assert.equal(
    normalizeAnnouncementPublishDate(publishDate),
    '2026-04-19T08:30:45.000Z',
  );
});

test('normalizeAnnouncementPublishDate accepts ISO date strings from DatePicker state', () => {
  assert.equal(
    normalizeAnnouncementPublishDate('2026-04-19T08:30:45.000Z'),
    '2026-04-19T08:30:45.000Z',
  );
});

test('normalizeAnnouncementPublishDate rejects invalid publishDate values', () => {
  assert.throws(
    () => normalizeAnnouncementPublishDate('not-a-date'),
    /发布日期格式无效/,
  );
});

test('ensureOptionUpdateSucceeded throws when option save fails', () => {
  assert.throws(
    () =>
      ensureOptionUpdateSucceeded({
        success: false,
        message: '第1个公告的发布日期格式错误',
      }),
    /第1个公告的发布日期格式错误/,
  );
});

test('ensureOptionUpdateSucceeded tolerates successful option saves', () => {
  assert.doesNotThrow(() =>
    ensureOptionUpdateSucceeded({
      success: true,
      message: '',
    }),
  );
});
