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

import React, { useEffect, useMemo, useState } from 'react';
import { Button, Input, Select, TextArea } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import OptionModeCard from './OptionModeCard';
import SimpleMapTableEditor from './SimpleMapTableEditor';
import {
  ALL_VENDOR_FILTER,
  UNKNOWN_VENDOR_FILTER,
  buildVendorFilterOptions,
  resolveModelVendor,
} from '../utils/vendorCatalog';

export default function ModelRatioOptionCard({
  title,
  description,
  card,
  valuePlaceholder,
  vendorCatalog,
  vendorCatalogLoading,
  onModeChange,
  onJsonChange,
  onRebuildRows,
  onAddRow,
  onDeleteRow,
  onChangeRow,
}) {
  const { t } = useTranslation();
  const [searchText, setSearchText] = useState('');
  const [vendorFilter, setVendorFilter] = useState(ALL_VENDOR_FILTER);

  const vendorOptions = useMemo(
    () => buildVendorFilterOptions(card.rows, vendorCatalog, t),
    [card.rows, t, vendorCatalog],
  );

  useEffect(() => {
    if (!vendorOptions.some((option) => option.value === vendorFilter)) {
      setVendorFilter(ALL_VENDOR_FILTER);
    }
  }, [vendorFilter, vendorOptions]);

  const filteredRows = useMemo(() => {
    const keyword = searchText.trim().toLowerCase();
    return card.rows.filter((row) => {
      const key = String(row.key || '');
      const resolvedVendor = resolveModelVendor(key, vendorCatalog);
      const matchesSearch = !keyword || key.toLowerCase().includes(keyword);
      const matchesVendor =
        vendorFilter === ALL_VENDOR_FILTER ||
        (vendorFilter === UNKNOWN_VENDOR_FILTER
          ? !resolvedVendor?.name
          : resolvedVendor?.name === vendorFilter);

      return matchesSearch && matchesVendor;
    });
  }, [card.rows, searchText, vendorCatalog, vendorFilter]);

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
              setVendorFilter(ALL_VENDOR_FILTER);
              onAddRow();
            }}
            onDeleteRow={onDeleteRow}
            onChangeRow={onChangeRow}
            toolbarExtra={
              <div
                style={{
                  display: 'flex',
                  gap: 8,
                  width: '100%',
                  flexWrap: 'wrap',
                }}
              >
                <Input
                  showClear
                  size='small'
                  value={searchText}
                  placeholder={t('搜索模型名称')}
                  onChange={(value) => setSearchText(value)}
                  style={{ flex: '1 1 220px' }}
                />
                <Select
                  size='small'
                  value={vendorFilter}
                  optionList={vendorOptions}
                  loading={vendorCatalogLoading}
                  onChange={(value) => setVendorFilter(value || ALL_VENDOR_FILTER)}
                  style={{ width: 180 }}
                />
              </div>
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
