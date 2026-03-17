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

import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import {
  Button,
  Card,
  Descriptions,
  Empty,
  Space,
  Spin,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { API, showError, showSuccess } from '../../helpers';
import { useTranslation } from 'react-i18next';
import {
  buildExternalShopOrderPayPath,
  isExternalShopOrderPayable,
} from './orderPaths';
import {
  getExternalShopStatusLabel,
  getMaskedExternalShopContact,
} from './shopUtils';

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

const isMissingOrderMessage = (message) =>
  ['订单不存在', '无权访问该订单'].includes(String(message || '').trim());

function parseDeliveryPayload(payload) {
  if (!payload) {
    return [];
  }
  try {
    const parsed = JSON.parse(payload);
    if (Array.isArray(parsed.cards)) {
      return parsed.cards;
    }
  } catch {}
  return [];
}

export default function OrderDetail() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { local_trade_no: localTradeNo = '' } = useParams();
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [order, setOrder] = useState(null);
  const [showFullContact, setShowFullContact] = useState(false);

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
          { skipErrorHandler: true },
        );
        if (!res.data.success) {
          throw new Error(res.data.message || t('获取订单失败'));
        }
        setOrder(res.data.data || null);
      } catch (error) {
        setOrder(null);
        if (!isMissingOrderMessage(error.message)) {
          showError(error);
        }
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
    setShowFullContact(false);
  }, [localTradeNo]);

  useEffect(() => {
    if (
      !['created', 'pending_payment', 'paid_waiting_delivery'].includes(
        order?.status,
      )
    ) {
      return undefined;
    }
    const timer = setInterval(() => {
      loadOrder(false).catch(() => {});
    }, 5000);
    return () => clearInterval(timer);
  }, [loadOrder, order]);

  const deliveryCards = useMemo(
    () => parseDeliveryPayload(order?.delivery_payload),
    [order?.delivery_payload],
  );
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

  const handleRefresh = async () => {
    setRefreshing(true);
    try {
      const res = await API.post(
        `/api/external-shop/orders/${encodeURIComponent(localTradeNo)}/refresh`,
      );
      if (!res.data.success) {
        throw new Error(res.data.message || t('刷新订单失败'));
      }
      setOrder(res.data.data || null);
      showSuccess(t('订单已刷新'));
    } catch (error) {
      showError(error.message || t('刷新订单失败'));
    } finally {
      setRefreshing(false);
    }
  };

  if (loading) {
    return (
      <div className='p-6 flex justify-center max-w-5xl mx-auto'>
        <Spin size='large' />
      </div>
    );
  }

  if (!order) {
    return (
      <div className='p-4 md:p-6 max-w-5xl mx-auto'>
        <Card className='!rounded-2xl shadow-sm border-0'>
          <Empty description={t('订单不存在')} />
        </Card>
      </div>
    );
  }

  return (
    <div className='p-4 md:p-6 space-y-4 max-w-5xl mx-auto'>
      <Card className='!rounded-2xl shadow-sm border-0'>
        <Space vertical align='start' style={{ width: '100%' }}>
          <Title heading={4}>{t('订单详情')}</Title>
          <Descriptions
            data={[
              { key: t('订单号'), value: order.local_trade_no || '-' },
              { key: t('商品名称'), value: order.goods_name || '-' },
              { key: t('数量'), value: order.quantity ?? '-' },
              { key: t('联系方式'), value: contactValue },
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
          <Space wrap>
            {isExternalShopOrderPayable(order) ? (
              <Button
                theme='solid'
                type='primary'
                onClick={() =>
                  navigate(buildExternalShopOrderPayPath(order.local_trade_no))
                }
              >
                {t('前往支付')}
              </Button>
            ) : null}
            <Button loading={refreshing} onClick={handleRefresh}>
              {t('检查结果')}
            </Button>
            <Button onClick={() => navigate('/console/shop')}>
              {t('返回商城')}
            </Button>
          </Space>
        </Space>
      </Card>

      <Card className='!rounded-2xl shadow-sm border-0' title={t('交付结果')}>
        {deliveryCards.length > 0 ? (
          <Space vertical align='start' style={{ width: '100%' }}>
            {deliveryCards.map((card, index) => (
              <Paragraph
                key={`${card}-${index}`}
                copyable
                style={{ width: '100%', marginBottom: 0 }}
              >
                {card}
              </Paragraph>
            ))}
          </Space>
        ) : (
          <Text type='tertiary'>
            {t('暂无交付内容，请在支付完成后检查订单结果。')}
          </Text>
        )}
      </Card>

      {order.last_error ? (
        <Card className='!rounded-2xl shadow-sm border-0' title={t('最近错误')}>
          <Paragraph type='danger' style={{ marginBottom: 0 }}>
            {order.last_error}
          </Paragraph>
        </Card>
      ) : null}
    </div>
  );
}
