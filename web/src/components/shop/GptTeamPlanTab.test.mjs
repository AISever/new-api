import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const source = fs.readFileSync(
  path.join(__dirname, 'GptTeamPlanTab.jsx'),
  'utf8',
);

test('GptTeamPlanTab uses user-readable datetime formatting for result fields', () => {
  assert.equal(source.includes('formatGPTTeamPlanDateTime'), true);
  assert.equal(source.includes('formatGPTTeamPlanDateTime(redeemResult.expires_at)'), true);
  assert.equal(
    source.includes('formatGPTTeamPlanDateTime(redeemResult.team_expires_at)'),
    true,
  );
  assert.equal(
    source.includes('formatGPTTeamPlanDateTime(\n                      redeemResult.warranty_expires_at,'),
    true,
  );
  assert.equal(
    source.includes('formatGPTTeamPlanDateTime(\n                      warrantyResult.warranty_expires_at,'),
    true,
  );
});

test('GptTeamPlanTab uses distinct placeholders for redeem and warranty inputs', () => {
  assert.equal(source.includes("placeholder={t('请输入兑换邮箱')}"), true);
  assert.equal(source.includes("placeholder={t('请输入待兑换的兑换码')}"), true);
  assert.equal(source.includes("placeholder={t('请输入需要查询的兑换码')}"), true);
});
