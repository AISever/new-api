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
import { Button, Input, Switch, Typography } from '@douyinfe/semi-ui';
import { IconDelete } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import usePendingAppendedRowFocus from '../hooks/usePendingAppendedRowFocus';
import useCompactEditorLayoutMode from '../hooks/useCompactEditorLayoutMode';
import { shouldUseSharedInlineHeaders } from '../utils/editorLayout';
import CompactEditorFrame from './CompactEditorFrame';

const { Text } = Typography;

export default function GroupRatioTableEditor({
  rows,
  onAddRow,
  onDeleteRow,
  onChangeRow,
  onToggleVisible,
  isGroupVisible,
  visibilityDisabled = false,
}) {
  const { t } = useTranslation();
  const layoutMode = useCompactEditorLayoutMode();
  const isMobile = layoutMode !== 'desktop';
  const useSharedHeader = shouldUseSharedInlineHeaders(layoutMode);
  const { containerRef, focusRowId, requestFocusOnNextAddedRow } =
    usePendingAppendedRowFocus(rows);
  const columns = [
    { key: 'key', label: t('分组名称'), width: 'minmax(220px, 1.5fr)' },
    { key: 'value', label: t('倍率'), width: 'minmax(100px, 0.8fr)' },
    { key: 'visible', label: t('显示'), width: '92px' },
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
      addLabel={t('新增一行')}
      onAdd={() => {
        requestFocusOnNextAddedRow();
        onAddRow();
      }}
      columns={columns}
      emptyText={t('暂无数据')}
      minWidth={720}
      mobileHeader={
        useSharedHeader ? (
          <div
            style={{
              display: 'grid',
              gridTemplateColumns:
                'minmax(0, 1.4fr) minmax(88px, 0.8fr) 80px 28px',
              gap: 8,
              alignItems: 'center',
            }}
          >
            <div>{t('分组名称')}</div>
            <div>{t('倍率')}</div>
            <div>{t('显示')}</div>
            <span />
          </div>
        ) : null
      }
    >
      <div ref={containerRef}>
        {rows.map((record, index) => {
          const visible = isGroupVisible(record.key);
          const switchDisabled =
            visibilityDisabled || String(record.key ?? '').trim() === '';

          return isMobile ? (
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
                      'minmax(0, 1.4fr) minmax(88px, 0.8fr) 80px 28px',
                    columnGap: 8,
                    alignItems: 'center',
                  }}
                >
                  <Input
                    size='small'
                    value={record.key}
                    placeholder={t('如 vip')}
                    onChange={(value) => onChangeRow(record.id, 'key', value)}
                  />
                  <Input
                    size='small'
                    value={record.value}
                    placeholder='1'
                    onChange={(value) => onChangeRow(record.id, 'value', value)}
                  />
                  <Switch
                    size='small'
                    checked={visible}
                    disabled={switchDisabled}
                    onChange={(checked) => onToggleVisible(record.id, checked)}
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
                      alignItems: 'center',
                      gap: 8,
                    }}
                  >
                    <Text type='tertiary' size='small'>
                      {t('用户可见')}
                    </Text>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                      <Switch
                        size='small'
                        checked={visible}
                        disabled={switchDisabled}
                        onChange={(checked) => onToggleVisible(record.id, checked)}
                      />
                      <Button
                        size='small'
                        type='danger'
                        theme='borderless'
                        icon={<IconDelete />}
                        onClick={() => onDeleteRow(record.id)}
                      />
                    </div>
                  </div>
                  <Text type='tertiary' size='small'>
                    {t('分组名称')}
                  </Text>
                  <div
                    data-row-focus-id={
                      record.id === focusRowId ? record.id : undefined
                    }
                  >
                    <Input
                      size='small'
                      value={record.key}
                      placeholder={t('如 vip')}
                      onChange={(value) => onChangeRow(record.id, 'key', value)}
                    />
                  </div>
                  <Text type='tertiary' size='small'>
                    {t('倍率')}
                  </Text>
                  <Input
                    size='small'
                    value={record.value}
                    placeholder='1'
                    onChange={(value) => onChangeRow(record.id, 'value', value)}
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
                  value={record.key}
                  placeholder={t('如 vip')}
                  onChange={(value) => onChangeRow(record.id, 'key', value)}
                />
              </div>
              <Input
                size='small'
                value={record.value}
                placeholder='1'
                onChange={(value) => onChangeRow(record.id, 'value', value)}
              />
              <Switch
                size='small'
                checked={visible}
                disabled={switchDisabled}
                onChange={(checked) => onToggleVisible(record.id, checked)}
              />
              <Button
                size='small'
                type='danger'
                theme='borderless'
                icon={<IconDelete />}
                onClick={() => onDeleteRow(record.id)}
              />
            </div>
          );
        })}
      </div>
    </CompactEditorFrame>
  );
}
