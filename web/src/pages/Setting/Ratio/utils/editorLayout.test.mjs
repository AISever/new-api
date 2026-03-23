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

const loadEditorLayout = async () => import('./editorLayout.js').catch(() => ({}));

test('getCompactEditorLayoutMode keeps desktop rows on non-mobile widths', async () => {
  const { getCompactEditorLayoutMode } = await loadEditorLayout();

  assert.equal(typeof getCompactEditorLayoutMode, 'function');
  assert.equal(
    getCompactEditorLayoutMode({
      isMobile: false,
      viewportWidth: 900,
    }),
    'desktop',
  );
});

test('getCompactEditorLayoutMode uses stacked cards for mobile widths', async () => {
  const { getCompactEditorLayoutMode } = await loadEditorLayout();

  assert.equal(typeof getCompactEditorLayoutMode, 'function');
  assert.equal(
    getCompactEditorLayoutMode({
      isMobile: true,
      viewportWidth: 390,
    }),
    'stacked',
  );
  assert.equal(
    getCompactEditorLayoutMode({
      isMobile: true,
      viewportWidth: 430,
    }),
    'stacked',
  );
});

test('shouldUseSharedInlineHeaders only enables shared headers for inline mobile layout', async () => {
  const { shouldUseSharedInlineHeaders } = await loadEditorLayout();

  assert.equal(typeof shouldUseSharedInlineHeaders, 'function');
  assert.equal(shouldUseSharedInlineHeaders('desktop'), false);
  assert.equal(shouldUseSharedInlineHeaders('stacked'), false);
  assert.equal(shouldUseSharedInlineHeaders('inline'), true);
});
