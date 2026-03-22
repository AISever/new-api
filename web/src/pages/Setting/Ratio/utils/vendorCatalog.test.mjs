import test from 'node:test';
import assert from 'node:assert/strict';

import {
  ALL_VENDOR_FILTER,
  UNKNOWN_VENDOR_FILTER,
  buildVendorFilterOptions,
  createVendorCatalog,
  resolveModelVendor,
} from './vendorCatalog.js';

const createCatalog = () =>
  createVendorCatalog({
    vendors: [
      { id: 3, name: 'OpenAI' },
      { id: 4, name: 'Anthropic' },
    ],
    models: [
      { model_name: 'gpt-4o', vendor_id: 3 },
      { model_name: 'gpt-4.1', vendor_id: 3 },
      { model_name: 'claude-3-7-sonnet', vendor_id: 4 },
    ],
  });

test('resolveModelVendor resolves exact match', () => {
  const vendor = resolveModelVendor('gpt-4o', createCatalog());
  assert.equal(vendor?.name, 'OpenAI');
});

test('resolveModelVendor resolves wildcard when matches a single vendor', () => {
  const vendor = resolveModelVendor('gpt-*', createCatalog());
  assert.equal(vendor?.name, 'OpenAI');
});

test('resolveModelVendor returns null when wildcard spans multiple vendors', () => {
  const vendor = resolveModelVendor('*', createCatalog());
  assert.equal(vendor, null);
});

test('buildVendorFilterOptions includes vendor and unknown buckets', () => {
  const options = buildVendorFilterOptions(
    [
      { key: 'gpt-4o' },
      { key: 'claude-3-7-sonnet' },
      { key: 'custom-private-model' },
    ],
    createCatalog(),
    (value) => value,
  );

  assert.deepEqual(
    options.map((option) => option.value),
    [ALL_VENDOR_FILTER, 'Anthropic', 'OpenAI', UNKNOWN_VENDOR_FILTER],
  );
});
