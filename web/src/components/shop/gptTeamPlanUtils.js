import { getMaskedExternalShopContact } from './shopUtils.js';

export function normalizeShopTabKey(
  value,
  externalShopReady = false,
  gptTeamEnabled = false,
) {
  if (value === 'gpt-team' && gptTeamEnabled) {
    return 'gpt-team';
  }
  if (value === 'goods' && externalShopReady) {
    return 'goods';
  }
  if (externalShopReady) {
    return 'goods';
  }
  if (gptTeamEnabled) {
    return 'gpt-team';
  }
  return 'goods';
}

export function getMaskedGPTTeamCode(value) {
  const raw = String(value || '').trim();
  if (!raw) {
    return '-';
  }
  if (raw.length <= 6) {
    return `${raw.slice(0, 1)}••••`;
  }
  return `${raw.slice(0, 3)}••••${raw.slice(-3)}`;
}

export function getMaskedGPTTeamEmail(value) {
  return getMaskedExternalShopContact(value);
}

export function formatGPTTeamPlanDateTime(value) {
  const raw = String(value || '').trim();
  if (!raw) {
    return '-';
  }
  const timestamp = raw.includes('T') ? raw : raw.replace(' ', 'T');
  const parsed = new Date(timestamp);
  if (Number.isNaN(parsed.getTime())) {
    return raw;
  }
  const pad = (part) => String(part).padStart(2, '0');
  return `${parsed.getFullYear()}-${pad(parsed.getMonth() + 1)}-${pad(parsed.getDate())} ${pad(parsed.getHours())}:${pad(parsed.getMinutes())}:${pad(parsed.getSeconds())}`;
}

export function formatGPTTeamTeamStatus(value) {
  const status = String(value || '')
    .trim()
    .toLowerCase();
  if (!status) {
    return '未知';
  }
  const labels = {
    active: '正常',
    full: '已满员',
    expired: '已过期',
    inactive: '未激活',
    banned: '已封禁',
  };
  return labels[status] || value;
}

export function formatGPTTeamWarrantyExpiry(value, hasWarranty, warrantyValid) {
  const raw = String(value || '').trim();
  if (!hasWarranty) {
    return '-';
  }
  if (!raw) {
    return warrantyValid ? '待激活' : '-';
  }
  return formatGPTTeamPlanDateTime(raw);
}

export function getGPTTeamWarrantyRecordTeamName(value) {
  const raw = String(value || '').trim();
  return raw || '未知 Team';
}

export function getGPTTeamWarrantyRecordExpiry(record) {
  if (!record) {
    return '-';
  }
  const raw = record.has_warranty
    ? String(record.user_expires_at || record.warranty_expires_at || '').trim()
    : String(record.team_expires_at || '').trim();
  return formatGPTTeamPlanDateTime(raw);
}
