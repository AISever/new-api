import { test, expect } from 'bun:test';
import { createRequire } from 'node:module';
const require = createRequire(import.meta.url);
const braces = require('braces');
test('braces ranges, alternatives and limits retain their behavior', () => {
  expect(braces.expand('a/{b,c}/{1..3}')).toEqual([
    'a/b/1',
    'a/b/2',
    'a/b/3',
    'a/c/1',
    'a/c/2',
    'a/c/3',
  ]);
  expect(braces.compile('a/{b,c}/d')).toBe('a/(b|c)/d');
  expect(braces.stringify('a/{b,c}/d')).toBe('a/{b,c}/d');
  expect(braces.expand('x/{a,{b,c}}')).toEqual(['x/a', 'x/b', 'x/c']);
  expect(braces.expand('{0001..0003}')).toEqual(['0001', '0002', '0003']);
  expect(() => braces.expand('{1..10001}')).toThrow(RangeError);
});
test('maximum parser nesting does not exhaust the stack', () => {
  const input = '{'.repeat(4998) + 'a,b' + '}'.repeat(4998);
  expect(braces.compile(input).length).toBe(input.length);
  expect(braces.stringify(input)).toBe(input);
  expect(braces.expand(input).length).toBe(2);
});
test('deep caller supplied AST does not exhaust the JS stack', () => {
  let ast = { type: 'text', value: 'safe' };
  for (let i = 0; i < 50000; i++) ast = { type: 'root', nodes: [ast] };
  expect(braces.compile(ast)).toBe('safe');
  expect(braces.stringify(ast)).toBe('safe');
});
