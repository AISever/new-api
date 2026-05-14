export const PUBLIC_SHELL_PATHS = new Set([
  '/',
  '/about',
  '/pricing',
  '/help',
]);

export function shouldUsePublicShell(pathname) {
  if (PUBLIC_SHELL_PATHS.has(pathname)) {
    return true;
  }

  return pathname === '/docs' || pathname.startsWith('/docs/');
}

export function shouldHidePublicFooter(pathname) {
  return pathname === '/pricing';
}
