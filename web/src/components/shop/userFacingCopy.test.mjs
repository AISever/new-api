import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const readSource = (name) =>
  fs.readFileSync(path.join(__dirname, name), 'utf8');

test('OrderPay does not expose polling cadence or local/internal labels to users', () => {
  const source = readSource('OrderPay.jsx');

  assert.equal(
    source.includes(
      '本地状态轮询：每 3 秒检查一次，远端订单刷新：每 15 秒触发一次。',
    ),
    false,
  );
  assert.equal(source.includes("key: t('本地订单号')"), false);
  assert.equal(source.includes("key: t('本地支付页')"), false);
  assert.equal(source.includes("key: t('轮询状态')"), false);
});

test('OrderDetail does not expose upstream or channel metadata to users', () => {
  const source = readSource('OrderDetail.jsx');

  assert.equal(source.includes("key: t('上游订单号')"), false);
  assert.equal(source.includes("key: t('支付渠道')"), false);
  assert.equal(
    source.includes("{ key: t('联系方式'), value: order.contact || '-' }"),
    false,
  );
});

test('OrderPay masks contact instead of showing raw values', () => {
  const source = readSource('OrderPay.jsx');

  assert.equal(
    source.includes("{ key: t('联系方式'), value: order.contact || '-' }"),
    false,
  );
});

test('Shop index hides upstream-process copy from the user-facing intro', () => {
  const source = readSource('index.jsx');

  assert.equal(source.includes('支付在上游页面完成'), false);
  assert.equal(source.includes("title: t('本地订单号')"), false);
});
