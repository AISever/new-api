import test from 'node:test';
import assert from 'node:assert/strict';
import {
  buildExternalShopOrderDetailPath,
  buildExternalShopOrderPayPath,
  isExternalShopOrderPayable,
} from './orderPaths.js';

test('builds encoded local pay and detail paths for shop orders', () => {
  assert.equal(
    buildExternalShopOrderPayPath('LD 26/03?A'),
    '/console/shop/orders/LD%2026%2F03%3FA/pay',
  );
  assert.equal(
    buildExternalShopOrderDetailPath('LD 26/03?A'),
    '/console/shop/orders/LD%2026%2F03%3FA',
  );
});

test('treats created and pending payment orders as payable', () => {
  assert.equal(isExternalShopOrderPayable({ status: 'created', pay_url: 'https://pay.example' }), true);
  assert.equal(isExternalShopOrderPayable({ status: 'pending_payment', pay_url: 'https://pay.example' }), true);
  assert.equal(isExternalShopOrderPayable({ status: 'delivered', pay_url: 'https://pay.example' }), false);
  assert.equal(isExternalShopOrderPayable({ status: 'pending_payment', pay_url: '' }), false);
});
