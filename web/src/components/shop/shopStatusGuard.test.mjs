import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const source = fs.readFileSync(path.join(__dirname, 'index.jsx'), 'utf8');

test('shop page loads a combined shop status endpoint before requesting storefront data', () => {
  const initialLoadSection = source
    .split('if (nextShopStatus.externalShopReady) {')
    .at(1)
    ?.split('} else {')
    .at(0);

  assert.equal(source.includes("API.get('/api/external-shop/status'"), true);
  assert.equal(source.includes('loadGPTTeamStatus'), false);
  assert.equal(source.includes('gptTeamRemainingSeats'), true);
  assert.equal(
    source.includes("res.data.data?.gptteamplan?.remaining_seats"),
    true,
  );
  assert.equal(
    source.includes('if (nextShopStatus.externalShopReady) {'),
    true,
  );
  assert.equal(initialLoadSection?.includes('loadOrders('), false);
  assert.equal(source.includes('const fallbackStatus = {'), false);
  assert.equal(source.includes('showError(error);'), true);
});

test('shop page passes entry seat counts into the GPT Team tab', () => {
  assert.equal(
    source.includes(
      '<GptTeamPlanTab remainingSeats={shopStatus.gptTeamRemainingSeats} />',
    ),
    true,
  );
});

test('shop page refreshes GPT Team seats in the background after initial render', () => {
  assert.equal(source.includes("API.get('/api/gptteamplan/status?refresh=1'"), true);
  assert.equal(source.includes('setShopStatus((current) => ({'), true);
  assert.equal(source.includes('gptTeamRemainingSeats:'), true);
});

test('shop page refreshes goods catalog in the background after cached first paint', () => {
  assert.equal(source.includes("'/api/external-shop/goods?refresh=1'"), true);
  assert.equal(source.includes("'/api/external-shop/categories?refresh=1'"), true);
  assert.equal(source.includes('loadGoods({ refresh: true, silent: true })'), true);
  assert.equal(
    source.includes('loadGoodsCategories({ refresh: true, silent: true })'),
    true,
  );
  assert.equal(source.includes('setGoods(res.data.data || [])'), true);
  assert.equal(source.includes('setGoodsCategories(res.data.data || [])'), true);
});

test('shop page refreshes payment channels in the background after cached first paint', () => {
  assert.equal(source.includes("'/api/external-shop/channels?refresh=1'"), true);
  assert.equal(
    source.includes('loadPaymentChannels({ refresh: true, silent: true })'),
    true,
  );
  assert.equal(source.includes('setPaymentChannels(res.data.data || [])'), true);
});

test('shop page refreshes orders in the background after cached first paint', () => {
  const orderRefreshSection = source
    .split('const refreshOrders = async () => {')
    .at(1)
    ?.split('};')
    .at(0);

  assert.equal(source.includes("params.set('refresh', '1')"), true);
  assert.equal(orderRefreshSection?.includes('await loadOrders({'), true);
  assert.equal(orderRefreshSection?.includes('page: orderPage'), true);
  assert.equal(orderRefreshSection?.includes('pageSize: orderPageSize'), true);
  assert.equal(orderRefreshSection?.includes('sort: orderSort'), true);
  assert.equal(orderRefreshSection?.includes('refresh: true'), true);
  assert.equal(orderRefreshSection?.includes('silent: true'), true);
});

test('shop page renders a non-error empty state when the external shop is unavailable', () => {
  assert.equal(source.includes("{t('商城暂未开放')}"), true);
  assert.equal(
    source.includes("{t('当前站点尚未完成商城配置，请联系管理员。')}"),
    true,
  );
});
