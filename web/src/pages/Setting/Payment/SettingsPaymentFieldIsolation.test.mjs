import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const readSource = (name) =>
  fs.readFileSync(path.join(__dirname, name), 'utf8');

test('GPT Team payment settings use unique form field names to avoid DOM id collisions', () => {
  const source = readSource('SettingsGPTTeamPlan.jsx');

  assert.equal(source.includes("field='enabled'"), false);
  assert.equal(source.includes("field='base_url'"), false);
  assert.equal(source.includes("field='gptteamplan_enabled'"), true);
  assert.equal(source.includes("field='gptteamplan_base_url'"), true);
});
