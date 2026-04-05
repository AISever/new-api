import test from 'node:test';
import assert from 'node:assert/strict';

import { applyBrandingToDocument } from './branding.js';

function createDocument() {
  const iconLink = { href: '/logo.png' };
  return {
    title: 'New API',
    querySelector(selector) {
      if (selector === "link[rel~='icon']") {
        return iconLink;
      }
      return null;
    },
    iconLink,
  };
}

test('applyBrandingToDocument uses custom logo as favicon', () => {
  const document = createDocument();

  applyBrandingToDocument(document, {
    systemName: 'Aisever Enterprise',
    logo: 'https://corp-api.aisever.cn/logo-my.png',
  });

  assert.equal(document.title, 'Aisever Enterprise');
  assert.equal(
    document.iconLink.href,
    'https://corp-api.aisever.cn/logo-my.png',
  );
});

test('applyBrandingToDocument keeps default favicon when logo is empty', () => {
  const document = createDocument();

  applyBrandingToDocument(document, {
    systemName: '',
    logo: '',
  });

  assert.equal(document.title, 'New API');
  assert.equal(document.iconLink.href, '/logo.png');
});
