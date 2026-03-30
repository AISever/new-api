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

import React, { useRef } from 'react';
import { Form, Button } from '@douyinfe/semi-ui';
import { IconSearch } from '@douyinfe/semi-icons';

const TopupOrdersFilters = ({
  formInitValues,
  setFormApi,
  searchTopupOrders,
  loadTopupOrders,
  pageSize,
  loading,
  searching,
  t,
}) => {
  const formApiRef = useRef(null);

  const handleReset = () => {
    if (!formApiRef.current) return;
    formApiRef.current.reset();
    setTimeout(() => {
      loadTopupOrders(1, pageSize);
    }, 100);
  };

  return (
    <Form
      initValues={formInitValues}
      getFormApi={(api) => {
        setFormApi(api);
        formApiRef.current = api;
      }}
      onSubmit={() => searchTopupOrders(1, pageSize)}
      allowEmpty
      autoComplete='off'
      layout='horizontal'
      trigger='change'
      stopValidateWithError={false}
      className='w-full'
    >
      <div className='flex flex-col lg:flex-row items-center gap-2 w-full'>
        <div className='relative w-full lg:w-80'>
          <Form.Input
            field='searchKeyword'
            prefix={<IconSearch />}
            placeholder={t('支持搜索订单号、第三方单号、用户名和显示名称')}
            showClear
            pure
            size='small'
          />
        </div>
        <div className='w-full lg:w-40'>
          <Form.Select
            field='status'
            placeholder={t('订单状态')}
            optionList={[
              { label: t('待支付'), value: 'pending' },
              { label: t('成功'), value: 'success' },
              { label: t('失败'), value: 'failed' },
              { label: t('已过期'), value: 'expired' },
            ]}
            showClear
            pure
            size='small'
          />
        </div>
        <div className='w-full lg:w-40'>
          <Form.Select
            field='paymentMethod'
            placeholder={t('支付方式')}
            optionList={[
              { label: 'Stripe', value: 'stripe' },
              { label: 'Creem', value: 'creem' },
              { label: 'Waffo', value: 'waffo' },
              { label: t('在线支付'), value: 'ldxp' },
              { label: t('支付宝'), value: 'alipay' },
              { label: t('微信'), value: 'wxpay' },
            ]}
            showClear
            pure
            size='small'
          />
        </div>
        <div className='flex gap-2 w-full lg:w-auto'>
          <Button
            type='tertiary'
            htmlType='submit'
            loading={loading || searching}
            size='small'
            className='flex-1 lg:flex-initial'
          >
            {t('查询')}
          </Button>
          <Button
            type='tertiary'
            onClick={handleReset}
            size='small'
            className='flex-1 lg:flex-initial'
          >
            {t('重置')}
          </Button>
        </div>
      </div>
    </Form>
  );
};

export default TopupOrdersFilters;

