import test from 'node:test';
import assert from 'node:assert/strict';

import {
  buildLdxpSubscriptionPayPath,
  buildLdxpTopupPayPath,
} from './ldxpOrderPaths.js';

test('buildLdxpTopupPayPath encodes trade number into in-site pay route', () => {
  assert.equal(
    buildLdxpTopupPayPath('LD 26/03?A'),
    '/console/topup/ldxp/orders/LD%2026%2F03%3FA/pay',
  );
});

test('buildLdxpSubscriptionPayPath encodes trade number into in-site pay route', () => {
  assert.equal(
    buildLdxpSubscriptionPayPath('SUB 26/03?A'),
    '/console/subscription/ldxp/orders/SUB%2026%2F03%3FA/pay',
  );
});
