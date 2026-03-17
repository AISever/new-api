import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const detailSource = fs.readFileSync(
  path.join(__dirname, 'OrderDetail.jsx'),
  'utf8',
);
const paySource = fs.readFileSync(
  path.join(__dirname, 'OrderPay.jsx'),
  'utf8',
);

test('order detail suppresses toast noise when the order is missing and shows the inline empty state instead', () => {
  assert.equal(detailSource.includes('const isMissingOrderMessage = (message) =>'), true);
  assert.equal(
    detailSource.includes('skipErrorHandler: true'),
    true,
  );
  assert.equal(detailSource.includes('if (!isMissingOrderMessage(error.message)) {'), true);
});

test('order pay suppresses toast noise when the order is missing and relies on the inline unavailable state', () => {
  assert.equal(paySource.includes('const isMissingOrderMessage = (message) =>'), true);
  assert.equal(
    paySource.includes('skipErrorHandler: true'),
    true,
  );
  assert.equal(paySource.includes('if (!isMissingOrderMessage(error.message)) {'), true);
});
