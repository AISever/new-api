export function shouldAutoStartLdxpPay({ isMobile, search }) {
  if (isMobile) {
    return false;
  }

  const params = new URLSearchParams(search || '');
  return params.get('autostart') === '1';
}
