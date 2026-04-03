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
import { Button, Input, Select, Typography } from '@douyinfe/semi-ui';
import { IconDelete } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import usePendingAppendedRowFocus from '../hooks/usePendingAppendedRowFocus';
import useCompactEditorLayoutMode from '../hooks/useCompactEditorLayoutMode';
import { shouldUseSharedInlineHeaders } from '../utils/editorLayout';
import CompactEditorFrame from './CompactEditorFrame';

const { Text } = Typography;

export default function OperationRuleTableEditor({
  rows,
  onAddRow,
  onDeleteRow,
  onChangeRow,
}) {
  const { t } = useTranslation();
  const layoutMode = useCompactEditorLayoutMode();
  const isMobile = layoutMode !== 'desktop';
  const useSharedHeader = shouldUseSharedInlineHeaders(layoutMode);
  const { containerRef, focusRowId, requestFocusOnNextAddedRow } =
    usePendingAppendedRowFocus(rows);
  const columns = [
    { key: 'group', label: t('用户分组'), width: 'minmax(120px, 1fr)' },
    { key: 'action', label: t('操作类型'), width: '120px' },
    { key: 'targetGroup', label: t('目标分组'), width: 'minmax(120px, 1fr)' },
    { key: 'description', label: t('描述'), width: 'minmax(160px, 1.2fr)' },
    { key: 'actions', label: t('操作'), width: '44px' },
  ];
  const rowStyle = {
    display: 'grid',
    gridTemplateColumns: columns.map((column) => column.width).join(' '),
    gap: 8,
    alignItems: 'center',
    padding: '8px 12px',
    borderBottom: '1px solid var(--semi-color-border)',
  };

  return (
    <CompactEditorFrame
      addLabel={t('新增规则')}
      onAdd={() => {
        requestFocusOnNextAddedRow();
        onAddRow();
      }}
      columns={columns}
      emptyText={t('暂无数据')}
      minWidth={760}
      maxHeight={300}
      mobileHeader={
        useSharedHeader ? (
          <div
            style={{
              display: 'grid',
              gridTemplateColumns:
                'minmax(0, 1fr) minmax(110px, 0.9fr) minmax(0, 1fr) minmax(0, 1fr) 28px',
              gap: 8,
              alignItems: 'center',
            }}
          >
            <div>{t('用户分组')}</div>
            <div>{t('操作类型')}</div>
            <div>{t('目标分组')}</div>
            <div>{t('描述')}</div>
            <span />
          </div>
        ) : null
      }
    >
      <div ref={containerRef}>
        {rows.map((record, index) =>
          isMobile ? (
            <div
              key={record.id}
              style={{
                border: '1px solid var(--semi-color-border)',
                borderRadius: 10,
                padding: useSharedHeader ? '8px 10px' : 10,
                display: 'flex',
                flexDirection: 'column',
                gap: useSharedHeader ? 6 : 8,
                marginBottom: index === rows.length - 1 ? 0 : 8,
                background: 'var(--semi-color-bg-1)',
              }}
            >
              {useSharedHeader ? (
                <div
                  data-row-focus-id={
                    record.id === focusRowId ? record.id : undefined
                  }
                  style={{
                    display: 'grid',
                    gridTemplateColumns:
                      'minmax(0, 1fr) minmax(110px, 0.9fr) minmax(0, 1fr) minmax(0, 1fr) 28px',
                    columnGap: 8,
                    alignItems: 'center',
                  }}
                >
                  <Input
                    size='small'
                    value={record.group}
                    placeholder={t('如 vip')}
                    onChange={(value) => onChangeRow(record.id, 'group', value)}
                  />
                  <Select
                    size='small'
                    value={record.action}
                    onChange={(value) => onChangeRow(record.id, 'action', value)}
                    optionList={[
                      { label: t('添加'), value: 'add' },
                      { label: t('移除'), value: 'remove' },
                      { label: t('直接追加'), value: 'direct' },
                    ]}
                  />
                  <Input
                    size='small'
                    value={record.targetGroup}
                    placeholder={t('如 premium')}
                    onChange={(value) =>
                      onChangeRow(record.id, 'targetGroup', value)
                    }
                  />
                  <Input
                    size='small'
                    value={record.description}
                    placeholder={t('分组描述')}
                    onChange={(value) =>
                      onChangeRow(record.id, 'description', value)
                    }
                  />
                  <Button
                    size='small'
                    type='danger'
                    theme='borderless'
                    icon={<IconDelete />}
                    onClick={() => onDeleteRow(record.id)}
                  />
                </div>
              ) : (
                <>
                  <div
                    style={{
                      display: 'flex',
                      justifyContent: 'space-between',
                      gap: 8,
                    }}
                  >
                    <Text type='tertiary' size='small'>
                      {t('特殊可用分组规则')}
                    </Text>
                    <Button
                      size='small'
                      type='danger'
                      theme='borderless'
                      icon={<IconDelete />}
                      onClick={() => onDeleteRow(record.id)}
                    />
                  </div>
                  <Text type='tertiary' size='small'>
                    {t('用户分组')}
                  </Text>
                  <Input
                    size='small'
                    data-row-focus-id={
                      record.id === focusRowId ? record.id : undefined
                    }
                    value={record.group}
                    placeholder={t('如 vip')}
                    onChange={(value) => onChangeRow(record.id, 'group', value)}
                  />
                  <Text type='tertiary' size='small'>
                    {t('操作类型')}
                  </Text>
                  <Select
                    size='small'
                    value={record.action}
                    onChange={(value) => onChangeRow(record.id, 'action', value)}
                    optionList={[
                      { label: t('添加'), value: 'add' },
                      { label: t('移除'), value: 'remove' },
                      { label: t('直接追加'), value: 'direct' },
                    ]}
                  />
                  <Text type='tertiary' size='small'>
                    {t('目标分组')}
                  </Text>
                  <Input
                    size='small'
                    value={record.targetGroup}
                    placeholder={t('如 premium')}
                    onChange={(value) =>
                      onChangeRow(record.id, 'targetGroup', value)
                    }
                  />
                  <Text type='tertiary' size='small'>
                    {t('描述')}
                  </Text>
                  <Input
                    size='small'
                    value={record.description}
                    placeholder={t('分组描述')}
                    onChange={(value) =>
                      onChangeRow(record.id, 'description', value)
                    }
                  />
                </>
              )}
            </div>
          ) : (
            <div
              key={record.id}
              style={{
                ...rowStyle,
                borderBottom:
                  index === rows.length - 1
                    ? 'none'
                    : '1px solid var(--semi-color-border)',
              }}
            >
              <div
                data-row-focus-id={record.id === focusRowId ? record.id : undefined}
              >
                <Input
                  size='small'
                  value={record.group}
                  placeholder={t('如 vip')}
                  onChange={(value) => onChangeRow(record.id, 'group', value)}
                />
              </div>
              <Select
                size='small'
                value={record.action}
                onChange={(value) => onChangeRow(record.id, 'action', value)}
                optionList={[
                  { label: t('添加'), value: 'add' },
                  { label: t('移除'), value: 'remove' },
                  { label: t('直接追加'), value: 'direct' },
                ]}
              />
              <Input
                size='small'
                value={record.targetGroup}
                placeholder={t('如 premium')}
                onChange={(value) => onChangeRow(record.id, 'targetGroup', value)}
              />
              <Input
                size='small'
                value={record.description}
                placeholder={t('分组描述')}
                onChange={(value) => onChangeRow(record.id, 'description', value)}
              />
              <Button
                size='small'
                type='danger'
                theme='borderless'
                icon={<IconDelete />}
                onClick={() => onDeleteRow(record.id)}
              />
            </div>
          ),
        )}
      </div>
    </CompactEditorFrame>
  );
}
