import test from 'node:test';
import assert from 'node:assert/strict';
import {
  AUTO_REFRESH_INTERVAL_MS,
  buildExternalShopCategoryCards,
  filterExternalShopGoods,
  getExternalShopDeliverySummary,
  getExternalShopDescriptionDetail,
  getExternalShopFulfillmentLabel,
  getExternalShopPurchaseLimitLabel,
  getExternalShopStockLabel,
  getMaskedExternalShopContact,
  parseExternalShopDeliveryCards,
  getExternalShopStatusLabel,
  getAutoRefreshOrderTradeNos,
  normalizeExternalShopDescription,
  shouldShowDescriptionToggle,
} from './shopUtils.js';

test('normalizeExternalShopDescription strips tags and decodes common html entities', () => {
  assert.equal(
    normalizeExternalShopDescription(
      '<p>Hello&nbsp;World &amp; &#x2705; &#39;ok&#39;</p>',
    ),
    "Hello World & ✅ 'ok'",
  );
});

test('normalizeExternalShopDescription hides direct links and support contacts', () => {
  const output = normalizeExternalShopDescription(
    '联系售后QQ: 578391246 教程地址 https://openai.com/zh-Hans-CN/codex 备用网址 GptTeamPlan.Tech 邮箱 help@example.com',
  );

  assert.equal(output.includes('578391246'), false);
  assert.equal(output.includes('openai.com'), false);
  assert.equal(output.includes('GptTeamPlan.Tech'), false);
  assert.equal(output.includes('help@example.com'), false);
  assert.equal(output.includes('【联系方式已隐藏】'), false);
  assert.equal(output.includes('【链接已隐藏】'), false);
});

test('normalizeExternalShopDescription removes user-irrelevant sales copy', () => {
  const output = normalizeExternalShopDescription(
    '使用过程遇到问题，联系售后QQ: 578391246 ChatGPT CodexAI编程官网教程地址在这里哦： https://openai.com/zh-Hans-CN/codex 不会使用勿拍 访问不了官网勿拍 不懂勿拍 Outlook.live.com登录微软邮箱可修改微软密码 在官网ChatGPT.COM通过接收验证码登录即可使用GPT',
  );

  assert.equal(output.includes('教程地址'), false);
  assert.equal(output.includes('不会使用勿拍'), false);
  assert.equal(output.includes('访问不了官网勿拍'), false);
  assert.equal(output.includes('不懂勿拍'), false);
  assert.equal(output.includes('Outlook'), false);
  assert.equal(output.includes('验证码登录即可使用GPT'), false);
});

test('normalizeExternalShopDescription removes upstream operational and delivery noise', () => {
  const output = normalizeExternalShopDescription(
    '发货格式: [微软邮箱--微软密码--接收验证码的地址] 注意:使用过程中，请不要频繁以不同IP登入登出 声明:如遇封控被停用问题可自用 ChatGPT密码可在登录时点击忘记密码进行改密操作 此商品为兑换码，在兑换系统中输入您的邮箱以及兑换码即可自助上车，全程质保一个月',
  );

  assert.equal(output.includes('发货格式'), false);
  assert.equal(output.includes('不同IP'), false);
  assert.equal(output.includes('声明'), false);
  assert.equal(output.includes('封控'), false);
  assert.equal(output.includes('忘记密码'), false);
  assert.equal(output.includes('兑换系统'), false);
  assert.equal(output.includes('自助上车'), false);
  assert.equal(output.includes('质保'), false);
  assert.equal(output, '此商品为兑换码');
});

test('normalizeExternalShopDescription keeps concise product-facing summary text', () => {
  const output = normalizeExternalShopDescription(
    'ChatGPT 官网 Plus 会员独享一个月，微软邮箱成品号。ChatGPT 官网 Plus 会员独享一个月，微软邮箱成品号。✅️长效微软邮箱,附带接收验证码的地址,可自行改密',
  );

  assert.equal(output, '微软邮箱成品号，长效微软邮箱');
});

test('getExternalShopDescriptionDetail preserves full decoded product content', () => {
  const output = getExternalShopDescriptionDetail(
    '<p><strong>完整说明</strong><br>教程：https://example.com/help<br>联系邮箱：help@example.com</p><p>第二段&nbsp;内容</p>',
  );

  assert.equal(output.includes('完整说明'), true);
  assert.equal(output.includes('教程：https://example.com/help'), true);
  assert.equal(output.includes('联系邮箱：help@example.com'), true);
  assert.equal(output.includes('第二段 内容'), true);
});

test('buildExternalShopCategoryCards groups goods into source-style category summaries', () => {
  const output = buildExternalShopCategoryCards([
    { category_name: 'ChatGPT', goods_key: '1' },
    { category_name: 'ChatGPT', goods_key: '2' },
    { category_name: 'Gemini', goods_key: '3' },
  ]);

  assert.deepEqual(output, [
    { key: 'all', name: '全部商品', count: 3 },
    { key: 'ChatGPT', name: 'ChatGPT', count: 2 },
    { key: 'Gemini', name: 'Gemini', count: 1 },
  ]);
});

test('buildExternalShopCategoryCards preserves source category order and zero-count categories', () => {
  const output = buildExternalShopCategoryCards(
    [
      { category_id: 20746, category_name: 'ChatGPT', goods_key: '1' },
      { category_id: 20746, category_name: 'ChatGPT', goods_key: '2' },
      {
        category_id: 22645,
        category_name: 'Google/gemini/反重力',
        goods_key: '3',
      },
    ],
    [
      { id: 20746, name: 'ChatGPT', goods_count: 4 },
      { id: 22645, name: 'Google/gemini/反重力', goods_count: 1 },
      { id: 21722, name: 'AI IDE账号', goods_count: 0 },
    ],
  );

  assert.deepEqual(output, [
    { key: 'all', name: '全部商品', count: 3 },
    { key: '20746', name: 'ChatGPT', count: 2 },
    { key: '22645', name: 'Google/gemini/反重力', count: 1 },
    { key: '21722', name: 'AI IDE账号', count: 0 },
  ]);
});

test('filterExternalShopGoods applies category and keyword search together', () => {
  const output = filterExternalShopGoods(
    [
      {
        goods_key: 'chatgpt-team',
        category_name: 'ChatGPT',
        name: 'ChatGPT Team 自动拉车',
        description: '<p>兑换码商品</p>',
      },
      {
        goods_key: 'gemini-pro',
        category_name: 'Gemini',
        name: 'Gemini Pro 一年',
        description: '<p>学生方案</p>',
      },
    ],
    { categoryKey: 'ChatGPT', searchTerm: '拉车' },
  );

  assert.deepEqual(output.map((item) => item.goods_key), ['chatgpt-team']);
});

test('getExternalShopStockLabel matches source-style inventory wording', () => {
  assert.equal(getExternalShopStockLabel({ stock_count: 0 }), '暂时缺货');
  assert.equal(getExternalShopStockLabel({ stock_count: 2 }), '库存一般');
  assert.equal(getExternalShopStockLabel({ stock_count: 20 }), '库存充足');
});

test('order confirmation helpers match source-style fulfillment and purchase limit copy', () => {
  assert.equal(getExternalShopFulfillmentLabel({ send_order: 0 }), '自动发货');
  assert.equal(getExternalShopFulfillmentLabel({ send_order: 1 }), '人工处理');
  assert.equal(getExternalShopPurchaseLimitLabel({ stock_count: 0 }), '仅剩0件');
  assert.equal(getExternalShopPurchaseLimitLabel({ stock_count: 1 }), '仅剩1件');
  assert.equal(getExternalShopPurchaseLimitLabel({ stock_count: 3 }), '最多3件');
});

test('getMaskedExternalShopContact masks email and fallback strings', () => {
  assert.equal(
    getMaskedExternalShopContact('verify@example.com'),
    'v***@example.com',
  );
  assert.equal(getMaskedExternalShopContact('12345678901'), '123****8901');
  assert.equal(getMaskedExternalShopContact('abcdefg'), 'ab***fg');
});

test('getAutoRefreshOrderTradeNos only refreshes one eligible order per tick', () => {
  const orders = [
    { local_trade_no: 'done', status: 'delivered', last_sync_at: 1 },
    { local_trade_no: 'older', status: 'pending_payment', last_sync_at: 10 },
    { local_trade_no: 'newer', status: 'created', last_sync_at: 20 },
  ];

  assert.deepEqual(getAutoRefreshOrderTradeNos(orders), ['older']);
});

test('AUTO_REFRESH_INTERVAL_MS uses a rate-limit-safe interval', () => {
  assert.equal(AUTO_REFRESH_INTERVAL_MS >= 30000, true);
});

test('getExternalShopStatusLabel returns localized-facing labels', () => {
  assert.equal(getExternalShopStatusLabel('delivered'), '已交付');
  assert.equal(getExternalShopStatusLabel('pending_payment'), '待支付');
  assert.equal(getExternalShopStatusLabel('unknown_status'), 'unknown_status');
});

test('getExternalShopDeliverySummary hides sensitive delivery content in lists', () => {
  assert.equal(getExternalShopDeliverySummary(''), '-');
  assert.equal(
    getExternalShopDeliverySummary('{"cards":["secret"]}'),
    '••••••••',
  );
});

test('parseExternalShopDeliveryCards returns cards for masked display and reveal', () => {
  assert.deepEqual(
    parseExternalShopDeliveryCards('{"cards":["secret","secret-2"]}'),
    ['secret', 'secret-2'],
  );
  assert.deepEqual(parseExternalShopDeliveryCards(''), []);
});

test('shouldShowDescriptionToggle only enables toggle for longer descriptions', () => {
  assert.equal(shouldShowDescriptionToggle('短描述', 10), false);
  assert.equal(
    shouldShowDescriptionToggle(
      '这是一个明显超过阈值的较长商品描述，用来测试展开和收起逻辑。',
      10,
    ),
    true,
  );
});
