/* Copyright (C) 2025 QuantumNous — Licensed under AGPL-3.0-or-later. */
import React, { useEffect, useRef, useState } from 'react';
import {
  Banner,
  Button,
  Input,
  Modal,
  Pagination,
  Space,
  Spin,
  Table,
  Typography,
} from '@douyinfe/semi-ui';
import { API } from '../../../helpers';
import {
  createInvitationRequestFence,
  decodeInvitationPage,
  invitationMoney,
} from '../utils/invitationDetails';

export default function InvitationUsersModal({
  visible,
  onCancel,
  viewerId,
  t,
}) {
  const [page, setPage] = useState(1);
  const [keyword, setKeyword] = useState('');
  const [query, setQuery] = useState('');
  const [refresh, setRefresh] = useState(0);
  const [state, setState] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(false);
  const fence = useRef(createInvitationRequestFence());
  const identity = String(viewerId || '');
  const data = visible && state?.identity === identity ? state.data : null;

  useEffect(() => {
    setPage(1);
    setKeyword('');
    setQuery('');
  }, [visible, identity]);
  useEffect(() => {
    if (!visible || !viewerId) {
      fence.current.invalidate();
      setState(null);
      return;
    }
    const ticket = fence.current.begin(`${identity}/${page}/${query}`);
    const abort = new AbortController();
    setState(null);
    setError(false);
    setLoading(true);
    API.get('/api/user/self/invitees', {
      disableDuplicate: true,
      headers: { 'New-API-User': viewerId },
      params: { p: page, page_size: 20, keyword: query },
      signal: abort.signal,
    })
      .then((response) => {
        const decoded = decodeInvitationPage(response.data);
        if (fence.current.accept(ticket)) setState({ identity, data: decoded });
      })
      .catch(() => {
        if (fence.current.accept(ticket)) {
          setState(null);
          setError(true);
        }
      })
      .finally(() => {
        if (fence.current.accept(ticket)) setLoading(false);
      });
    return () => {
      fence.current.invalidate();
      abort.abort();
    };
  }, [visible, identity, viewerId, page, query, refresh]);
  useEffect(() => {
    if (!visible) return;
    // Refresh the authorization with the data, rather than retaining a stale grant.
    const timer = setInterval(() => setRefresh((v) => v + 1), 15000);
    return () => clearInterval(timer);
  }, [visible]);

  const money = (value) => (
    <Space vertical align='start' style={{ width: '100%' }}>
      {value.currencies.length === 0 && (
        <Typography.Text type='tertiary'>{t('暂无金额记录')}</Typography.Text>
      )}
      {value.currencies.map((c) => (
        <div
          key={`${c.currency}/${c.currency_symbol}`}
          className='grid grid-cols-1 sm:grid-cols-4 gap-3 w-full py-2'
        >
          {[
            [t('充值实付'), c.topup],
            [t('订阅实付'), c.subscription],
            [t('兑换折算'), c.redemption],
            [t('合计'), c.total],
          ].map(([label, n]) => (
            <div key={label}>
              <div className='text-xs opacity-60'>{label}</div>
              <div className='break-all'>
                {invitationMoney(n, c.currency_symbol || c.currency)}
              </div>
            </div>
          ))}
          {c.admin_topup && !/^0(?:\.0+)?$/.test(c.admin_topup) && (
            <Typography.Text
              type='tertiary'
              className='col-span-full sm:col-span-4'
            >
              {t('其中管理员充值')}：
              {invitationMoney(c.admin_topup, c.currency_symbol || c.currency)}
            </Typography.Text>
          )}
          {(c.conversions || []).map((f) => (
            <Typography.Text
              key={`${f.quota_per_unit}/${f.exchange_rate}/${f.estimated}`}
              type='tertiary'
              className='col-span-full sm:col-span-4'
            >
              {t(
                '折算基准：{{quota}} 额度/USD；1 USD = {{rate}} {{currency}}',
                {
                  quota: f.quota_per_unit,
                  rate: f.exchange_rate,
                  currency: c.currency_symbol || c.currency,
                },
              )}
              {f.estimated ? ` (${t('按当前配置估算')})` : ''}
            </Typography.Text>
          ))}
          {c.conversion_incomplete && (
            <Typography.Text
              type='warning'
              className='col-span-full sm:col-span-4'
            >
              {t('部分历史折算基准缺失或过多，请联系管理员核对')}
            </Typography.Text>
          )}
          {c.estimated && (
            <Typography.Text
              type='warning'
              className='col-span-full sm:col-span-4'
            >
              {t('部分兑换金额按当前配置估算')}
            </Typography.Text>
          )}
        </div>
      ))}
      {value.unknown_count > 0 && (
        <Typography.Text type='warning'>
          {t('实付或币种未确认记录')}：{value.unknown_count}
        </Typography.Text>
      )}
      {value.legacy_recorded.length > 0 && (
        <details>
          <summary>{t('历史记录金额（未确认）')}</summary>
          {value.legacy_recorded.map((r) => (
            <div key={`${r.category}/${r.amount}`}>
              <Typography.Text type='tertiary'>
                {r.category === 'topup' ? t('充值实付') : t('订阅实付')}：
                {r.amount} × {r.count}
              </Typography.Text>
            </div>
          ))}
        </details>
      )}
      {value.legacy_truncated && (
        <Typography.Text type='tertiary'>
          {t('仅展示前50种历史金额，未确认记录总数保持完整')}
        </Typography.Text>
      )}
      {value.unpriced_count > 0 && (
        <Typography.Text type='warning'>
          {t('无法按额度折算的兑换记录')}：{value.unpriced_count}
        </Typography.Text>
      )}
    </Space>
  );
  const columns = [
    {
      title: t('用户名'),
      dataIndex: 'username',
      width: data?.can_view_amounts ? '25%' : undefined,
      render: (username) => <span className='break-all'>{username}</span>,
    },
  ];
  if (data?.can_view_amounts)
    columns.push({
      title: t('充值、兑换与订阅金额'),
      dataIndex: 'amounts',
      render: (a) => (a ? money(a) : '—'),
    });
  return (
    <Modal
      title={t('邀请用户名单')}
      visible={visible}
      onCancel={onCancel}
      footer={null}
      width='min(1000px, 96vw)'
      centered
      bodyStyle={{ maxHeight: 'calc(100vh - 180px)', overflowY: 'auto' }}
    >
      <Space vertical align='start' style={{ width: '100%' }}>
        <Space wrap>
          <Input
            aria-label={t('搜索受邀用户名')}
            placeholder={t('搜索受邀用户名')}
            value={keyword}
            maxLength={100}
            onChange={setKeyword}
            onEnterPress={() => {
              setQuery(keyword.trim());
              setPage(1);
            }}
          />
          <Button
            onClick={() => {
              setQuery(keyword.trim());
              setPage(1);
            }}
          >
            {t('搜索')}
          </Button>
          <Button onClick={() => setRefresh((v) => v + 1)}>{t('刷新')}</Button>
        </Space>
        <Typography.Text type='tertiary'>
          {t(
            '仅显示您直接邀请且当前可查看的用户，历史邀请计数可能包含已注销用户',
          )}
        </Typography.Text>
        {data?.can_view_amounts && (
          <Banner
            type='info'
            closeIcon={null}
            description={t(
              '按币种分别汇总：充值实付＋订阅实付＋兑换折算；未确认记录不计入实付合计，兑换折算不代表现金付款。',
            )}
          />
        )}
        {error && (
          <Banner
            type='danger'
            closeIcon={null}
            description={
              <Space>
                {t('无法读取邀请名单，请重试')}
                <Button onClick={() => setRefresh((v) => v + 1)}>
                  {t('重试')}
                </Button>
              </Space>
            }
          />
        )}
        <Spin spinning={loading} style={{ width: '100%' }}>
          <Table
            rowKey='username'
            columns={columns}
            dataSource={data?.items || []}
            pagination={false}
            empty={t('暂无受邀用户')}
            size='small'
          />
        </Spin>
        {data && (
          <Pagination
            total={data.total}
            currentPage={page}
            pageSize={20}
            onPageChange={setPage}
            showTotal
          />
        )}
      </Space>
    </Modal>
  );
}
