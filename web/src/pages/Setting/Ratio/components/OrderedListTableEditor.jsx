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
import { Button, Input, Space, Typography } from '@douyinfe/semi-ui';
import { IconArrowDown, IconArrowUp, IconDelete } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import CompactEditorFrame from './CompactEditorFrame';

const { Text } = Typography;

export default function OrderedListTableEditor({
  rows,
  onAddRow,
  onDeleteRow,
  onChangeRow,
  onMoveRow,
}) {
  const { t } = useTranslation();
  const columns = [
    { key: 'index', label: t('顺序'), width: '56px' },
    { key: 'value', label: t('分组'), width: 'minmax(180px, 1fr)' },
    { key: 'actions', label: t('操作'), width: '112px' },
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
      addLabel={t('新增分组')}
      onAdd={onAddRow}
      columns={columns}
      emptyText={t('暂无数据')}
      minWidth={520}
      maxHeight={260}
    >
      {rows.map((record, index) => (
        <div
          key={record.id}
          style={{
            ...rowStyle,
            borderBottom:
              index === rows.length - 1 ? 'none' : '1px solid var(--semi-color-border)',
          }}
        >
          <Text type='tertiary'>{index + 1}</Text>
          <Input
            size='small'
            value={record.value}
            placeholder={t('如 default')}
            onChange={(value) => onChangeRow(record.id, 'value', value)}
          />
          <Space spacing={4}>
            <Button
              size='small'
              theme='borderless'
              icon={<IconArrowUp />}
              disabled={index === 0}
              onClick={() => onMoveRow(index, index - 1)}
            />
            <Button
              size='small'
              theme='borderless'
              icon={<IconArrowDown />}
              disabled={index === rows.length - 1}
              onClick={() => onMoveRow(index, index + 1)}
            />
            <Button
              size='small'
              type='danger'
              theme='borderless'
              icon={<IconDelete />}
              onClick={() => onDeleteRow(record.id)}
            />
          </Space>
        </div>
      ))}
    </CompactEditorFrame>
  );
}
