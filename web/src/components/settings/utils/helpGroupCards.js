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

const DEFAULT_IMAGE_URL = '/WeChat-qun.jpg';

const DEFAULT_HELP_GROUP_CARDS = [
  {
    id: 'wechat-group-default',
    enabled: true,
    label: '群聊：AIGC交流二群',
    title: '欢迎加微信群沟通交流',
    description:
      '扫描二维码加入 AIGC 交流群，与更多开发者一起探讨 AI 编程工具的使用技巧和最佳实践。',
    image_url: DEFAULT_IMAGE_URL,
    sort_order: 0,
  },
];

const cloneCards = (cards) => cards.map((card) => ({ ...card }));

export const createDefaultHelpGroupCards = () =>
  cloneCards(DEFAULT_HELP_GROUP_CARDS);

export const createEmptyHelpGroupCard = (sortOrder = 0) => ({
  id: `help-group-card-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
  enabled: true,
  label: '',
  title: '',
  description: '',
  image_url: '',
  sort_order: sortOrder,
});

const normalizeHelpGroupCard = (card, index) => ({
  id: typeof card?.id === 'string' && card.id.trim() ? card.id.trim() : `help-group-card-${index + 1}`,
  enabled: card?.enabled !== false,
  label: typeof card?.label === 'string' ? card.label : '',
  title: typeof card?.title === 'string' ? card.title : '',
  description: typeof card?.description === 'string' ? card.description : '',
  image_url:
    typeof card?.image_url === 'string' && card.image_url.trim()
      ? card.image_url
      : DEFAULT_IMAGE_URL,
  sort_order:
    typeof card?.sort_order === 'number' && Number.isFinite(card.sort_order)
      ? card.sort_order
      : index,
});

const sortCards = (cards) =>
  [...cards].sort((left, right) => {
    if (left.sort_order === right.sort_order) {
      return left.id.localeCompare(right.id);
    }
      return left.sort_order - right.sort_order;
  });

export const readHelpGroupCardsOption = (
  rawValue,
  fallbackCards = createDefaultHelpGroupCards(),
) => {
  if (!rawValue || !rawValue.trim()) {
    return {
      cards: cloneCards(fallbackCards),
      error: '',
    };
  }

  try {
    const parsed = JSON.parse(rawValue);
    if (!Array.isArray(parsed)) {
      return {
        cards: cloneCards(fallbackCards),
        error: 'invalid_shape',
      };
    }
    return {
      cards: parsed.map(normalizeHelpGroupCard),
      error: '',
    };
  } catch {
    return {
      cards: cloneCards(fallbackCards),
      error: 'invalid_json',
    };
  }
};

export const parseHelpGroupCardsOption = (rawValue, fallbackCards = createDefaultHelpGroupCards()) => {
  return readHelpGroupCardsOption(rawValue, fallbackCards).cards;
};

export const stringifyHelpGroupCardsOption = (cards) =>
  JSON.stringify(
    cards.map((card, index) => ({
      ...normalizeHelpGroupCard(card, index),
    })),
    null,
    2,
  );

export const buildDisplayHelpGroupCards = (cards) =>
  sortCards(cards.map(normalizeHelpGroupCard)).filter((card) => card.enabled);

export const getDefaultHelpGroupCardImageURL = () => DEFAULT_IMAGE_URL;

export const getHelpGroupCardsValidationError = (cards) => {
  const seenIds = new Set();

  for (let index = 0; index < cards.length; index += 1) {
    const card = normalizeHelpGroupCard(cards[index], index);

    if (!card.id.trim()) {
      return {
        type: 'missing_id',
        index,
      };
    }

    if (!card.title.trim()) {
      return {
        type: 'missing_title',
        index,
      };
    }

    if (seenIds.has(card.id)) {
      return {
        type: 'duplicate_id',
        index,
        id: card.id,
      };
    }

    seenIds.add(card.id);
  }

  return null;
};
