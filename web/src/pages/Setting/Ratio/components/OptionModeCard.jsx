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
import {
  Banner,
  Card,
  Divider,
  Radio,
  RadioGroup,
  Typography,
} from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { useIsMobile } from '../../../../hooks/common/useIsMobile';

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
  const isMobile = useIsMobile();

  return (
    <Card
      style={{ marginBottom: 12 }}
      bodyStyle={{ padding: 0 }}
      headerStyle={{ padding: '12px 16px' }}
      title={
        <div
          style={{
            display: 'flex',
            flexDirection: 'column',
            gap: 4,
          }}
        >
          <span>{title}</span>
          {description ? (
            <Text type='tertiary' size='small'>
              {description}
            </Text>
          ) : null}
        </div>
      }
      headerExtraContent={
        <RadioGroup
          type='button'
          buttonSize='small'
          size='small'
          value={mode}
          direction={isMobile ? 'vertical' : 'horizontal'}
          onChange={(event) => onModeChange?.(event.target.value)}
        >
          <Radio value='table'>{t('表格模式')}</Radio>
          <Radio value='json'>JSON</Radio>
        </RadioGroup>
      }
    >
      <Divider margin='0' />
      <div style={{ padding: 16 }}>
        {error ? (
          <Banner
            type='danger'
            bordered
            fullMode={false}
            closeIcon={null}
            style={{ marginBottom: 12 }}
            description={error}
          />
        ) : null}
        {mode === 'table' ? tableContent : jsonContent}
        {footer}
      </div>
    </Card>
  );
}
