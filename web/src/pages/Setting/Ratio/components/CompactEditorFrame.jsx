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
import { Button, Empty } from '@douyinfe/semi-ui';
import { IconPlus } from '@douyinfe/semi-icons';
import { useIsMobile } from '../../../../hooks/common/useIsMobile';

const createHeaderGrid = (gridTemplateColumns) => ({
  display: 'grid',
  gridTemplateColumns,
  gap: 8,
  alignItems: 'center',
});

export default function CompactEditorFrame({
  addLabel,
  onAdd,
  columns,
  children,
  emptyText,
  toolbarExtra = null,
  minWidth = 640,
  maxHeight = 360,
}) {
  const isMobile = useIsMobile();
  const headerStyle = createHeaderGrid(columns.map((column) => column.width).join(' '));

  return (
    <div style={{ width: '100%' }}>
      <div
        style={{
          display: 'flex',
          alignItems: isMobile ? 'stretch' : 'center',
          justifyContent: 'space-between',
          flexDirection: isMobile ? 'column' : 'row',
          gap: 8,
          flexWrap: isMobile ? 'nowrap' : 'wrap',
          marginBottom: 8,
        }}
      >
        <Button
          size='small'
          icon={<IconPlus />}
          onClick={onAdd}
          style={isMobile ? { width: '100%' } : undefined}
        >
          {addLabel}
        </Button>
        {toolbarExtra ? (
          <div
            style={{
              flex: isMobile ? '1 1 auto' : '1 1 240px',
              minWidth: isMobile ? 0 : 220,
              width: isMobile ? '100%' : undefined,
            }}
          >
            {toolbarExtra}
          </div>
        ) : null}
      </div>
      <div
        style={{
          border: '1px solid var(--semi-color-border)',
          borderRadius: 12,
          background: 'var(--semi-color-bg-1)',
          overflow: 'hidden',
        }}
      >
        <div style={{ overflowX: 'auto' }}>
          <div style={{ minWidth: isMobile ? '100%' : minWidth }}>
            {!isMobile ? (
              <div
                style={{
                  ...headerStyle,
                  padding: '8px 12px',
                  background: 'var(--semi-color-fill-0)',
                  color: 'var(--semi-color-text-2)',
                  fontSize: 12,
                  fontWeight: 600,
                  borderBottom: '1px solid var(--semi-color-border)',
                }}
              >
                {columns.map((column) => (
                  <div key={column.key}>{column.label}</div>
                ))}
              </div>
            ) : null}
            <div style={{ maxHeight, overflowY: 'auto', padding: isMobile ? 8 : 0 }}>
              {children?.length ? (
                children
              ) : (
                <div style={{ padding: '20px 12px' }}>
                  <Empty description={emptyText} imageStyle={{ height: 48 }} />
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
