import test from 'node:test';
import assert from 'node:assert/strict';

import {
  buildDisplayHelpGroupCards,
  getHelpGroupCardsValidationError,
  parseHelpGroupCardsOption,
  readHelpGroupCardsOption,
  stringifyHelpGroupCardsOption,
} from './helpGroupCards.js';

test('parseHelpGroupCardsOption falls back to defaults when option is empty', () => {
  const parsed = parseHelpGroupCardsOption('', [
    {
      id: 'default',
      enabled: true,
      label: '群聊：默认',
      title: '默认群',
      description: '默认描述',
      image_url: '/WeChat-qun.jpg',
      sort_order: 0,
    },
  ]);

  assert.equal(parsed.length, 1);
  assert.equal(parsed[0].id, 'default');
});

test('stringifyHelpGroupCardsOption serializes cards into formatted json', () => {
  assert.equal(
    stringifyHelpGroupCardsOption([
      {
        id: 'group-a',
        enabled: true,
        label: '群聊：A',
        title: 'A 群',
        description: '描述 A',
        image_url: '/site-assets/help-group-card/a.png',
        sort_order: 2,
      },
    ]),
    '[\n  {\n    "id": "group-a",\n    "enabled": true,\n    "label": "群聊：A",\n    "title": "A 群",\n    "description": "描述 A",\n    "image_url": "/site-assets/help-group-card/a.png",\n    "sort_order": 2\n  }\n]',
  );
});

test('buildDisplayHelpGroupCards sorts by sort_order and filters disabled cards', () => {
  const displayCards = buildDisplayHelpGroupCards([
    {
      id: 'group-b',
      enabled: false,
      label: '群聊：B',
      title: 'B 群',
      description: '描述 B',
      image_url: '/site-assets/help-group-card/b.png',
      sort_order: 2,
    },
    {
      id: 'group-a',
      enabled: true,
      label: '群聊：A',
      title: 'A 群',
      description: '描述 A',
      image_url: '/site-assets/help-group-card/a.png',
      sort_order: 1,
    },
  ]);

  assert.deepEqual(displayCards.map((item) => item.id), ['group-a']);
});

test('readHelpGroupCardsOption reports invalid json instead of silently accepting it', () => {
  const result = readHelpGroupCardsOption('{', [
    {
      id: 'default',
      enabled: true,
      label: '群聊：默认',
      title: '默认群',
      description: '默认描述',
      image_url: '/WeChat-qun.jpg',
      sort_order: 0,
    },
  ]);

  assert.equal(result.error, 'invalid_json');
  assert.equal(result.cards.length, 1);
  assert.equal(result.cards[0].id, 'default');
});

test('getHelpGroupCardsValidationError catches duplicate ids and missing titles', () => {
  assert.deepEqual(
    getHelpGroupCardsValidationError([
      {
        id: 'group-a',
        enabled: true,
        label: '群聊：A',
        title: '',
        description: '描述 A',
        image_url: '/site-assets/help-group-card/a.png',
        sort_order: 0,
      },
    ]),
    { type: 'missing_title', index: 0 },
  );

  assert.deepEqual(
    getHelpGroupCardsValidationError([
      {
        id: 'group-a',
        enabled: true,
        label: '群聊：A',
        title: 'A 群',
        description: '描述 A',
        image_url: '/site-assets/help-group-card/a.png',
        sort_order: 0,
      },
      {
        id: 'group-a',
        enabled: true,
        label: '群聊：B',
        title: 'B 群',
        description: '描述 B',
        image_url: '/site-assets/help-group-card/b.png',
        sort_order: 1,
      },
    ]),
    { type: 'duplicate_id', index: 1, id: 'group-a' },
  );
});
