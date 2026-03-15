import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const readSource = (name) =>
  fs.readFileSync(path.join(__dirname, name), 'utf8');

test('PageLayout reserves console space below the fixed header on desktop and mobile', () => {
  const source = readSource('PageLayout.jsx');

  assert.equal(
    source.includes("paddingTop: isConsoleRoute ? '64px' : undefined"),
    true,
  );
  assert.equal(
    source.includes("paddingTop: shouldInnerPadding && isMobile ? '64px' : undefined"),
    false,
  );
  assert.equal(
    source.includes("padding: shouldInnerPadding ? (isMobile ? '5px' : '24px') : '0'"),
    true,
  );
});
