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
import { Banner, Button, Card, Space, Typography } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';

const { Text } = Typography;

export default function OptionModeCard({
  title,
  description = '',
  mode = 'table',
  onModeChange,
  error = '',
  tableContent,
  jsonContent,
  footer = null,
}) {
  const { t } = useTranslation();

  return (
    <Card
      style={{ marginBottom: 12 }}
      bodyStyle={{ padding: 12 }}
      headerStyle={{ padding: '12px 16px' }}
      title={title}
      headerExtraContent={
        <Space>
          <Button
            theme={mode === 'table' ? 'solid' : 'light'}
            type='primary'
            size='small'
            onClick={() => onModeChange?.('table')}
          >
            {t('表格模式')}
          </Button>
          <Button
            theme={mode === 'json' ? 'solid' : 'light'}
            type='tertiary'
            size='small'
            onClick={() => onModeChange?.('json')}
          >
            JSON
          </Button>
        </Space>
      }
    >
      {description ? (
        <Text type='tertiary' style={{ display: 'block', marginBottom: 8 }}>
          {description}
        </Text>
      ) : null}
      {error ? (
        <Banner
          type='danger'
          bordered
          fullMode={false}
          closeIcon={null}
          style={{ marginBottom: 8 }}
          description={error}
        />
      ) : null}
      {mode === 'table' ? tableContent : jsonContent}
      {footer}
    </Card>
  );
}
