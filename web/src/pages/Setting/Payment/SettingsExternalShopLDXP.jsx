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

import React, { useEffect, useRef, useState } from 'react';
import {
  Button,
  Form,
  Input,
  Select,
  Space,
  Spin,
  Table,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import {
  API,
  removeTrailingSlash,
  showError,
  showSuccess,
} from '../../../helpers';
import { useTranslation } from 'react-i18next';
import { getExternalShopStatusLabel } from '../../../components/shop/shopUtils';

const { Text } = Typography;
const statusColorMap = {
  created: 'grey',
  pending_payment: 'blue',
  paid_waiting_delivery: 'orange',
  delivered: 'green',
  failed: 'red',
  expired: 'grey',
  manual_review: 'pink',
};

export default function SettingsExternalShopLDXP(props) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [orderLoading, setOrderLoading] = useState(false);
  const [orders, setOrders] = useState([]);
  const [orderPage, setOrderPage] = useState(1);
  const [orderTotal, setOrderTotal] = useState(0);
  const [orderKeyword, setOrderKeyword] = useState('');
  const [orderStatus, setOrderStatus] = useState('');
  const [inputs, setInputs] = useState({
    enabled: false,
    base_url: 'https://pay.ldxp.cn',
    shop_name: '',
    shop_token: '',
    allowed_category_ids: '',
    allowed_goods_keys: '',
  });
  const formApiRef = useRef(null);

  useEffect(() => {
    if (props.options && formApiRef.current) {
      const nextValues = {
        enabled: props.options['external_shop.ldxp.enabled'] === 'true',
        base_url:
          props.options['external_shop.ldxp.base_url'] || 'https://pay.ldxp.cn',
        shop_name: props.options['external_shop.ldxp.shop_name'] || '',
        shop_token: '',
        allowed_category_ids:
          props.options['external_shop.ldxp.allowed_category_ids'] || '',
        allowed_goods_keys:
          props.options['external_shop.ldxp.allowed_goods_keys'] || '',
      };
      setInputs(nextValues);
      formApiRef.current.setValues(nextValues);
    }
  }, [props.options]);

  const loadOrders = async (
    page = orderPage,
    keyword = orderKeyword,
    status = orderStatus,
  ) => {
    setOrderLoading(true);
    try {
      const res = await API.get('/api/external-shop/admin/orders', {
        params: {
          p: page,
          page_size: 10,
          keyword,
          status,
        },
      });
      if (!res.data.success) {
        throw new Error(res.data.message || t('获取订单失败'));
      }
      setOrders(res.data.data?.items || []);
      setOrderTotal(res.data.data?.total || 0);
      setOrderPage(page);
    } catch (error) {
      showError(error.message || t('获取订单失败'));
    } finally {
      setOrderLoading(false);
    }
  };

  useEffect(() => {
    loadOrders(1, '', '');
  }, []);

  const handleSubmit = async () => {
    setLoading(true);
    try {
      const options = [
        {
          key: 'external_shop.ldxp.enabled',
          value: inputs.enabled ? 'true' : 'false',
        },
        {
          key: 'external_shop.ldxp.base_url',
          value: removeTrailingSlash(inputs.base_url || 'https://pay.ldxp.cn'),
        },
        {
          key: 'external_shop.ldxp.shop_name',
          value: inputs.shop_name || '',
        },
        {
          key: 'external_shop.ldxp.allowed_category_ids',
          value: inputs.allowed_category_ids || '',
        },
        {
          key: 'external_shop.ldxp.allowed_goods_keys',
          value: inputs.allowed_goods_keys || '',
        },
      ];

      if (inputs.shop_token) {
        options.push({
          key: 'external_shop.ldxp.shop_token',
          value: inputs.shop_token,
        });
      }

      const results = await Promise.all(
        options.map((opt) =>
          API.put('/api/option/', {
            key: opt.key,
            value: opt.value,
          }),
        ),
      );

      const failed = results.find((res) => !res.data.success);
      if (failed) {
        showError(failed.data.message || t('更新失败'));
        return;
      }
      showSuccess(t('更新成功'));
      props.refresh?.();
    } catch (error) {
      showError(t('更新失败'));
    } finally {
      setLoading(false);
    }
  };

  const handleSync = async () => {
    setLoading(true);
    try {
      const res = await API.post('/api/external-shop/admin/sync');
      if (res.data.success) {
        showSuccess(t('同步成功'));
        await loadOrders(1);
      } else {
        showError(res.data.message || t('同步失败'));
      }
    } catch (error) {
      showError(t('同步失败'));
    } finally {
      setLoading(false);
    }
  };

  const handleRefreshOrder = async (record) => {
    setOrderLoading(true);
    try {
      const res = await API.post(
        `/api/external-shop/admin/orders/${record.local_trade_no}/refresh`,
      );
      if (!res.data.success) {
        throw new Error(res.data.message || t('刷新订单失败'));
      }
      showSuccess(t('订单已刷新'));
      await loadOrders(orderPage, orderKeyword, orderStatus);
    } catch (error) {
      showError(error.message || t('刷新订单失败'));
    } finally {
      setOrderLoading(false);
    }
  };

  const handleSyncPendingOrders = async () => {
    setOrderLoading(true);
    try {
      const res = await API.post(
        '/api/external-shop/admin/orders/sync-pending',
      );
      if (!res.data.success) {
        throw new Error(res.data.message || t('同步挂单失败'));
      }
      showSuccess(t('挂单同步完成'));
      await loadOrders(orderPage, orderKeyword, orderStatus);
    } catch (error) {
      showError(error.message || t('同步挂单失败'));
    } finally {
      setOrderLoading(false);
    }
  };

  const columns = [
    {
      title: t('本地订单号'),
      dataIndex: 'local_trade_no',
    },
    {
      title: t('用户 ID'),
      dataIndex: 'user_id',
    },
    {
      title: t('商品'),
      dataIndex: 'goods_name',
    },
    {
      title: t('金额'),
      dataIndex: 'amount',
      render: (value) => `￥${Number(value || 0).toFixed(2)}`,
    },
    {
      title: t('状态'),
      dataIndex: 'status',
      render: (value) => (
        <Tag color={statusColorMap[value] || 'grey'}>
          {getExternalShopStatusLabel(value, t)}
        </Tag>
      ),
    },
    {
      title: t('最近错误'),
      dataIndex: 'last_error',
      render: (value) => value || '-',
    },
    {
      title: t('操作'),
      render: (_, record) => (
        <Space>
          <Button size='small' onClick={() => handleRefreshOrder(record)}>
            {t('刷新订单')}
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <Spin spinning={loading}>
      <Form
        initValues={inputs}
        getFormApi={(api) => (formApiRef.current = api)}
        onValueChange={(values) => setInputs(values)}
      >
        <Form.Section text={t('LDXP 外部商城')}>
          <Text type='tertiary'>
            {t(
              '用于同步链动小铺商品目录，并在站内创建订单后跳转到上游支付页面。',
            )}
          </Text>
          <Form.Switch field='enabled' label={t('启用外部商城')} />
          <Form.Input
            field='base_url'
            label={t('商城地址')}
            placeholder='https://pay.ldxp.cn'
          />
          <Form.Input
            field='shop_name'
            label={t('店铺名称')}
            placeholder='AI工具圈'
          />
          <Form.Input
            field='shop_token'
            label={t('店铺 Token')}
            placeholder={t('如需更新请重新输入')}
          />
          <Form.Input
            field='allowed_category_ids'
            label={t('允许分类 ID')}
            placeholder='20746,21722'
          />
          <Form.Input
            field='allowed_goods_keys'
            label={t('允许商品 Key')}
            placeholder='qnx5gk,z7wkl8'
          />
          <div className='flex gap-2'>
            <Button onClick={handleSubmit}>{t('保存配置')}</Button>
            <Button type='primary' theme='solid' onClick={handleSync}>
              {t('立即同步商品')}
            </Button>
          </div>
        </Form.Section>
      </Form>
      <div className='mt-6 space-y-3'>
        <Text strong>{t('商城订单管理')}</Text>
        <Space wrap>
          <Input
            value={orderKeyword}
            onChange={setOrderKeyword}
            placeholder={t('搜索订单号 / 商品 / 联系方式')}
            style={{ width: 260 }}
          />
          <Select
            value={orderStatus}
            onChange={setOrderStatus}
            style={{ width: 180 }}
            optionList={[
              { label: t('全部状态'), value: '' },
              {
                label: getExternalShopStatusLabel('created', t),
                value: 'created',
              },
              {
                label: getExternalShopStatusLabel('pending_payment', t),
                value: 'pending_payment',
              },
              {
                label: getExternalShopStatusLabel('paid_waiting_delivery', t),
                value: 'paid_waiting_delivery',
              },
              {
                label: getExternalShopStatusLabel('delivered', t),
                value: 'delivered',
              },
              {
                label: getExternalShopStatusLabel('failed', t),
                value: 'failed',
              },
              {
                label: getExternalShopStatusLabel('expired', t),
                value: 'expired',
              },
              {
                label: getExternalShopStatusLabel('manual_review', t),
                value: 'manual_review',
              },
            ]}
          />
          <Button onClick={() => loadOrders(1, orderKeyword, orderStatus)}>
            {t('查询')}
          </Button>
          <Button
            type='primary'
            theme='solid'
            onClick={handleSyncPendingOrders}
          >
            {t('同步挂单')}
          </Button>
        </Space>
        <Table
          rowKey='local_trade_no'
          loading={orderLoading}
          dataSource={orders}
          columns={columns}
          pagination={{
            currentPage: orderPage,
            pageSize: 10,
            total: orderTotal,
            onPageChange: (page) => loadOrders(page, orderKeyword, orderStatus),
          }}
        />
      </div>
    </Spin>
  );
}
