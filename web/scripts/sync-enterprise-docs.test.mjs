import test from 'node:test';
import assert from 'node:assert/strict';

import {
  parseLlmsIndex,
  rewriteMarkdownContent,
} from './sync-enterprise-docs.mjs';

test('parseLlmsIndex extracts section, hierarchy and source ids', () => {
  const entries = parseLlmsIndex(`
# 钱多多-全链路API聚合

## API Docs
- 模型接口 > 聊天(Chat) > OpenAI [基础文本对话](https://ob6nfbpu76.apifox.cn/124951880e0.md): 本中转所有模型均已适配v1/chat/completions
`);

  assert.equal(entries.length, 1);
  assert.equal(entries[0].section, 'API Docs');
  assert.deepEqual(entries[0].hierarchy, [
    '模型接口',
    '聊天(Chat)',
    'OpenAI',
  ]);
  assert.equal(entries[0].title, '基础文本对话');
  assert.equal(entries[0].sourceId, '124951880e0');
});

test('rewriteMarkdownContent rewrites upstream origin and internal apifox docs links', () => {
  const output = rewriteMarkdownContent(
    [
      '# 发出请求',
      '',
      '访问 [控制台](https://api2.aigcbest.top/console/token)',
      '参考 [基础文本对话](https://ob6nfbpu76.apifox.cn/124951880e0.md)',
      '',
      'curl https://api2.aigcbest.top/v1/chat/completions',
    ].join('\n'),
    {
      targetOrigin: 'https://corp-api.aisever.cn',
      docsBasePath: '/docs',
    },
  );

  assert.match(output, /https:\/\/corp-api\.aisever\.cn\/console\/token/);
  assert.match(output, /\(\/docs\/124951880e0\)/);
  assert.match(output, /curl https:\/\/corp-api\.aisever\.cn\/v1\/chat\/completions/);
});
