/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import React, { useState, useMemo } from 'react';
import {
  Card,
  Input,
  Button,
  Banner,
  Typography,
  Tag,
  Spin,
  Table,
  Empty,
} from '@douyinfe/semi-ui';
import { IconSearch, IconKey } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import { API } from '../../helpers';

const { Title, Text } = Typography;

// 令牌状态常量
const TokenStatus = {
  ENABLED: 1,
  DISABLED: 2,
  EXPIRED: 3,
  EXHAUSTED: 4,
};

const Query = () => {
  const { t } = useTranslation();
  const [tokenKey, setTokenKey] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [tokenInfo, setTokenInfo] = useState(null);
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize] = useState(10);
  const [logsTotal, setLogsTotal] = useState(0);
  const [logsLoading, setLogsLoading] = useState(false);
  
  // 模型列表相关状态
  const [modelSearch, setModelSearch] = useState('');
  const [modelPage, setModelPage] = useState(1);
  const [modelPageSize] = useState(20);

  const fetchData = async (key, page = 1) => {
    try {
      const res = await API.post('/api/token/query', { 
        key: key.trim(),
        page: page,
        size: pageSize
      });
      const { success, message, data } = res.data;

      if (success) {
        setTokenInfo(data);
        setLogsTotal(data.logs_total || 0);
        return true;
      } else {
        setError(message || t('查询失败'));
        return false;
      }
    } catch (err) {
      if (err.response?.status === 429) {
        setError(t('请求过于频繁，请稍后重试'));
      } else {
        setError(t('网络错误，请稍后重试'));
      }
      return false;
    }
  };

  const handleQuery = async () => {
    if (!tokenKey.trim()) {
      setError(t('请输入 API 令牌'));
      setTokenInfo(null);
      return;
    }

    setLoading(true);
    setError('');
    setTokenInfo(null);
    setCurrentPage(1);
    setModelPage(1);
    setModelSearch('');

    await fetchData(tokenKey, 1);
    setLoading(false);
  };

  const handlePageChange = async (page) => {
    setLogsLoading(true);
    setCurrentPage(page);
    await fetchData(tokenKey, page);
    setLogsLoading(false);
  };

  const handleKeyPress = (e) => {
    if (e.key === 'Enter') {
      handleQuery();
    }
  };

  // 格式化额度显示
  const formatQuota = (quota, unlimited) => {
    if (unlimited) {
      return t('无限');
    }
    // 假设 QuotaPerUnit = 500000，转换为美元显示
    const quotaPerUnit = 500000;
    const dollars = (quota / quotaPerUnit).toFixed(4);
    return `${dollars}`;
  };

  // 格式化过期时间
  const formatExpireTime = (timestamp) => {
    if (timestamp === 0 || timestamp === -1) {
      return t('永不过期');
    }
    const date = new Date(timestamp * 1000);
    return date.toLocaleString();
  };

  // 格式化时间戳
  const formatTime = (timestamp) => {
    if (!timestamp) return '-';
    const date = new Date(timestamp * 1000);
    return date.toLocaleString();
  };

  // 获取状态标签
  const getStatusTag = (status) => {
    switch (status) {
      case TokenStatus.ENABLED:
        return <Tag color="green">{t('正常')}</Tag>;
      case TokenStatus.DISABLED:
        return <Tag color="grey">{t('已禁用')}</Tag>;
      case TokenStatus.EXPIRED:
        return <Tag color="orange">{t('已过期')}</Tag>;
      case TokenStatus.EXHAUSTED:
        return <Tag color="red">{t('额度耗尽')}</Tag>;
      default:
        return <Tag color="grey">{t('未知')}</Tag>;
    }
  };

  // 日志表格列定义
  const logColumns = [
    {
      title: t('时间'),
      dataIndex: 'created_at',
      key: 'created_at',
      width: 180,
      render: (text) => formatTime(text),
    },
    {
      title: t('模型'),
      dataIndex: 'model_name',
      key: 'model_name',
      width: 150,
    },
    {
      title: t('提示tokens'),
      dataIndex: 'prompt_tokens',
      key: 'prompt_tokens',
      width: 100,
      align: 'right',
    },
    {
      title: t('补全tokens'),
      dataIndex: 'completion_tokens',
      key: 'completion_tokens',
      width: 100,
      align: 'right',
    },
    {
      title: t('消耗额度'),
      dataIndex: 'quota',
      key: 'quota',
      width: 100,
      align: 'right',
      render: (text) => formatQuota(text, false),
    },
  ];

  // 过滤后的模型列表
  const filteredModels = useMemo(() => {
    if (!tokenInfo?.available_models) return [];
    const models = tokenInfo.available_models;
    if (!modelSearch.trim()) return models;
    const search = modelSearch.toLowerCase().trim();
    return models.filter(m => m.model_name.toLowerCase().includes(search));
  }, [tokenInfo?.available_models, modelSearch]);

  // 分页后的模型列表
  const paginatedModels = useMemo(() => {
    const start = (modelPage - 1) * modelPageSize;
    return filteredModels.slice(start, start + modelPageSize);
  }, [filteredModels, modelPage, modelPageSize]);

  // 模型表格列定义
  const modelColumns = [
    {
      title: t('模型名称'),
      dataIndex: 'model_name',
      key: 'model_name',
      width: 200,
    },
    {
      title: t('计费类型'),
      dataIndex: 'quota_type',
      key: 'quota_type',
      width: 100,
      render: (type) => (
        <Tag color={type === 1 ? 'green' : 'blue'}>
          {type === 1 ? t('按次计费') : t('按量计费')}
        </Tag>
      ),
    },
    {
      title: t('模型倍率'),
      dataIndex: 'model_ratio',
      key: 'model_ratio',
      width: 100,
      align: 'right',
      render: (ratio, record) => {
        if (record.quota_type === 1) {
          return '-';
        }
        return `${ratio}x`;
      },
    },
    {
      title: t('补全倍率'),
      dataIndex: 'completion_ratio',
      key: 'completion_ratio',
      width: 100,
      align: 'right',
      render: (ratio, record) => {
        if (record.quota_type === 1) {
          return '-';
        }
        return ratio ? `${ratio}x` : '-';
      },
    },
    {
      title: t('输入价格'),
      dataIndex: 'input_price',
      key: 'input_price',
      width: 120,
      align: 'right',
      render: (price, record) => {
        if (record.quota_type === 1) {
          return `$${record.model_price.toFixed(4)}/次`;
        }
        return `$${price.toFixed(4)}/M`;
      },
    },
    {
      title: t('输出价格'),
      dataIndex: 'output_price',
      key: 'output_price',
      width: 120,
      align: 'right',
      render: (price, record) => {
        if (record.quota_type === 1) {
          return '-';
        }
        return `$${price.toFixed(4)}/M`;
      },
    },
  ];

  return (
    <div className="mt-[60px] px-4 py-6 max-w-4xl mx-auto">
      <Title heading={2} className="mb-6 text-center">
        <IconKey className="mr-2" />
        {t('令牌查询')}
      </Title>

      <Card className="mb-6">
        <div className="flex flex-col gap-4">
          <Text type="secondary">
            {t('输入您的 API 令牌以查询使用情况和余额信息')}
          </Text>
          <div className="flex gap-2">
            <Input
              prefix={<IconKey />}
              placeholder={t('请输入 API 令牌（支持带或不带 sk- 前缀）')}
              value={tokenKey}
              onChange={setTokenKey}
              onKeyPress={handleKeyPress}
              size="large"
              className="flex-1"
              disabled={loading}
            />
            <Button
              type="primary"
              icon={<IconSearch />}
              onClick={handleQuery}
              loading={loading}
              size="large"
            >
              {t('查询')}
            </Button>
          </div>
        </div>
      </Card>

      {error && (
        <Banner
          type="danger"
          description={error}
          className="mb-4"
          closeIcon={null}
        />
      )}

      {loading && (
        <div className="flex justify-center py-8">
          <Spin size="large" />
        </div>
      )}

      {tokenInfo && !loading && (
        <>
          <Card title={t('令牌信息')} className="mb-6">
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
              <div className="flex flex-col">
                <Text type="tertiary" size="small">{t('状态')}</Text>
                <div>{getStatusTag(tokenInfo.status)}</div>
              </div>
              <div className="flex flex-col">
                <Text type="tertiary" size="small">{t('过期时间')}</Text>
                <Text strong>{formatExpireTime(tokenInfo.expires_at)}</Text>
              </div>
              <div className="flex flex-col">
                <Text type="tertiary" size="small">{t('已使用')}</Text>
                <Text strong>${formatQuota(tokenInfo.total_used, false)}</Text>
              </div>
              <div className="flex flex-col">
                <Text type="tertiary" size="small">{t('剩余额度')}</Text>
                <Text strong>${formatQuota(tokenInfo.total_available, false)}</Text>
              </div>
            </div>
          </Card>

          {/* 可用模型表格 */}
          <Card 
            title={t('可用模型')} 
            className="mb-6"
            headerExtraContent={
              <Input
                prefix={<IconSearch />}
                placeholder={t('搜索模型')}
                value={modelSearch}
                onChange={setModelSearch}
                style={{ width: 200 }}
                showClear
              />
            }
          >
            {tokenInfo.available_models && tokenInfo.available_models.length > 0 ? (
              <Table
                columns={modelColumns}
                dataSource={paginatedModels}
                pagination={{
                  currentPage: modelPage,
                  pageSize: modelPageSize,
                  total: filteredModels.length,
                  onPageChange: setModelPage,
                  showSizeChanger: false,
                  showTotal: true,
                  formatShowTotal: (total) => `${t('共')} ${total} ${t('个模型')}`,
                }}
                size="small"
                rowKey="model_name"
              />
            ) : (
              <Empty
                image={<IconSearch style={{ fontSize: 48, color: 'var(--semi-color-text-2)' }} />}
                description={t('该分组暂无可用模型')}
              />
            )}
          </Card>

          <Card title={t('详细调用信息')}>
            {tokenInfo.logs && tokenInfo.logs.length > 0 ? (
              <Table
                columns={logColumns}
                dataSource={tokenInfo.logs}
                loading={logsLoading}
                pagination={{
                  currentPage: currentPage,
                  pageSize: pageSize,
                  total: logsTotal,
                  onPageChange: handlePageChange,
                  showSizeChanger: false,
                  showTotal: true,
                  formatShowTotal: (total) => `${t('共')} ${total} ${t('条记录')}`,
                }}
                size="small"
                rowKey="id"
              />
            ) : (
              <Empty
                image={<IconSearch style={{ fontSize: 48, color: 'var(--semi-color-text-2)' }} />}
                description={t('暂无调用记录')}
              />
            )}
          </Card>
        </>
      )}
    </div>
  );
};

export default Query;
