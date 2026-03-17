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
    source.includes('redeemResult.warranty_expires_at'),
    true,
  );
  assert.equal(
    source.includes('warrantyResult.warranty_expires_at'),
    true,
  );
});

test('GptTeamPlanTab uses distinct placeholders for redeem and warranty inputs', () => {
  assert.equal(source.includes("placeholder={t('请输入兑换邮箱')}"), true);
  assert.equal(source.includes("placeholder={t('请输入待兑换的兑换码')}"), true);
  assert.equal(source.includes("placeholder={t('请输入需要查询的兑换码')}"), true);
});

test('GptTeamPlanTab bypasses the global error interceptor when rendering inline failure states', () => {
  assert.equal(
    source.includes("'/api/gptteamplan/redeem'"),
    true,
  );
  assert.equal(
    source.includes("'/api/gptteamplan/warranty/check'"),
    true,
  );
  assert.equal(source.includes('skipErrorHandler: true'), true);
});

test('GptTeamPlanTab keeps remaining seat details in redeem results', () => {
  assert.equal(source.includes("{t('剩余车位')}"), true);
  assert.equal(source.includes('redeemResult.remaining_seats'), true);
  assert.equal(source.includes('redeemResult.warranty_expires_at || redeemResult.expires_at'), true);
});

test('GptTeamPlanTab renders the entry remaining seats from parent status data', () => {
  assert.equal(
    source.includes(
      'export default function GptTeamPlanTab({ remainingSeats: initialRemainingSeats = null })',
    ),
    true,
  );
  assert.equal(source.includes('const currentRemainingSeats ='), true);
  assert.equal(source.includes('remainingSeatsDisplay'), true);
});

test('GptTeamPlanTab follows source-site warranty fallback rendering', () => {
  assert.equal(source.includes('getGPTTeamWarrantyRecordTeamName(record.team_name)'), true);
  assert.equal(source.includes('getGPTTeamWarrantyRecordExpiry(record)'), true);
  assert.equal(source.includes('formatGPTTeamWarrantyExpiry('), true);
  assert.equal(
    source.includes('warrantyResult.records.length === 0'),
    true,
  );
});

test('GptTeamPlanTab shows warranty record code and email directly without reveal toggles', () => {
  assert.equal(source.includes('getMaskedGPTTeamCode(value)'), false);
  assert.equal(source.includes('getMaskedGPTTeamEmail(value)'), false);
  assert.equal(source.includes('revealedWarrantyCodes'), false);
  assert.equal(source.includes('revealedWarrantyEmails'), false);
});

test('GptTeamPlanTab renders warranty records as full detail blocks instead of a table row', () => {
  assert.equal(source.includes('import {\n  Button,\n  Card,\n  Empty,\n  Input,\n  Space,\n  Table,'), false);
  assert.equal(source.includes('warrantyResult.records.map((record, index) => ('), true);
  assert.equal(source.includes("label={t('兑换码')}"), true);
  assert.equal(source.includes("label={t('邮箱')}"), true);
  assert.equal(source.includes("label={t('兑换时间')}"), true);
  assert.equal(source.includes("label={t('到期时间')}"), true);
  assert.equal(source.includes("className='space-y-3'"), true);
  assert.equal(source.includes("className='grid grid-cols-1 gap-3 lg:grid-cols-2'"), false);
});
