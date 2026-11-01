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
      items: [
        {
          username: 'alice',
          amounts: {
            currency: 'USD',
            currency_symbol: '$',
            total: '0.00',
            estimated: false,
            unknown_count: 0,
            unpriced_count: 0,
          },
        },
        {
          username: 'bob',
          amounts: {
            currency: 'USD',
            currency_symbol: '$',
            total: '999999999999999.01',
            estimated: false,
            unknown_count: 1,
            unpriced_count: 0,
          },
        },
      ],
    },
  });
  assert.equal(
    invitationMoney(
      page.items[0].amounts.total,
      page.items[0].amounts.currency_symbol,
    ),
    '$0.00',
  );
  assert.equal(page.items[1].amounts.total, '999999999999999.01');
  assert.equal(page.items[1].amounts.unknown_count, 1);
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
test('authorized list retains one total while discarding source breakdown and payment details', () => {
  const page = decodeInvitationPage({
    success: true,
    data: {
      can_view_amounts: true,
      items: [
        {
          username: 'alice',
          amounts: {
            currency: 'USD',
            currency_symbol: '$',
            total: '6.30',
            estimated: false,
            unknown_count: 1,
            unpriced_count: 0,
            topup: '2.1',
            subscription: '3.2',
            redemption: '1',
            legacy_recorded: [{ amount: '9.99' }],
          },
        },
      ],
    },
  });
  assert.deepEqual(page.items[0].amounts, {
    currency: 'USD',
    currency_symbol: '$',
    total: '6.30',
    estimated: false,
    unknown_count: 1,
    unpriced_count: 0,
  });
});
test('one site-currency total is shown without retaining any currency list', () => {
  const page = decodeInvitationPage({
    success: true,
    data: {
      can_view_amounts: true,
      items: [
        {
          username: 'alice',
          amounts: {
            currency: 'USD',
            currency_symbol: '$',
            total: '7.30',
            estimated: true,
            unknown_count: 0,
            unpriced_count: 0,
            currencies: [{ currency: 'CNY', total: '10' }],
          },
        },
      ],
    },
  });
  assert.deepEqual(page.items[0].amounts, {
    currency: 'USD',
    currency_symbol: '$',
    total: '7.30',
    estimated: true,
    unknown_count: 0,
    unpriced_count: 0,
  });
  assert.equal(invitationMoney(page.items[0].amounts.total, '$'), '$7.30');
});
