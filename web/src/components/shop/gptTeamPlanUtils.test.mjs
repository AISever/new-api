import test from 'node:test';
import assert from 'node:assert/strict';
import {
  formatGPTTeamPlanDateTime,
  formatGPTTeamTeamStatus,
  formatGPTTeamWarrantyExpiry,
  getGPTTeamWarrantyRecordExpiry,
  getGPTTeamWarrantyRecordTeamName,
  getMaskedGPTTeamCode,
  getMaskedGPTTeamEmail,
  normalizeShopTabKey,
} from './gptTeamPlanUtils.js';

test('normalizeShopTabKey falls back to the available tab when shop or gpt team is unavailable', () => {
  assert.equal(normalizeShopTabKey('gpt-team', true, true), 'gpt-team');
  assert.equal(normalizeShopTabKey('gpt-team', true, false), 'goods');
  assert.equal(normalizeShopTabKey('goods', true, false), 'goods');
  assert.equal(normalizeShopTabKey('goods', false, true), 'gpt-team');
  assert.equal(normalizeShopTabKey(null, false, true), 'gpt-team');
  assert.equal(normalizeShopTabKey(null, false, false), 'goods');
});

test('getMaskedGPTTeamCode hides sensitive code fragments', () => {
  assert.equal(getMaskedGPTTeamCode('ABC-123-XYZ'), 'ABC••••XYZ');
  assert.equal(getMaskedGPTTeamCode('ABC123'), 'A••••');
});

test('getMaskedGPTTeamEmail reuses contact masking', () => {
  assert.equal(getMaskedGPTTeamEmail('user@example.com'), 'u***@example.com');
});

test('formatGPTTeamPlanDateTime formats ISO timestamps for users', () => {
  assert.equal(
    formatGPTTeamPlanDateTime('2026-04-14T09:37:33.857408'),
    '2026-04-14 09:37:33',
  );
  assert.equal(formatGPTTeamPlanDateTime(''), '-');
  assert.equal(formatGPTTeamPlanDateTime('not-a-date'), 'not-a-date');
});

test('formatGPTTeamTeamStatus maps upstream status to user-facing labels', () => {
  assert.equal(formatGPTTeamTeamStatus('full'), '已满员');
  assert.equal(formatGPTTeamTeamStatus('active'), '正常');
  assert.equal(formatGPTTeamTeamStatus('expired'), '已过期');
  assert.equal(formatGPTTeamTeamStatus(''), '未知');
  assert.equal(formatGPTTeamTeamStatus('unknown_status'), 'unknown_status');
});

test('formatGPTTeamWarrantyExpiry follows source-site pending activation rules', () => {
  assert.equal(formatGPTTeamWarrantyExpiry('', true, true), '待激活');
  assert.equal(formatGPTTeamWarrantyExpiry('', false, false), '-');
  assert.equal(
    formatGPTTeamWarrantyExpiry('2026-04-14T09:37:33.857408', true, true),
    '2026-04-14 09:37:33',
  );
});

test('GPT Team warranty helpers keep source-site fallback values', () => {
  assert.equal(getGPTTeamWarrantyRecordTeamName(''), '未知 Team');
  assert.equal(
    getGPTTeamWarrantyRecordExpiry({
      has_warranty: true,
      user_expires_at: '',
      warranty_expires_at: '',
      team_expires_at: '',
    }),
    '-',
  );
});
