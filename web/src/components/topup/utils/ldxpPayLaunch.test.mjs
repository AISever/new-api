import test from 'node:test';
import assert from 'node:assert/strict';

import { launchLdxpExternalPay } from './ldxpPayLaunch.js';

test('launchLdxpExternalPay keeps the current page when a popup window is available', () => {
  const popupLocation = {
    replaceCalls: [],
    replace(url) {
      this.replaceCalls.push(url);
    },
  };
  const popupWindow = {
    location: popupLocation,
    opener: { existing: true },
  };
  const assignCalls = [];

  const result = launchLdxpExternalPay({
    paymentUrl: 'https://pay.ldxp.cn/order/123',
    openWindow: (url, target) => {
      assert.equal(url, 'about:blank');
      assert.equal(target, '_blank');
      return popupWindow;
    },
    assignLocation: (url) => assignCalls.push(url),
  });

  assert.equal(result.openedInNewWindow, true);
  assert.deepEqual(popupLocation.replaceCalls, ['https://pay.ldxp.cn/order/123']);
  assert.deepEqual(assignCalls, []);
  assert.equal(popupWindow.opener, null);
});

test('launchLdxpExternalPay falls back to current-page navigation when popup creation is blocked', () => {
  const assignCalls = [];

  const result = launchLdxpExternalPay({
    paymentUrl: 'https://pay.ldxp.cn/order/456',
    openWindow: () => null,
    assignLocation: (url) => assignCalls.push(url),
  });

  assert.equal(result.openedInNewWindow, false);
  assert.deepEqual(assignCalls, ['https://pay.ldxp.cn/order/456']);
});
