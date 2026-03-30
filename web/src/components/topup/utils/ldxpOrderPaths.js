export function buildLdxpTopupPayPath(tradeNo) {
  return `/console/topup/ldxp/orders/${encodeURIComponent(tradeNo)}/pay`;
}

export function buildLdxpSubscriptionPayPath(tradeNo) {
  return `/console/subscription/ldxp/orders/${encodeURIComponent(tradeNo)}/pay`;
}
