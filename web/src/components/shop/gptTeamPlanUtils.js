import { getMaskedExternalShopContact } from './shopUtils.js';

export function normalizeShopTabKey(value, gptTeamEnabled = false) {
  if (value === 'gpt-team' && gptTeamEnabled) {
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
  const status = String(value || '').trim().toLowerCase();
  if (!status) {
    return '-';
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
