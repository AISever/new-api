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
import { Button, Input } from '@douyinfe/semi-ui';
import { IconDelete } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import CompactEditorFrame from './CompactEditorFrame';

export default function RelationTableEditor({
  rows,
  onAddRow,
  onDeleteRow,
  onChangeRow,
}) {
  const { t } = useTranslation();
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
      ))}
    </CompactEditorFrame>
  );
}
