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
import { Button, Input, Typography } from '@douyinfe/semi-ui';
import { IconDelete } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import useCompactEditorLayoutMode from '../hooks/useCompactEditorLayoutMode';
import { shouldUseSharedInlineHeaders } from '../utils/editorLayout';
import CompactEditorFrame from './CompactEditorFrame';

const { Text } = Typography;

export default function SimpleMapTableEditor({
  rows,
  keyLabel,
  valueLabel,
  keyPlaceholder,
  valuePlaceholder,
  onAddRow,
  onDeleteRow,
  onChangeRow,
  valueSuffix,
  toolbarExtra,
  addLabel,
}) {
  const { t } = useTranslation();
  const layoutMode = useCompactEditorLayoutMode();
  const isMobile = layoutMode !== 'desktop';
  const useSharedHeader = shouldUseSharedInlineHeaders(layoutMode);

  const columns = [
    { key: 'key', label: keyLabel, width: 'minmax(220px, 1.6fr)' },
    { key: 'value', label: valueLabel, width: 'minmax(120px, 1fr)' },
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
      addLabel={addLabel || t('新增一行')}
      onAdd={onAddRow}
      columns={columns}
      toolbarExtra={toolbarExtra}
      emptyText={t('暂无数据')}
      mobileHeader={
        useSharedHeader ? (
          <div
            style={{
              display: 'grid',
              gridTemplateColumns: 'minmax(0, 1.7fr) minmax(88px, 0.9fr) 28px',
              gap: 8,
              alignItems: 'center',
            }}
          >
            <div>{keyLabel}</div>
            <div>{valueLabel}</div>
            <span />
          </div>
        ) : null
      }
    >
      {rows.map((record, index) => (
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
                style={{
                  display: 'grid',
                  gridTemplateColumns: 'minmax(0, 1.7fr) minmax(88px, 0.9fr) 28px',
                  columnGap: 8,
                  alignItems: 'center',
                }}
              >
                <Input
                  size='small'
                  value={record.key}
                  placeholder={keyPlaceholder}
                  onChange={(value) => onChangeRow(record.id, 'key', value)}
                />
                <Input
                  size='small'
                  value={record.value}
                  placeholder={valuePlaceholder}
                  suffix={valueSuffix}
                  onChange={(value) => onChangeRow(record.id, 'value', value)}
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
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    gap: 8,
                  }}
                >
                  <Text type='tertiary' size='small'>
                    {keyLabel}
                  </Text>
                  <Button
                    size='small'
                    type='danger'
                    theme='borderless'
                    icon={<IconDelete />}
                    onClick={() => onDeleteRow(record.id)}
                  />
                </div>
                <Input
                  size='small'
                  value={record.key}
                  placeholder={keyPlaceholder}
                  onChange={(value) => onChangeRow(record.id, 'key', value)}
                />
                <Text type='tertiary' size='small'>
                  {valueLabel}
                </Text>
                <Input
                  size='small'
                  value={record.value}
                  placeholder={valuePlaceholder}
                  suffix={valueSuffix}
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
                index === rows.length - 1 ? 'none' : '1px solid var(--semi-color-border)',
            }}
          >
            <Input
              size='small'
              value={record.key}
              placeholder={keyPlaceholder}
              onChange={(value) => onChangeRow(record.id, 'key', value)}
            />
            <Input
              size='small'
              value={record.value}
              placeholder={valuePlaceholder}
              suffix={valueSuffix}
              onChange={(value) => onChangeRow(record.id, 'value', value)}
            />
            <Button
              size='small'
              type='danger'
              theme='borderless'
              icon={<IconDelete />}
              onClick={() => onDeleteRow(record.id)}
            />
          </div>
        )
      ))}
    </CompactEditorFrame>
  );
}
