/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { QRCodeSVG } from 'qrcode.react';
import { useLocation, useNavigate, useParams } from 'react-router-dom';
import {
  Button,
  Card,
  ImagePreview,
  Spin,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { API, showError, showSuccess } from '../../helpers';
import { useTranslation } from 'react-i18next';
import { useIsMobile } from '../../hooks/common/useIsMobile';
import { getLdxpPayLayout } from './utils/ldxpPayLayout';
import { shouldAutoStartLdxpPay } from './utils/ldxpPayAutostart';
import { launchLdxpExternalPay } from './utils/ldxpPayLaunch';
import {
  resolveLdxpDesktopPaySurfaceMode,
  shouldUseLdxpExternalPayFlow,
  shouldUseLdxpQrImageSurface,
} from './utils/ldxpPaySurface';
import { formatUserFacingLdxpPlanTitle } from './utils/ldxpDisplay';

const { Text, Title } = Typography;

const statusColorMap = {
  pending: 'blue',
  success: 'green',
  failed: 'red',
  expired: 'grey',
};

const statusLabelMap = {
  pending: '待支付',
  success: '支付成功',
  failed: '支付失败',
  expired: '已过期',
};

const iframeSandbox = 'allow-same-origin allow-scripts allow-forms allow-popups';

function isOrderPayable(status) {
  return status === 'pending';
}

function formatMoney(value) {
  return `$${Number(value || 0).toFixed(2)}`;
}

const LdxpOrderPay = ({ mode = 'topup' }) => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const location = useLocation();
  const { trade_no: tradeNo = '' } = useParams();
  const isMobile = useIsMobile();
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [order, setOrder] = useState(null);
  const [showIframe, setShowIframe] = useState(false);
  const [pollingActive, setPollingActive] = useState(false);
  const [iframeLoaded, setIframeLoaded] = useState(false);
  const [iframeStalled, setIframeStalled] = useState(false);
  const [iframeVersion, setIframeVersion] = useState(0);
  const [mobilePayLaunched, setMobilePayLaunched] = useState(false);
  const [qrPreviewVisible, setQrPreviewVisible] = useState(false);
  const redirectingRef = useRef(false);

  const isSubscription = mode === 'subscription';
  const endpointBase = isSubscription
    ? '/api/subscription/ldxp/orders'
    : '/api/user/ldxp/orders';
  const backPath = '/console/topup';
  const successPath = isSubscription ? '/console/topup' : '/console/topup?show_history=true';

  const loadOrder = useCallback(
    async (showLoading = false) => {
      if (!tradeNo) {
        return null;
      }
      if (showLoading) {
        setLoading(true);
      }
      try {
        const res = await API.get(`${endpointBase}/${encodeURIComponent(tradeNo)}`, {
          skipErrorHandler: true,
        });
        if (!res.data?.success) {
          throw new Error(res.data?.message || t('获取订单失败'));
        }
        const nextOrder = res.data.data || null;
        setOrder(nextOrder);
        return nextOrder;
      } catch (error) {
        setOrder(null);
        throw error;
      } finally {
        if (showLoading) {
          setLoading(false);
        }
      }
    },
    [endpointBase, t, tradeNo],
  );

  useEffect(() => {
    loadOrder(true).catch((error) => {
      showError(error.message || t('获取订单失败'));
    });
  }, [loadOrder, t]);

  const layout = useMemo(() => getLdxpPayLayout(isMobile), [isMobile]);
  const browserUserAgent = typeof navigator === 'undefined' ? '' : navigator.userAgent;
  const usesExternalPayFlow = useMemo(
    () => shouldUseLdxpExternalPayFlow({ isMobile, userAgent: browserUserAgent }),
    [browserUserAgent, isMobile],
  );
  const desktopPaySurfaceMode = useMemo(
    () => resolveLdxpDesktopPaySurfaceMode({
      isMobile,
      userAgent: browserUserAgent,
      qrImageUrl: order?.qr_img_url,
      qrCode: order?.qr_code,
      paymentUrl: order?.payment_url,
    }),
    [browserUserAgent, isMobile, order?.payment_url, order?.qr_code, order?.qr_img_url],
  );
  const showQrImageSurface = useMemo(
    () => (
      showIframe
      && shouldUseLdxpQrImageSurface({
        isMobile,
        userAgent: browserUserAgent,
        qrImageUrl: order?.qr_img_url,
      })
    ),
    [browserUserAgent, isMobile, order?.qr_img_url, showIframe],
  );
  const showQrCodeSurface = showIframe && desktopPaySurfaceMode === 'qr-code';
  const blockUnsafeIframeFallback = showIframe && desktopPaySurfaceMode === 'blocked';
  const shouldRenderEmbeddedIframe = (
    showIframe
    && !showQrImageSurface
    && !showQrCodeSurface
    && !blockUnsafeIframeFallback
    && !usesExternalPayFlow
  );

  useEffect(() => {
    if (!shouldRenderEmbeddedIframe || iframeLoaded) {
      setIframeStalled(false);
      return undefined;
    }
    const timer = setTimeout(() => {
      setIframeStalled(true);
    }, 12000);
    return () => clearTimeout(timer);
  }, [iframeLoaded, shouldRenderEmbeddedIframe]);

  useEffect(() => {
    if (!pollingActive || !showIframe) {
      return undefined;
    }
    const syncOrder = async () => {
      try {
        const nextOrder = await loadOrder(false);
        if (!nextOrder || redirectingRef.current) {
          return;
        }
        if (nextOrder.status === 'success') {
          redirectingRef.current = true;
          setPollingActive(false);
          showSuccess(
            t(isSubscription ? '订阅购买成功，正在返回钱包页面' : '充值成功，正在返回钱包页面'),
          );
          setTimeout(() => {
            navigate(successPath, { replace: true });
          }, 1000);
          return;
        }
        if (nextOrder.status === 'failed' || nextOrder.status === 'expired') {
          setPollingActive(false);
          showError(t('支付未完成'));
        }
      } catch {
        // ignore transient polling errors
      }
    };

    void syncOrder();
    const timer = setInterval(() => {
      void syncOrder();
    }, 3000);
    return () => clearInterval(timer);
  }, [isSubscription, loadOrder, navigate, pollingActive, showIframe, successPath, t]);

  const autoStartRequested = useMemo(
    () => shouldAutoStartLdxpPay({ isMobile, userAgent: browserUserAgent, search: location.search }),
    [browserUserAgent, isMobile, location.search],
  );

  useEffect(() => {
    if (!autoStartRequested || showIframe || !order?.payment_url || !isOrderPayable(order.status)) {
      return;
    }
    setIframeLoaded(false);
    setIframeStalled(false);
    setIframeVersion((current) => current + 1);
    setShowIframe(true);
    setMobilePayLaunched(false);
    setPollingActive(true);
  }, [autoStartRequested, order, showIframe]);

  useEffect(() => {
    if (!usesExternalPayFlow || !mobilePayLaunched) {
      return undefined;
    }
    const refreshAfterReturn = () => {
      if (document.visibilityState === 'visible') {
        void loadOrder(false);
      }
    };
    document.addEventListener('visibilitychange', refreshAfterReturn);
    window.addEventListener('focus', refreshAfterReturn);
    return () => {
      document.removeEventListener('visibilitychange', refreshAfterReturn);
      window.removeEventListener('focus', refreshAfterReturn);
    };
  }, [loadOrder, mobilePayLaunched, usesExternalPayFlow]);

  const payHint = useMemo(() => {
    if (usesExternalPayFlow) {
      return mobilePayLaunched
        ? t('官方支付页已拉起，支付完成后回到当前订单页，再点击“检查结果”。')
        : t('移动端会拉起官方支付页或支付宝应用，当前订单页会保留在 new-api 中。');
    }
    if (showQrImageSurface) {
      return t('桌面端已切换为站内二维码展示，请使用支付宝扫码完成支付。');
    }
    if (showQrCodeSurface) {
      return t('上游未返回二维码图片，当前已切换为站内生成二维码，请使用支付宝扫码完成支付。');
    }
    if (blockUnsafeIframeFallback) {
      return t('当前未能获取安全的站内二维码，系统已阻止加载高风险支付页。请点击“重新获取二维码”后重试。');
    }
    if (!showIframe) {
      return t('请先核对订单信息，确认无误后点击“开始支付”。');
    }
    if (iframeStalled) {
      return t('支付页面加载较慢，请点击“重新加载支付页”后重试。');
    }
    if (iframeLoaded) {
      return t('支付页面已加载，完成支付后请点击“我已完成支付，立即检查”。');
    }
    return t('正在加载支付页面，请稍候。');
  }, [
    iframeLoaded,
    iframeStalled,
    mobilePayLaunched,
    blockUnsafeIframeFallback,
    showIframe,
    showQrCodeSurface,
    showQrImageSurface,
    t,
    usesExternalPayFlow,
  ]);

  const handleRefresh = async () => {
    setRefreshing(true);
    try {
      const nextOrder = await loadOrder(false);
      if (nextOrder?.status === 'success' && !redirectingRef.current) {
        redirectingRef.current = true;
        showSuccess(
          t(isSubscription ? '订阅购买成功，正在返回钱包页面' : '充值成功，正在返回钱包页面'),
        );
        setTimeout(() => {
          navigate(successPath, { replace: true });
        }, 1000);
        return;
      }
      if (nextOrder?.status === 'failed' || nextOrder?.status === 'expired') {
        showError(t('支付未完成'));
        return;
      }
      showSuccess(t('订单状态已刷新'));
    } catch (error) {
      showError(error.message || t('刷新订单失败'));
    } finally {
      setRefreshing(false);
    }
  };

  if (loading) {
    return (
      <div className={`${layout.pageClassName} ${layout.shellClassName} flex justify-center`}>
        <Spin size='large' />
      </div>
    );
  }

  if (!order) {
    return (
      <div className={`${layout.pageClassName} ${layout.shellClassName}`}>
        <Card className={layout.sideCardClassName}>
          <Text>{t('订单不存在或已无法访问')}</Text>
        </Card>
      </div>
    );
  }

  const summaryItems = [
    { key: t('订单号'), value: order.trade_no || '-' },
    ...(isSubscription
      ? [{ key: t('套餐名称'), value: formatUserFacingLdxpPlanTitle(order.plan_title) || '-' }]
      : []),
    { key: t('支付金额'), value: formatMoney(order.money) },
    {
      key: t('当前状态'),
      value: (
        <Tag color={statusColorMap[order.status] || 'grey'}>
          {t(statusLabelMap[order.status] || order.status || '未知状态')}
        </Tag>
      ),
    },
  ];

  const compactButtonClassName = 'w-full justify-center';
  const startButtonLabel = usesExternalPayFlow ? t('打开官方支付页') : t('开始支付');
  const showStatusBanner = (
    iframeStalled
    || blockUnsafeIframeFallback
    || (usesExternalPayFlow && mobilePayLaunched)
  );
  const workspaceHint = usesExternalPayFlow
    ? mobilePayLaunched
      ? t('支付已在官方页面或支付宝应用中发起，完成后回到当前订单页检查结果。')
      : t('支付将在官方页面或支付宝应用中完成，当前订单页会保留在 new-api 中。')
    : t('点击左侧“开始支付”后，这里会展示站内支付宝二维码。');

  return (
    <div className={layout.pageClassName}>
      <div className={layout.shellClassName}>
        <div className={layout.contentGridClassName}>
          <Card className={layout.sideCardClassName}>
            <div className='space-y-4'>
              <div className='space-y-3'>
                <div className='flex items-start justify-between gap-3'>
                  <div className='min-w-0 space-y-1'>
                    <Title heading={isMobile ? 5 : 4} className='!mb-0'>
                      {t(isSubscription ? '订阅支付' : '充值支付')}
                    </Title>
                    <Text type='tertiary'>{t('请先核对订单信息，再进入支付流程。')}</Text>
                  </div>
                  <Tag color={statusColorMap[order.status] || 'grey'}>
                    {t(statusLabelMap[order.status] || order.status || '未知状态')}
                  </Tag>
                </div>
                {showStatusBanner && (
                  <div className='rounded-2xl border border-[var(--semi-color-border)] bg-[var(--semi-color-fill-0)] px-4 py-3'>
                    <Text type='secondary'>{payHint}</Text>
                  </div>
                )}
              </div>

              <div className={layout.summaryGridClassName}>
                {summaryItems.map((item) => (
                  <div key={item.key} className={layout.summaryRowClassName}>
                    <Text type='tertiary' size='small'>
                      {item.key}
                    </Text>
                    <div className={layout.summaryValueClassName}>
                      {item.value}
                    </div>
                  </div>
                ))}
              </div>

              {!showIframe && (
                <div className={layout.summaryActionsClassName}>
                  <Button
                    theme='solid'
                    type='primary'
                    className={compactButtonClassName}
                    onClick={() => {
                      if (!order.payment_url) {
                        showError(t('支付链接不存在'));
                        return;
                      }
                      if (usesExternalPayFlow) {
                        setMobilePayLaunched(true);
                        launchLdxpExternalPay({ paymentUrl: order.payment_url });
                        return;
                      }
                      setShowIframe(true);
                      setMobilePayLaunched(false);
                      setPollingActive(true);
                      setIframeLoaded(false);
                      setIframeStalled(false);
                      setIframeVersion((current) => current + 1);
                    }}
                    disabled={!isOrderPayable(order.status) || !order.payment_url}
                  >
                    {startButtonLabel}
                  </Button>
                  {usesExternalPayFlow && (
                    <Button
                      className={compactButtonClassName}
                      onClick={() => setMobilePayLaunched(false)}
                      disabled={!mobilePayLaunched}
                    >
                      {t('重置支付状态')}
                    </Button>
                  )}
                  <Button className={compactButtonClassName} loading={refreshing} onClick={handleRefresh}>
                    {t('检查结果')}
                  </Button>
                  <Button className={compactButtonClassName} onClick={() => navigate(backPath)}>
                    {t('返回钱包页面')}
                  </Button>
                </div>
              )}
            </div>
          </Card>

          <Card className={layout.mainCardClassName}>
            {!showIframe ? (
              <div className='space-y-4'>
                <div className='space-y-1'>
                  <Title heading={isMobile ? 6 : 5} className='!mb-0'>
                    {t('支付工作区')}
                  </Title>
                  <Text type='tertiary'>{workspaceHint}</Text>
                </div>

                <div className={layout.workspaceSplitClassName}>
                  <div
                    className={layout.placeholderClassName}
                    style={{ minHeight: `${layout.placeholderHeight}px` }}
                  >
                    <div className='max-w-xl space-y-3'>
                      <Title heading={isMobile ? 6 : 5} className='!mb-0'>
                        {t('支付页面')}
                      </Title>
                      <Text type='tertiary'>{workspaceHint}</Text>
                    </div>
                  </div>

                  <div className={layout.workspaceAsideClassName}>
                    <div className='space-y-3'>
                      <Title heading={6} className='!mb-0'>
                        {t('支付流程')}
                      </Title>
                      <div className='space-y-2 text-sm text-[var(--semi-color-text-1)]'>
                        <div>{t('1. 先核对左侧订单金额与状态。')}</div>
                        <div>
                          {t(
                            usesExternalPayFlow
                              ? '2. 点击“打开官方支付页”，系统可能新开页面或唤起支付宝。'
                              : '2. 点击“开始支付”，在当前页面展示二维码并使用支付宝扫码。',
                          )}
                        </div>
                        <div>{t('3. 支付完成后返回当前订单页，点击“检查结果”。')}</div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            ) : (
              <div className='space-y-3'>
                <div className='flex flex-col gap-2 md:flex-row md:items-center md:justify-between'>
                  <div className='min-w-0'>
                    <Title heading={isMobile ? 6 : 5} className='!mb-0'>
                      {t('支付页面')}
                    </Title>
                    <Text type='tertiary'>{payHint}</Text>
                  </div>
                  <Tag color={statusColorMap[order.status] || 'grey'}>
                    {t(statusLabelMap[order.status] || order.status || '未知状态')}
                  </Tag>
                </div>

                {showQrImageSurface || showQrCodeSurface ? (
                  <div className='rounded-2xl border bg-[var(--semi-color-bg-0)] px-6 py-8'>
                    <div className='mx-auto flex max-w-xl flex-col items-center gap-4 text-center'>
                      <div className='space-y-2'>
                        <Title heading={6} className='!mb-0'>
                          {t('支付宝扫码支付')}
                        </Title>
                        <Text type='secondary'>
                          {t(
                            showQrCodeSurface
                              ? '当前二维码由站内生成，请使用支付宝 App 扫描并在完成支付后返回当前订单页检查结果。'
                              : '请使用支付宝 App 扫描下方二维码，支付完成后返回当前订单页检查结果。',
                          )}
                        </Text>
                      </div>
                      <div className='rounded-3xl border border-slate-200 bg-white p-6 shadow-[0_18px_45px_rgba(15,23,42,0.18)]'>
                        {showQrImageSurface ? (
                          <img
                            src={order.qr_img_url}
                            alt={t('支付宝收款二维码')}
                            className='block h-auto max-w-full rounded-2xl bg-white p-3 cursor-zoom-in'
                            onClick={() => setQrPreviewVisible(true)}
                          />
                        ) : (
                          <div className='rounded-2xl bg-white p-3'>
                            <QRCodeSVG
                              value={order.qr_code}
                              size={260}
                              includeMargin
                              aria-label={t('支付宝收款二维码')}
                            />
                          </div>
                        )}
                      </div>
                      {showQrImageSurface && (
                        <Text type='tertiary' size='small'>
                          {t('二维码按原图尺寸展示，点击可在站内查看原图。')}
                        </Text>
                      )}
                    </div>
                  </div>
                ) : usesExternalPayFlow ? (
                  <div className='rounded-2xl border border-[var(--semi-color-border)] bg-[var(--semi-color-bg-0)] px-6 py-8'>
                    <div className='mx-auto flex max-w-xl flex-col items-center gap-4 text-center'>
                      <div className='space-y-2'>
                        <Title heading={6} className='!mb-0'>
                          {t('请在官方支付页完成支付')}
                        </Title>
                        <Text type='secondary'>
                          {mobilePayLaunched
                            ? t('官方支付页已拉起。支付完成后回到当前订单页，点击“我已完成支付，立即检查”。')
                            : t('当前设备按移动端支付流程处理。为了避免风控，系统不会在页面内嵌入支付页。')}
                        </Text>
                      </div>
                      {!mobilePayLaunched && (
                        <Button
                          theme='solid'
                          type='primary'
                          onClick={() => {
                            if (!order.payment_url) {
                              showError(t('支付链接不存在'));
                              return;
                            }
                            setMobilePayLaunched(true);
                            launchLdxpExternalPay({ paymentUrl: order.payment_url });
                          }}
                        >
                          {t('打开官方支付页')}
                        </Button>
                      )}
                    </div>
                  </div>
                ) : blockUnsafeIframeFallback ? (
                  <div className='rounded-2xl border border-[var(--semi-color-warning)] bg-[var(--semi-color-warning-light-default)] px-6 py-8'>
                    <div className='mx-auto flex max-w-xl flex-col items-center gap-4 text-center'>
                      <div className='space-y-2'>
                        <Title heading={6} className='!mb-0'>
                          {t('正在安全获取支付二维码')}
                        </Title>
                        <Text type='secondary'>
                          {t('当前未拿到可安全展示的二维码信息。为了避免触发支付风险，系统不会回退到嵌入式支付页。')}
                        </Text>
                      </div>
                      <Text type='secondary'>
                        {t('请点击下方“重新获取二维码”刷新订单信息；若持续失败，请稍后重试或联系管理员检查上游支付配置。')}
                      </Text>
                    </div>
                  </div>
                ) : (
                  <div className={layout.iframeWrapperClassName}>
                    <iframe
                      key={iframeVersion}
                      src={order.payment_url}
                      title={t('支付页面')}
                      onLoad={() => {
                        setIframeLoaded(true);
                        setIframeStalled(false);
                      }}
                      style={{
                        width: '100%',
                        height: `${layout.iframeHeight}px`,
                        border: '0',
                        background: 'var(--semi-color-bg-0)',
                      }}
                      sandbox={iframeSandbox}
                    />
                  </div>
                )}

                <div className={layout.iframeActionsClassName}>
                  <Button
                    theme='solid'
                    type='primary'
                    className={isMobile ? 'w-full' : undefined}
                    onClick={handleRefresh}
                    loading={refreshing}
                  >
                    {t('我已完成支付，立即检查')}
                  </Button>
                  <Button
                    className={isMobile ? 'w-full' : undefined}
                    onClick={() => {
                      if (showQrImageSurface || showQrCodeSurface || blockUnsafeIframeFallback) {
                        void loadOrder(false);
                        return;
                      }
                      setIframeLoaded(false);
                      setIframeStalled(false);
                      setIframeVersion((current) => current + 1);
                    }}
                  >
                    {showQrImageSurface || showQrCodeSurface || blockUnsafeIframeFallback
                      ? t('重新获取二维码')
                      : t('重新加载支付页')}
                  </Button>
                  <Button
                    className={isMobile ? 'w-full' : undefined}
                    onClick={() => {
                      setShowIframe(false);
                      setPollingActive(false);
                    }}
                  >
                    {t('返回订单信息')}
                  </Button>
                  <Button className={isMobile ? 'w-full' : undefined} onClick={() => navigate(backPath)}>
                    {t('返回钱包页面')}
                  </Button>
                </div>
              </div>
            )}
          </Card>
        </div>
      </div>
      <ImagePreview
        src={order?.qr_img_url}
        visible={qrPreviewVisible}
        onVisibleChange={(visible) => setQrPreviewVisible(visible)}
      />
    </div>
  );
};

export default LdxpOrderPay;
