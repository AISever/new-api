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
  assert.equal(shouldAutoStartLdxpPay({ isMobile: true, search: '?autostart=1' }), false);
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
