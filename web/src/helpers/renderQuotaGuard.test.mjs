import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const source = fs.readFileSync(path.join(__dirname, 'render.jsx'), 'utf8');

test('render quota helpers guard against missing quota_per_unit and invalid quota values', () => {
  assert.equal(
    source.includes(
      "return Number.isFinite(quotaPerUnit) && quotaPerUnit > 0 ? quotaPerUnit : 1;",
    ),
    true,
  );
  assert.equal(
    source.includes(
      'const safeQuota = Number.isFinite(quotaValue) ? quotaValue : 0;',
    ),
    true,
  );
});
