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

const INVALID_PUBLISH_DATE_MESSAGE = '发布日期格式无效';

export const normalizeAnnouncementPublishDate = (value) => {
  if (value == null || value === '') {
    throw new Error(INVALID_PUBLISH_DATE_MESSAGE);
  }

  if (value instanceof Date) {
    if (Number.isNaN(value.getTime())) {
      throw new Error(INVALID_PUBLISH_DATE_MESSAGE);
    }
    return value.toISOString();
  }

  if (typeof value === 'string' || typeof value === 'number') {
    const parsed = new Date(value);
    if (Number.isNaN(parsed.getTime())) {
      throw new Error(INVALID_PUBLISH_DATE_MESSAGE);
    }
    return parsed.toISOString();
  }

  if (typeof value?.toISOString === 'function') {
    const isoString = value.toISOString();
    const parsed = new Date(isoString);
    if (Number.isNaN(parsed.getTime())) {
      throw new Error(INVALID_PUBLISH_DATE_MESSAGE);
    }
    return parsed.toISOString();
  }

  throw new Error(INVALID_PUBLISH_DATE_MESSAGE);
};

export const ensureOptionUpdateSucceeded = (result) => {
  if (!result?.success) {
    throw new Error(result?.message || '系统公告更新失败');
  }

  return result;
};
