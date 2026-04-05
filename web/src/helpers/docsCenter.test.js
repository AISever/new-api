import test from 'node:test';
import assert from 'node:assert/strict';

import {
  getDocsAvailability,
  normalizeDocsManifestPath,
  shouldUseLocalDocs,
} from './docsCenter.js';

test('normalizeDocsManifestPath accepts enterprise static manifest paths', () => {
  assert.equal(
    normalizeDocsManifestPath('/enterprise-docs/apifox/manifest.json'),
    '/enterprise-docs/apifox/manifest.json',
  );
});

test('normalizeDocsManifestPath rejects remote or unsafe paths', () => {
  assert.equal(normalizeDocsManifestPath('https://evil.com/manifest.json'), '');
  assert.equal(normalizeDocsManifestPath('/docs/manifest.json'), '');
  assert.equal(normalizeDocsManifestPath('/enterprise-docs/apifox/index.json'), '');
  assert.equal(
    normalizeDocsManifestPath('/enterprise-docs/../secret/manifest.json'),
    '',
  );
  assert.equal(
    normalizeDocsManifestPath('/enterprise-docs//apifox/manifest.json'),
    '',
  );
  assert.equal(
    normalizeDocsManifestPath('/enterprise-docs\\apifox\\manifest.json'),
    '',
  );
});

test('getDocsAvailability prefers local docs when manifest exists', () => {
  assert.deepEqual(
    getDocsAvailability({
      docs_manifest_path: '/enterprise-docs/apifox/manifest.json',
      docs_link: 'https://legacy.example.com',
    }),
    {
      hasLocalDocs: true,
      hasLegacyDocsLink: true,
      manifestPath: '/enterprise-docs/apifox/manifest.json',
      docsLink: 'https://legacy.example.com',
    },
  );
});

test('getDocsAvailability keeps legacy docs link when local docs are absent', () => {
  assert.deepEqual(
    getDocsAvailability({
      docs_link: 'https://docs.example.com',
    }),
    {
      hasLocalDocs: false,
      hasLegacyDocsLink: true,
      manifestPath: '',
      docsLink: 'https://docs.example.com',
    },
  );
});

test('shouldUseLocalDocs only enables the in-app docs experience for valid manifests', () => {
  assert.equal(
    shouldUseLocalDocs({
      docs_manifest_path: '/enterprise-docs/apifox/manifest.json',
    }),
    true,
  );
  assert.equal(
    shouldUseLocalDocs({
      docs_manifest_path: 'https://evil.com/manifest.json',
    }),
    false,
  );
  assert.equal(shouldUseLocalDocs({}), false);
});
