/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

export function parseProviderPayload(payload) {
  if (!payload || typeof payload !== 'string') {
    return null;
  }
  try {
    return JSON.parse(payload);
  } catch {
    return null;
  }
}

export function extractLdxpAuditInfo(payload) {
  const parsed = typeof payload === 'string' ? parseProviderPayload(payload) : payload;
  const orderInfo = parsed?.order_info?.data;
  const response = orderInfo?.response;

  return {
    cards: Array.isArray(response?.cards) ? response.cards.filter(Boolean) : [],
    exportCardsUrl: response?.export_cards_url || '',
    transactionId: orderInfo?.transaction_id || '',
    successTime: Number(orderInfo?.success_time || 0),
  };
}

export function getPaymentMethodLabel(method, t = (key) => key) {
  const labelMap = {
    stripe: 'Stripe',
    creem: 'Creem',
    waffo: 'Waffo',
    ldxp: '在线支付',
    wxpay: '微信',
    alipay: '支付宝',
  };
  const label = labelMap[method];
  return label ? t(label) : method || '-';
}

export function isSubscriptionTopupRecord(record) {
  const tradeNo = String(record?.trade_no || '').toLowerCase();
  return Number(record?.amount || 0) === 0 && tradeNo.startsWith('sub');
}

function formatTopupAmountValue(value) {
  const numericValue = Number(value);
  if (!Number.isFinite(numericValue) || numericValue <= 0) {
    return '0';
  }
  return String(Number(numericValue.toFixed(2)));
}

export function getTopupAmountDisplay(record) {
  if (isSubscriptionTopupRecord(record)) {
    return {
      isSubscription: true,
      value: '订阅套餐',
    };
  }

  if (Number(record?.amount_value || 0) > 0) {
    return {
      isSubscription: false,
      value: formatTopupAmountValue(record.amount_value),
    };
  }

  return {
    isSubscription: false,
    value: String(record?.amount ?? 0),
  };
}
