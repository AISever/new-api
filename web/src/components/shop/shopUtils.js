const namedEntities = {
  nbsp: ' ',
  amp: '&',
  lt: '<',
  gt: '>',
  quot: '"',
  apos: "'",
  '#39': "'",
};

function decodeHtmlEntity(entity) {
  const normalized = String(entity || '').toLowerCase();
  if (namedEntities[normalized]) {
    return namedEntities[normalized];
  }
  if (normalized.startsWith('#x')) {
    const codePoint = Number.parseInt(normalized.slice(2), 16);
    return Number.isNaN(codePoint) ? ' ' : String.fromCodePoint(codePoint);
  }
  if (normalized.startsWith('#')) {
    const codePoint = Number.parseInt(normalized.slice(1), 10);
    return Number.isNaN(codePoint) ? ' ' : String.fromCodePoint(codePoint);
  }
  return ' ';
}

const refreshableStatuses = new Set([
  'created',
  'pending_payment',
  'paid_waiting_delivery',
]);

export const AUTO_REFRESH_INTERVAL_MS = 30000;

const statusLabelMap = {
  created: '待支付',
  pending_payment: '待支付',
  paid_waiting_delivery: '待交付',
  delivered: '已交付',
  failed: '处理失败',
  expired: '已过期',
  manual_review: '待人工处理',
};

const removableSalesCopyPatterns = [
  /教程地址[^。！!？?\n]*/gi,
  /使用过程遇到问题[^。！!？?\n]*/gi,
  /使用过程中/gi,
  /联系售后[^。！!？?\n]*/gi,
  /咨询售后[^。！!？?\n]*/gi,
  /不会使用勿拍/gi,
  /访问不了官网勿拍/gi,
  /不懂勿拍/gi,
  /官网教程[^。！!？?\n]*/gi,
  /发货格式[:：]?\s*\[[^\]]+\]/gi,
  /(?:声明|注意)[:：]?/gi,
  /如遇[^。！!？?\n]*/gi,
  /(?:不同IP|IP)[^。！!？?\n]*/gi,
  /请不要频繁以\s*不同IP\s*登入登出/gi,
  /全程\s*质保[^。！!？?\n]*/gi,
  /质保[^。！!？?\n]*/gi,
  /欢迎购买测试[^。！!？?\n]*/gi,
  /车队共\d+个车位[^。！!？?\n]*/gi,
  /拉人[^。！!？?\n]*/gi,
  /接收验证码的地址[^。！!？?\n]*/gi,
  /忘记密码[^。！!？?\n]*/gi,
  /(?:可自行)?改密[^。！!？?\n]*/gi,
  /验证码[^。！!？?\n]*/gi,
  /微软密码[^。！!？?\n]*/gi,
  /兑换系统[^。！!？?\n]*/gi,
  /自助上车[^。！!？?\n]*/gi,
  /封控[^。！!？?\n]*/gi,
  /独家新测方案[^。！!？?\n]*/gi,
  /输入\s*您?的?邮箱[^。！!？?\n]*/gi,
];

const hiddenDescriptionFragmentPatterns = [
  /教程/i,
  /售后/i,
  /补发/i,
  /QQ/i,
  /Outlook/i,
  /验证码登录即可使用GPT/i,
  /通过接收验证码登录即可使用GPT/i,
  /勿拍/i,
  /使用过程中/i,
  /发货格式/i,
  /声明/i,
  /注意[:：]?/i,
  /质保/i,
  /封控/i,
  /验证码/i,
  /改密/i,
  /密码/i,
  /兑换系统/i,
  /上车/i,
  /官网/i,
];

function normalizeDescriptionFragment(value) {
  let fragment = String(value || '')
    .replace(/\s+/g, ' ')
    .trim();

  fragment = fragment
    .replace(/^[✅️🔝\s]+/u, '')
    .replace(/[✅️🔝\s]+$/u, '')
    .trim();

  for (let index = 0; index < 3; index += 1) {
    const duplicated = fragment.match(/^(.+?)\s+\1(?:\s|$)/u);
    if (!duplicated) {
      break;
    }
    fragment = duplicated[1].trim();
  }

  fragment = fragment
    .replace(/^(?:在|是|可|并|且|附带|提供|支持)\s*/u, '')
    .replace(/\s*(?:在|是|可|并|且|附带|提供|支持)$/u, '')
    .trim();

  return fragment;
}

export function normalizeExternalShopDescription(value) {
  const rawValue = String(value || '');
  const sanitized = rawValue
    .replace(/<[^>]+>/g, ' ')
    .replace(/&(#x?[0-9a-fA-F]+|[a-zA-Z]+);/g, (_, entity) => {
      return decodeHtmlEntity(entity);
    })
    .replace(/\bhttps?:\/\/[^\s]+/gi, ' ')
    .replace(/\b[a-z0-9._%+-]+@[a-z0-9.-]+\.[a-z]{2,}\b/gi, ' ')
    .replace(/\b(?:[a-z0-9-]+\.)+[a-z]{2,}(?:\/[^\s]*)?/gi, ' ')
    .replace(/(?:售后|联系|咨询)?\s*Q{1,2}\s*Q?\s*[:：]?\s*\d{5,}/gi, ' ')
    .replace(/【链接已隐藏】|【联系方式已隐藏】/g, ' ')
    .replace(
      new RegExp(
        removableSalesCopyPatterns.map((pattern) => pattern.source).join('|'),
        'gi',
      ),
      ' ',
    );

  const seenFragments = new Set();

  const normalized = sanitized
    .split(/[，,。！!？?\n]+/)
    .map((fragment) => normalizeDescriptionFragment(fragment))
    .filter(Boolean)
    .filter(
      (fragment) =>
        !hiddenDescriptionFragmentPatterns.some((pattern) =>
          pattern.test(fragment),
        ),
    )
    .filter((fragment) => fragment !== '在官网' && fragment !== '可')
    .filter((fragment) => !/^[A-Za-z]+$/u.test(fragment))
    .filter((fragment) => fragment.length >= 4)
    .filter((fragment) => {
      const normalized = fragment.replace(/\s+/g, ' ').trim();
      if (!normalized || seenFragments.has(normalized)) {
        return false;
      }
      seenFragments.add(normalized);
      return true;
    })
    .join('，')
    .replace(/[，,]{2,}/g, '，')
    .trim();

  if (normalized) {
    return normalized;
  }
  if (/兑换码/u.test(rawValue)) {
    return '此商品为兑换码';
  }
  if (/微软邮箱/u.test(rawValue)) {
    return '微软邮箱成品号';
  }
  if (/ChatGPT\s*Team/iu.test(rawValue)) {
    return '成品账号商品';
  }
  return '';
}

export function getExternalShopDescriptionDetail(value) {
  const detail = String(value || '')
    .replace(/<\s*br\s*\/?>/gi, '\n')
    .replace(/<\/p\s*>/gi, '\n\n')
    .replace(/<\/div\s*>/gi, '\n')
    .replace(/<li[^>]*>/gi, '- ')
    .replace(/<\/li\s*>/gi, '\n')
    .replace(/<[^>]+>/g, ' ')
    .replace(/&(#x?[0-9a-fA-F]+|[a-zA-Z]+);/g, (_, entity) =>
      decodeHtmlEntity(entity),
    )
    .replace(/\r/g, '')
    .replace(/[ \t]+\n/g, '\n')
    .replace(/\n[ \t]+/g, '\n')
    .replace(/[ \t]{2,}/g, ' ')
    .replace(/\n{3,}/g, '\n\n')
    .trim();

  return detail || normalizeExternalShopDescription(value);
}

export function buildExternalShopCategoryCards(goods, sourceCategories = []) {
  const normalizedGoods = Array.isArray(goods) ? goods : [];
  const normalizedCategories = Array.isArray(sourceCategories)
    ? sourceCategories
    : [];

  if (normalizedCategories.length > 0) {
    const categoryCounts = new Map();
    for (const good of normalizedGoods) {
      const categoryId = String(good?.category_id || '').trim();
      if (!categoryId) {
        continue;
      }
      categoryCounts.set(categoryId, (categoryCounts.get(categoryId) || 0) + 1);
    }

    return [
      {
        key: 'all',
        name: '全部商品',
        count: normalizedGoods.length,
      },
      ...normalizedCategories.map((category) => {
        const categoryId = String(category?.id || '').trim();
        return {
          key: categoryId || String(category?.name || '').trim() || '未分类',
          name: String(category?.name || '').trim() || '未分类',
          count:
            categoryCounts.get(categoryId) ??
            Number(category?.goods_count || 0),
        };
      }),
    ];
  }

  const categories = new Map();
  for (const good of normalizedGoods) {
    const categoryName = String(good?.category_name || '').trim() || '未分类';
    categories.set(categoryName, (categories.get(categoryName) || 0) + 1);
  }

  return [
    {
      key: 'all',
      name: '全部商品',
      count: normalizedGoods.length,
    },
    ...Array.from(categories.entries()).map(([name, count]) => ({
      key: name,
      name,
      count,
    })),
  ];
}

export function filterExternalShopGoods(
  goods,
  { categoryKey = 'all', searchTerm = '' } = {},
) {
  const normalizedSearchTerm = String(searchTerm || '').trim().toLowerCase();

  return (goods || []).filter((good) => {
    const categoryName = String(good?.category_name || '').trim() || '未分类';
    const categoryId = String(good?.category_id || '').trim();
    if (
      categoryKey &&
      categoryKey !== 'all' &&
      categoryName !== categoryKey &&
      categoryId !== String(categoryKey)
    ) {
      return false;
    }

    if (!normalizedSearchTerm) {
      return true;
    }

    const searchHaystack = [
      good?.name,
      categoryName,
      normalizeExternalShopDescription(good?.description),
    ]
      .filter(Boolean)
      .join(' ')
      .toLowerCase();

    return searchHaystack.includes(normalizedSearchTerm);
  });
}

export function getExternalShopStockLabel(good) {
  const stockCount = Number(good?.stock_count || 0);

  if (stockCount <= 0) {
    return '暂时缺货';
  }
  if (stockCount <= 5) {
    return '库存一般';
  }
  return '库存充足';
}

export function getExternalShopFulfillmentLabel(good) {
  return Number(good?.send_order || 0) === 0 ? '自动发货' : '人工处理';
}

export function getExternalShopPurchaseLimitLabel(good) {
  const stockCount = Number(good?.stock_count || 0);
  if (stockCount <= 1) {
    return `仅剩${Math.max(stockCount, 0)}件`;
  }
  return `最多${stockCount}件`;
}

export function getMaskedExternalShopContact(value) {
  const contact = String(value || '').trim();
  if (!contact) {
    return '-';
  }

  if (contact.includes('@')) {
    const [localPart, domain] = contact.split('@');
    if (!domain) {
      return `${contact.slice(0, 2)}***`;
    }
    const visibleLocal =
      localPart.length <= 1 ? localPart : localPart.slice(0, 1);
    return `${visibleLocal}***@${domain}`;
  }

  if (/^\d{7,}$/.test(contact)) {
    if (contact.length <= 7) {
      return `${contact.slice(0, 2)}***${contact.slice(-2)}`;
    }
    return `${contact.slice(0, 3)}****${contact.slice(-4)}`;
  }

  if (contact.length <= 4) {
    return `${contact[0] || ''}***`;
  }
  return `${contact.slice(0, 2)}***${contact.slice(-2)}`;
}

export function getExternalShopStatusLabel(status, t) {
  if (!status) {
    return '-';
  }
  const text = statusLabelMap[status] || status;
  return typeof t === 'function' ? t(text) : text;
}

export function parseExternalShopDeliveryCards(value) {
  if (!value) {
    return [];
  }
  try {
    const parsed = JSON.parse(value);
    if (Array.isArray(parsed?.cards)) {
      return parsed.cards.filter(Boolean);
    }
  } catch {}
  return [];
}

export function getExternalShopDeliverySummary(value, t) {
  const cards = parseExternalShopDeliveryCards(value);
  if (cards.length === 0) {
    return '-';
  }
  return typeof t === 'function' ? t('••••••••') : '••••••••';
}

export function shouldShowDescriptionToggle(value, threshold = 140) {
  return normalizeExternalShopDescription(value).length > threshold;
}

export function getAutoRefreshOrderTradeNos(orders, maxCount = 1) {
  if (!Array.isArray(orders) || maxCount <= 0) {
    return [];
  }
  return orders
    .filter(
      (order) => order?.local_trade_no && refreshableStatuses.has(order.status),
    )
    .sort((left, right) => {
      const leftSyncAt = Number(left?.last_sync_at || 0);
      const rightSyncAt = Number(right?.last_sync_at || 0);
      if (leftSyncAt !== rightSyncAt) {
        return leftSyncAt - rightSyncAt;
      }
      return Number(right?.created_at || 0) - Number(left?.created_at || 0);
    })
    .slice(0, maxCount)
    .map((order) => order.local_trade_no);
}
