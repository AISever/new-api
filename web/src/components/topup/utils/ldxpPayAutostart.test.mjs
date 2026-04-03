import test from 'node:test';
import assert from 'node:assert/strict';

import {
  buildLdxpSubscriptionPayPath,
  buildLdxpTopupPayPath,
} from './ldxpOrderPaths.js';
import { shouldAutoStartLdxpPay } from './ldxpPayAutostart.js';

test('desktop autostarts only when query explicitly requests it', () => {
  assert.equal(shouldAutoStartLdxpPay({ isMobile: false, search: '?autostart=1' }), true);
  assert.equal(shouldAutoStartLdxpPay({ isMobile: false, search: '' }), false);
  assert.equal(shouldAutoStartLdxpPay({ isMobile: false, search: '?autostart=0' }), false);
});

test('mobile never autostarts embedded ldxp pay', () => {
  assert.equal(
    shouldAutoStartLdxpPay({
      isMobile: true,
      userAgent:
        'Mozilla/5.0 (iPhone; CPU iPhone OS 18_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.3 Mobile/15E148 Safari/604.1',
      search: '?autostart=1',
    }),
    false,
  );
});

test('narrow desktop windows can still autostart in-site payment flow', () => {
  assert.equal(
    shouldAutoStartLdxpPay({
      isMobile: true,
      userAgent:
        'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36',
      search: '?autostart=1',
    }),
    true,
  );
});

test('ldxp pay paths can request autostart explicitly', () => {
  assert.equal(
    buildLdxpTopupPayPath('TOP123', { autoStart: true }),
    '/console/topup/ldxp/orders/TOP123/pay?autostart=1',
  );
  assert.equal(
    buildLdxpSubscriptionPayPath('SUB123', { autoStart: true }),
    '/console/subscription/ldxp/orders/SUB123/pay?autostart=1',
  );
});
