/* Copyright (C) 2025 QuantumNous — Licensed under AGPL-3.0-or-later. */
import React, { useEffect, useReducer, useRef, useState } from 'react';
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
  invitationViewReducer,
  initialInvitationView,
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
  const [state, dispatch] = useReducer(
    invitationViewReducer,
    initialInvitationView,
  );
  const fence = useRef(createInvitationRequestFence());
  const identity = String(viewerId || '');
  const scope = JSON.stringify([identity, page, query]);
  const current = visible && state.scope === scope;
  const data = current ? state.data : null;
  const loading = current && state.loading;
  const error = current && state.error;

  useEffect(() => {
    setPage(1);
    setKeyword('');
    setQuery('');
  }, [visible, identity]);
  useEffect(() => {
    if (!visible || !viewerId) {
      fence.current.invalidate();
      dispatch({ type: 'clear' });
      return;
    }
    let inFlight = false;
    let abort;
    const load = () => {
      // Slow requests must finish instead of being aborted by the next poll.
      if (inFlight) return;
      inFlight = true;
      const ticket = fence.current.begin(scope);
      abort = new AbortController();
      dispatch({ type: 'start', scope });
      API.get('/api/user/self/invitees', {
        disableDuplicate: true,
        timeout: 10000,
        headers: { 'New-API-User': viewerId },
        params: { p: page, page_size: 20, keyword: query },
        signal: abort.signal,
      })
        .then((response) => {
          const data = decodeInvitationPage(response.data);
          if (fence.current.accept(ticket))
            dispatch({ type: 'success', scope, data });
        })
        .catch(() => {
          if (fence.current.accept(ticket))
            dispatch({ type: 'failure', scope });
        })
        .finally(() => {
          inFlight = false;
        });
    };
    load();
    const timer = setInterval(load, 15000);
    return () => {
      clearInterval(timer);
      fence.current.invalidate();
      abort?.abort();
    };
  }, [visible, viewerId, scope, page, query, refresh]);

  const money = (value) => (
    <Space vertical align='start'>
      {value.currencies.length === 0 && (
        <Typography.Text type='tertiary'>—</Typography.Text>
      )}
      {value.currencies.map((c) => (
        <Typography.Text key={`${c.currency}/${c.currency_symbol}`}>
          {invitationMoney(c.total, c.currency_symbol || c.currency)}
          {c.estimated ? ` (${t('含估算')})` : ''}
        </Typography.Text>
      ))}
      {(value.unknown_count > 0 || value.unpriced_count > 0) && (
        <Typography.Text type='warning'>
          {t('部分金额未确认，合计可能不完整')}
        </Typography.Text>
      )}
    </Space>
  );
  const columns = [
    {
      title: t('用户名'),
      dataIndex: 'username',
      width: data?.can_view_amounts ? '45%' : undefined,
      shouldCellUpdate: (next, previous) =>
        next.record !== previous.record ||
        next.column.title !== previous.column.title,
      render: (username) => <span className='break-all'>{username}</span>,
    },
  ];
  if (data?.can_view_amounts)
    columns.push({
      title: t('累计总金额'),
      shouldCellUpdate: (next, previous) =>
        next.record !== previous.record ||
        next.column.title !== previous.column.title,
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
        {data?.can_view_amounts && (
          <Typography.Text type='tertiary'>
            {t(
              '合计包含兑换折算；未确认或已清除且无快照的历史记录可能未计入，未扣除退款。',
            )}
          </Typography.Text>
        )}
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
