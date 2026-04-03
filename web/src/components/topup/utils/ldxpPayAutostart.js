import { shouldUseLdxpExternalPayFlow } from './ldxpPaySurface.js';

export function shouldAutoStartLdxpPay({ isMobile, userAgent, search }) {
  if (shouldUseLdxpExternalPayFlow({ isMobile, userAgent })) {
    return false;
  }

  const params = new URLSearchParams(search || '');
  return params.get('autostart') === '1';
}
