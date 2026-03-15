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

import React, { useEffect, useMemo, useState } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import {
  Button,
  Card,
  Empty,
  Input,
  Modal,
  Pagination,
  Select,
  Space,
  Spin,
  TabPane,
  Table,
  Tabs,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { API, showError, showSuccess } from '../../helpers';
import { useTranslation } from 'react-i18next';
import {
  buildExternalShopOrderDetailPath,
  buildExternalShopOrderPayPath,
  isExternalShopOrderPayable,
} from './orderPaths';
import {
  AUTO_REFRESH_INTERVAL_MS,
  getExternalShopDeliverySummary,
  getExternalShopStatusLabel,
  getAutoRefreshOrderTradeNos,
  normalizeExternalShopDescription,
  parseExternalShopDeliveryCards,
  shouldShowDescriptionToggle,
} from './shopUtils';
import GptTeamPlanTab from './GptTeamPlanTab';
import { normalizeShopTabKey } from './gptTeamPlanUtils';

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

export default function Shop() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const location = useLocation();
  const [loading, setLoading] = useState(true);
  const [goods, setGoods] = useState([]);
  const [orders, setOrders] = useState([]);
  const [gptTeamEnabled, setGptTeamEnabled] = useState(false);
  const [gptTeamStatusLoaded, setGptTeamStatusLoaded] = useState(false);
  const [activeTab, setActiveTab] = useState('goods');
  const [orderTotal, setOrderTotal] = useState(0);
  const [createLoading, setCreateLoading] = useState(false);
  const [orderLoading, setOrderLoading] = useState(false);
  const [selectedGood, setSelectedGood] = useState(null);
  const [contact, setContact] = useState('');
  const [expandedGoods, setExpandedGoods] = useState({});
  const [revealedDeliveries, setRevealedDeliveries] = useState({});
  const [goodsSort, setGoodsSort] = useState('default');
  const [goodsPage, setGoodsPage] = useState(1);
  const [goodsPageSize, setGoodsPageSize] = useState(4);
  const [orderPage, setOrderPage] = useState(1);
  const [orderPageSize, setOrderPageSize] = useState(10);
  const [orderSort, setOrderSort] = useState('created_at:desc');

  const loadGoods = async () => {
    const res = await API.get('/api/external-shop/goods');
    if (!res.data.success) {
      throw new Error(res.data.message || t('获取商品失败'));
    }
    setGoods(res.data.data || []);
  };

  const loadGPTTeamStatus = async () => {
    try {
      const res = await API.get('/api/gptteamplan/status', {
        skipErrorHandler: true,
      });
      if (res?.data?.success) {
        setGptTeamEnabled(Boolean(res.data.data?.enabled));
        setGptTeamStatusLoaded(true);
        return;
      }
    } catch {
      // Keep the existing shop tab available even if the GPT Team status endpoint is unavailable.
    }
    setGptTeamEnabled(false);
    setGptTeamStatusLoaded(true);
  };

  const loadOrders = async (options = {}) => {
    const {
      silent = false,
      page = orderPage,
      pageSize = orderPageSize,
      sort = orderSort,
    } = options;
    if (!silent) {
      setOrderLoading(true);
    }
    try {
      const [sortBy = 'created_at', sortOrder = 'desc'] = String(
        sort || 'created_at:desc',
      ).split(':');
      const params = new URLSearchParams({
        p: String(page),
        page_size: String(pageSize),
        sort_by: sortBy,
        sort_order: sortOrder,
      });
      const res = await API.get(
        `/api/external-shop/orders?${params.toString()}`,
      );
      if (!res.data.success) {
        throw new Error(res.data.message || t('获取订单失败'));
      }
      setOrders(res.data.data?.items || []);
      setOrderTotal(res.data.data?.total || 0);
    } finally {
      if (!silent) {
        setOrderLoading(false);
      }
    }
  };

  useEffect(() => {
    const load = async () => {
      setLoading(true);
      try {
        await Promise.all([
          loadGoods(),
          loadOrders({ page: 1, pageSize: orderPageSize, sort: orderSort }),
          loadGPTTeamStatus(),
        ]);
      } catch (error) {
        showError(error.message || t('加载失败'));
      } finally {
        setLoading(false);
      }
    };
    load();
  }, []);

  useEffect(() => {
    if (!gptTeamStatusLoaded) {
      return;
    }
    const searchParams = new URLSearchParams(location.search);
    const nextTab = normalizeShopTabKey(
      searchParams.get('tab'),
      gptTeamEnabled,
    );
    setActiveTab(nextTab);
    if (searchParams.get('tab') !== nextTab) {
      navigate(`${location.pathname}?tab=${nextTab}`, { replace: true });
    }
  }, [gptTeamEnabled, gptTeamStatusLoaded, location.pathname, location.search, navigate]);

  useEffect(() => {
    if (loading) {
      return;
    }
    loadOrders({
      page: orderPage,
      pageSize: orderPageSize,
      sort: orderSort,
    }).catch((error) => {
      showError(error.message || t('获取订单失败'));
    });
  }, [loading, orderPage, orderPageSize, orderSort, t]);

  useEffect(() => {
    const refreshTradeNos = getAutoRefreshOrderTradeNos(orders);
    if (refreshTradeNos.length === 0) return undefined;

    const timer = setInterval(async () => {
      for (const localTradeNo of refreshTradeNos) {
        try {
          await API.post(
            `/api/external-shop/orders/${localTradeNo}/refresh`,
            null,
            { skipErrorHandler: true },
          );
        } catch {}
      }
      try {
        await loadOrders({
          silent: true,
          page: orderPage,
          pageSize: orderPageSize,
          sort: orderSort,
        });
      } catch {}
    }, AUTO_REFRESH_INTERVAL_MS);

    return () => clearInterval(timer);
  }, [orderPage, orderPageSize, orderSort, orders]);

  const openOrderModal = (good) => {
    setSelectedGood(good);
    setContact('');
  };

  const createOrder = async () => {
    if (!selectedGood) return;
    if (!contact.trim()) {
      showError(t('请输入联系方式'));
      return;
    }
    setCreateLoading(true);
    try {
      const res = await API.post('/api/external-shop/orders', {
        goods_key: selectedGood.goods_key,
        quantity: 1,
        contact: contact.trim(),
        channel_id: 0,
      });
      if (!res.data.success) {
        throw new Error(res.data.message || t('创建订单失败'));
      }
      const data = res.data.data || {};
      showSuccess(t('订单已创建'));
      setSelectedGood(null);
      setOrderPage(1);
      await loadOrders({ page: 1, pageSize: orderPageSize, sort: orderSort });
      if (data.local_trade_no) {
        navigate(buildExternalShopOrderPayPath(data.local_trade_no));
      }
    } catch (error) {
      showError(error.message || t('创建订单失败'));
    } finally {
      setCreateLoading(false);
    }
  };

  const refreshOrder = async (record) => {
    try {
      const res = await API.post(
        `/api/external-shop/orders/${record.local_trade_no}/refresh`,
      );
      if (!res.data.success) {
        throw new Error(res.data.message || t('刷新订单失败'));
      }
      showSuccess(t('订单已刷新'));
      await loadOrders({
        page: orderPage,
        pageSize: orderPageSize,
        sort: orderSort,
      });
    } catch (error) {
      showError(error.message || t('刷新订单失败'));
    }
  };

  const toggleGoodDescription = (goodsKey) => {
    setExpandedGoods((current) => ({
      ...current,
      [goodsKey]: !current[goodsKey],
    }));
  };

  const toggleDeliveryVisibility = (localTradeNo) => {
    setRevealedDeliveries((current) => ({
      ...current,
      [localTradeNo]: !current[localTradeNo],
    }));
  };

  const sortedGoods = useMemo(() => {
    const nextGoods = [...(goods || [])];
    switch (goodsSort) {
      case 'price:asc':
        nextGoods.sort(
          (left, right) => Number(left.price || 0) - Number(right.price || 0),
        );
        break;
      case 'price:desc':
        nextGoods.sort(
          (left, right) => Number(right.price || 0) - Number(left.price || 0),
        );
        break;
      case 'stock:desc':
        nextGoods.sort(
          (left, right) =>
            Number(right.stock_count || 0) - Number(left.stock_count || 0),
        );
        break;
      case 'name:asc':
        nextGoods.sort((left, right) =>
          String(left.name || '').localeCompare(
            String(right.name || ''),
            'zh-Hans-CN',
          ),
        );
        break;
      default:
        break;
    }
    return nextGoods;
  }, [goods, goodsSort]);

  const paginatedGoods = useMemo(() => {
    const start = (goodsPage - 1) * goodsPageSize;
    return sortedGoods.slice(start, start + goodsPageSize);
  }, [goodsPage, goodsPageSize, sortedGoods]);

  const columns = useMemo(
    () => [
      {
        title: t('订单号'),
        dataIndex: 'local_trade_no',
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
        title: t('交付内容'),
        dataIndex: 'delivery_payload',
        render: (value, record) => {
          const cards = parseExternalShopDeliveryCards(value);
          if (cards.length === 0) {
            return '-';
          }
          const revealed = Boolean(revealedDeliveries[record.local_trade_no]);
          return (
            <Space>
              <Text code>
                {revealed
                  ? cards.join(' / ')
                  : getExternalShopDeliverySummary(value, t)}
              </Text>
              <Button
                size='small'
                theme='borderless'
                type='tertiary'
                onClick={() => toggleDeliveryVisibility(record.local_trade_no)}
              >
                {revealed ? t('隐藏') : t('显示')}
              </Button>
            </Space>
          );
        },
      },
      {
        title: t('操作'),
        render: (_, record) => (
          <Space>
            {record.pay_url ? (
              isExternalShopOrderPayable(record) ? (
                <Button
                  size='small'
                  onClick={() =>
                    navigate(
                      buildExternalShopOrderPayPath(record.local_trade_no),
                    )
                  }
                >
                  {t('继续支付')}
                </Button>
              ) : null
            ) : null}
            <Button
              size='small'
              onClick={() =>
                navigate(
                  buildExternalShopOrderDetailPath(record.local_trade_no),
                )
              }
            >
              {t('查看详情')}
            </Button>
            <Button size='small' onClick={() => refreshOrder(record)}>
              {t('检查结果')}
            </Button>
          </Space>
        ),
      },
    ],
    [navigate, revealedDeliveries, t],
  );

  if (loading) {
    return (
      <div className='p-6 flex justify-center'>
        <Spin size='large' />
      </div>
    );
  }

  const renderExternalShopContent = () => (
    <div className='px-4 pb-6 pt-8 md:px-6 md:pb-6 md:pt-10 space-y-6'>
      <div style={{ scrollMarginTop: 96 }}>
        <Title heading={4}>{t('商品商城')}</Title>
        <Text type='tertiary'>
          {t('选择商品并完成支付后，可在这里查看订单进度和交付结果。')}
        </Text>
      </div>

      <Card>
        <Space wrap style={{ width: '100%', justifyContent: 'space-between' }}>
          <Text type='tertiary'>
            {t('可按价格、库存或名称排序，并分页查看商品。')}
          </Text>
          <Space wrap>
            <Select
              value={goodsSort}
              onChange={(value) => {
                setGoodsSort(value);
                setGoodsPage(1);
              }}
              style={{ width: 180 }}
            >
              <Select.Option value='default'>{t('默认排序')}</Select.Option>
              <Select.Option value='price:asc'>
                {t('价格从低到高')}
              </Select.Option>
              <Select.Option value='price:desc'>
                {t('价格从高到低')}
              </Select.Option>
              <Select.Option value='stock:desc'>
                {t('库存从高到低')}
              </Select.Option>
              <Select.Option value='name:asc'>{t('名称排序')}</Select.Option>
            </Select>
          </Space>
        </Space>
      </Card>

      <div className='grid grid-cols-1 lg:grid-cols-2 gap-4'>
        {paginatedGoods.map((good) => (
          <Card
            key={good.goods_key}
            title={good.name}
            headerExtraContent={
              <Space>
                <Tag color='blue'>{good.category_name}</Tag>
                {Number(good.stock_count || 0) <= 0 ? (
                  <Tag color='red'>{t('缺货')}</Tag>
                ) : null}
              </Space>
            }
          >
            <Space vertical align='start' style={{ width: '100%' }}>
              <Text strong>{`￥${Number(good.price || 0).toFixed(2)}`}</Text>
              {(() => {
                const normalizedDescription = normalizeExternalShopDescription(
                  good.description,
                );
                const expanded = Boolean(expandedGoods[good.goods_key]);
                const canToggle = shouldShowDescriptionToggle(
                  normalizedDescription,
                );
                return (
                  <Space
                    vertical
                    align='start'
                    spacing={4}
                    style={{ width: '100%' }}
                  >
                    {normalizedDescription ? (
                      <Paragraph
                        style={{
                          width: '100%',
                          marginBottom: 0,
                          whiteSpace: 'pre-wrap',
                          ...(expanded
                            ? {}
                            : {
                                display: '-webkit-box',
                                WebkitLineClamp: 4,
                                WebkitBoxOrient: 'vertical',
                                overflow: 'hidden',
                              }),
                        }}
                      >
                        {normalizedDescription}
                      </Paragraph>
                    ) : null}
                    {normalizedDescription && canToggle ? (
                      <Button
                        size='small'
                        theme='borderless'
                        type='tertiary'
                        onClick={() => toggleGoodDescription(good.goods_key)}
                      >
                        {expanded ? t('收起') : t('展开')}
                      </Button>
                    ) : null}
                  </Space>
                );
              })()}
              <Text type='tertiary'>{`${t('库存')}: ${good.stock_count}`}</Text>
              <Button
                theme='solid'
                type='primary'
                disabled={Number(good.stock_count || 0) <= 0}
                onClick={() => openOrderModal(good)}
              >
                {Number(good.stock_count || 0) <= 0
                  ? t('暂时缺货')
                  : t('立即购买')}
              </Button>
            </Space>
          </Card>
        ))}
      </div>

      {(goods || []).length === 0 ? (
        <Empty description={t('暂无商品')} />
      ) : null}

      {(goods || []).length > 0 ? (
        <div className='flex justify-end'>
          <Pagination
            currentPage={goodsPage}
            pageSize={goodsPageSize}
            total={sortedGoods.length}
            pageSizeOpts={[4, 8, 12]}
            showSizeChanger
            onPageChange={setGoodsPage}
            onPageSizeChange={(size) => {
              setGoodsPageSize(size);
              setGoodsPage(1);
            }}
          />
        </div>
      ) : null}

      <Card title={t('我的订单')}>
        <Space
          wrap
          style={{
            width: '100%',
            justifyContent: 'space-between',
            marginBottom: 12,
          }}
        >
          <Text type='tertiary'>
            {t('支持按创建时间、金额或状态排序，并分页查看订单。')}
          </Text>
          <Select
            value={orderSort}
            onChange={(value) => {
              setOrderSort(value);
              setOrderPage(1);
            }}
            style={{ width: 200 }}
          >
            <Select.Option value='created_at:desc'>
              {t('最新订单优先')}
            </Select.Option>
            <Select.Option value='created_at:asc'>
              {t('最早订单优先')}
            </Select.Option>
            <Select.Option value='amount:desc'>
              {t('金额从高到低')}
            </Select.Option>
            <Select.Option value='amount:asc'>
              {t('金额从低到高')}
            </Select.Option>
            <Select.Option value='status:asc'>{t('状态排序')}</Select.Option>
          </Select>
        </Space>
        <Table
          rowKey='local_trade_no'
          loading={orderLoading}
          dataSource={orders}
          columns={columns}
          pagination={{
            currentPage: orderPage,
            pageSize: orderPageSize,
            total: orderTotal,
            pageSizeOpts: [10, 20, 50],
            showSizeChanger: true,
            onPageChange: setOrderPage,
            onPageSizeChange: (size) => {
              setOrderPageSize(size);
              setOrderPage(1);
            },
          }}
        />
      </Card>

      <Modal
        title={selectedGood?.name || t('创建订单')}
        visible={!!selectedGood}
        onCancel={() => setSelectedGood(null)}
        onOk={createOrder}
        confirmLoading={createLoading}
      >
        <Space vertical style={{ width: '100%' }}>
          <Text>{`${t('商品价格')}: ￥${Number(selectedGood?.price || 0).toFixed(2)}`}</Text>
          <Input
            value={contact}
            onChange={setContact}
            placeholder={t('请输入联系方式')}
          />
        </Space>
      </Modal>
    </div>
  );

  return (
    <div className='px-4 pb-6 pt-8 md:px-6 md:pb-6 md:pt-10'>
      <Tabs
        type='card'
        activeKey={activeTab}
        onChange={(key) => {
          const nextTab = normalizeShopTabKey(key, gptTeamEnabled);
          setActiveTab(nextTab);
          navigate(`${location.pathname}?tab=${nextTab}`, { replace: true });
        }}
      >
        <TabPane itemKey='goods' tab={t('商品商城')}>
          {activeTab === 'goods' ? renderExternalShopContent() : null}
        </TabPane>
        {gptTeamEnabled ? (
          <TabPane itemKey='gpt-team' tab={t('GPT Team 兑换')}>
            {activeTab === 'gpt-team' ? <GptTeamPlanTab /> : null}
          </TabPane>
        ) : null}
      </Tabs>
    </div>
  );
}
