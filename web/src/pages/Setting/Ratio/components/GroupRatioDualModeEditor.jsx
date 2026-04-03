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
  Spin,
  Switch,
  Tag,
  TextArea,
  Typography,
} from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import useGroupSettingsEditorState from '../hooks/useGroupSettingsEditorState';
import {
  parseGroupRelationOption,
  parseOrderedStringListOption,
  parseSimpleMapOption,
  parseSpecialUsableGroupOption,
} from '../utils/optionTransformers';
import GroupRatioTableEditor from './GroupRatioTableEditor';
import OptionModeCard from './OptionModeCard';
import OrderedListTableEditor from './OrderedListTableEditor';
import OperationRuleTableEditor from './OperationRuleTableEditor';
import RelationTableEditor from './RelationTableEditor';
import SimpleMapTableEditor from './SimpleMapTableEditor';

const { Text } = Typography;

export default function GroupRatioDualModeEditor(props) {
  const { t } = useTranslation();
  const {
    cards,
    loading,
    defaultUseAutoGroup,
    setDefaultUseAutoGroup,
    setCardMode,
    setCardJson,
    updateRow,
    addRow,
    deleteRow,
    moveRow,
    isGroupVisible,
    setGroupVisibility,
    hasUserUsableGroupsJsonError,
    rebuildCardRowsFromJson,
    submit,
  } = useGroupSettingsEditorState({
    options: props.options,
    refresh: props.refresh,
    t,
  });
  const isUserUsableGroupLinked = (groupKey) =>
    cards.groupRatio.rows.some(
      (row) => String(row.key ?? '').trim() === String(groupKey ?? '').trim(),
    );

  return (
    <Spin spinning={loading}>
      <OptionModeCard
        title={t('分组倍率')}
        description={t('维护基础分组倍率。键为分组名称，值为倍率。')}
        mode={cards.groupRatio.mode}
        onModeChange={(mode) => setCardMode('groupRatio', mode)}
        error={cards.groupRatio.error}
        tableContent={
          <>
            <GroupRatioTableEditor
              rows={cards.groupRatio.rows}
              onAddRow={() => addRow('groupRatio', { key: '', value: '' })}
              onDeleteRow={(rowId) => deleteRow('groupRatio', rowId)}
              onChangeRow={(rowId, field, value) =>
                updateRow('groupRatio', rowId, field, value)
              }
              onToggleVisible={setGroupVisibility}
              isGroupVisible={isGroupVisible}
              visibilityDisabled={hasUserUsableGroupsJsonError}
            />
            <Text
              type='tertiary'
              size='small'
              style={{ display: 'block', marginTop: 8 }}
            >
              {t('显示开关会同步维护“用户可选分组”；展示名称仍可在下方单独编辑。')}
            </Text>
            {hasUserUsableGroupsJsonError ? (
              <Text
                type='warning'
                size='small'
                style={{ display: 'block', marginTop: 4 }}
              >
                {t('用户可选分组 JSON 有错误时，显示联动将暂时禁用。')}
              </Text>
            ) : null}
          </>
        }
        jsonContent={
          <>
            <TextArea
              autosize={{ minRows: 6, maxRows: 12 }}
              value={cards.groupRatio.rawJson}
              onChange={(value) => setCardJson('groupRatio', value)}
            />
            <div style={{ marginTop: 8 }}>
              <Button
                size='small'
                theme='light'
                onClick={() =>
                  rebuildCardRowsFromJson('groupRatio', (rawJson) =>
                    parseSimpleMapOption(rawJson, 'number'),
                  )
                }
              >
                {t('用当前 JSON 刷新表格')}
              </Button>
            </div>
          </>
        }
      />

      <OptionModeCard
        title={t('用户可选分组')}
        description={t('维护用户在创建令牌时可见的分组与描述。')}
        mode={cards.userUsableGroups.mode}
        onModeChange={(mode) => setCardMode('userUsableGroups', mode)}
        error={cards.userUsableGroups.error}
        tableContent={
          <>
            <SimpleMapTableEditor
              rows={cards.userUsableGroups.rows}
              keyLabel={t('分组名称')}
              valueLabel={t('展示名称')}
              keyPlaceholder={t('如 vip')}
              valuePlaceholder={t('如 VIP 用户')}
              onAddRow={() => addRow('userUsableGroups', { key: '', value: '' })}
              onDeleteRow={(rowId) => deleteRow('userUsableGroups', rowId)}
              onChangeRow={(rowId, field, value) =>
                updateRow('userUsableGroups', rowId, field, value)
              }
              renderRowActionPrefix={(record) =>
                isUserUsableGroupLinked(record.key) ? (
                  <Tag size='small' color='blue'>
                    {t('联动')}
                  </Tag>
                ) : null
              }
            />
            <Text
              type='tertiary'
              size='small'
              style={{ display: 'block', marginTop: 8 }}
            >
              {t('带“联动”标记的分组由上方分组倍率中的显示开关控制。')}
            </Text>
          </>
        }
        jsonContent={
          <>
            <TextArea
              autosize={{ minRows: 6, maxRows: 12 }}
              value={cards.userUsableGroups.rawJson}
              onChange={(value) => setCardJson('userUsableGroups', value)}
            />
            <div style={{ marginTop: 8 }}>
              <Button
                size='small'
                theme='light'
                onClick={() =>
                  rebuildCardRowsFromJson('userUsableGroups', (rawJson) =>
                    parseSimpleMapOption(rawJson, 'string'),
                  )
                }
              >
                {t('用当前 JSON 刷新表格')}
              </Button>
            </div>
          </>
        }
      />

      <OptionModeCard
        title={t('分组特殊倍率')}
        description={t('为指定用户分组覆盖使用特定分组时的倍率。')}
        mode={cards.groupGroupRatio.mode}
        onModeChange={(mode) => setCardMode('groupGroupRatio', mode)}
        error={cards.groupGroupRatio.error}
        tableContent={
          <RelationTableEditor
            rows={cards.groupGroupRatio.rows}
            onAddRow={() =>
              addRow('groupGroupRatio', {
                group: '',
                targetGroup: '',
                value: '',
              })
            }
            onDeleteRow={(rowId) => deleteRow('groupGroupRatio', rowId)}
            onChangeRow={(rowId, field, value) =>
              updateRow('groupGroupRatio', rowId, field, value)
            }
          />
        }
        jsonContent={
          <>
            <TextArea
              autosize={{ minRows: 6, maxRows: 12 }}
              value={cards.groupGroupRatio.rawJson}
              onChange={(value) => setCardJson('groupGroupRatio', value)}
            />
            <div style={{ marginTop: 8 }}>
              <Button
                size='small'
                theme='light'
                onClick={() =>
                  rebuildCardRowsFromJson(
                    'groupGroupRatio',
                    parseGroupRelationOption,
                  )
                }
              >
                {t('用当前 JSON 刷新表格')}
              </Button>
            </div>
          </>
        }
      />

      <OptionModeCard
        title={t('分组特殊可用分组')}
        description={t(
          '为指定用户分组追加或移除可用分组。保存时会自动转换为 +: / -: 规则键。',
        )}
        mode={cards.groupSpecialUsableGroup.mode}
        onModeChange={(mode) => setCardMode('groupSpecialUsableGroup', mode)}
        error={cards.groupSpecialUsableGroup.error}
        tableContent={
          <OperationRuleTableEditor
            rows={cards.groupSpecialUsableGroup.rows}
            onAddRow={() =>
              addRow('groupSpecialUsableGroup', {
                group: '',
                action: 'add',
                targetGroup: '',
                description: '',
              })
            }
            onDeleteRow={(rowId) => deleteRow('groupSpecialUsableGroup', rowId)}
            onChangeRow={(rowId, field, value) =>
              updateRow('groupSpecialUsableGroup', rowId, field, value)
            }
          />
        }
        jsonContent={
          <>
            <TextArea
              autosize={{ minRows: 6, maxRows: 12 }}
              value={cards.groupSpecialUsableGroup.rawJson}
              onChange={(value) => setCardJson('groupSpecialUsableGroup', value)}
            />
            <div style={{ marginTop: 8 }}>
              <Button
                size='small'
                theme='light'
                onClick={() =>
                  rebuildCardRowsFromJson(
                    'groupSpecialUsableGroup',
                    parseSpecialUsableGroupOption,
                  )
                }
              >
                {t('用当前 JSON 刷新表格')}
              </Button>
            </div>
          </>
        }
      />

      <OptionModeCard
        title={t('自动分组 auto')}
        description={t('按顺序维护 auto 分组会尝试的分组列表。')}
        mode={cards.autoGroups.mode}
        onModeChange={(mode) => setCardMode('autoGroups', mode)}
        error={cards.autoGroups.error}
        tableContent={
          <OrderedListTableEditor
            rows={cards.autoGroups.rows}
            onAddRow={() => addRow('autoGroups', { value: '' })}
            onDeleteRow={(rowId) => deleteRow('autoGroups', rowId)}
            onChangeRow={(rowId, field, value) =>
              updateRow('autoGroups', rowId, field, value)
            }
            onMoveRow={(fromIndex, toIndex) =>
              moveRow('autoGroups', fromIndex, toIndex)
            }
          />
        }
        jsonContent={
          <>
            <TextArea
              autosize={{ minRows: 6, maxRows: 12 }}
              value={cards.autoGroups.rawJson}
              onChange={(value) => setCardJson('autoGroups', value)}
            />
            <div style={{ marginTop: 8 }}>
              <Button
                size='small'
                theme='light'
                onClick={() =>
                  rebuildCardRowsFromJson(
                    'autoGroups',
                    parseOrderedStringListOption,
                  )
                }
              >
                {t('用当前 JSON 刷新表格')}
              </Button>
            </div>
          </>
        }
        footer={
          <div style={{ marginTop: 16 }}>
            <Text style={{ display: 'block', marginBottom: 8 }}>
              {t(
                '创建令牌默认选择 auto 分组，初始令牌也将设为 auto（否则留空，为用户默认分组）',
              )}
            </Text>
            <Switch
              checked={defaultUseAutoGroup}
              onChange={(value) => setDefaultUseAutoGroup(value)}
            />
          </div>
        }
      />

      <Button type='primary' onClick={submit}>
        {t('保存分组相关设置')}
      </Button>
    </Spin>
  );
}
