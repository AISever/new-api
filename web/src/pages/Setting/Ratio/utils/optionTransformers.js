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

const EMPTY_OBJECT_JSON = '{}';
const EMPTY_ARRAY_JSON = '[]';

const isNonEmptyString = (value) =>
  typeof value === 'string' && value.trim() !== '';

const formatJson = (value) => JSON.stringify(value, null, 2);

export const parseJsonObject = (rawValue) => {
  if (!isNonEmptyString(rawValue)) {
    return {};
  }

  try {
    const parsed = JSON.parse(rawValue);
    return parsed && typeof parsed === 'object' && !Array.isArray(parsed)
      ? parsed
      : {};
  } catch (error) {
    return {};
  }
};

const parseJsonArray = (rawValue) => {
  if (!isNonEmptyString(rawValue)) {
    return [];
  }

  try {
    const parsed = JSON.parse(rawValue);
    return Array.isArray(parsed) ? parsed : [];
  } catch (error) {
    return [];
  }
};

const stringifyNumberValue = (value) => {
  const numericValue = Number(value);
  return Number.isFinite(numericValue) ? numericValue : null;
};

export const parseSimpleMapOption = (rawValue, valueType = 'string') => {
  const parsed = parseJsonObject(rawValue);
  return Object.entries(parsed).map(([key, value]) => ({
    key,
    value:
      valueType === 'number'
        ? Number.isFinite(Number(value))
          ? String(value)
          : ''
        : String(value ?? ''),
  }));
};

export const stringifySimpleMapOption = (rows, valueType = 'string') => {
  const result = {};

  rows.forEach((row) => {
    if (!isNonEmptyString(row?.key)) {
      return;
    }

    if (valueType === 'number') {
      const numericValue = stringifyNumberValue(row.value);
      if (numericValue === null) {
        return;
      }
      result[row.key.trim()] = numericValue;
      return;
    }

    result[row.key.trim()] = String(row.value ?? '');
  });

  return Object.keys(result).length === 0
    ? EMPTY_OBJECT_JSON
    : formatJson(result);
};

export const parseGroupRelationOption = (rawValue) => {
  const parsed = parseJsonObject(rawValue);
  const rows = [];

  Object.entries(parsed).forEach(([group, targets]) => {
    if (!targets || typeof targets !== 'object' || Array.isArray(targets)) {
      return;
    }

    Object.entries(targets).forEach(([targetGroup, value]) => {
      rows.push({
        group,
        targetGroup,
        value: Number.isFinite(Number(value)) ? String(value) : '',
      });
    });
  });

  return rows;
};

export const stringifyGroupRelationOption = (rows) => {
  const result = {};

  rows.forEach((row) => {
    if (!isNonEmptyString(row?.group) || !isNonEmptyString(row?.targetGroup)) {
      return;
    }

    const numericValue = stringifyNumberValue(row.value);
    if (numericValue === null) {
      return;
    }

    const group = row.group.trim();
    const targetGroup = row.targetGroup.trim();

    if (!result[group]) {
      result[group] = {};
    }
    result[group][targetGroup] = numericValue;
  });

  return Object.keys(result).length === 0
    ? EMPTY_OBJECT_JSON
    : formatJson(result);
};

export const parseSpecialUsableGroupOption = (rawValue) => {
  const parsed = parseJsonObject(rawValue);
  const rows = [];

  Object.entries(parsed).forEach(([group, rules]) => {
    if (!rules || typeof rules !== 'object' || Array.isArray(rules)) {
      return;
    }

    Object.entries(rules).forEach(([targetGroup, description]) => {
      if (targetGroup.startsWith('+:')) {
        rows.push({
          group,
          action: 'add',
          targetGroup: targetGroup.slice(2),
          description: String(description ?? ''),
        });
        return;
      }

      if (targetGroup.startsWith('-:')) {
        rows.push({
          group,
          action: 'remove',
          targetGroup: targetGroup.slice(2),
          description: String(description ?? ''),
        });
        return;
      }

      rows.push({
        group,
        action: 'direct',
        targetGroup,
        description: String(description ?? ''),
      });
    });
  });

  return rows;
};

export const stringifySpecialUsableGroupOption = (rows) => {
  const result = {};

  rows.forEach((row) => {
    if (!isNonEmptyString(row?.group) || !isNonEmptyString(row?.targetGroup)) {
      return;
    }

    const group = row.group.trim();
    const targetGroup = row.targetGroup.trim();
    const description = String(row.description ?? '');

    if (!result[group]) {
      result[group] = {};
    }

    if (row.action === 'add') {
      result[group][`+:${targetGroup}`] = description;
      return;
    }

    if (row.action === 'remove') {
      result[group][`-:${targetGroup}`] = description;
      return;
    }

    result[group][targetGroup] = description;
  });

  return Object.keys(result).length === 0
    ? EMPTY_OBJECT_JSON
    : formatJson(result);
};

export const parseOrderedStringListOption = (rawValue) =>
  parseJsonArray(rawValue)
    .filter((value) => isNonEmptyString(value))
    .map((value) => ({ value }));

export const stringifyOrderedStringListOption = (rows) => {
  const result = rows
    .map((row) => row?.value)
    .filter((value) => isNonEmptyString(value))
    .map((value) => value.trim());

  return result.length === 0 ? EMPTY_ARRAY_JSON : formatJson(result);
};
