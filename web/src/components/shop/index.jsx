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
import { IconSearch } from '@douyinfe/semi-icons';
import {
  Button,
  Card,
  Empty,
  Input,
  InputNumber,
  Modal,
  Pagination,
  Popconfirm,
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
import { useActualTheme } from '../../context/Theme';
import {
  AUTO_REFRESH_INTERVAL_MS,
  buildExternalShopCategoryCards,
  filterExternalShopGoods,
  getExternalShopDescriptionDetail,
  getExternalShopDeliverySummary,
  getExternalShopFulfillmentLabel,
  getExternalShopPurchaseLimitLabel,
  getExternalShopStockLabel,
  getExternalShopStatusLabel,
  getAutoRefreshOrderTradeNos,
  parseExternalShopDeliveryCards,
} from './shopUtils';
import GptTeamPlanTab from './GptTeamPlanTab';
import { normalizeShopTabKey } from './gptTeamPlanUtils';

const { Text, Paragraph } = Typography;

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
  const actualTheme = useActualTheme();
  const isDark = actualTheme === 'dark';
  const navigate = useNavigate();
  const location = useLocation();
  const [loading, setLoading] = useState(true);
  const [goods, setGoods] = useState([]);
  const [goodsCategories, setGoodsCategories] = useState([]);
  const [orders, setOrders] = useState([]);
  const [shopStatus, setShopStatus] = useState({
    externalShopEnabled: false,
    externalShopReady: false,
    gptTeamEnabled: false,
    gptTeamRemainingSeats: null,
  });
  const [shopStatusLoaded, setShopStatusLoaded] = useState(false);
  const [shopStatusError, setShopStatusError] = useState(null);
  const [activeTab, setActiveTab] = useState('goods');
  const [orderTotal, setOrderTotal] = useState(0);
  const [createLoading, setCreateLoading] = useState(false);
  const [orderLoading, setOrderLoading] = useState(false);
  const [selectedGood, setSelectedGood] = useState(null);
  const [selectedDetailGood, setSelectedDetailGood] = useState(null);
  const [contact, setContact] = useState('');
  const [orderQuantity, setOrderQuantity] = useState(1);
  const [paymentChannels, setPaymentChannels] = useState([]);
  const [selectedChannelId, setSelectedChannelId] = useState(0);
  const [revealedDeliveries, setRevealedDeliveries] = useState({});
  const [activeGoodsCategory, setActiveGoodsCategory] = useState('all');
  const [goodsSearch, setGoodsSearch] = useState('');
  const [goodsSort, setGoodsSort] = useState('default');
  const [goodsPage, setGoodsPage] = useState(1);
  const [goodsPageSize, setGoodsPageSize] = useState(4);
  const [orderPage, setOrderPage] = useState(1);
  const [orderPageSize, setOrderPageSize] = useState(10);
  const [orderSort, setOrderSort] = useState('created_at:desc');

  const loadGoods = async (options = {}) => {
    const { refresh = false, silent = false } = options;
    const url = refresh
      ? '/api/external-shop/goods?refresh=1'
      : '/api/external-shop/goods';
    const res = await API.get(url, silent ? { skipErrorHandler: true } : undefined);
    if (!res.data.success) {
      throw new Error(res.data.message || t('获取商品失败'));
    }
    setGoods(res.data.data || []);
  };

  const loadGoodsCategories = async (options = {}) => {
    const { refresh = false, silent = false } = options;
    const url = refresh
      ? '/api/external-shop/categories?refresh=1'
      : '/api/external-shop/categories';
    const res = await API.get(url, silent ? { skipErrorHandler: true } : undefined);
    if (!res.data.success) {
      throw new Error(res.data.message || t('获取商品分类失败'));
    }
    setGoodsCategories(res.data.data || []);
  };

  const loadShopStatus = async () => {
    const res = await API.get('/api/external-shop/status', {
      skipErrorHandler: true,
    });
    if (!res?.data?.success) {
      throw new Error(res?.data?.message || t('获取商城状态失败'));
    }
    const nextStatus = {
      externalShopEnabled: Boolean(res.data.data?.external_shop?.enabled),
      externalShopReady: Boolean(res.data.data?.external_shop?.ready),
      gptTeamEnabled: Boolean(res.data.data?.gptteamplan?.enabled),
      gptTeamRemainingSeats:
        res.data.data?.gptteamplan?.remaining_seats === null ||
        res.data.data?.gptteamplan?.remaining_seats === undefined
          ? null
          : Number(res.data.data.gptteamplan.remaining_seats),
    };
    setShopStatusError(null);
    setShopStatus(nextStatus);
    setShopStatusLoaded(true);
    return nextStatus;
  };

  const loadPaymentChannels = async (options = {}) => {
    const { refresh = false, silent = false } = options;
    const url = refresh
      ? '/api/external-shop/channels?refresh=1'
      : '/api/external-shop/channels';
    const res = await API.get(url, silent ? { skipErrorHandler: true } : undefined);
    if (!res.data.success) {
      throw new Error(res.data.message || t('获取支付方式失败'));
    }
    setPaymentChannels(res.data.data || []);
  };

  const loadOrders = async (options = {}) => {
    const {
      silent = false,
      page = orderPage,
      pageSize = orderPageSize,
      sort = orderSort,
      refresh = false,
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
      if (refresh) {
        params.set('refresh', '1');
      }
      const res = await API.get(
        `/api/external-shop/orders?${params.toString()}`,
        silent ? { skipErrorHandler: true } : undefined,
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
        const nextShopStatus = await loadShopStatus();
        if (nextShopStatus.externalShopReady) {
          await Promise.all([
            loadGoods(),
            loadGoodsCategories(),
            loadPaymentChannels(),
          ]);
        } else {
          setGoods([]);
          setGoodsCategories([]);
          setOrders([]);
          setOrderTotal(0);
          setPaymentChannels([]);
        }
      } catch (error) {
        setShopStatusError(error);
        setShopStatusLoaded(true);
        setShopStatus({
          externalShopEnabled: false,
          externalShopReady: false,
          gptTeamEnabled: false,
          gptTeamRemainingSeats: null,
        });
        setGoodsCategories([]);
        if (error?.config?.skipErrorHandler) {
          showError(error);
        } else if (error?.name !== 'AxiosError') {
          showError(error.message || t('加载失败'));
        }
      } finally {
        setLoading(false);
      }
    };
    load();
  }, []);

  useEffect(() => {
    if (!shopStatusLoaded) {
      return;
    }
    const searchParams = new URLSearchParams(location.search);
    const nextTab = normalizeShopTabKey(
      searchParams.get('tab'),
      shopStatus.externalShopReady,
      shopStatus.gptTeamEnabled,
    );
    setActiveTab(nextTab);
    if (searchParams.get('tab') !== nextTab) {
      navigate(`${location.pathname}?tab=${nextTab}`, { replace: true });
    }
  }, [
    location.pathname,
    location.search,
    navigate,
    shopStatus.externalShopReady,
    shopStatus.gptTeamEnabled,
    shopStatusLoaded,
  ]);

  useEffect(() => {
    if (!shopStatusLoaded || !shopStatus.gptTeamEnabled) {
      return;
    }
    let mounted = true;
    const refreshGPTTeamSeats = async () => {
      try {
        const res = await API.get('/api/gptteamplan/status?refresh=1', {
          skipErrorHandler: true,
          disableDuplicate: true,
        });
        if (!mounted || !res?.data?.success) {
          return;
        }
        const nextRemainingSeats =
          res.data.data?.remaining_seats === null ||
          res.data.data?.remaining_seats === undefined
            ? null
            : Number(res.data.data.remaining_seats);
        setShopStatus((current) => ({
          ...current,
          gptTeamRemainingSeats: nextRemainingSeats,
        }));
      } catch {
        // Keep the cached value on screen when the background refresh fails.
      }
    };
    refreshGPTTeamSeats();
    return () => {
      mounted = false;
    };
  }, [shopStatusLoaded, shopStatus.gptTeamEnabled]);

  useEffect(() => {
    if (!shopStatusLoaded || !shopStatus.externalShopReady) {
      return;
    }
    let mounted = true;
    const refreshCatalog = async () => {
      try {
        await Promise.all([
          loadGoods({ refresh: true, silent: true }),
          loadGoodsCategories({ refresh: true, silent: true }),
        ]);
      } catch {
        if (!mounted) {
          return;
        }
      }
    };
    refreshCatalog();
    return () => {
      mounted = false;
    };
  }, [shopStatusLoaded, shopStatus.externalShopReady]);

  useEffect(() => {
    if (!shopStatusLoaded || !shopStatus.externalShopReady) {
      return;
    }
    let mounted = true;
    const refreshChannels = async () => {
      try {
        await loadPaymentChannels({ refresh: true, silent: true });
      } catch {
        if (!mounted) {
          return;
        }
      }
    };
    refreshChannels();
    return () => {
      mounted = false;
    };
  }, [shopStatusLoaded, shopStatus.externalShopReady]);

  useEffect(() => {
    if (loading || !shopStatus.externalShopReady) {
      return;
    }
    let mounted = true;
    const refreshOrders = async () => {
      try {
        await loadOrders({
          page: orderPage,
          pageSize: orderPageSize,
          sort: orderSort,
        });
      } catch (error) {
        if (!mounted) {
          return;
        }
        showError(error.message || t('获取订单失败'));
        return;
      }

      try {
        await loadOrders({
          silent: true,
          page: orderPage,
          pageSize: orderPageSize,
          sort: orderSort,
          refresh: true,
        });
      } catch {}
    };
    refreshOrders();
    return () => {
      mounted = false;
    };
  }, [
    loading,
    orderPage,
    orderPageSize,
    orderSort,
    shopStatus.externalShopReady,
    t,
  ]);

  useEffect(() => {
    if (!shopStatus.externalShopReady) {
      return undefined;
    }
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
  }, [
    orderPage,
    orderPageSize,
    orderSort,
    orders,
    shopStatus.externalShopReady,
  ]);

  useEffect(() => {
    if (paymentChannels.length === 0) {
      setSelectedChannelId(0);
      return;
    }
    setSelectedChannelId((current) =>
      paymentChannels.some((channel) => channel.id === current)
        ? current
        : paymentChannels[0].id,
    );
  }, [paymentChannels]);

  const openOrderModal = (good) => {
    setSelectedGood(good);
    setContact('');
    setOrderQuantity(1);
  };

  const getNormalizedOrderQuantity = (quantity, good) => {
    const parsed = Number.parseInt(String(quantity ?? 1), 10);
    const minQuantity = 1;
    const parsedStockCount = Number.parseInt(String(good?.stock_count ?? 1), 10);
    const maxQuantity =
      Number.isFinite(parsedStockCount) && parsedStockCount >= minQuantity
        ? parsedStockCount
        : minQuantity;
    let nextQuantity = Number.isFinite(parsed) ? parsed : minQuantity;
    if (nextQuantity < minQuantity) {
      nextQuantity = minQuantity;
    }
    if (nextQuantity > maxQuantity) {
      nextQuantity = maxQuantity;
    }
    return nextQuantity;
  };

  const createOrder = async () => {
    if (!selectedGood) return;
    if (!contact.trim()) {
      showError(t('请输入联系方式'));
      return;
    }
    const quantity = getNormalizedOrderQuantity(orderQuantity, selectedGood);
    setCreateLoading(true);
    try {
      const res = await API.post('/api/external-shop/orders', {
        goods_key: selectedGood.goods_key,
        quantity,
        contact: contact.trim(),
        channel_id: selectedChannelId || 0,
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
        const nextPath = buildExternalShopOrderPayPath(data.local_trade_no);
        navigate(nextPath);
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
        null,
        { skipErrorHandler: true },
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

  const deleteOrder = async (record) => {
    const nextPage =
      orderPage > 1 && orders.length === 1 ? orderPage - 1 : orderPage;
    try {
      const res = await API.delete(`/api/external-shop/orders/${record.local_trade_no}`);
      if (!res.data.success) {
        throw new Error(res.data.message || t('删除订单失败'));
      }
      showSuccess(t('订单已删除'));
      if (nextPage !== orderPage) {
        setOrderPage(nextPage);
      }
      await loadOrders({
        page: nextPage,
        pageSize: orderPageSize,
        sort: orderSort,
      });
    } catch (error) {
      showError(error.message || t('删除订单失败'));
    }
  };

  const toggleDeliveryVisibility = (localTradeNo) => {
    setRevealedDeliveries((current) => ({
      ...current,
      [localTradeNo]: !current[localTradeNo],
    }));
  };

  const goodsCategoryCards = useMemo(
    () => buildExternalShopCategoryCards(goods, goodsCategories),
    [goods, goodsCategories],
  );

  useEffect(() => {
    if (activeGoodsCategory === 'all') {
      return;
    }
    if (
      goodsCategoryCards.some((category) => category.key === activeGoodsCategory)
    ) {
      return;
    }
    setActiveGoodsCategory('all');
  }, [activeGoodsCategory, goodsCategoryCards]);

  const filteredGoods = useMemo(
    () =>
      filterExternalShopGoods(goods, {
        categoryKey: activeGoodsCategory,
        searchTerm: goodsSearch,
      }),
    [activeGoodsCategory, goods, goodsSearch],
  );

  const sortedGoods = useMemo(() => {
    const nextGoods = [...filteredGoods];
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
  }, [filteredGoods, goodsSort]);

  const paginatedGoods = useMemo(() => {
    const start = (goodsPage - 1) * goodsPageSize;
    return sortedGoods.slice(start, start + goodsPageSize);
  }, [goodsPage, goodsPageSize, sortedGoods]);

  const selectedDetailDescription = useMemo(
    () => getExternalShopDescriptionDetail(selectedDetailGood?.description),
    [selectedDetailGood],
  );

  const selectedGoodFullDescription = useMemo(
    () => getExternalShopDescriptionDetail(selectedGood?.description),
    [selectedGood],
  );

  const normalizedSelectedGoodQuantity = useMemo(
    () => getNormalizedOrderQuantity(orderQuantity, selectedGood),
    [orderQuantity, selectedGood],
  );

  const selectedGoodTotalPrice = useMemo(
    () => Number(selectedGood?.price || 0) * normalizedSelectedGoodQuantity,
    [normalizedSelectedGoodQuantity, selectedGood],
  );

  const selectedGoodTotalMarketPrice = useMemo(
    () =>
      Number(selectedGood?.market_price || 0) * normalizedSelectedGoodQuantity,
    [normalizedSelectedGoodQuantity, selectedGood],
  );

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
            <Popconfirm
              title={t('确定删除这个订单记录吗？')}
              content={t('删除后将无法在列表中查看该订单。')}
              okText={t('删除')}
              cancelText={t('取消')}
              okType='danger'
              onConfirm={() => deleteOrder(record)}
            >
              <Button size='small' type='danger' theme='borderless'>
                {t('删除')}
              </Button>
            </Popconfirm>
          </Space>
        ),
      },
    ],
    [navigate, orderPage, orderPageSize, orderSort, orders.length, revealedDeliveries, t],
  );

  if (loading) {
    return (
      <div className='p-6 flex justify-center'>
        <Spin size='large' />
      </div>
    );
  }

  const renderExternalShopContent = () => (
    <div className='space-y-6'>
      {shopStatusError ? (
        <Card className='!rounded-2xl shadow-sm border-0'>
          <Empty
            description={
              <Space vertical align='center' spacing='tight'>
                <Text strong>{t('商城状态加载失败')}</Text>
                <Text type='tertiary'>
                  {t('请刷新后重试，若持续失败请检查服务日志。')}
                </Text>
              </Space>
            }
          />
        </Card>
      ) : null}

      {shopStatusLoaded && !shopStatusError && !shopStatus.externalShopReady ? (
        <Card className='!rounded-2xl shadow-sm border-0'>
          <Empty
            description={
              <Space vertical align='center' spacing='tight'>
                <Text strong>{t('商城暂未开放')}</Text>
                <Text type='tertiary'>
                  {t('当前站点尚未完成商城配置，请联系管理员。')}
                </Text>
              </Space>
            }
          />
        </Card>
      ) : null}

      {!shopStatusError && shopStatus.externalShopReady ? (
        <>
          <Card
            className='!rounded-[28px] border-0 shadow-sm overflow-hidden'
            bodyStyle={{ padding: 0 }}
          >
            <div
              className='relative overflow-hidden rounded-[28px] p-4 sm:p-6'
              style={{
                background: isDark
                  ? 'radial-gradient(circle at top left, rgba(37,99,235,0.18), transparent 28%), radial-gradient(circle at bottom right, rgba(8,145,178,0.18), transparent 26%), linear-gradient(180deg, rgba(2,6,23,0.96) 0%, rgba(15,23,42,0.96) 100%)'
                  : 'radial-gradient(circle at top left, rgba(59,130,246,0.14), transparent 28%), radial-gradient(circle at bottom right, rgba(125,211,252,0.16), transparent 26%), linear-gradient(180deg, #f8fbff 0%, #ffffff 100%)',
              }}
            >
              <div
                className='relative space-y-5 rounded-[24px] p-4 backdrop-blur sm:p-6'
                style={{
                  background: isDark
                    ? 'rgba(2, 6, 23, 0.72)'
                    : 'rgba(255, 255, 255, 0.9)',
                  boxShadow: isDark
                    ? '0 24px 48px rgba(2, 6, 23, 0.45)'
                    : '0 18px 40px rgba(148, 163, 184, 0.12)',
                }}
              >
                <div className='flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between'>
                  <div>
                    <Text
                      strong
                      style={{
                        fontSize: 15,
                        color: 'var(--semi-color-text-0)',
                      }}
                    >
                      {t('选择商品')}
                    </Text>
                  </div>
                  <div className='flex w-full flex-col gap-3 sm:flex-row xl:w-auto'>
                    <Input
                      value={goodsSearch}
                      onChange={(value) => {
                        setGoodsSearch(value);
                        setGoodsPage(1);
                      }}
                      prefix={<IconSearch />}
                      showClear
                      placeholder={t('搜索商品')}
                      style={{ width: '100%', maxWidth: 280 }}
                    />
                    <Select
                      value={goodsSort}
                      onChange={(value) => {
                        setGoodsSort(value);
                        setGoodsPage(1);
                      }}
                      style={{ width: 180 }}
                    >
                      <Select.Option value='default'>
                        {t('默认排序')}
                      </Select.Option>
                      <Select.Option value='price:asc'>
                        {t('价格从低到高')}
                      </Select.Option>
                      <Select.Option value='price:desc'>
                        {t('价格从高到低')}
                      </Select.Option>
                      <Select.Option value='stock:desc'>
                        {t('库存从高到低')}
                      </Select.Option>
                      <Select.Option value='name:asc'>
                        {t('名称排序')}
                      </Select.Option>
                    </Select>
                  </div>
                </div>

                <div className='flex gap-3 overflow-x-auto pb-1'>
                  {goodsCategoryCards.map((category) => {
                    const selected = activeGoodsCategory === category.key;

                    return (
                      <button
                        key={category.key}
                        type='button'
                        className={`min-w-[170px] rounded-2xl border px-4 py-4 text-left transition ${
                          selected
                            ? 'border-blue-400 bg-[linear-gradient(135deg,_#3b82f6_0%,_#2563eb_100%)] text-white shadow-[0_16px_30px_rgba(37,99,235,0.28)] dark:border-cyan-400/60 dark:bg-[linear-gradient(135deg,_rgba(37,99,235,0.92)_0%,_rgba(8,145,178,0.88)_100%)] dark:shadow-[0_16px_36px_rgba(8,145,178,0.2)]'
                            : 'border-slate-200 bg-white text-slate-700 shadow-sm hover:border-blue-200 hover:bg-slate-50 dark:border-slate-700 dark:bg-slate-900/95 dark:text-slate-100 dark:hover:border-cyan-500/50 dark:hover:bg-slate-900'
                        }`}
                        style={
                          selected
                            ? undefined
                            : {
                                borderColor: isDark
                                  ? 'rgba(71, 85, 105, 0.9)'
                                  : undefined,
                                background: isDark
                                  ? 'rgba(15, 23, 42, 0.92)'
                                  : undefined,
                                color: isDark ? '#e2e8f0' : undefined,
                              }
                        }
                        onClick={() => {
                          setActiveGoodsCategory(category.key);
                          setGoodsPage(1);
                        }}
                      >
                        <div className='text-sm font-semibold leading-5'>
                          {category.key === 'all'
                            ? t(category.name)
                            : category.name}
                        </div>
                        <div
                          className={`mt-1 text-xs leading-5 ${
                            selected
                              ? 'text-blue-50 dark:text-cyan-50'
                              : 'text-slate-400 dark:text-slate-400'
                          }`}
                        >
                          {`${category.count}${t('种商品')}`}
                        </div>
                      </button>
                    );
                  })}
                </div>

                <div>
                  <Text
                    strong
                    style={{ fontSize: 15, color: 'var(--semi-color-text-1)' }}
                  >
                    {t('选择商品')}
                  </Text>

                  <div className='mt-4 sm:mt-5'>
                    <div className='grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 gap-5'>
                      {paginatedGoods.map((good) => (
                        <div
                          key={good.goods_key}
                          className='group flex h-full cursor-pointer flex-col overflow-hidden rounded-[22px] bg-white shadow-[0_14px_32px_rgba(148,163,184,0.12)] transition duration-200 hover:-translate-y-1 hover:shadow-[0_20px_40px_rgba(59,130,246,0.18)] dark:bg-slate-950 dark:shadow-[0_16px_34px_rgba(2,6,23,0.3)] dark:hover:shadow-[0_22px_42px_rgba(8,145,178,0.18)]'
                          style={{
                            background: isDark
                              ? 'linear-gradient(180deg, rgba(15, 23, 42, 0.98) 0%, rgba(2, 6, 23, 0.98) 100%)'
                              : '#ffffff',
                            boxShadow: isDark
                              ? '0 18px 40px rgba(2, 6, 23, 0.32)'
                              : '0 18px 40px rgba(148, 163, 184, 0.12)',
                          }}
                          onClick={() => setSelectedDetailGood(good)}
                        >
                          <div className='relative h-48 overflow-hidden bg-[linear-gradient(180deg,_#f8fafc_0%,_#e2e8f0_100%)] dark:bg-[linear-gradient(180deg,_rgba(15,23,42,1)_0%,_rgba(30,41,59,1)_100%)]'>
                            {good.image ? (
                              <img
                                src={good.image}
                                alt={good.name}
                                className='h-48 w-full object-cover transition duration-300 group-hover:scale-[1.03]'
                              />
                            ) : (
                              <div className='flex h-full w-full items-center justify-center bg-[linear-gradient(135deg,_#2563eb_0%,_#60a5fa_60%,_#bfdbfe_100%)] p-6 text-center text-white dark:bg-[linear-gradient(135deg,_rgba(30,64,175,0.96)_0%,_rgba(14,116,144,0.92)_60%,_rgba(15,23,42,0.88)_100%)]'>
                                <div>
                                  <div className='text-xs font-medium leading-5 tracking-[0.08em] text-blue-100/90'>
                                    {good.category_name || t('商品')}
                                  </div>
                                  <div className='mt-3 text-base font-semibold leading-6 sm:text-lg sm:leading-7'>
                                    {good.name}
                                  </div>
                                </div>
                              </div>
                            )}
                            <div className='absolute left-3 top-3 flex gap-2'>
                              {good.category_name ? (
                                <Tag color='blue' size='small'>
                                  {good.category_name}
                                </Tag>
                              ) : null}
                              {Number(good.stock_count || 0) <= 0 ? (
                                <Tag color='red' size='small'>
                                  {t('缺货')}
                                </Tag>
                              ) : null}
                            </div>
                          </div>

                          <div className='flex flex-1 flex-col gap-3 p-3.5'>
                            <div>
                              <div className='line-clamp-2 text-sm font-semibold leading-6 text-slate-900 dark:text-slate-100'>
                                {good.name}
                              </div>
                            </div>

                            <div className='mt-auto space-y-2.5'>
                              <div className='flex items-end justify-between gap-3'>
                                <div className='flex items-baseline gap-2'>
                                  <Text
                                    strong
                                    style={{
                                      fontSize: 18,
                                      lineHeight: '22px',
                                      color: 'var(--semi-color-primary)',
                                    }}
                                  >
                                    {`￥${Number(good.price || 0).toFixed(2)}`}
                                  </Text>
                                  {Number(good.market_price || 0) >
                                  Number(good.price || 0) ? (
                                    <Text
                                      type='quaternary'
                                      className='dark:!text-slate-500'
                                      delete
                                    >{`￥${Number(good.market_price || 0).toFixed(2)}`}</Text>
                                  ) : null}
                                </div>
                                <Tag
                                  color={
                                    Number(good.stock_count || 0) <= 0
                                      ? 'red'
                                      : Number(good.stock_count || 0) <= 5
                                        ? 'orange'
                                        : 'blue'
                                  }
                                >
                                  {t(getExternalShopStockLabel(good))}
                                </Tag>
                              </div>

                              <Text
                                type='tertiary'
                                className='dark:!text-slate-400'
                              >{`${t('库存')}: ${good.stock_count}`}</Text>

                              <Button
                                block
                                theme='solid'
                                type='primary'
                                disabled={Number(good.stock_count || 0) <= 0}
                                onClick={(event) => {
                                  event.stopPropagation();
                                  openOrderModal(good);
                                }}
                              >
                                {Number(good.stock_count || 0) <= 0
                                  ? t('暂时缺货')
                                  : t('立即购买')}
                              </Button>
                            </div>
                          </div>
                        </div>
                      ))}
                    </div>

                    {sortedGoods.length === 0 ? (
                      <Empty
                        description={
                          goodsSearch || activeGoodsCategory !== 'all'
                            ? t('没有找到匹配的商品')
                            : t('暂无商品')
                        }
                      />
                    ) : null}

                    {sortedGoods.length > 0 ? (
                      <div className='mt-5 flex justify-center lg:justify-end'>
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
                  </div>
                </div>
              </div>
            </div>
          </Card>

          <Card
            className='!rounded-2xl shadow-sm border-0'
            title={t('我的订单')}
          >
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
                <Select.Option value='status:asc'>
                  {t('状态排序')}
                </Select.Option>
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
            title={t('订单确认')}
            visible={!!selectedGood}
            onCancel={() => setSelectedGood(null)}
            footer={null}
            width={480}
          >
            <div className='space-y-4'>
              <div className='space-y-1'>
                <Text strong style={{ fontSize: 16 }}>
                  {selectedGood?.name || '-'}
                </Text>
                <Text type='tertiary'>{`x${normalizedSelectedGoodQuantity}`}</Text>
              </div>

              <Paragraph
                style={{
                  width: '100%',
                  marginBottom: 0,
                  maxHeight: '32vh',
                  overflowY: 'auto',
                  whiteSpace: 'pre-wrap',
                  lineHeight: 1.75,
                }}
              >
                {selectedGoodFullDescription || t('暂无商品详情')}
              </Paragraph>

              <div className='flex flex-wrap gap-2'>
                {selectedGood ? (
                  <Tag color='green'>
                    {t(getExternalShopFulfillmentLabel(selectedGood))}
                  </Tag>
                ) : null}
                {selectedGood ? (
                  <Tag color='blue'>
                    {t(getExternalShopPurchaseLimitLabel(selectedGood))}
                  </Tag>
                ) : null}
                {selectedGood?.category_name ? (
                  <Tag color='grey'>{selectedGood.category_name}</Tag>
                ) : null}
              </div>

              <div className='space-y-1.5'>
                <Text strong>
                  {t('联系方式')} <Text type='danger'>*</Text>
                </Text>
                <Input
                  value={contact}
                  onChange={setContact}
                  placeholder={t('请输入联系方式方便查询订单')}
                />
                <Text type='tertiary'>
                  {t('【建议】填写手机号，如填邮箱，卡密也会同步发送至邮箱。')}
                </Text>
              </div>

              <div className='flex flex-col gap-4 rounded-2xl border border-slate-200/80 bg-slate-50/80 p-4 dark:border-slate-800 dark:bg-slate-900/70 sm:flex-row sm:items-end sm:justify-between'>
                <div className='space-y-1.5'>
                  <Text strong>{t('商品价格')}</Text>
                  <div className='flex items-baseline gap-2'>
                    <Text
                      strong
                      style={{
                        fontSize: 20,
                        lineHeight: '24px',
                        color: 'var(--semi-color-primary)',
                      }}
                    >
                      {`￥${selectedGoodTotalPrice.toFixed(2)}`}
                    </Text>
                    {selectedGoodTotalMarketPrice > selectedGoodTotalPrice ? (
                      <Text type='quaternary' delete>{`￥${Number(
                        selectedGoodTotalMarketPrice,
                      ).toFixed(2)}`}</Text>
                    ) : null}
                  </div>
                  <Text type='tertiary'>
                    {`${t('单价')}: ￥${Number(selectedGood?.price || 0).toFixed(2)}`}
                  </Text>
                </div>

                <div className='space-y-1.5'>
                  <Text strong>{t('购买数量')}</Text>
                  <InputNumber
                    min={1}
                    max={Number(selectedGood?.stock_count || 1)}
                    step={1}
                    precision={0}
                    value={normalizedSelectedGoodQuantity}
                    onChange={(value) =>
                      setOrderQuantity(
                        getNormalizedOrderQuantity(value, selectedGood),
                      )
                    }
                    style={{ width: 160 }}
                  />
                </div>
              </div>

              <div className='space-y-3'>
                <Text strong>{t('支付方式')}</Text>
                <div className='grid gap-3 sm:grid-cols-2'>
                  {paymentChannels.map((channel) => {
                    const active = selectedChannelId === channel.id;

                    return (
                      <button
                        key={channel.id}
                        type='button'
                        className={`rounded-2xl border px-4 py-3 text-left transition ${
                          active
                            ? 'border-blue-500 bg-blue-50 text-blue-700 dark:border-cyan-400 dark:bg-cyan-500/10 dark:text-cyan-100'
                            : 'border-slate-200 bg-white text-slate-700 hover:border-blue-200 dark:border-slate-700 dark:bg-slate-900/80 dark:text-slate-200'
                        }`}
                        onClick={() => setSelectedChannelId(channel.id)}
                      >
                        <div className='flex items-center justify-between gap-3'>
                          <div>
                            <div className='text-sm font-semibold'>
                              {channel.show_name || channel.name}
                            </div>
                            <div className='mt-1 text-xs text-slate-400 dark:text-slate-500'>
                              {channel.paytype?.name || t('支付页确认')}
                            </div>
                          </div>
                          {channel.paytype?.icon ? (
                            <img
                              src={channel.paytype.icon}
                              alt={channel.show_name || channel.name}
                              className='h-7 w-7 rounded-full object-contain'
                            />
                          ) : null}
                        </div>
                      </button>
                    );
                  })}
                </div>
                {paymentChannels.length === 0 ? (
                  <Text type='tertiary'>{t('支付方式将在下一步确认')}</Text>
                ) : null}
              </div>

              <div className='flex justify-end gap-3 pt-3'>
                <Button onClick={() => setSelectedGood(null)}>
                  {t('取消')}
                </Button>
                <Button
                  theme='solid'
                  type='primary'
                  loading={createLoading}
                  disabled={
                    Number(selectedGood?.stock_count || 0) <= 0 ||
                    !contact.trim()
                  }
                  onClick={createOrder}
                >
                  {t('去支付')}
                </Button>
              </div>
            </div>
          </Modal>
        </>
      ) : null}

      <Modal
        title={selectedDetailGood?.name || t('商品详情')}
        visible={!!selectedDetailGood}
        onCancel={() => setSelectedDetailGood(null)}
        footer={
          <Button onClick={() => setSelectedDetailGood(null)}>
            {t('关闭')}
          </Button>
        }
      >
        <Space vertical align='start' style={{ width: '100%' }}>
          {selectedDetailGood ? (
            <Text
              strong
            >{`￥${Number(selectedDetailGood.price || 0).toFixed(2)}`}</Text>
          ) : null}
          {selectedDetailGood?.category_name ? (
            <Tag color='blue'>{selectedDetailGood.category_name}</Tag>
          ) : null}
          <Paragraph
            style={{
              width: '100%',
              marginBottom: 0,
              maxHeight: '60vh',
              overflowY: 'auto',
              whiteSpace: 'pre-wrap',
            }}
          >
            {selectedDetailDescription || t('暂无商品详情')}
          </Paragraph>
        </Space>
      </Modal>
    </div>
  );

  const shopTabs = [];
  if (shopStatus.externalShopReady || !shopStatus.gptTeamEnabled) {
    shopTabs.push({
      itemKey: 'goods',
      tab: t('商品商城'),
      content: activeTab === 'goods' ? renderExternalShopContent() : null,
    });
  }
  if (shopStatus.gptTeamEnabled) {
    shopTabs.push({
      itemKey: 'gpt-team',
      tab: t('GPT Team 兑换'),
      content:
        activeTab === 'gpt-team' ? (
          <GptTeamPlanTab remainingSeats={shopStatus.gptTeamRemainingSeats} />
        ) : null,
    });
  }

  return (
    <div className='px-4 pb-6 pt-0 md:px-6 md:pb-6 md:pt-10 max-w-7xl mx-auto'>
      <Tabs
        type='card'
        activeKey={activeTab}
        onChange={(key) => {
          const nextTab = normalizeShopTabKey(
            key,
            shopStatus.externalShopReady,
            shopStatus.gptTeamEnabled,
          );
          setActiveTab(nextTab);
          navigate(`${location.pathname}?tab=${nextTab}`, { replace: true });
        }}
      >
        {shopTabs.map(({ itemKey, tab, content }) => (
          <TabPane key={itemKey} itemKey={itemKey} tab={tab}>
            {content}
          </TabPane>
        ))}
      </Tabs>
    </div>
  );
}
