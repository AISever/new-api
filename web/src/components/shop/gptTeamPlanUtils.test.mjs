import test from 'node:test';
import assert from 'node:assert/strict';
import {
  formatGPTTeamPlanDateTime,
  formatGPTTeamTeamStatus,
  getMaskedGPTTeamCode,
  getMaskedGPTTeamEmail,
  normalizeShopTabKey,
} from './gptTeamPlanUtils.js';

test('normalizeShopTabKey keeps gpt team tab only when enabled', () => {
  assert.equal(normalizeShopTabKey('gpt-team', true), 'gpt-team');
  assert.equal(normalizeShopTabKey('gpt-team', false), 'goods');
  assert.equal(normalizeShopTabKey('goods', true), 'goods');
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
  assert.equal(formatGPTTeamTeamStatus('unknown_status'), 'unknown_status');
});
