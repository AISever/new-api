/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

export const ALL_VENDOR_FILTER = 'all';
export const UNKNOWN_VENDOR_FILTER = 'unknown';

const normalizeText = (value) => String(value ?? '').trim();

const escapeRegex = (value) => value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');

const globToRegex = (pattern) =>
  new RegExp(
    `^${escapeRegex(pattern).replace(/\\\*/g, '.*')}$`,
    'i',
  );

export const createEmptyVendorCatalog = () => ({
  vendorsById: {},
  exactVendorByModel: new Map(),
  knownModelNames: [],
  wildcardCache: new Map(),
});

export const createVendorCatalog = ({ models = [], vendors = [] } = {}) => {
  const vendorsById = {};
  const exactVendorByModel = new Map();
  const knownModelNames = [];

  vendors.forEach((vendor) => {
    if (vendor?.id === undefined || vendor?.id === null) {
      return;
    }

    vendorsById[String(vendor.id)] = {
      id: String(vendor.id),
      name: normalizeText(vendor.name),
      icon: normalizeText(vendor.icon),
    };
  });

  models.forEach((model) => {
    const modelName = normalizeText(model?.model_name);
    if (!modelName) {
      return;
    }

    knownModelNames.push(modelName);
    const vendor = vendorsById[String(model?.vendor_id)] || null;
    exactVendorByModel.set(modelName, vendor);
  });

  return {
    vendorsById,
    exactVendorByModel,
    knownModelNames,
    wildcardCache: new Map(),
  };
};

export const resolveModelVendor = (modelName, catalog = createEmptyVendorCatalog()) => {
  const normalizedModelName = normalizeText(modelName);
  if (!normalizedModelName) {
    return null;
  }

  if (catalog.exactVendorByModel.has(normalizedModelName)) {
    return catalog.exactVendorByModel.get(normalizedModelName);
  }

  if (!normalizedModelName.includes('*')) {
    return null;
  }

  if (catalog.wildcardCache.has(normalizedModelName)) {
    return catalog.wildcardCache.get(normalizedModelName);
  }

  const matcher = globToRegex(normalizedModelName);
  const matchedVendorIds = new Set();
  let matchedCount = 0;

  catalog.knownModelNames.forEach((knownModelName) => {
    if (!matcher.test(knownModelName)) {
      return;
    }

    matchedCount += 1;
    const vendor = catalog.exactVendorByModel.get(knownModelName);
    matchedVendorIds.add(vendor?.id || UNKNOWN_VENDOR_FILTER);
  });

  let resolvedVendor = null;
  if (matchedCount > 0 && matchedVendorIds.size === 1) {
    const [vendorId] = [...matchedVendorIds];
    resolvedVendor =
      vendorId === UNKNOWN_VENDOR_FILTER ? null : catalog.vendorsById[vendorId] || null;
  }

  catalog.wildcardCache.set(normalizedModelName, resolvedVendor);
  return resolvedVendor;
};

export const buildVendorFilterOptions = (
  rows,
  catalog = createEmptyVendorCatalog(),
  t = (value) => value,
) => {
  const vendorCounts = new Map();
  let unknownCount = 0;
  let nonEmptyCount = 0;

  rows.forEach((row) => {
    const modelName = normalizeText(row?.key);
    if (!modelName) {
      return;
    }

    nonEmptyCount += 1;
    const vendor = resolveModelVendor(modelName, catalog);
    if (!vendor?.name) {
      unknownCount += 1;
      return;
    }

    vendorCounts.set(vendor.name, {
      vendor,
      count: (vendorCounts.get(vendor.name)?.count || 0) + 1,
    });
  });

  const options = [
    {
      value: ALL_VENDOR_FILTER,
      label: `${t('全部供应商')} (${nonEmptyCount})`,
    },
  ];

  [...vendorCounts.entries()]
    .sort((left, right) => left[0].localeCompare(right[0]))
    .forEach(([, { vendor, count }]) => {
      options.push({
        value: vendor.name,
        label: `${vendor.name} (${count})`,
      });
    });

  if (unknownCount > 0) {
    options.push({
      value: UNKNOWN_VENDOR_FILTER,
      label: `${t('未识别')} (${unknownCount})`,
    });
  }

  return options;
};
