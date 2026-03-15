import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

test('OrderPay imports useCallback when the component uses it', () => {
  const source = fs.readFileSync(path.join(__dirname, 'OrderPay.jsx'), 'utf8');
  const usesCallback = source.includes('useCallback(');
  const reactImportBlock =
    source.match(/import React,[\s\S]*?from 'react';/)?.[0] || '';

  assert.equal(usesCallback, true);
  assert.equal(reactImportBlock.includes('useCallback'), true);
});

test('OrderPay imports useRef when the component uses ref-backed polling state', () => {
  const source = fs.readFileSync(path.join(__dirname, 'OrderPay.jsx'), 'utf8');
  const reactImportBlock =
    source.match(/import React,[\s\S]*?from 'react';/)?.[0] || '';

  assert.equal(source.includes('useRef('), true);
  assert.equal(reactImportBlock.includes('useRef'), true);
});

test('OrderPay keeps polling off the whole order object dependency to avoid request storms', () => {
  const source = fs.readFileSync(path.join(__dirname, 'OrderPay.jsx'), 'utf8');

  assert.equal(source.includes('orderRef.current'), true);
  assert.equal(
    /\[\s*localTradeNo,\s*navigate,\s*order\?\.status,\s*pollingActive,\s*recordPayDebugEvent,\s*showIframe,\s*t,\s*\]/.test(
      source,
    ),
    true,
  );
});

test('OrderPay declares debug recorder before confirmation-mode helper', () => {
  const source = fs.readFileSync(path.join(__dirname, 'OrderPay.jsx'), 'utf8');

  assert.equal(
    source.indexOf('const recordPayDebugEvent') <
      source.indexOf('const enterConfirmationMode'),
    true,
  );
});

test('OrderPay auto-confirmation timer does not depend on the whole order object', () => {
  const source = fs.readFileSync(path.join(__dirname, 'OrderPay.jsx'), 'utf8');

  assert.equal(source.includes('post-load-delay'), false);
});
