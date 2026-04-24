export function normalizeClientDeviceUserCode(value) {
  const compact = String(value || '')
    .trim()
    .toUpperCase()
    .replace(/\s+/g, '')
    .replace(/-/g, '');

  if (compact.length <= 4) {
    return compact;
  }

  return `${compact.slice(0, 4)}-${compact.slice(4, 8)}`;
}
