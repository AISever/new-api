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

const hasText = (value) => String(value ?? '').trim() !== '';

export const validateUserUsableGroupRows = (rows, t) => {
  for (const row of rows) {
    const hasKey = hasText(row?.key);
    const hasValue = hasText(row?.value);

    if (!hasKey && !hasValue) {
      continue;
    }

    if (!hasKey) {
      return t('用户可选分组中存在缺少分组名称的行');
    }
  }

  return '';
};

export const validateSpecialUsableGroupRows = (rows, t) => {
  for (const row of rows) {
    const hasGroup = hasText(row?.group);
    const hasTargetGroup = hasText(row?.targetGroup);
    const hasDescription = hasText(row?.description);

    if (!hasGroup && !hasTargetGroup && !hasDescription) {
      continue;
    }

    if (!hasGroup) {
      return t('分组特殊可用分组中存在缺少用户分组的行');
    }

    if (!hasTargetGroup) {
      return t('分组特殊可用分组中存在缺少目标分组的行');
    }
  }

  return '';
};
