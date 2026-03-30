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

import React, { useMemo } from 'react';
import {
  Banner,
  Button,
  Card,
  Descriptions,
  Empty,
  Modal,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { useIsMobile } from '../../../../hooks/common/useIsMobile';
import { copy, timestamp2string } from '../../../../helpers';
import {
  extractLdxpAuditInfo,
  getPaymentMethodLabel,
  getTopupAmountDisplay,
  parseProviderPayload,
} from '../topupOrderUtils';

const { Text } = Typography;

const statusConfigMap = {
  pending: { color: 'blue', label: '待支付' },
  success: { color: 'green', label: '成功' },
  failed: { color: 'red', label: '失败' },
  expired: { color: 'grey', label: '已过期' },
};

const TopupOrderDetailModal = ({ visible, onCancel, order, t }) => {
  const isMobile = useIsMobile();

  const auditInfo = useMemo(
    () => extractLdxpAuditInfo(order?.provider_payload),
    [order?.provider_payload],
  );
  const parsedPayload = useMemo(
    () => parseProviderPayload(order?.provider_payload),
    [order?.provider_payload],
  );

  const baseRows = useMemo(() => {
    if (!order) {
      return [];
    }
    const statusConfig = statusConfigMap[order.status] || {
      color: 'grey',
      label: order.status || '-',
    };
    const amountDisplay = getTopupAmountDisplay(order);

    return [
      {
        key: t('订单号'),
        value: <Text copyable>{order.trade_no}</Text>,
      },
      {
        key: t('用户'),
        value: (
          <div className='flex flex-col'>
            <Text>{order.display_name || order.username || '-'}</Text>
            <Text type='tertiary' size='small'>
              {order.username ? `@${order.username}` : `${t('用户ID')}: ${order.user_id}`}
            </Text>
          </div>
        ),
      },
      {
        key: t('支付方式'),
        value: getPaymentMethodLabel(order.payment_method, t),
      },
      {
        key: t('订单状态'),
        value: <Tag color={statusConfig.color}>{t(statusConfig.label)}</Tag>,
      },
      {
        key: t('充值额度'),
        value: amountDisplay.isSubscription
          ? t(amountDisplay.value)
          : amountDisplay.value,
      },
      {
        key: t('支付金额'),
        value: `¥${Number(order.money || 0).toFixed(2)}`,
      },
      {
        key: t('第三方单号'),
        value: order.provider_trade_no ? <Text copyable>{order.provider_trade_no}</Text> : '-',
      },
      {
        key: t('创建时间'),
        value: order.create_time ? timestamp2string(order.create_time) : '-',
      },
      {
        key: t('完成时间'),
        value: order.complete_time ? timestamp2string(order.complete_time) : '-',
      },
    ];
  }, [order, t]);

  const handleCopyPayload = async () => {
    if (!order?.provider_payload) return;
    await copy(order.provider_payload);
  };

  return (
    <Modal
      title={t('充值订单详情')}
      visible={visible}
      onCancel={onCancel}
      footer={
        <div className='flex justify-between'>
          <Button
            theme='borderless'
            disabled={!order?.provider_payload}
            onClick={handleCopyPayload}
          >
            {t('复制原始回包')}
          </Button>
          <Button onClick={onCancel}>{t('关闭')}</Button>
        </div>
      }
      size={isMobile ? 'full-width' : 'large'}
    >
      {order ? (
        <div className='space-y-4 max-h-[70vh] overflow-y-auto'>
          <Card className='border-0 shadow-sm'>
            <Descriptions data={baseRows} />
          </Card>

          {order.payment_method === 'ldxp' && (
            <Card className='border-0 shadow-sm' title={t('在线支付审计信息')}>
              {auditInfo.cards.length > 0 || auditInfo.exportCardsUrl || auditInfo.transactionId ? (
                <div className='space-y-3'>
                  <Descriptions
                    data={[
                      {
                        key: t('上游交易号'),
                        value: auditInfo.transactionId ? (
                          <Text copyable>{auditInfo.transactionId}</Text>
                        ) : '-',
                      },
                      {
                        key: t('卡密导出地址'),
                        value: auditInfo.exportCardsUrl ? (
                          <Typography.Link
                            href={auditInfo.exportCardsUrl}
                            target='_blank'
                            rel='noreferrer'
                          >
                            {t('打开链接')}
                          </Typography.Link>
                        ) : '-',
                      },
                      {
                        key: t('上游成功时间'),
                        value: auditInfo.successTime
                          ? timestamp2string(auditInfo.successTime)
                          : '-',
                      },
                    ]}
                  />

                  {auditInfo.cards.length > 0 && (
                    <div className='space-y-2'>
                      <Text strong>{t('卡密列表')}</Text>
                      <div className='flex flex-wrap gap-2'>
                        {auditInfo.cards.map((card) => (
                          <Tag key={card} color='blue' shape='circle'>
                            {card}
                          </Tag>
                        ))}
                      </div>
                    </div>
                  )}
                </div>
              ) : (
                <Banner
                  type='info'
                  closeIcon={null}
                  description={t('当前订单还没有同步到上游卡密或交付信息。')}
                />
              )}
            </Card>
          )}

          <Card className='border-0 shadow-sm' title={t('原始回包')}>
            {parsedPayload ? (
              <pre className='m-0 whitespace-pre-wrap break-all text-xs leading-6 text-[var(--semi-color-text-1)]'>
                {JSON.stringify(parsedPayload, null, 2)}
              </pre>
            ) : (
              <Empty description={t('暂无原始回包')} />
            )}
          </Card>
        </div>
      ) : null}
    </Modal>
  );
};

export default TopupOrderDetailModal;
