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
  return amount(value) === null
    ? '—'
    : `${currency}${/^[A-Z]{3,8}$/.test(currency) ? ' ' : ''}${value}`;
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
          if (
            typeof a.currency !== 'string' ||
            !/^[A-Z]{3,8}$/.test(a.currency)
          )
            throw new Error('Invalid site currency');
          row.amounts = {
            currency: a.currency,
            currency_symbol:
              typeof a.currency_symbol === 'string' ? a.currency_symbol : '',
            total: amount(a.total),
            estimated: a.estimated === true,
            unknown_count: count(a.unknown_count),
            unpriced_count: count(a.unpriced_count),
          };
        }
        return row;
      }),
  };
}

export const initialInvitationView = {
  scope: null,
  data: null,
  loading: false,
  error: false,
};

function reconcileInvitationPage(previous, next) {
  if (!previous) return next;
  const byUsername = new Map(previous.items.map((row) => [row.username, row]));
  const items = next.items.map((row) => {
    const old = byUsername.get(row.username);
    return old && JSON.stringify(old) === JSON.stringify(row) ? old : row;
  });
  const sameItems =
    items.length === previous.items.length &&
    items.every((row, i) => row === previous.items[i]);
  if (
    sameItems &&
    previous.total === next.total &&
    previous.page === next.page &&
    previous.page_size === next.page_size &&
    previous.can_view_amounts === next.can_view_amounts
  )
    return previous;
  return { ...next, items: sameItems ? previous.items : items };
}

export function invitationViewReducer(state, action) {
  if (action.type === 'clear') return initialInvitationView;
  if (action.type === 'start') {
    if (state.scope === action.scope && state.data) {
      return state.loading || state.error
        ? { ...state, loading: false, error: false }
        : state;
    }
    return { scope: action.scope, data: null, loading: true, error: false };
  }
  if (state.scope !== action.scope) return state;
  if (action.type === 'success') {
    const data = reconcileInvitationPage(state.data, action.data);
    return data === state.data && !state.loading && !state.error
      ? state
      : { ...state, data, loading: false, error: false };
  }
  if (action.type === 'failure') {
    // Keep only public usernames when authorization could not be revalidated.
    const safe = state.data && {
      ...state.data,
      can_view_amounts: false,
      items: state.data.items.map(({ username }) => ({ username })),
    };
    return {
      ...state,
      data: safe && reconcileInvitationPage(state.data, safe),
      loading: false,
      error: true,
    };
  }
  return state;
}
