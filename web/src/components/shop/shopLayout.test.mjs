import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const readSource = (name) =>
  fs.readFileSync(path.join(__dirname, name), 'utf8');

test('shop index uses stable page shell and source-style goods section shell', () => {
  const source = readSource('index.jsx');

  assert.equal(source.includes("max-w-7xl mx-auto"), true);
  assert.equal(source.includes('useActualTheme'), true);
  assert.equal(source.includes("radial-gradient(circle at top left"), true);
  assert.equal(source.includes("rounded-[28px]"), true);
  assert.equal(source.includes("pt-0"), true);
  assert.equal(source.includes("md:pt-10"), true);
});

test('shop cards open the detail modal directly without a separate detail button', () => {
  const source = readSource('index.jsx');

  assert.equal(source.includes('selectedDetailGood'), true);
  assert.equal(
    source.includes("selectedDetailGood?.name || t('商品详情')"),
    true,
  );
  assert.equal(source.includes('onClick={() => setSelectedDetailGood(good)}'), true);
  assert.equal(source.includes('<Space wrap>'), false);
});

test('shop index exposes source-style category and search controls for goods', () => {
  const source = readSource('index.jsx');

  assert.equal(source.includes('activeGoodsCategory'), true);
  assert.equal(source.includes('goodsSearch'), true);
  assert.equal(source.includes("placeholder={t('搜索商品')}"), true);
  assert.equal(source.includes("{t('选择商品')}"), true);
});

test('shop goods use compact fixed-height image cards without extra descriptions', () => {
  const source = readSource('index.jsx');

  assert.equal(source.includes("grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 gap-5"), true);
  assert.equal(source.includes('good.image'), true);
  assert.equal(source.includes("aspect-[4/5]"), false);
  assert.equal(source.includes("h-48 w-full object-cover"), true);
  assert.equal(source.includes('normalizeExternalShopDescription'), false);
});

test('shop goods section includes explicit dark theme styling for the source-style panel', () => {
  const source = readSource('index.jsx');

  assert.equal(source.includes("const isDark = actualTheme === 'dark';"), true);
  assert.equal(source.includes("rgba(2, 6, 23, 0.72)"), true);
  assert.equal(source.includes("rgba(15, 23, 42, 0.98)"), true);
});

test('shop goods section removes redundant nested borders around the product grid', () => {
  const source = readSource('index.jsx');

  assert.equal(source.includes("className='border-t border-slate-100 pt-5'"), false);
  assert.equal(source.includes("className='mt-4 rounded-[22px] border border-slate-200/80 bg-white p-4 sm:p-5 dark:border-slate-800/80 dark:bg-slate-900/95'"), false);
  assert.equal(source.includes("border: isDark"), false);
});

test('shop goods typography stays aligned with the console module scale', () => {
  const source = readSource('index.jsx');

  assert.equal(source.includes("style={{ fontSize: 20, color: 'var(--semi-color-text-0)' }}"), false);
  assert.equal(source.includes("text-[17px]"), false);
  assert.equal(source.includes('fontSize: 28,'), false);
  assert.equal(source.includes('fontSize: 30,'), false);
  assert.equal(source.includes('fontSize: 16,'), false);
  assert.equal(source.includes('fontSize: 15,'), true);
  assert.equal(source.includes("color: 'var(--semi-color-text-0)'"), true);
  assert.equal(source.includes('line-clamp-2 text-base font-semibold leading-6'), false);
  assert.equal(source.includes('text-sm font-semibold leading-5'), true);
  assert.equal(source.includes('line-clamp-2 text-sm font-semibold leading-6'), true);
  assert.equal(source.includes('fontSize: 24,'), false);
  assert.equal(source.includes('fontSize: 20,'), true);
  assert.equal(source.includes('fontSize: 18,'), true);
  assert.equal(source.includes("mt-1 text-xs leading-5"), true);
  assert.equal(source.includes("uppercase tracking-[0.28em]"), false);
  assert.equal(source.includes("text-xs font-medium leading-5 tracking-[0.08em]"), true);
});

test('shop order confirmation follows the source storefront modal flow', () => {
  const source = readSource('index.jsx');
  const confirmationSection = source
    .split("title={t('订单确认')}")
    .at(1)
    ?.split("<Modal")[0];

  assert.equal(source.includes("'/api/external-shop/channels?refresh=1'"), true);
  assert.equal(source.includes("'/api/external-shop/channels'"), true);
  assert.equal(source.includes('loadPaymentChannels()'), true);
  assert.equal(source.includes("title={t('订单确认')}"), true);
  assert.equal(source.includes("{t('联系方式')}"), true);
  assert.equal(source.includes("{t('支付方式')}"), true);
  assert.equal(source.includes("{t('去支付')}"), true);
  assert.equal(source.includes('quantity: 1,'), false);
  assert.equal(source.includes('quantity,'), true);
  assert.equal(source.includes('width={720}'), false);
  assert.equal(source.includes('width={480}'), true);
  assert.equal(source.includes('window.open('), false);
  assert.equal(source.includes('showPayRedirectPrompt'), false);
  assert.equal(source.includes('navigate(nextPath);'), true);
  assert.equal(confirmationSection?.includes("{t('查看详情')}"), false);
});

test('shop confirmation modal keeps tighter spacing around contact, price, and footer actions', () => {
  const source = readSource('index.jsx');

  assert.equal(source.includes("className='space-y-5'"), false);
  assert.equal(source.includes("className='space-y-4'"), true);
  assert.equal(source.includes("className='space-y-2'"), false);
  assert.equal(source.includes("className='space-y-1.5'"), true);
  assert.equal(source.includes("className='flex justify-end gap-3 pt-3'"), true);
  assert.equal(source.includes('InputNumber'), true);
  assert.equal(source.includes("当前仅支持单件下单"), false);
  assert.equal(source.includes("当前商品不限制下单数量"), false);
  assert.equal(source.includes("支持多件下单，实际以下游库存为准"), false);
  assert.equal(source.includes("{`x${normalizedSelectedGoodQuantity}`}"), true);
  assert.equal(source.includes('max={Number(selectedGood?.stock_count || 1)}'), true);
  assert.equal(source.includes("<Text type='danger'>*</Text>"), true);
});

test('shop goods cards avoid full-height card body that pushes actions outside the card', () => {
  const source = readSource('index.jsx');

  assert.equal(source.includes("bodyStyle={{ height: '100%' }}"), false);
});

test('order pay and detail pages use the same centered page shell', () => {
  const paySource = readSource('OrderPay.jsx');
  const detailSource = readSource('OrderDetail.jsx');

  assert.equal(paySource.includes("max-w-5xl mx-auto"), true);
  assert.equal(detailSource.includes("max-w-5xl mx-auto"), true);
  assert.equal(paySource.includes("!rounded-2xl shadow-sm border-0"), true);
  assert.equal(detailSource.includes("!rounded-2xl shadow-sm border-0"), true);
});

test('shop orders expose a confirmed delete action instead of only refresh and detail', () => {
  const source = readSource('index.jsx');

  assert.equal(source.includes('Popconfirm'), true);
  assert.equal(
    source.includes("API.delete(`/api/external-shop/orders/${record.local_trade_no}`)"),
    true,
  );
  assert.equal(source.includes("{t('删除')}"), true);
  assert.equal(source.includes("{t('确定删除这个订单记录吗？')}"), true);
});

test('order pay refresh uses local error handling and avoids background-tab refresh churn', () => {
  const source = readSource('OrderPay.jsx');
  const shopSource = readSource('index.jsx');

  assert.equal(source.includes("document.visibilityState === 'visible'"), true);
  assert.equal(source.includes('skipErrorHandler: true'), true);
  assert.equal(
    shopSource.includes(
      "API.post(\n        `/api/external-shop/orders/${record.local_trade_no}/refresh`,\n        null,\n        { skipErrorHandler: true },",
    ),
    true,
  );
});
