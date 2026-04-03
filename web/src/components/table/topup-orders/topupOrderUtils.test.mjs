import test from 'node:test';
import assert from 'node:assert/strict';

import {
  extractLdxpAuditInfo,
  getTopupAmountDisplay,
  parseProviderPayload,
} from './topupOrderUtils.js';

test('parseProviderPayload returns null for invalid json', () => {
  assert.equal(parseProviderPayload('{'), null);
});

test('extractLdxpAuditInfo reads cards and export url from merged provider payload', () => {
  const payload = JSON.stringify({
    query: {
      code: 1,
      data: {
        payurl: 'https://pay.ldxp.cn/shopApi/Pay/payment?trade_no=T-1',
      },
    },
    order_info: {
      data: {
        transaction_id: 'ALI-001',
        success_time: 1774851710,
        response: {
          cards: ['CARD-1', 'CARD-2'],
          export_cards_url: 'https://pay.ldxp.cn/export/cards/T-1',
        },
      },
    },
  });

  assert.deepEqual(extractLdxpAuditInfo(payload), {
    cards: ['CARD-1', 'CARD-2'],
    exportCardsUrl: 'https://pay.ldxp.cn/export/cards/T-1',
    transactionId: 'ALI-001',
    successTime: 1774851710,
  });
});

test('extractLdxpAuditInfo tolerates legacy payloads without order info', () => {
  const payload = JSON.stringify({
    data: {
      payurl: 'https://pay.ldxp.cn/shopApi/Pay/payment?trade_no=T-2',
    },
  });

  assert.deepEqual(extractLdxpAuditInfo(payload), {
    cards: [],
    exportCardsUrl: '',
    transactionId: '',
    successTime: 0,
  });
});

test('getTopupAmountDisplay keeps recharge amount aligned with list display', () => {
  assert.deepEqual(
    getTopupAmountDisplay({
      amount: 100,
      trade_no: 'LDXPUSR1NOWgCsVT1774851710',
    }),
    {
      isSubscription: false,
      value: '100',
    },
  );
});

test('getTopupAmountDisplay prefers decimal amount_value when present', () => {
  assert.deepEqual(
    getTopupAmountDisplay({
      amount: 0,
      amount_value: 0.1,
      payment_method: 'ldxp',
      trade_no: 'LDXPUSR1NO123456',
    }),
    {
      isSubscription: false,
      value: '0.1',
    },
  );
});

test('getTopupAmountDisplay marks subscription records explicitly', () => {
  assert.deepEqual(
    getTopupAmountDisplay({
      amount: 0,
      trade_no: 'SUBUSR1NO123456',
    }),
    {
      isSubscription: true,
      value: '订阅套餐',
    },
  );
});
