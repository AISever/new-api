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

import { MESSAGE_ROLES } from '../constants/playground.constants';

export function getUserIdFromLocalStorage() {
  const user = localStorage.getItem('user');
  if (!user) return -1;

  try {
    return JSON.parse(user).id;
  } catch (error) {
    console.error('Failed to parse user from localStorage', error);
    return -1;
  }
}

export const formatMessageForAPI = (message) => {
  if (message.role === 'system') {
    return { role: MESSAGE_ROLES.SYSTEM, content: message.content };
  }
  if (message.role === 'user') {
    return { role: MESSAGE_ROLES.USER, content: message.content };
  }
  return { role: MESSAGE_ROLES.ASSISTANT, content: message.content };
};

export const isValidMessage = (message) => {
  return (
    message &&
    typeof message === 'object' &&
    typeof message.role === 'string' &&
    typeof message.content === 'string' &&
    message.content.trim() !== ''
  );
};
