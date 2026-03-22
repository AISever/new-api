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

import { useEffect, useMemo, useRef, useState } from 'react';
import { API, showError, showSuccess, showWarning } from '../../../../helpers';
import { getSubmitBlockingError } from '../utils/editorErrorHelpers';
import {
  parseGroupRelationOption,
  parseOrderedStringListOption,
  parseSimpleMapOption,
  parseSpecialUsableGroupOption,
  stringifyGroupRelationOption,
  stringifyOrderedStringListOption,
  stringifySimpleMapOption,
  stringifySpecialUsableGroupOption,
} from '../utils/optionTransformers';
import {
  validateSpecialUsableGroupRows,
  validateUserUsableGroupRows,
} from '../utils/rowValidators';

const createRowIdFactory = () => {
  let currentId = 0;
  return () => `ratio-editor-row-${currentId++}`;
};

const withRowIds = (rows, createId) =>
  rows.map((row) => ({
    id: createId(),
    ...row,
  }));

const withStableRowIds = (rows, previousRows, createId) =>
  rows.map((row, index) => {
    const previousRow = previousRows[index];
    const previousComparable =
      previousRow &&
      JSON.stringify(
        Object.fromEntries(
          Object.entries(previousRow).filter(([key]) => key !== 'id'),
        ),
      );
    const nextComparable = JSON.stringify(row);

    return {
      id:
        previousComparable === nextComparable && previousRow?.id
          ? previousRow.id
          : createId(),
      ...row,
    };
  });

const stripId = (rows) => rows.map(({ id, ...row }) => row);

const moveItem = (list, fromIndex, toIndex) => {
  if (
    fromIndex < 0 ||
    toIndex < 0 ||
    fromIndex >= list.length ||
    toIndex >= list.length
  ) {
    return list;
  }

  const nextList = [...list];
  const [item] = nextList.splice(fromIndex, 1);
  nextList.splice(toIndex, 0, item);
  return nextList;
};

const createEmptyCardState = (mode = 'table') => ({
  mode,
  error: '',
  rows: [],
  rawJson: '',
});

const CARD_PARSERS = {
  groupRatio: (rawJson) => parseSimpleMapOption(rawJson, 'number'),
  userUsableGroups: (rawJson) => parseSimpleMapOption(rawJson, 'string'),
  groupGroupRatio: parseGroupRelationOption,
  groupSpecialUsableGroup: parseSpecialUsableGroupOption,
  autoGroups: parseOrderedStringListOption,
};

const parseJsonError = (value, t) => {
  if (!value || value.trim() === '') {
    return '';
  }
  try {
    JSON.parse(value);
    return '';
  } catch (error) {
    return t('JSON 格式错误：{{message}}', { message: error.message });
  }
};

const validateSimpleNumericRows = (rows, t) => {
  for (const row of rows) {
    if (!row.key?.trim() && !row.value?.trim()) {
      continue;
    }
    if (!row.key?.trim()) {
      return t('表格中存在缺少键名的行');
    }
    if (!row.value?.trim()) {
      return t('表格中存在缺少数值的行');
    }
    if (!Number.isFinite(Number(row.value))) {
      return t('表格中存在非法数值：{{value}}', { value: row.value });
    }
  }
  return '';
};

const validateRelationRows = (rows, t) => {
  for (const row of rows) {
    if (!row.group?.trim() && !row.targetGroup?.trim() && !row.value?.trim()) {
      continue;
    }
    if (!row.group?.trim() || !row.targetGroup?.trim()) {
      return t('分组特殊倍率中存在缺少分组名称的行');
    }
    if (!row.value?.trim()) {
      return t('分组特殊倍率中存在缺少倍率的行');
    }
    if (!Number.isFinite(Number(row.value))) {
      return t('分组特殊倍率中存在非法数值：{{value}}', { value: row.value });
    }
  }
  return '';
};

const CARD_ROW_VALIDATORS = {
  groupRatio: validateSimpleNumericRows,
  userUsableGroups: validateUserUsableGroupRows,
  groupGroupRatio: validateRelationRows,
  groupSpecialUsableGroup: validateSpecialUsableGroupRows,
  autoGroups: () => '',
};

export default function useGroupSettingsEditorState({ options, refresh, t }) {
  const createIdRef = useRef(createRowIdFactory());
  const [loading, setLoading] = useState(false);
  const [defaultUseAutoGroup, setDefaultUseAutoGroup] = useState(false);
  const [baseline, setBaseline] = useState({
    GroupRatio: '{}',
    UserUsableGroups: '{}',
    GroupGroupRatio: '{}',
    GroupSpecialUsableGroup: '{}',
    AutoGroups: '[]',
    DefaultUseAutoGroup: false,
  });
  const [cards, setCards] = useState({
    groupRatio: createEmptyCardState(),
    userUsableGroups: createEmptyCardState(),
    groupGroupRatio: createEmptyCardState(),
    groupSpecialUsableGroup: createEmptyCardState(),
    autoGroups: createEmptyCardState(),
  });

  useEffect(() => {
    const createId = createIdRef.current;
    const nextBaseline = {
      GroupRatio: options.GroupRatio || '{}',
      UserUsableGroups: options.UserUsableGroups || '{}',
      GroupGroupRatio: options.GroupGroupRatio || '{}',
      GroupSpecialUsableGroup:
        options['group_ratio_setting.group_special_usable_group'] || '{}',
      AutoGroups: options.AutoGroups || '[]',
      DefaultUseAutoGroup: Boolean(options.DefaultUseAutoGroup),
    };

    setBaseline(nextBaseline);
    setDefaultUseAutoGroup(nextBaseline.DefaultUseAutoGroup);
    setCards({
      groupRatio: {
        mode: 'table',
        error: '',
        rawJson: nextBaseline.GroupRatio,
        rows: withRowIds(
          parseSimpleMapOption(nextBaseline.GroupRatio, 'number'),
          createId,
        ),
      },
      userUsableGroups: {
        mode: 'table',
        error: '',
        rawJson: nextBaseline.UserUsableGroups,
        rows: withRowIds(
          parseSimpleMapOption(nextBaseline.UserUsableGroups, 'string'),
          createId,
        ),
      },
      groupGroupRatio: {
        mode: 'table',
        error: '',
        rawJson: nextBaseline.GroupGroupRatio,
        rows: withRowIds(
          parseGroupRelationOption(nextBaseline.GroupGroupRatio),
          createId,
        ),
      },
      groupSpecialUsableGroup: {
        mode: 'table',
        error: '',
        rawJson: nextBaseline.GroupSpecialUsableGroup,
        rows: withRowIds(
          parseSpecialUsableGroupOption(nextBaseline.GroupSpecialUsableGroup),
          createId,
        ),
      },
      autoGroups: {
        mode: 'table',
        error: '',
        rawJson: nextBaseline.AutoGroups,
        rows: withRowIds(parseOrderedStringListOption(nextBaseline.AutoGroups), createId),
      },
    });
  }, [options]);

  const updateCard = (cardKey, updater) => {
    setCards((previous) => ({
      ...previous,
      [cardKey]:
        typeof updater === 'function'
          ? updater(previous[cardKey])
          : { ...previous[cardKey], ...updater },
    }));
  };

  const setCardMode = (cardKey, mode) => {
    updateCard(cardKey, { mode });
  };

  const setCardJson = (cardKey, rawJson) => {
    const error = parseJsonError(rawJson, t);
    if (error) {
      updateCard(cardKey, {
        rawJson,
        error,
      });
      return;
    }

    updateCard(cardKey, {
      rawJson,
      error: '',
      rows: withStableRowIds(
        CARD_PARSERS[cardKey](rawJson),
        cards[cardKey].rows,
        createIdRef.current,
      ),
    });
  };

  const replaceCardRows = (cardKey, nextRows) => {
    const validationError = CARD_ROW_VALIDATORS[cardKey]?.(nextRows, t) || '';
    updateCard(cardKey, (card) => ({
      ...card,
      error:
        card.mode === 'table'
          ? validationError
          : card.error && card.error.startsWith('JSON ')
            ? card.error
            : validationError,
      rows: nextRows,
    }));
  };

  const updateRow = (cardKey, rowId, field, value) => {
    replaceCardRows(
      cardKey,
      cards[cardKey].rows.map((row) =>
        row.id === rowId ? { ...row, [field]: value } : row,
      ),
    );
  };

  const addRow = (cardKey, row) => {
    replaceCardRows(cardKey, [
      ...cards[cardKey].rows,
      { id: createIdRef.current(), ...row },
    ]);
  };

  const deleteRow = (cardKey, rowId) => {
    replaceCardRows(
      cardKey,
      cards[cardKey].rows.filter((row) => row.id !== rowId),
    );
  };

  const moveRow = (cardKey, fromIndex, toIndex) => {
    replaceCardRows(cardKey, moveItem(cards[cardKey].rows, fromIndex, toIndex));
  };

  const rebuildCardRowsFromJson = (cardKey, parser) => {
    const card = cards[cardKey];
    const error = parseJsonError(card.rawJson, t);
    if (error) {
      updateCard(cardKey, { error });
      return false;
    }

    replaceCardRows(
      cardKey,
      withStableRowIds(parser(card.rawJson), card.rows, createIdRef.current),
    );
    updateCard(cardKey, { error: '' });
    return true;
  };

  const serializedValues = useMemo(() => {
    const currentValues = {
      GroupRatio: stringifySimpleMapOption(
        stripId(cards.groupRatio.rows),
        'number',
      ),
      UserUsableGroups: stringifySimpleMapOption(
        stripId(cards.userUsableGroups.rows),
        'string',
      ),
      GroupGroupRatio: stringifyGroupRelationOption(
        stripId(cards.groupGroupRatio.rows),
      ),
      'group_ratio_setting.group_special_usable_group':
        stringifySpecialUsableGroupOption(
          stripId(cards.groupSpecialUsableGroup.rows),
        ),
      AutoGroups: stringifyOrderedStringListOption(stripId(cards.autoGroups.rows)),
      DefaultUseAutoGroup: defaultUseAutoGroup,
    };

    return currentValues;
  }, [cards, defaultUseAutoGroup]);

  const hasPendingErrors = Object.values(cards).some((card) => Boolean(card.error));

  const hasChanges = useMemo(
    () =>
      serializedValues.GroupRatio !== baseline.GroupRatio ||
      serializedValues.UserUsableGroups !== baseline.UserUsableGroups ||
      serializedValues.GroupGroupRatio !== baseline.GroupGroupRatio ||
      serializedValues['group_ratio_setting.group_special_usable_group'] !==
        baseline.GroupSpecialUsableGroup ||
      serializedValues.AutoGroups !== baseline.AutoGroups ||
      serializedValues.DefaultUseAutoGroup !== baseline.DefaultUseAutoGroup,
    [baseline, serializedValues],
  );

  const submit = async () => {
    if (hasPendingErrors) {
      showError(getSubmitBlockingError(cards) || t('请先修正表单中的错误'));
      return;
    }
    if (!hasChanges) {
      showWarning(t('你似乎并没有修改什么'));
      return;
    }

    const payload = [
      ['GroupRatio', serializedValues.GroupRatio],
      ['UserUsableGroups', serializedValues.UserUsableGroups],
      ['GroupGroupRatio', serializedValues.GroupGroupRatio],
      [
        'group_ratio_setting.group_special_usable_group',
        serializedValues['group_ratio_setting.group_special_usable_group'],
      ],
      ['AutoGroups', serializedValues.AutoGroups],
      ['DefaultUseAutoGroup', String(serializedValues.DefaultUseAutoGroup)],
    ];

    setLoading(true);
    try {
      const results = await Promise.all(
        payload.map(([key, value]) => API.put('/api/option/', { key, value })),
      );

      for (const result of results) {
        if (!result?.data?.success) {
          throw new Error(result?.data?.message || t('保存失败'));
        }
      }

      showSuccess(t('保存成功'));
      await refresh();
    } catch (error) {
      showError(error.message || t('保存失败，请重试'));
    } finally {
      setLoading(false);
    }
  };

  return {
    cards,
    loading,
    defaultUseAutoGroup,
    setDefaultUseAutoGroup,
    hasChanges,
    setCardMode,
    setCardJson,
    updateRow,
    addRow,
    deleteRow,
    moveRow,
    rebuildCardRowsFromJson,
    submit,
  };
}
