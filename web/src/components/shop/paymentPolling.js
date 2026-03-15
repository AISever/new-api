export const DETAIL_POLL_INTERVAL_MS = 3000;
export const REMOTE_REFRESH_INTERVAL_MS = 15000;
export const IFRAME_LOAD_TIMEOUT_MS = 12000;
export const RATE_LIMIT_BACKOFF_MS = 30000;
export const PAYMENT_IFRAME_SANDBOX =
  'allow-same-origin allow-scripts allow-forms allow-popups';
export const EXTERNAL_PAY_URL_EXPOSURE_ALLOWED = false;

export function isExternalShopOrderPaymentActive(order) {
  const status = order?.status;
  return (
    status === 'created' ||
    status === 'pending_payment' ||
    status === 'paid_waiting_delivery'
  );
}

export function shouldTriggerRemoteRefresh(nowMs, lastRemoteRefreshAtMs) {
  if (!lastRemoteRefreshAtMs) {
    return false;
  }
  return nowMs - lastRemoteRefreshAtMs >= REMOTE_REFRESH_INTERVAL_MS;
}

export function shouldNavigateFromPayPage(order) {
  const status = order?.status;
  return status === 'paid_waiting_delivery' || status === 'delivered';
}

export function isPayPagePollingEnabled(
  showIframe,
  order,
  isDocumentVisible = true,
) {
  return (
    Boolean(isDocumentVisible) &&
    Boolean(showIframe) &&
    isExternalShopOrderPaymentActive(order)
  );
}

export function shouldBackoffPolling(nowMs, blockedUntilMs) {
  return Boolean(blockedUntilMs) && nowMs < blockedUntilMs;
}

export function shouldShowEmbeddedReload(showIframe) {
  return Boolean(showIframe);
}

export function shouldHideIframeAfterLoad(loadCount, order) {
  return loadCount >= 1 && !shouldNavigateFromPayPage(order);
}

export function shouldDisableRefreshAction(nowMs, blockedUntilMs) {
  return shouldBackoffPolling(nowMs, blockedUntilMs);
}

export function shouldEnterEmbeddedConfirmationMode(
  showIframe,
  iframeLoaded,
  order,
) {
  return (
    Boolean(showIframe) &&
    Boolean(iframeLoaded) &&
    !shouldNavigateFromPayPage(order)
  );
}
