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

function stripLeadingLdxpMarker(value) {
  return String(value || '')
    .replace(/^\s*LDXP[\s:：-]*/i, '')
    .trim();
}

export function formatUserFacingLdxpPlanTitle(value) {
  const normalized = stripLeadingLdxpMarker(value);
  if (!normalized) {
    return '';
  }

  const monthlyMatch = normalized.match(/^包月\s+(.+)$/);
  if (monthlyMatch) {
    return `${monthlyMatch[1].trim()} 包月套餐`;
  }

  const yearlyMatch = normalized.match(/^包年\s+(.+)$/);
  if (yearlyMatch) {
    return `${yearlyMatch[1].trim()} 包年套餐`;
  }

  return normalized;
}

export function formatUserFacingLdxpProductLabel(value) {
  return stripLeadingLdxpMarker(value);
}
