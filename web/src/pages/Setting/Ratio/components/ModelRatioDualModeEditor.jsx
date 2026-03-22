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
  Button,
  Popconfirm,
  Space,
  Spin,
  Switch,
  Typography,
} from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import useModelRatioOptionEditorState from '../hooks/useModelRatioOptionEditorState';
import ModelRatioOptionCard from './ModelRatioOptionCard';

const { Text } = Typography;

const OPTION_CARD_CONFIG = [
  {
    key: 'ModelPrice',
    title: '模型固定价格',
    description: '一次调用消耗多少刀，优先级大于模型倍率。',
    valuePlaceholder: '如 0.1',
  },
  {
    key: 'ModelRatio',
    title: '模型倍率',
    description: '按量计费的基础倍率，键为模型名称，值为倍率。',
    valuePlaceholder: '如 1.25',
  },
  {
    key: 'CacheRatio',
    title: '提示缓存倍率',
    description: '缓存读取的倍率，保存时继续写回官方 JSON 配置。',
    valuePlaceholder: '如 0.1',
  },
  {
    key: 'CreateCacheRatio',
    title: '缓存创建倍率',
    description: '默认用于 5m 缓存创建倍率；1h 倍率仍由后端按固定乘法计算。',
    valuePlaceholder: '如 1.25',
  },
  {
    key: 'CompletionRatio',
    title: '模型补全倍率（仅对自定义模型有效）',
    description: '仅对自定义模型有效，键为模型名称，值为补全倍率。',
    valuePlaceholder: '如 4',
  },
  {
    key: 'ImageRatio',
    title: '图片输入倍率（仅部分模型支持该计费）',
    description: '图片输入相关倍率设置。',
    valuePlaceholder: '如 2',
  },
  {
    key: 'AudioRatio',
    title: '音频倍率（仅部分模型支持该计费）',
    description: '音频输入相关倍率设置。',
    valuePlaceholder: '如 16',
  },
  {
    key: 'AudioCompletionRatio',
    title: '音频补全倍率（仅部分模型支持该计费）',
    description: '音频输出补全相关倍率设置。',
    valuePlaceholder: '如 2',
  },
];

export default function ModelRatioDualModeEditor(props) {
  const { t } = useTranslation();
  const {
    cards,
    loading,
    vendorCatalog,
    vendorCatalogLoading,
    exposeRatioEnabled,
    setExposeRatioEnabled,
    setCardMode,
    setCardJson,
    updateRow,
    addRow,
    deleteRow,
    rebuildRowsFromJson,
    submit,
    resetModelRatio,
  } = useModelRatioOptionEditorState({
    options: props.options,
    refresh: props.refresh,
    t,
  });

  return (
    <Spin spinning={loading}>
      {OPTION_CARD_CONFIG.map((config) => (
        <ModelRatioOptionCard
          key={config.key}
          title={t(config.title)}
          description={t(config.description)}
          valuePlaceholder={config.valuePlaceholder}
          card={cards[config.key]}
          vendorCatalog={vendorCatalog}
          vendorCatalogLoading={vendorCatalogLoading}
          onModeChange={(mode) => setCardMode(config.key, mode)}
          onJsonChange={(value) => setCardJson(config.key, value)}
          onRebuildRows={() => rebuildRowsFromJson(config.key)}
          onAddRow={() => addRow(config.key)}
          onDeleteRow={(rowId) => deleteRow(config.key, rowId)}
          onChangeRow={(rowId, field, value) =>
            updateRow(config.key, rowId, field, value)
          }
        />
      ))}

      <div style={{ marginBottom: 16 }}>
        <Text style={{ display: 'block', marginBottom: 8 }}>
          {t('暴露倍率接口')}
        </Text>
        <Switch
          checked={exposeRatioEnabled}
          onChange={(value) => setExposeRatioEnabled(value)}
        />
      </div>

      <Space>
        <Button type='primary' onClick={submit}>
          {t('保存模型倍率设置')}
        </Button>
        <Popconfirm
          title={t('确定重置模型倍率吗？')}
          content={t('此修改将不可逆')}
          okType='danger'
          position='top'
          onConfirm={resetModelRatio}
        >
          <Button type='danger'>{t('重置模型倍率')}</Button>
        </Popconfirm>
      </Space>
    </Spin>
  );
}
