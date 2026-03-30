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

import React from 'react';
import { Button, Tag, Typography } from '@douyinfe/semi-ui';
import { Coins } from 'lucide-react';
import { timestamp2string } from '../../../helpers';
import {
  getPaymentMethodLabel,
  isSubscriptionTopupRecord,
} from './topupOrderUtils';

const { Text } = Typography;

const statusConfigMap = {
  pending: { color: 'blue', label: '待支付' },
  success: { color: 'green', label: '成功' },
  failed: { color: 'red', label: '失败' },
  expired: { color: 'grey', label: '已过期' },
};

export function getTopupOrdersColumns({ t, openDetail }) {
  return [
    {
      title: t('订单号'),
      dataIndex: 'trade_no',
      key: 'trade_no',
      width: 220,
      render: (text) => <Text copyable>{text}</Text>,
    },
    {
      title: t('用户'),
      dataIndex: 'username',
      key: 'username',
      width: 180,
      render: (_, record) => (
        <div className='flex flex-col'>
          <Text>{record.display_name || record.username || '-'}</Text>
          <Text type='tertiary' size='small'>
            {record.username ? `@${record.username}` : `${t('用户ID')}: ${record.user_id}`}
          </Text>
        </div>
      ),
    },
    {
      title: t('支付方式'),
      dataIndex: 'payment_method',
      key: 'payment_method',
      width: 120,
      render: (value) => getPaymentMethodLabel(value, t),
    },
    {
      title: t('充值额度'),
      dataIndex: 'amount',
      key: 'amount',
      width: 120,
      render: (value, record) => {
        if (isSubscriptionTopupRecord(record)) {
          return (
            <Tag color='purple' shape='circle' size='small'>
              {t('订阅套餐')}
            </Tag>
          );
        }
        return (
          <span className='flex items-center gap-1'>
            <Coins size={16} />
            <Text>{value}</Text>
          </span>
        );
      },
    },
    {
      title: t('支付金额'),
      dataIndex: 'money',
      key: 'money',
      width: 120,
      render: (money) => <Text type='danger'>¥{Number(money || 0).toFixed(2)}</Text>,
    },
    {
      title: t('状态'),
      dataIndex: 'status',
      key: 'status',
      width: 110,
      render: (status) => {
        const statusConfig = statusConfigMap[status] || {
          color: 'grey',
          label: status || '-',
        };
        return <Tag color={statusConfig.color}>{t(statusConfig.label)}</Tag>;
      },
    },
    {
      title: t('第三方单号'),
      dataIndex: 'provider_trade_no',
      key: 'provider_trade_no',
      width: 220,
      render: (value) => (value ? <Text copyable>{value}</Text> : '-'),
    },
    {
      title: t('创建时间'),
      dataIndex: 'create_time',
      key: 'create_time',
      width: 180,
      render: (value) => (value ? timestamp2string(value) : '-'),
    },
    {
      title: t('完成时间'),
      dataIndex: 'complete_time',
      key: 'complete_time',
      width: 180,
      render: (value) => (value ? timestamp2string(value) : '-'),
    },
    {
      title: t('操作'),
      dataIndex: 'operate',
      key: 'operate',
      fixed: 'right',
      width: 100,
      render: (_, record) => (
        <Button size='small' type='tertiary' onClick={() => openDetail(record)}>
          {t('详情')}
        </Button>
      ),
    },
  ];
}

