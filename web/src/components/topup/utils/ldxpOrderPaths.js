function appendAutoStart(path, options = {}) {
  if (!options?.autoStart) {
    return path;
  }
  return `${path}?autostart=1`;
}

export function buildLdxpTopupPayPath(tradeNo, options = {}) {
  return appendAutoStart(
    `/console/topup/ldxp/orders/${encodeURIComponent(tradeNo)}/pay`,
    options,
  );
}

export function buildLdxpSubscriptionPayPath(tradeNo, options = {}) {
  return appendAutoStart(
    `/console/subscription/ldxp/orders/${encodeURIComponent(tradeNo)}/pay`,
    options,
  );
}
