import test from 'node:test';
import assert from 'node:assert/strict';
import {
  invitationViewReducer,
  initialInvitationView,
} from './invitationDetails.js';
const scope = 'owner-2/page-1';
const page = () => ({
  total: 2,
  page: 1,
  page_size: 20,
  can_view_amounts: true,
  items: [
    {
      username: 'alice',
      amounts: {
        currency: 'USD',
        currency_symbol: '$',
        total: '6.30',
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
        total: '10.00',
        estimated: false,
        unknown_count: 0,
        unpriced_count: 0,
      },
    },
  ],
});
const loaded = () => ({ scope, data: page(), loading: false, error: false });
test('background request preserves visible rows and does not show the initial loader', () => {
  const current = loaded();
  const next = invitationViewReducer(current, { type: 'start', scope });
  assert.strictEqual(next, current);
  assert.equal(next.loading, false);
  assert.equal(next.data.items[0].amounts.total, '6.30');
});
test('unchanged background results reuse the complete state without rendering the table again', () => {
  const current = loaded();
  assert.strictEqual(
    invitationViewReducer(current, { type: 'success', scope, data: page() }),
    current,
  );
});
test('a changed total updates only its own row and preserves every unchanged row reference', () => {
  const current = loaded();
  const response = page();
  response.items[0].amounts.total = '9.30';
  const next = invitationViewReducer(current, {
    type: 'success',
    scope,
    data: response,
  });
  assert.equal(next.data.items[0].amounts.total, '9.30');
  assert.notStrictEqual(next.data.items[0], current.data.items[0]);
  assert.strictEqual(next.data.items[1], current.data.items[1]);
  assert.equal(current.data.items[0].amounts.total, '6.30');
});
test('new and deleted invitees follow server order without replacing unchanged survivors', () => {
  const current = loaded();
  const response = page();
  response.items = [
    response.items[1],
    {
      username: 'charlie',
      amounts: {
        currency: 'USD',
        currency_symbol: '$',
        total: '0.00',
        estimated: false,
        unknown_count: 0,
        unpriced_count: 0,
      },
    },
  ];
  const next = invitationViewReducer(current, {
    type: 'success',
    scope,
    data: response,
  });
  assert.deepEqual(
    next.data.items.map((r) => r.username),
    ['bob', 'charlie'],
  );
  assert.strictEqual(next.data.items[0], current.data.items[1]);
});
test('revocation immediately removes all amounts while retaining the username list', () => {
  const next = invitationViewReducer(loaded(), {
    type: 'success',
    scope,
    data: {
      ...page(),
      can_view_amounts: false,
      items: [{ username: 'alice' }, { username: 'bob' }],
    },
  });
  assert.equal(next.data.can_view_amounts, false);
  assert.deepEqual(next.data.items, [
    { username: 'alice' },
    { username: 'bob' },
  ]);
});
test('failed refresh hides old financial data but keeps non-sensitive usernames and retry state', () => {
  const next = invitationViewReducer(loaded(), { type: 'failure', scope });
  assert.equal(next.error, true);
  assert.equal(next.loading, false);
  assert.equal(next.data.can_view_amounts, false);
  assert.deepEqual(next.data.items, [
    { username: 'alice' },
    { username: 'bob' },
  ]);
});
test('account or page changes clear old data and late responses cannot repopulate another scope', () => {
  const next = invitationViewReducer(loaded(), {
    type: 'start',
    scope: 'owner-30/page-1',
  });
  assert.equal(next.data, null);
  assert.equal(next.loading, true);
  assert.strictEqual(
    invitationViewReducer(next, { type: 'success', scope, data: page() }),
    next,
  );
  assert.strictEqual(
    invitationViewReducer(next, { type: 'failure', scope }),
    next,
  );
});
test('closing the dialog clears financial state and permits a clean initial load', () => {
  const cleared = invitationViewReducer(loaded(), { type: 'clear' });
  assert.strictEqual(cleared, initialInvitationView);
  const next = invitationViewReducer(cleared, { type: 'start', scope });
  assert.equal(next.loading, true);
  assert.equal(next.data, null);
  const ready = invitationViewReducer(next, {
    type: 'success',
    scope,
    data: page(),
  });
  assert.equal(ready.loading, false);
  assert.equal(ready.data.items[0].amounts.total, '6.30');
});
