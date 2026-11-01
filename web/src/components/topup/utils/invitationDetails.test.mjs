import test from 'node:test';
import assert from 'node:assert/strict';
import {
  decodeInvitationPage,
  parseInvitationVisibility,
  serializeInvitationVisibility,
  createInvitationRequestFence,
  invitationMoney,
} from './invitationDetails.js';

test('unauthorized response discards every money field even if supplied', () => {
  const page = decodeInvitationPage({
    success: true,
    data: {
      can_view_amounts: false,
      total: 1,
      page: 1,
      page_size: 20,
      items: [
        {
          username: 'alice',
          email: 'secret',
          amounts: { currencies: [{ currency: 'USD', total: '900' }] },
        },
      ],
    },
  });
  assert.deepEqual(page.items, [{ username: 'alice' }]);
  assert.equal(page.can_view_amounts, false);
});
test('authorized zero and exact decimal amounts survive without numeric coercion', () => {
  const page = decodeInvitationPage({
    success: true,
    data: {
      can_view_amounts: true,
      total: 1,
      items: [
        {
          username: 'alice',
          amounts: {
            currencies: [
              {
                currency: 'JPY',
                topup: '0',
                subscription: '0',
                redemption: '0',
                total: '0',
              },
              {
                currency: 'USD',
                topup: '999999999999999.01',
                subscription: '0',
                redemption: '0',
                total: '999999999999999.01',
              },
            ],
            unknown_count: 1,
            unpriced_count: 0,
            legacy_recorded: [{ category: 'topup', amount: '9.99', count: 1 }],
          },
        },
      ],
    },
  });
  assert.equal(
    invitationMoney(page.items[0].amounts.currencies[0].total, 'JPY'),
    'JPY 0',
  );
  assert.equal(page.items[0].amounts.currencies[1].total, '999999999999999.01');
  assert.equal(page.items[0].amounts.unknown_count, 1);
});
test('selected policy is one canonical JSON value and refuses invalid IDs', () => {
  assert.equal(
    serializeInvitationVisibility({ mode: 'selected', user_ids: [9, 2, 9] }),
    '{"mode":"selected","user_ids":[2,9]}',
  );
  assert.deepEqual(parseInvitationVisibility(), { mode: 'off', user_ids: [] });
  for (const policy of [
    { mode: 'wrong', user_ids: [] },
    { mode: 'all', user_ids: [0] },
    { mode: 'selected', user_ids: [2.2] },
    { mode: 'selected', user_ids: [Number.MAX_SAFE_INTEGER + 1] },
  ])
    assert.throws(() => serializeInvitationVisibility(policy));
});
test('account changes, later requests and closing invalidate previous sensitive responses', () => {
  const fence = createInvitationRequestFence();
  const old = fence.begin('user1/page1');
  const current = fence.begin('user2/page1');
  assert.equal(fence.accept(old), false);
  assert.equal(fence.accept(current), true);
  fence.invalidate();
  assert.equal(fence.accept(current), false);
});
test('business failures and malformed list payload are errors rather than saved/success state', () => {
  assert.throws(() =>
    decodeInvitationPage({ success: false, data: { items: [] } }),
  );
  assert.throws(() =>
    decodeInvitationPage({ success: true, data: { items: {} } }),
  );
});
test('redemption conversion evidence is retained rather than relabeled with current exchange rates', () => {
  const page = decodeInvitationPage({
    success: true,
    data: {
      can_view_amounts: true,
      items: [
        {
          username: 'alice',
          amounts: {
            currencies: [
              {
                currency: 'CNY',
                topup: '0',
                subscription: '0',
                redemption: '7',
                total: '7',
                conversions: [
                  {
                    quota_per_unit: '500000',
                    exchange_rate: '7',
                    estimated: false,
                  },
                ],
              },
            ],
          },
        },
      ],
    },
  });
  assert.deepEqual(page.items[0].amounts.currencies[0].conversions, [
    { quota_per_unit: '500000', exchange_rate: '7', estimated: false },
  ]);
});
