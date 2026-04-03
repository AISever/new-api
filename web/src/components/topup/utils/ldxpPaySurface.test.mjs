import test from 'node:test';
import assert from 'node:assert/strict';

import {
  resolveLdxpDesktopPaySurfaceMode,
  shouldUseLdxpExternalPayFlow,
  shouldUseLdxpQrImageSurface,
} from './ldxpPaySurface.js';

test('desktop browsers use qr image surface when qr image is available', () => {
  assert.equal(
    shouldUseLdxpQrImageSurface({
      isMobile: false,
      userAgent:
        'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.3 Safari/605.1.15',
      qrImageUrl: 'https://mobilecodec.alipay.com/show.htm?code=demo',
    }),
    true,
  );
  assert.equal(
    shouldUseLdxpQrImageSurface({
      isMobile: false,
      userAgent:
        'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36',
      qrImageUrl: 'https://mobilecodec.alipay.com/show.htm?code=demo',
    }),
    true,
  );
});

test('missing qr image never switches to safari qr surface', () => {
  assert.equal(
    shouldUseLdxpQrImageSurface({
      isMobile: false,
      userAgent:
        'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.3 Safari/605.1.15',
      qrImageUrl: '',
    }),
    false,
  );
});

test('desktop falls back to local qr rendering when qr code exists without qr image', () => {
  assert.equal(
    resolveLdxpDesktopPaySurfaceMode({
      isMobile: false,
      userAgent:
        'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36',
      qrImageUrl: '',
      qrCode: 'https://qr.alipay.com/demo-code',
      paymentUrl: 'https://pay.ldxp.cn/shopApi/Pay/payment?trade_no=demo',
    }),
    'qr-code',
  );
});

test('desktop blocks unsafe iframe fallback when no qr data is available', () => {
  assert.equal(
    resolveLdxpDesktopPaySurfaceMode({
      isMobile: false,
      userAgent:
        'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36',
      qrImageUrl: '',
      qrCode: '',
      paymentUrl: 'https://pay.ldxp.cn/shopApi/Pay/payment?trade_no=demo',
    }),
    'blocked',
  );
});

test('narrow desktop windows still keep desktop in-site payment flow', () => {
  assert.equal(
    shouldUseLdxpExternalPayFlow({
      isMobile: true,
      userAgent:
        'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36',
    }),
    false,
  );
  assert.equal(
    resolveLdxpDesktopPaySurfaceMode({
      isMobile: true,
      userAgent:
        'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36',
      qrImageUrl: 'https://mobilecodec.alipay.com/show.htm?code=demo',
      qrCode: '',
      paymentUrl: 'https://pay.ldxp.cn/shopApi/Pay/payment?trade_no=demo',
    }),
    'qr-image',
  );
});

test('real mobile devices still use external payment flow', () => {
  assert.equal(
    shouldUseLdxpExternalPayFlow({
      isMobile: true,
      userAgent:
        'Mozilla/5.0 (iPhone; CPU iPhone OS 18_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.3 Mobile/15E148 Safari/604.1',
    }),
    true,
  );
});
