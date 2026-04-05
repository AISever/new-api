import fs from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const DEFAULT_INDEX_URL = 'https://ob6nfbpu76.apifox.cn/llms.txt';
const DEFAULT_TARGET_ORIGIN = 'https://corp-api.aisever.cn';
const DEFAULT_DOCS_BASE_PATH = '/docs';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const WEB_ROOT = path.resolve(__dirname, '..');
const DEFAULT_OUTPUT_DIR = path.join(
  WEB_ROOT,
  'public',
  'enterprise-docs',
  'apifox',
);

function extractSourceId(sourceUrl) {
  const pathname = new URL(sourceUrl).pathname;
  return pathname.split('/').pop().replace(/\.md$/i, '');
}

export function parseLlmsIndex(content) {
  const entries = [];
  let currentSection = 'Docs';

  for (const rawLine of content.split('\n')) {
    const line = rawLine.trim();
    if (!line) {
      continue;
    }

    const headingMatch = line.match(/^##\s+(.+)$/);
    if (headingMatch) {
      currentSection = headingMatch[1].trim();
      continue;
    }

    const itemMatch = line.match(
      /^-\s+(.*?)\[(.+?)\]\((https?:\/\/[^)\s]+\.md)\):?\s*(.*)$/,
    );
    if (!itemMatch) {
      continue;
    }

    const prefix = itemMatch[1].trim().replace(/\s*>\s*$/, '');
    const title = itemMatch[2].trim();
    const sourceUrl = itemMatch[3].trim();
    const description = itemMatch[4].trim();

    entries.push({
      section: currentSection,
      hierarchy: prefix ? prefix.split(/\s*>\s*/).filter(Boolean) : [],
      title,
      description,
      sourceUrl,
      sourceId: extractSourceId(sourceUrl),
    });
  }

  return entries;
}

function escapeRegExp(value) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

export function rewriteMarkdownContent(
  content,
  {
    targetOrigin = DEFAULT_TARGET_ORIGIN,
    docsBasePath = DEFAULT_DOCS_BASE_PATH,
    upstreamOrigin = 'https://api2.aigcbest.top',
    apifoxOrigin = 'https://ob6nfbpu76.apifox.cn',
  } = {},
) {
  let output = content;

  output = output.replace(
    new RegExp(`${escapeRegExp(apifoxOrigin)}/([A-Za-z0-9_-]+)\\.md`, 'g'),
    `${docsBasePath}/$1`,
  );
  output = output.replace(
    new RegExp(`${escapeRegExp(upstreamOrigin)}`, 'g'),
    targetOrigin,
  );

  return output;
}

function buildManifest(entries, docsBasePath) {
  return {
    version: 1,
    generatedAt: new Date().toISOString(),
    docsBasePath,
    items: entries.map((entry, index) => ({
      id: entry.sourceId,
      title: entry.title,
      description: entry.description,
      section: entry.section,
      hierarchy: entry.hierarchy,
      order: index,
      sourceUrl: entry.sourceUrl,
      docPath: `${docsBasePath}/${entry.sourceId}`,
      markdownPath: `/enterprise-docs/apifox/pages/${entry.sourceId}.md`,
    })),
  };
}

async function fetchText(url) {
  const response = await fetch(url, {
    headers: {
      'User-Agent': 'AISever Enterprise Docs Sync',
    },
  });
  if (!response.ok) {
    throw new Error(`Failed to fetch ${url}: ${response.status}`);
  }
  return response.text();
}

async function ensureDir(dirPath) {
  await fs.mkdir(dirPath, { recursive: true });
}

async function writeFile(filePath, content) {
  await ensureDir(path.dirname(filePath));
  await fs.writeFile(filePath, content, 'utf8');
}

export async function syncEnterpriseDocs({
  indexUrl = DEFAULT_INDEX_URL,
  outputDir = DEFAULT_OUTPUT_DIR,
  targetOrigin = DEFAULT_TARGET_ORIGIN,
  docsBasePath = DEFAULT_DOCS_BASE_PATH,
} = {}) {
  const pagesDir = path.join(outputDir, 'pages');
  await ensureDir(pagesDir);

  const llmsIndex = await fetchText(indexUrl);
  const entries = parseLlmsIndex(llmsIndex);

  for (const entry of entries) {
    const markdown = await fetchText(entry.sourceUrl);
    const transformedMarkdown = rewriteMarkdownContent(markdown, {
      targetOrigin,
      docsBasePath,
    });
    await writeFile(
      path.join(pagesDir, `${entry.sourceId}.md`),
      transformedMarkdown,
    );
  }

  const manifest = buildManifest(entries, docsBasePath);
  await writeFile(path.join(outputDir, 'manifest.json'), JSON.stringify(manifest, null, 2));
  await writeFile(path.join(outputDir, 'llms.txt'), llmsIndex);

  return {
    count: entries.length,
    manifestPath: path.join(outputDir, 'manifest.json'),
  };
}

async function main() {
  const result = await syncEnterpriseDocs();
  console.log(
    `Synced ${result.count} enterprise docs to ${result.manifestPath}`,
  );
}

if (import.meta.url === `file://${process.argv[1]}`) {
  main().catch((error) => {
    console.error(error);
    process.exitCode = 1;
  });
}
