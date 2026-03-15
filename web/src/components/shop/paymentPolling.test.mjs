import test from 'node:test';
import assert from 'node:assert/strict';
import {
  DETAIL_POLL_INTERVAL_MS,
  EXTERNAL_PAY_URL_EXPOSURE_ALLOWED,
  IFRAME_LOAD_TIMEOUT_MS,
  PAYMENT_IFRAME_SANDBOX,
  RATE_LIMIT_BACKOFF_MS,
  REMOTE_REFRESH_INTERVAL_MS,
  isExternalShopOrderPaymentActive,
  shouldEnterEmbeddedConfirmationMode,
  shouldDisableRefreshAction,
  shouldHideIframeAfterLoad,
  isPayPagePollingEnabled,
  shouldShowEmbeddedReload,
  shouldBackoffPolling,
  shouldNavigateFromPayPage,
  shouldTriggerRemoteRefresh,
} from './paymentPolling.js';

test('keeps payment polling active only for payable or waiting-delivery orders', () => {
  assert.equal(isExternalShopOrderPaymentActive({ status: 'created' }), true);
  assert.equal(
    isExternalShopOrderPaymentActive({ status: 'pending_payment' }),
    true,
  );
  assert.equal(
    isExternalShopOrderPaymentActive({ status: 'paid_waiting_delivery' }),
    true,
  );
  assert.equal(
    isExternalShopOrderPaymentActive({ status: 'delivered' }),
    false,
  );
  assert.equal(isExternalShopOrderPaymentActive({ status: 'failed' }), false);
});

test('only triggers remote refresh after the safe interval elapses', () => {
  assert.equal(shouldTriggerRemoteRefresh(20_000, 0), false);
  assert.equal(shouldTriggerRemoteRefresh(20_000, 10_001), false);
  assert.equal(shouldTriggerRemoteRefresh(20_000, 5_000), true);
});

test('navigates away from pay page once payment is confirmed', () => {
  assert.equal(
    shouldNavigateFromPayPage({ status: 'paid_waiting_delivery' }),
    true,
  );
  assert.equal(shouldNavigateFromPayPage({ status: 'delivered' }), true);
  assert.equal(shouldNavigateFromPayPage({ status: 'pending_payment' }), false);
});

test('polling intervals stay within expected values', () => {
  assert.equal(DETAIL_POLL_INTERVAL_MS, 3000);
  assert.equal(REMOTE_REFRESH_INTERVAL_MS, 15000);
  assert.equal(IFRAME_LOAD_TIMEOUT_MS, 12000);
  assert.equal(RATE_LIMIT_BACKOFF_MS, 30000);
});

test('pay page polling stops once the embedded pay page is hidden', () => {
  assert.equal(
    isPayPagePollingEnabled(true, { status: 'pending_payment' }, true),
    true,
  );
  assert.equal(
    isPayPagePollingEnabled(false, { status: 'pending_payment' }, true),
    false,
  );
  assert.equal(
    isPayPagePollingEnabled(true, { status: 'pending_payment' }, false),
    false,
  );
  assert.equal(
    isPayPagePollingEnabled(true, { status: 'delivered' }, true),
    false,
  );
});

test('polling backs off while a recent rate limit window is still active', () => {
  assert.equal(shouldBackoffPolling(20_000, 0), false);
  assert.equal(shouldBackoffPolling(20_000, 20_001), true);
  assert.equal(shouldBackoffPolling(20_000, 19_999), false);
});

test('iframe sandbox keeps payment inside the embedded frame', () => {
  assert.equal(PAYMENT_IFRAME_SANDBOX.includes('allow-top-navigation'), false);
  assert.equal(PAYMENT_IFRAME_SANDBOX.includes('allow-forms'), true);
  assert.equal(PAYMENT_IFRAME_SANDBOX.includes('allow-scripts'), true);
});

test('pay page policy keeps the upstream site url out of the user-facing flow', () => {
  assert.equal(EXTERNAL_PAY_URL_EXPOSURE_ALLOWED, false);
  assert.equal(shouldShowEmbeddedReload(true), true);
  assert.equal(shouldShowEmbeddedReload(false), false);
});

test('iframe hides itself after a second load until local confirmation catches up', () => {
  assert.equal(
    shouldHideIframeAfterLoad(0, { status: 'pending_payment' }),
    false,
  );
  assert.equal(
    shouldHideIframeAfterLoad(1, { status: 'pending_payment' }),
    true,
  );
  assert.equal(shouldHideIframeAfterLoad(1, { status: 'delivered' }), false);
});

test('manual refresh action is disabled during the rate-limit backoff window', () => {
  assert.equal(shouldDisableRefreshAction(31_000, 60_000), true);
  assert.equal(shouldDisableRefreshAction(61_000, 60_000), false);
  assert.equal(shouldDisableRefreshAction(31_000, 0), false);
});

test('embedded confirmation mode hides the iframe once the user returns to the page', () => {
  assert.equal(
    shouldEnterEmbeddedConfirmationMode(true, true, {
      status: 'pending_payment',
    }),
    true,
  );
  assert.equal(
    shouldEnterEmbeddedConfirmationMode(true, false, {
      status: 'pending_payment',
    }),
    false,
  );
  assert.equal(
    shouldEnterEmbeddedConfirmationMode(false, true, {
      status: 'pending_payment',
    }),
    false,
  );
  assert.equal(
    shouldEnterEmbeddedConfirmationMode(true, true, { status: 'delivered' }),
    false,
  );
});
