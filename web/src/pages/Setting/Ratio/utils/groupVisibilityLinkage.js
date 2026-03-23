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

const normalizeGroupKey = (value) => String(value ?? '').trim();

export const isGroupVisibleInUserUsableRows = (rows, groupKey) => {
  const normalizedGroupKey = normalizeGroupKey(groupKey);
  if (!normalizedGroupKey) {
    return false;
  }

  return rows.some((row) => normalizeGroupKey(row?.key) === normalizedGroupKey);
};

export const setGroupVisibilityInUserUsableRows = (rows, groupKey, visible) => {
  const normalizedGroupKey = normalizeGroupKey(groupKey);
  if (!normalizedGroupKey) {
    return rows;
  }

  if (visible) {
    if (isGroupVisibleInUserUsableRows(rows, normalizedGroupKey)) {
      return rows;
    }

    return [...rows, { key: normalizedGroupKey, value: normalizedGroupKey }];
  }

  return rows.filter(
    (row) => normalizeGroupKey(row?.key) !== normalizedGroupKey,
  );
};

export const renameLinkedUserUsableGroup = (rows, previousGroupKey, nextGroupKey) => {
  const normalizedPreviousKey = normalizeGroupKey(previousGroupKey);
  const normalizedNextKey = normalizeGroupKey(nextGroupKey);

  if (
    !normalizedPreviousKey ||
    !normalizedNextKey ||
    normalizedPreviousKey === normalizedNextKey
  ) {
    return rows;
  }

  const previousIndex = rows.findIndex(
    (row) => normalizeGroupKey(row?.key) === normalizedPreviousKey,
  );

  if (previousIndex === -1) {
    return rows;
  }

  const existingNextIndex = rows.findIndex(
    (row) => normalizeGroupKey(row?.key) === normalizedNextKey,
  );

  if (existingNextIndex !== -1) {
    return rows.filter((_, index) => index !== previousIndex);
  }

  return rows.map((row, index) =>
    index === previousIndex ? { ...row, key: normalizedNextKey } : row,
  );
};
