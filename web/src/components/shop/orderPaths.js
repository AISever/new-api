export function buildExternalShopOrderDetailPath(localTradeNo) {
  return `/console/shop/orders/${encodeURIComponent(localTradeNo)}`;
}

export function buildExternalShopOrderPayPath(localTradeNo) {
  return `${buildExternalShopOrderDetailPath(localTradeNo)}/pay`;
}

export function isExternalShopOrderPayable(order) {
  if (!order?.pay_url) {
    return false;
  }
  return ['created', 'pending_payment'].includes(order.status);
}
