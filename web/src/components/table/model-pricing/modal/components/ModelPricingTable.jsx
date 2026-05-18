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
import { Avatar, Typography, Table, Tag } from '@douyinfe/semi-ui';
import { IconCoinMoneyStroked } from '@douyinfe/semi-icons';
import { calculateModelPrice, getModelPriceItems } from '../../../../../helpers';
import { useIsMobile } from '../../../../../hooks/common/useIsMobile';

const { Text } = Typography;

const priceSummaryRowStyle = {
  display: 'grid',
  gridTemplateColumns: 'minmax(0, 1fr) auto',
  alignItems: 'center',
  gap: 12,
  padding: '10px 18px',
  borderBottom: '1px solid var(--semi-color-border)',
};

const nestedPricingTableStyle = {
  borderRadius: 8,
  overflow: 'hidden',
  width: '100%',
};

const compactGroupCardStyle = {
  borderTop: '1px solid var(--semi-color-border)',
  padding: '14px 0',
};

const mobilePerCallCardStyle = {
  border: '1px solid var(--semi-color-border)',
  borderRadius: 8,
  overflow: 'hidden',
  background: 'var(--semi-color-bg-1)',
};

const mobilePerCallMetaStyle = {
  display: 'grid',
  gridTemplateColumns: '1fr',
  gap: 10,
  padding: 14,
  borderBottom: '1px solid var(--semi-color-border)',
};

const mobilePerCallMetaItemStyle = {
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'space-between',
  gap: 12,
};

const ModelPricingTable = ({
  modelData,
  groupRatio,
  currency,
  siteDisplayType,
  tokenUnit,
  displayPrice,
  showRatio,
  usableGroup,
  autoGroups = [],
  t,
}) => {
  const isMobile = useIsMobile();
  const isPerCallExpr = modelData?.billing_mode === 'per_call_expr';
  const modelEnableGroups = Array.isArray(modelData?.enable_groups)
    ? modelData.enable_groups
    : [];
  const autoChain = autoGroups.filter((g) => modelEnableGroups.includes(g));

  const renderGroupTag = (group, withSuffix = false) => (
    <Tag color='white' size='small' shape='circle'>
      {group}
      {withSuffix ? t('分组') : ''}
    </Tag>
  );

  const renderPriceSummary = (items) => {
    if (items.length === 1 && items[0].isDynamic) {
      return (
        <Text type='tertiary' size='small'>
          {t('见上方动态计费详情')}
        </Text>
      );
    }

    if (isPerCallExpr) {
      const priceItems = items.filter((item) => !item.isDynamic);
      if (priceItems.length === 0) {
        return <Text type='tertiary'>-</Text>;
      }

      const columns = [
        {
          title: t('模型'),
          dataIndex: 'label',
          render: (text) => <Text>{text || t('默认')}</Text>,
        },
        {
          title: t('价格'),
          dataIndex: 'value',
          render: (text) => (
            <Text strong style={{ color: 'var(--semi-color-warning)' }}>
              {text}
            </Text>
          ),
        },
      ];
      const dataSource = priceItems.map((item) => ({
        key: item.key,
        label: item.label,
        value: item.value,
      }));

      return (
        <div style={nestedPricingTableStyle}>
          <Table
            dataSource={dataSource}
            columns={columns}
            pagination={false}
            size='small'
            bordered={false}
            className='!rounded-lg'
          />
        </div>
      );
    }

    return (
      <div style={{ minWidth: 240 }}>
        {items.map((item, index) => (
          <div
            key={item.key}
            style={{
              ...priceSummaryRowStyle,
              paddingLeft: 0,
              paddingRight: 0,
              borderBottom:
                index === items.length - 1
                  ? 'none'
                  : priceSummaryRowStyle.borderBottom,
            }}
          >
            <Text size='small'>{item.label || t('默认')}</Text>
            <div style={{ textAlign: 'right', whiteSpace: 'nowrap' }}>
              <Text strong>{item.value}</Text>
              {item.suffix && (
                <Text type='tertiary' size='small'>
                  {item.suffix}
                </Text>
              )}
            </div>
          </div>
        ))}
      </div>
    );
  };

  const renderGroupPriceTable = () => {
    // 仅展示模型可用的分组：模型 enable_groups 与用户可用分组的交集

    const availableGroups = Object.keys(usableGroup || {})
      .filter((g) => g !== '')
      .filter((g) => g !== 'auto')
      .filter((g) => modelEnableGroups.includes(g));

    // 准备表格数据
    const tableData = availableGroups.map((group) => {
      const priceData = modelData
        ? calculateModelPrice({
            record: modelData,
            selectedGroup: group,
            groupRatio,
            tokenUnit,
            displayPrice,
            currency,
            quotaDisplayType: siteDisplayType,
          })
        : { inputPrice: '-', outputPrice: '-', price: '-' };

      // 获取分组倍率
      const groupRatioValue =
        groupRatio && groupRatio[group] ? groupRatio[group] : 1;

      return {
        key: group,
        group: group,
        ratio: groupRatioValue,
        billingType:
          modelData?.billing_mode === 'tiered_expr' ||
          modelData?.billing_mode === 'per_call_expr'
            ? modelData?.billing_mode === 'per_call_expr'
              ? t('按次计费')
              : t('动态计费')
            : modelData?.quota_type === 0
              ? t('按量计费')
              : modelData?.quota_type === 1
                ? t('按次计费')
                : '-',
        priceItems: getModelPriceItems(priceData, t, siteDisplayType),
      };
    });

    if (isPerCallExpr) {
      if (isMobile) {
        return (
          <div style={{ display: 'grid', gap: 12 }}>
            {tableData.map((row) => (
              <div key={row.key} style={mobilePerCallCardStyle}>
                <div style={mobilePerCallMetaStyle}>
                  <div style={mobilePerCallMetaItemStyle}>
                    <Text type='tertiary' size='small'>{t('分组')}</Text>
                    {renderGroupTag(row.group)}
                  </div>
                  <div style={mobilePerCallMetaItemStyle}>
                    <Text type='tertiary' size='small'>{t('计费类型')}</Text>
                    <Tag color='teal' size='small' shape='circle'>
                      {row.billingType || '-'}
                    </Tag>
                  </div>
                </div>
                <div style={{ padding: 12 }}>
                  <Text
                    type='tertiary'
                    size='small'
                    style={{ display: 'block', marginBottom: 8 }}
                  >
                    {t('价格')}
                  </Text>
                  {renderPriceSummary(row.priceItems)}
                </div>
              </div>
            ))}
          </div>
        );
      }

      const columns = [
        {
          title: t('分组'),
          dataIndex: 'group',
          render: (text) => renderGroupTag(text, true),
        },
        {
          title: t('计费类型'),
          dataIndex: 'billingType',
          render: (text) => (
            <Tag color='teal' size='small' shape='circle'>
              {text || '-'}
            </Tag>
          ),
        },
        {
          title: t('价格'),
          dataIndex: 'priceItems',
          render: renderPriceSummary,
        },
      ];

      return (
        <Table
          dataSource={tableData}
          columns={columns}
          pagination={false}
          size='small'
          bordered={false}
          className='!rounded-lg'
          scroll={isMobile ? { x: 'max-content' } : undefined}
        />
      );
    }

    if (isMobile || tableData.length === 1) {
      return (
        <div>
          {tableData.map((row) => (
            <div
              key={row.key}
              style={compactGroupCardStyle}
            >
              <div
                style={{
                  display: isMobile ? 'block' : 'grid',
                  gridTemplateColumns: '220px 220px minmax(0, 1fr)',
                  alignItems: 'center',
                  gap: 18,
                }}
              >
                <div style={{ marginBottom: isMobile ? 10 : 0 }}>
                  {renderGroupTag(row.group)}
                </div>
                <div style={{ marginBottom: isMobile ? 10 : 0 }}>
                  <Tag
                    color={
                      row.billingType === t('动态计费')
                        ? 'amber'
                        : row.billingType === t('按次计费')
                          ? 'teal'
                          : 'violet'
                    }
                    size='small'
                    shape='circle'
                  >
                    {row.billingType || '-'}
                  </Tag>
                </div>
                <div>{renderPriceSummary(row.priceItems)}</div>
              </div>
            </div>
          ))}
        </div>
      );
    }

    // 定义表格列
    const columns = [
      {
        title: t('分组'),
        dataIndex: 'group',
        render: (text) => renderGroupTag(text, true),
      },
    ];

    if (showRatio && !isPerCallExpr) {
      columns.push({
        title: t('分组倍率'),
        dataIndex: 'ratio',
        render: (text) => (
          <Tag color='blue' size='small' shape='circle'>
            {text}x
          </Tag>
        ),
      });
    }

    columns.push({
      title: t('计费类型'),
      dataIndex: 'billingType',
      render: (text) => {
        let color = 'white';
        if (text === t('按量计费')) color = 'violet';
        else if (text === t('按次计费')) color = 'teal';
        else if (text === t('动态计费')) color = 'amber';
        return (
          <Tag color={color} size='small' shape='circle'>
            {text || '-'}
          </Tag>
        );
      },
    });

    columns.push({
      title: isPerCallExpr
        ? t('价格')
        : siteDisplayType === 'TOKENS'
          ? t('计费摘要')
          : t('价格摘要'),
      dataIndex: 'priceItems',
      render: renderPriceSummary,
    });

    return (
      <Table
        dataSource={tableData}
        columns={columns}
        pagination={false}
        size='small'
        bordered={false}
        className='!rounded-lg'
      />
    );
  };

  return (
    <div>
      <div className='flex items-center mb-4'>
        <Avatar size='small' color='orange' className='mr-2 shadow-md'>
          <IconCoinMoneyStroked size={16} />
        </Avatar>
        <div>
          <Text className='text-lg font-medium'>{t('分组价格')}</Text>
          <div className='text-xs text-gray-600'>
            {t('不同用户分组的价格信息')}
          </div>
        </div>
      </div>
      {autoChain.length > 0 && (
        <div className='flex flex-wrap items-center gap-1 mb-4'>
          <span className='text-sm text-gray-600'>{t('auto分组调用链路')}</span>
          <span className='text-sm'>→</span>
          {autoChain.map((g, idx) => (
            <React.Fragment key={g}>
              <Tag color='white' size='small' shape='circle'>
                {g}
                {t('分组')}
              </Tag>
              {idx < autoChain.length - 1 && <span className='text-sm'>→</span>}
            </React.Fragment>
          ))}
        </div>
      )}
      {renderGroupPriceTable()}
    </div>
  );
};

export default ModelPricingTable;
