const modes = new Set(['off', 'all', 'selected']);
const moneyPattern = /^\d+(?:\.\d+)?$/;
export function parseInvitationVisibility(raw) {
  if (!raw) return { mode: 'off', user_ids: [] };
  const value = typeof raw === 'string' ? JSON.parse(raw) : raw;
  if (
    !modes.has(value?.mode) ||
    !Array.isArray(value.user_ids) ||
    value.user_ids.length > 1000
  )
    throw new Error('Invalid invitation policy');
  if (value.user_ids.some((id) => !Number.isSafeInteger(id) || id <= 0))
    throw new Error('Invalid selected user');
  return {
    mode: value.mode,
    user_ids: [...new Set(value.user_ids)].sort((a, b) => a - b),
  };
}
export function serializeInvitationVisibility(value) {
  return JSON.stringify(parseInvitationVisibility(value));
}
export function createInvitationRequestFence() {
  let sequence = 0;
  return {
    begin(identity) {
      return { sequence: ++sequence, identity };
    },
    accept(ticket) {
      return ticket.sequence === sequence;
    },
    invalidate() {
      sequence += 1;
    },
  };
}
const amount = (value) =>
  typeof value === 'string' && moneyPattern.test(value) ? value : null;
export function invitationMoney(value, currency) {
  return amount(value) === null ? '—' : `${currency} ${value}`;
}
export function decodeInvitationPage(response) {
  const page = response?.data;
  if (response?.success !== true || !Array.isArray(page?.items))
    throw new Error('Unable to read invitations');
  const allowed = page.can_view_amounts === true;
  const count = (n) => (Number.isSafeInteger(n) && n >= 0 ? n : 0);
  return {
    total: count(page.total),
    page: page.page || 1,
    page_size: page.page_size || 20,
    can_view_amounts: allowed,
    items: page.items
      .filter((item) => typeof item?.username === 'string')
      .map((item) => {
        const row = { username: item.username };
        if (allowed && item.amounts) {
          const a = item.amounts;
          row.amounts = {
            currencies: (Array.isArray(a.currencies) ? a.currencies : [])
              .filter((c) => /^[A-Z]{3,8}$/.test(c.currency))
              .map((c) => ({
                currency: c.currency,
                currency_symbol:
                  typeof c.currency_symbol === 'string'
                    ? c.currency_symbol
                    : '',
                admin_topup: amount(c.admin_topup),
                topup: amount(c.topup),
                subscription: amount(c.subscription),
                redemption: amount(c.redemption),
                total: amount(c.total),
                estimated: c.estimated === true,
                conversion_incomplete: c.conversion_incomplete === true,
                conversions: (Array.isArray(c.conversions) ? c.conversions : [])
                  .filter(
                    (f) =>
                      amount(f.quota_per_unit) !== null &&
                      amount(f.exchange_rate) !== null,
                  )
                  .slice(0, 50)
                  .map((f) => ({
                    quota_per_unit: f.quota_per_unit,
                    exchange_rate: f.exchange_rate,
                    estimated: f.estimated === true,
                  })),
              })),
            legacy_truncated: a.legacy_truncated === true,
            unknown_count: count(a.unknown_count),
            unpriced_count: count(a.unpriced_count),
            legacy_recorded: (Array.isArray(a.legacy_recorded)
              ? a.legacy_recorded
              : []
            )
              .filter(
                (r) =>
                  amount(r.amount) !== null &&
                  ['topup', 'subscription'].includes(r.category),
              )
              .map((r) => ({
                category: r.category,
                amount: r.amount,
                count: count(r.count),
              })),
          };
        }
        return row;
      }),
  };
}
