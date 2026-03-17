import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const source = fs.readFileSync(path.join(__dirname, 'index.jsx'), 'utf8');

test('shop tabs avoid null tab panes when only the GPT Team entry is available', () => {
  assert.equal(source.includes('const shopTabs = []'), true);
  assert.equal(source.includes('shopTabs.map(({ itemKey, tab, content }) => ('), true);
  assert.equal(
    source.includes(
      "{shopStatus.externalShopReady || !shopStatus.gptTeamEnabled ? (",
    ),
    false,
  );
});
