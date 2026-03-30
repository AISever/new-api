import test from 'node:test';
import assert from 'node:assert/strict';

import {
  formatUserFacingLdxpPlanTitle,
  formatUserFacingLdxpProductLabel,
} from './ldxpDisplay.js';

test('formatUserFacingLdxpPlanTitle removes leading LDXP marker and normalizes monthly plans', () => {
  assert.equal(
    formatUserFacingLdxpPlanTitle('LDXP 包月 Codex Lite'),
    'Codex Lite 包月套餐',
  );
});

test('formatUserFacingLdxpPlanTitle removes leading LDXP marker and normalizes yearly plans', () => {
  assert.equal(
    formatUserFacingLdxpPlanTitle('LDXP 包年 Codex Pro'),
    'Codex Pro 包年套餐',
  );
});

test('formatUserFacingLdxpPlanTitle falls back to trimmed title for unknown formats', () => {
  assert.equal(
    formatUserFacingLdxpPlanTitle('LDXP 高级版'),
    '高级版',
  );
});

test('formatUserFacingLdxpProductLabel removes leading LDXP marker', () => {
  assert.equal(
    formatUserFacingLdxpProductLabel('LDXP 24h 卡'),
    '24h 卡',
  );
});
