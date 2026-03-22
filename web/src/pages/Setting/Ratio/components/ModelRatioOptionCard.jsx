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

import React, { useMemo, useState } from 'react';
import { Button, Input, TextArea } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import OptionModeCard from './OptionModeCard';
import SimpleMapTableEditor from './SimpleMapTableEditor';

export default function ModelRatioOptionCard({
  title,
  description,
  card,
  valuePlaceholder,
  onModeChange,
  onJsonChange,
  onRebuildRows,
  onAddRow,
  onDeleteRow,
  onChangeRow,
}) {
  const { t } = useTranslation();
  const [searchText, setSearchText] = useState('');

  const filteredRows = useMemo(() => {
    const keyword = searchText.trim().toLowerCase();
    if (!keyword) {
      return card.rows;
    }
    return card.rows.filter((row) => row.key.toLowerCase().includes(keyword));
  }, [card.rows, searchText]);

  return (
    <OptionModeCard
      title={title}
      description={description}
      mode={card.mode}
      onModeChange={onModeChange}
      error={card.error}
      tableContent={
        <>
          <SimpleMapTableEditor
            rows={filteredRows}
            keyLabel={t('模型名称')}
            valueLabel={t('数值')}
            keyPlaceholder={t('如 gpt-4o')}
            valuePlaceholder={valuePlaceholder}
            onAddRow={() => {
              setSearchText('');
              onAddRow();
            }}
            onDeleteRow={onDeleteRow}
            onChangeRow={onChangeRow}
            toolbarExtra={
              <Input
                showClear
                size='small'
                value={searchText}
                placeholder={t('搜索模型名称')}
                onChange={(value) => setSearchText(value)}
              />
            }
          />
        </>
      }
      jsonContent={
        <>
          <TextArea
            autosize={{ minRows: 6, maxRows: 12 }}
            value={card.rawJson}
            onChange={onJsonChange}
          />
          <div style={{ marginTop: 8 }}>
            <Button size='small' theme='light' onClick={onRebuildRows}>
              {t('用当前 JSON 刷新表格')}
            </Button>
          </div>
        </>
      }
    />
  );
}
