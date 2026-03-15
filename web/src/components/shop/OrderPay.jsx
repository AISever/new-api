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

import React, {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import {
  Banner,
  Button,
  Card,
  Descriptions,
  Space,
  Spin,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { API, showError, showSuccess } from '../../helpers';
import { useTranslation } from 'react-i18next';
import {
  buildExternalShopOrderDetailPath,
  buildExternalShopOrderPayPath,
} from './orderPaths';
import {
  getExternalShopStatusLabel,
  getMaskedExternalShopContact,
} from './shopUtils';
import {
  DETAIL_POLL_INTERVAL_MS,
  IFRAME_LOAD_TIMEOUT_MS,
  isExternalShopOrderPaymentActive,
  isPayPagePollingEnabled,
  PAYMENT_IFRAME_SANDBOX,
  RATE_LIMIT_BACKOFF_MS,
  shouldBackoffPolling,
  shouldDisableRefreshAction,
  shouldEnterEmbeddedConfirmationMode,
  shouldHideIframeAfterLoad,
  shouldNavigateFromPayPage,
  shouldShowEmbeddedReload,
  shouldTriggerRemoteRefresh,
} from './paymentPolling';

const { Title, Text, Paragraph } = Typography;

const statusColorMap = {
  created: 'grey',
  pending_payment: 'blue',
  paid_waiting_delivery: 'orange',
  delivered: 'green',
  failed: 'red',
  expired: 'grey',
  manual_review: 'pink',
};

export default function OrderPay() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { local_trade_no: localTradeNo = '' } = useParams();
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [order, setOrder] = useState(null);
  const [showIframe, setShowIframe] = useState(false);
  const [pollingActive, setPollingActive] = useState(false);
  const [iframeLoaded, setIframeLoaded] = useState(false);
  const [iframeStalled, setIframeStalled] = useState(false);
  const [pollingBlockedUntil, setPollingBlockedUntil] = useState(0);
  const [iframeLoadCount, setIframeLoadCount] = useState(0);
  const [iframeHiddenForConfirmation, setIframeHiddenForConfirmation] =
    useState(false);
  const [iframeVersion, setIframeVersion] = useState(0);
  const [showFullContact, setShowFullContact] = useState(false);
  const orderRef = useRef(null);
  const pollingBlockedUntilRef = useRef(0);
  const lastRemoteRefreshAtRef = useRef(0);

  const recordPayDebugEvent = useCallback(
    (event, extra = {}) => {
      if (typeof window === 'undefined') {
        return;
      }
      const previous = window.__externalShopPayDebug || {};
      const nextEvent = {
        event,
        at: Date.now(),
        localTradeNo,
        orderStatus: orderRef.current?.status || null,
        iframeLoadCount: extra.iframeLoadCount ?? iframeLoadCount,
        iframeHiddenForConfirmation:
          extra.iframeHiddenForConfirmation ?? iframeHiddenForConfirmation,
        showIframe: extra.showIframe ?? showIframe,
      };
      window.__externalShopPayDebug = {
        ...previous,
        localTradeNo,
        orderStatus: nextEvent.orderStatus,
        iframeLoadCount: nextEvent.iframeLoadCount,
        iframeHiddenForConfirmation: nextEvent.iframeHiddenForConfirmation,
        showIframe: nextEvent.showIframe,
        lastEvent: nextEvent,
        events: [...(previous.events || []), nextEvent].slice(-20),
      };
    },
    [iframeHiddenForConfirmation, iframeLoadCount, localTradeNo, showIframe],
  );

  const enterConfirmationMode = useCallback(
    (reason) => {
      if (
        !shouldEnterEmbeddedConfirmationMode(
          showIframe,
          iframeLoaded,
          orderRef.current,
        )
      ) {
        return;
      }
      setIframeHiddenForConfirmation(true);
      recordPayDebugEvent('iframe-hidden-for-confirmation', {
        iframeLoadCount,
        iframeHiddenForConfirmation: true,
        showIframe: true,
        reason,
      });
    },
    [iframeLoadCount, iframeLoaded, recordPayDebugEvent, showIframe],
  );

  const loadOrder = useCallback(
    async (showLoading = false) => {
      if (!localTradeNo) {
        return;
      }
      if (showLoading) {
        setLoading(true);
      }
      try {
        const res = await API.get(
          `/api/external-shop/orders/${encodeURIComponent(localTradeNo)}`,
        );
        if (!res.data.success) {
          throw new Error(res.data.message || t('获取订单失败'));
        }
        setOrder(res.data.data || null);
      } catch (error) {
        showError(error.message || t('获取订单失败'));
      } finally {
        if (showLoading) {
          setLoading(false);
        }
      }
    },
    [localTradeNo, t],
  );

  useEffect(() => {
    loadOrder(true);
  }, [loadOrder]);

  useEffect(() => {
    orderRef.current = order;
  }, [order]);

  useEffect(() => {
    setShowFullContact(false);
  }, [localTradeNo]);

  useEffect(() => {
    if (order?.status !== 'expired' || !showIframe) {
      return;
    }
    setShowIframe(false);
    setPollingActive(false);
    setIframeHiddenForConfirmation(false);
    recordPayDebugEvent('order-expired');
    showError(t('支付超时，订单已过期，请重新创建订单。'));
  }, [order?.status, recordPayDebugEvent, showIframe, t]);

  useEffect(() => {
    recordPayDebugEvent('state-sync');
  }, [
    iframeHiddenForConfirmation,
    iframeLoadCount,
    order?.status,
    recordPayDebugEvent,
    showIframe,
  ]);

  useEffect(() => {
    pollingBlockedUntilRef.current = pollingBlockedUntil;
  }, [pollingBlockedUntil]);

  const contactValue = useMemo(() => {
    const contact = String(order?.contact || '').trim();
    if (!contact) {
      return '-';
    }
    return (
      <Space spacing={6}>
        <Text>
          {showFullContact ? contact : getMaskedExternalShopContact(contact)}
        </Text>
        <Button
          size='small'
          theme='borderless'
          type='tertiary'
          onClick={() => setShowFullContact((current) => !current)}
        >
          {showFullContact ? t('隐藏') : t('完整显示')}
        </Button>
      </Space>
    );
  }, [order?.contact, showFullContact, t]);

  useEffect(() => {
    if (!pollingActive) {
      return undefined;
    }
    if (!isPayPagePollingEnabled(showIframe, orderRef.current)) {
      setPollingActive(false);
      return undefined;
    }
    const syncOrder = async (forceRemoteRefresh = false) => {
      const now = Date.now();
      if (shouldBackoffPolling(now, pollingBlockedUntilRef.current)) {
        return;
      }
      try {
        const orderRes = await API.get(
          `/api/external-shop/orders/${encodeURIComponent(localTradeNo)}`,
          { skipErrorHandler: true },
        );
        if (!orderRes.data.success) {
          return;
        }
        let updatedOrder = orderRes.data.data;
        setOrder(updatedOrder);
        if (shouldNavigateFromPayPage(updatedOrder)) {
          setPollingActive(false);
          recordPayDebugEvent('navigate-order-detail');
          showSuccess(t('支付已确认，正在跳转到订单详情...'));
          setTimeout(() => {
            navigate(buildExternalShopOrderDetailPath(localTradeNo));
          }, 1200);
          return;
        }

        if (
          !forceRemoteRefresh &&
          !shouldTriggerRemoteRefresh(now, lastRemoteRefreshAtRef.current)
        ) {
          return;
        }

        const refreshRes = await API.post(
          `/api/external-shop/orders/${encodeURIComponent(localTradeNo)}/refresh`,
          null,
          { skipErrorHandler: true },
        );
        if (refreshRes.data.success) {
          lastRemoteRefreshAtRef.current = now;
          updatedOrder = refreshRes.data.data;
          setOrder(updatedOrder);
          if (shouldNavigateFromPayPage(updatedOrder)) {
            setPollingActive(false);
            recordPayDebugEvent('navigate-order-detail');
            showSuccess(t('支付已确认，正在跳转到订单详情...'));
            setTimeout(() => {
              navigate(buildExternalShopOrderDetailPath(localTradeNo));
            }, 1200);
          }
        }
      } catch (error) {
        if (error?.response?.status === 429) {
          setPollingBlockedUntil(now + RATE_LIMIT_BACKOFF_MS);
          recordPayDebugEvent('polling-rate-limited');
        }
      }
    };

    void syncOrder(false);
    const timer = setInterval(() => {
      void syncOrder(false);
    }, DETAIL_POLL_INTERVAL_MS);
    return () => clearInterval(timer);
  }, [
    localTradeNo,
    navigate,
    order?.status,
    pollingActive,
    recordPayDebugEvent,
    showIframe,
    t,
  ]);

  useEffect(() => {
    if (!showIframe || iframeLoaded) {
      setIframeStalled(false);
      return undefined;
    }
    const timer = setTimeout(() => {
      setIframeStalled(true);
    }, IFRAME_LOAD_TIMEOUT_MS);
    return () => clearTimeout(timer);
  }, [iframeLoaded, showIframe]);

  useEffect(() => {
    if (!showIframe) {
      return undefined;
    }
    const handleFocus = () => {
      enterConfirmationMode('window-focus');
    };
    const handleVisibilityChange = () => {
      if (document.visibilityState === 'visible') {
        enterConfirmationMode('visibility-visible');
      }
    };
    window.addEventListener('focus', handleFocus);
    document.addEventListener('visibilitychange', handleVisibilityChange);
    return () => {
      window.removeEventListener('focus', handleFocus);
      document.removeEventListener('visibilitychange', handleVisibilityChange);
    };
  }, [enterConfirmationMode, showIframe]);

  useEffect(() => {
    if (!pollingBlockedUntil) {
      return undefined;
    }
    const remainingMs = pollingBlockedUntil - Date.now();
    if (remainingMs <= 0) {
      setPollingBlockedUntil(0);
      return undefined;
    }
    const timer = setTimeout(() => {
      setPollingBlockedUntil(0);
    }, remainingMs);
    return () => clearTimeout(timer);
  }, [pollingBlockedUntil]);

  const handleRefresh = async () => {
    enterConfirmationMode('manual-refresh');
    setRefreshing(true);
    try {
      const res = await API.post(
        `/api/external-shop/orders/${encodeURIComponent(localTradeNo)}/refresh`,
      );
      if (!res.data.success) {
        throw new Error(res.data.message || t('刷新订单失败'));
      }
      lastRemoteRefreshAtRef.current = Date.now();
      setOrder(res.data.data || null);
      recordPayDebugEvent('manual-refresh-success');
      showSuccess(t('订单已刷新'));
    } catch (error) {
      if (error?.response?.status === 429) {
        setPollingBlockedUntil(Date.now() + RATE_LIMIT_BACKOFF_MS);
        recordPayDebugEvent('manual-refresh-rate-limited');
        showError(t('检查过于频繁，请等待 30 秒后再试。'));
        return;
      }
      showError(error.message || t('刷新订单失败'));
    } finally {
      setRefreshing(false);
    }
  };

  const handleContinuePay = () => {
    if (!order?.pay_url) {
      showError(t('支付链接不存在'));
      return;
    }
    setIframeLoaded(false);
    setIframeStalled(false);
    setShowIframe(true);
    setPollingActive(true);
    lastRemoteRefreshAtRef.current = Date.now();
    setPollingBlockedUntil(0);
    setIframeVersion((current) => current + 1);
    setIframeLoadCount(0);
    setIframeHiddenForConfirmation(false);
    recordPayDebugEvent('start-payment', {
      iframeLoadCount: 0,
      iframeHiddenForConfirmation: false,
      showIframe: true,
    });
  };

  const handleReloadEmbeddedPayPage = () => {
    if (!order?.pay_url) {
      showError(t('支付链接不存在'));
      return;
    }
    setIframeLoaded(false);
    setIframeStalled(false);
    setPollingBlockedUntil(0);
    lastRemoteRefreshAtRef.current = Date.now();
    setIframeVersion((current) => current + 1);
    setIframeLoadCount(0);
    setIframeHiddenForConfirmation(false);
    recordPayDebugEvent('reload-payment-frame', {
      iframeLoadCount: 0,
      iframeHiddenForConfirmation: false,
      showIframe: true,
    });
  };

  const refreshActionDisabled = shouldDisableRefreshAction(
    Date.now(),
    pollingBlockedUntil,
  );

  const payPageHint = useMemo(() => {
    if (!showIframe) {
      return t('请先核对订单信息，确认无误后点击"开始支付"按钮。');
    }
    if (shouldNavigateFromPayPage(order)) {
      return t('支付已确认，正在准备订单详情页。');
    }
    if (iframeHiddenForConfirmation) {
      return t('正在确认支付结果，请稍候。');
    }
    if (shouldBackoffPolling(Date.now(), pollingBlockedUntil)) {
      return t('检查过于频繁，请稍后再试。');
    }
    if (iframeStalled) {
      return t('支付页面加载较慢，请点击“重新加载支付页”后重试。');
    }
    if (iframeLoaded) {
      return t('支付页面已加载，完成支付后请点击“我已完成支付，立即检查”。');
    }
    return t('正在加载支付页面，如长时间无响应请点击“重新加载支付页”。');
  }, [
    iframeHiddenForConfirmation,
    iframeLoaded,
    iframeStalled,
    order,
    pollingBlockedUntil,
    showIframe,
    t,
  ]);

  if (loading) {
    return (
      <div className='p-6 flex justify-center'>
        <Spin size='large' />
      </div>
    );
  }

  if (!order) {
    return (
      <div className='p-6'>
        <Card>
          <Text>{t('订单不存在或已无法访问')}</Text>
        </Card>
      </div>
    );
  }

  return (
    <div className='p-4 md:p-6 space-y-4'>
      <Card>
        <Space vertical align='start' style={{ width: '100%' }}>
          <Title heading={4}>{t('订单支付')}</Title>
          <Banner
            type={iframeStalled ? 'warning' : 'info'}
            style={{ width: '100%' }}
            description={
              <div>
                <Paragraph style={{ marginBottom: 4 }}>{payPageHint}</Paragraph>
              </div>
            }
          />
          {!showIframe && (
            <>
              <Descriptions
                data={[
                  { key: t('订单号'), value: order.local_trade_no || '-' },
                  { key: t('商品名称'), value: order.goods_name || '-' },
                  {
                    key: t('订单金额'),
                    value: `￥${Number(order.amount || 0).toFixed(2)}`,
                  },
                  { key: t('联系方式'), value: contactValue },
                  {
                    key: t('当前状态'),
                    value: (
                      <Tag color={statusColorMap[order.status] || 'grey'}>
                        {getExternalShopStatusLabel(order.status, t)}
                      </Tag>
                    ),
                  },
                ]}
              />
              <Space wrap>
                <Button
                  theme='solid'
                  type='primary'
                  onClick={handleContinuePay}
                  disabled={
                    !isExternalShopOrderPaymentActive(order) || !order?.pay_url
                  }
                >
                  {t('开始支付')}
                </Button>
                <Button loading={refreshing} onClick={handleRefresh}>
                  {t('检查结果')}
                </Button>
                <Button
                  onClick={() =>
                    navigate(
                      buildExternalShopOrderDetailPath(order.local_trade_no),
                    )
                  }
                >
                  {t('查看详情')}
                </Button>
                <Button onClick={() => navigate('/console/shop')}>
                  {t('返回商城')}
                </Button>
              </Space>
            </>
          )}
          {showIframe && (
            <>
              <Descriptions
                data={[
                  { key: t('订单号'), value: order.local_trade_no || '-' },
                  {
                    key: t('订单金额'),
                    value: `￥${Number(order.amount || 0).toFixed(2)}`,
                  },
                  {
                    key: t('当前状态'),
                    value: (
                      <Tag color={statusColorMap[order.status] || 'grey'}>
                        {getExternalShopStatusLabel(order.status, t)}
                      </Tag>
                    ),
                  },
                ]}
              />
              <div style={{ width: '100%', position: 'relative' }}>
                <iframe
                  key={iframeVersion}
                  src={order.pay_url}
                  title={t('支付页面')}
                  onLoad={() => {
                    setIframeLoaded(true);
                    setIframeStalled(false);
                    setIframeLoadCount((current) => {
                      const nextLoadCount = current + 1;
                      recordPayDebugEvent('iframe-load', {
                        iframeLoadCount: nextLoadCount,
                        iframeHiddenForConfirmation,
                        showIframe: true,
                      });
                      if (shouldHideIframeAfterLoad(current, order)) {
                        setIframeHiddenForConfirmation(true);
                        recordPayDebugEvent('iframe-hidden-for-confirmation', {
                          iframeLoadCount: nextLoadCount,
                          iframeHiddenForConfirmation: true,
                          showIframe: true,
                          reason: 'iframe-reload',
                        });
                      }
                      return nextLoadCount;
                    });
                  }}
                  style={{
                    width: '100%',
                    height: '700px',
                    border: '1px solid var(--semi-color-border)',
                    borderRadius: '4px',
                    visibility: iframeHiddenForConfirmation
                      ? 'hidden'
                      : 'visible',
                  }}
                  sandbox={PAYMENT_IFRAME_SANDBOX}
                />
                {iframeHiddenForConfirmation && (
                  <div
                    style={{
                      position: 'absolute',
                      inset: 0,
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      background: 'var(--semi-color-bg-0)',
                      border: '1px solid var(--semi-color-border)',
                      borderRadius: '4px',
                      padding: 16,
                    }}
                  >
                    <Card
                      bodyStyle={{ padding: 24 }}
                      style={{ width: '100%', maxWidth: 720 }}
                    >
                      <Space vertical align='start' style={{ width: '100%' }}>
                        <Title heading={6}>{t('正在确认支付结果')}</Title>
                        <Text>{t('请稍候，系统正在检查订单状态。')}</Text>
                        <Text type='tertiary'>
                          {t(
                            '如果页面没有自动更新，请点击“我已完成支付，立即检查”。如需继续付款，可点击“继续支付”。',
                          )}
                        </Text>
                      </Space>
                    </Card>
                  </div>
                )}
              </div>
              <Space wrap>
                <Button
                  theme='solid'
                  type='primary'
                  onClick={() => handleRefresh()}
                  disabled={refreshActionDisabled}
                >
                  {t('我已完成支付，立即检查')}
                </Button>
                {iframeHiddenForConfirmation && (
                  <Button
                    onClick={() => {
                      setIframeHiddenForConfirmation(false);
                      recordPayDebugEvent('resume-embedded-payment', {
                        iframeHiddenForConfirmation: false,
                        showIframe: true,
                      });
                    }}
                  >
                    {t('继续支付')}
                  </Button>
                )}
                {shouldShowEmbeddedReload(showIframe) && (
                  <Button onClick={handleReloadEmbeddedPayPage}>
                    {t('重新加载支付页')}
                  </Button>
                )}
                <Button
                  loading={refreshing}
                  onClick={handleRefresh}
                  disabled={refreshActionDisabled}
                >
                  {t('检查结果')}
                </Button>
                <Button
                  onClick={() => {
                    setShowIframe(false);
                    setPollingActive(false);
                    setIframeHiddenForConfirmation(false);
                    recordPayDebugEvent('return-to-order-info', {
                      iframeHiddenForConfirmation: false,
                      showIframe: false,
                    });
                  }}
                >
                  {t('返回订单信息')}
                </Button>
                <Button
                  onClick={() =>
                    navigate(
                      buildExternalShopOrderDetailPath(order.local_trade_no),
                    )
                  }
                >
                  {t('查看详情')}
                </Button>
              </Space>
            </>
          )}
        </Space>
      </Card>
    </div>
  );
}
