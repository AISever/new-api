export function shouldUseLdxpExternalPayFlow({ isMobile, userAgent }) {
  if (!isMobile) {
    return false;
  }

  const normalizedUserAgent = String(userAgent || '');
  return /Android|webOS|iPhone|iPad|iPod|Mobile|Windows Phone/i.test(normalizedUserAgent);
}

export function resolveLdxpDesktopPaySurfaceMode({
  isMobile,
  userAgent,
  qrImageUrl,
  qrCode,
  paymentUrl,
}) {
  if (shouldUseLdxpExternalPayFlow({ isMobile, userAgent })) {
    return 'mobile-external';
  }
  if (qrImageUrl) {
    return 'qr-image';
  }
  if (qrCode) {
    return 'qr-code';
  }
  if (paymentUrl) {
    return 'blocked';
  }
  return 'unavailable';
}

export function shouldUseLdxpQrImageSurface({ isMobile, userAgent, qrImageUrl }) {
  return resolveLdxpDesktopPaySurfaceMode({
    isMobile,
    userAgent,
    qrImageUrl,
    qrCode: '',
    paymentUrl: '',
  }) === 'qr-image';
}
