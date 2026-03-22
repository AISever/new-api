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

export default function RelationTableEditor({
  rows,
  onAddRow,
  onDeleteRow,
  onChangeRow,
}) {
  const { t } = useTranslation();
  const layoutMode = useCompactEditorLayoutMode();
  const isMobile = layoutMode !== 'desktop';
  const useSharedHeader = shouldUseSharedInlineHeaders(layoutMode);
  const columns = [
    { key: 'group', label: t('用户分组'), width: 'minmax(140px, 1fr)' },
    { key: 'targetGroup', label: t('使用分组'), width: 'minmax(140px, 1fr)' },
    { key: 'value', label: t('倍率'), width: 'minmax(100px, 0.8fr)' },
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
      onAdd={onAddRow}
      columns={columns}
      emptyText={t('暂无数据')}
      minWidth={680}
      maxHeight={300}
      mobileHeader={
        useSharedHeader ? (
          <div
            style={{
              display: 'grid',
              gridTemplateColumns: 'minmax(0, 1fr) minmax(0, 1fr) minmax(88px, 0.8fr) 28px',
              gap: 8,
              alignItems: 'center',
            }}
          >
            <div>{t('用户分组')}</div>
            <div>{t('使用分组')}</div>
            <div>{t('倍率')}</div>
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
                  gridTemplateColumns: 'minmax(0, 1fr) minmax(0, 1fr) minmax(88px, 0.8fr) 28px',
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
                <Input
                  size='small'
                  value={record.targetGroup}
                  placeholder={t('如 default')}
                  onChange={(value) => onChangeRow(record.id, 'targetGroup', value)}
                />
                <Input
                  size='small'
                  value={record.value}
                  placeholder='1'
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
                <div style={{ display: 'flex', justifyContent: 'space-between', gap: 8 }}>
                  <Text type='tertiary' size='small'>
                    {t('特殊倍率规则')}
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
                  value={record.group}
                  placeholder={t('如 vip')}
                  onChange={(value) => onChangeRow(record.id, 'group', value)}
                />
                <Text type='tertiary' size='small'>
                  {t('使用分组')}
                </Text>
                <Input
                  size='small'
                  value={record.targetGroup}
                  placeholder={t('如 default')}
                  onChange={(value) => onChangeRow(record.id, 'targetGroup', value)}
                />
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
                index === rows.length - 1 ? 'none' : '1px solid var(--semi-color-border)',
            }}
          >
            <Input
              size='small'
              value={record.group}
              placeholder={t('如 vip')}
              onChange={(value) => onChangeRow(record.id, 'group', value)}
            />
            <Input
              size='small'
              value={record.targetGroup}
              placeholder={t('如 default')}
              onChange={(value) => onChangeRow(record.id, 'targetGroup', value)}
            />
            <Input
              size='small'
              value={record.value}
              placeholder='1'
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
