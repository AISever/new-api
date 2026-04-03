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
import { DEFAULT_OPTION_EDITOR_MODE } from '../utils/editorMode';
import {
  isGroupVisibleInUserUsableRows,
  renameLinkedUserUsableGroup,
  setGroupVisibilityInUserUsableRows,
} from '../utils/groupVisibilityLinkage';
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
  validateRelationRows,
  validateSimpleNumericRows,
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

const createEmptyCardState = (mode = DEFAULT_OPTION_EDITOR_MODE) => ({
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

const CARD_ROW_VALIDATORS = {
  groupRatio: validateSimpleNumericRows,
  userUsableGroups: validateUserUsableGroupRows,
  groupGroupRatio: validateRelationRows,
  groupSpecialUsableGroup: validateSpecialUsableGroupRows,
  autoGroups: () => '',
};

const CARD_ROW_SERIALIZERS = {
  groupRatio: (rows) => stringifySimpleMapOption(stripId(rows), 'number'),
  userUsableGroups: (rows) => stringifySimpleMapOption(stripId(rows), 'string'),
  groupGroupRatio: (rows) => stringifyGroupRelationOption(stripId(rows)),
  groupSpecialUsableGroup: (rows) =>
    stringifySpecialUsableGroupOption(stripId(rows)),
  autoGroups: (rows) => stringifyOrderedStringListOption(stripId(rows)),
};

const buildUpdatedCardState = (cardKey, card, rows, t) => {
  const validationError = CARD_ROW_VALIDATORS[cardKey]?.(rows, t) || '';
  return {
    ...card,
    error:
      card.mode === 'table'
        ? validationError
        : card.error && card.error.startsWith('JSON ')
          ? card.error
          : validationError,
    rawJson: CARD_ROW_SERIALIZERS[cardKey](rows),
    rows,
  };
};

export default function useGroupSettingsEditorState({ options, refresh, t }) {
  const createIdRef = useRef(createRowIdFactory());
  const linkedVisibleGroupKeyRef = useRef({});
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
    linkedVisibleGroupKeyRef.current = {};
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
        mode: DEFAULT_OPTION_EDITOR_MODE,
        error: '',
        rawJson: nextBaseline.GroupRatio,
        rows: withRowIds(
          parseSimpleMapOption(nextBaseline.GroupRatio, 'number'),
          createId,
        ),
      },
      userUsableGroups: {
        mode: DEFAULT_OPTION_EDITOR_MODE,
        error: '',
        rawJson: nextBaseline.UserUsableGroups,
        rows: withRowIds(
          parseSimpleMapOption(nextBaseline.UserUsableGroups, 'string'),
          createId,
        ),
      },
      groupGroupRatio: {
        mode: DEFAULT_OPTION_EDITOR_MODE,
        error: '',
        rawJson: nextBaseline.GroupGroupRatio,
        rows: withRowIds(
          parseGroupRelationOption(nextBaseline.GroupGroupRatio),
          createId,
        ),
      },
      groupSpecialUsableGroup: {
        mode: DEFAULT_OPTION_EDITOR_MODE,
        error: '',
        rawJson: nextBaseline.GroupSpecialUsableGroup,
        rows: withRowIds(
          parseSpecialUsableGroupOption(nextBaseline.GroupSpecialUsableGroup),
          createId,
        ),
      },
      autoGroups: {
        mode: DEFAULT_OPTION_EDITOR_MODE,
        error: '',
        rawJson: nextBaseline.AutoGroups,
        rows: withRowIds(
          parseOrderedStringListOption(nextBaseline.AutoGroups),
          createId,
        ),
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
    updateCard(cardKey, (card) =>
      buildUpdatedCardState(cardKey, card, nextRows, t),
    );
  };

  const hasUserUsableGroupsJsonError =
    cards.userUsableGroups.mode === 'json' &&
    Boolean(cards.userUsableGroups.error?.startsWith('JSON '));

  const syncUserUsableGroupRows = (previousRows, nextRows) =>
    withStableRowIds(nextRows, previousRows, createIdRef.current);

  const renameGroupVisibilityLinkage = ({ rowId, previousKey, nextKey }) => {
    const normalizedPreviousKey = String(previousKey ?? '').trim();
    const normalizedNextKey = String(nextKey ?? '').trim();

    if (!normalizedPreviousKey || normalizedPreviousKey === normalizedNextKey) {
      return false;
    }

    if (
      hasUserUsableGroupsJsonError &&
      isGroupVisibleInUserUsableRows(
        stripId(cards.userUsableGroups.rows),
        normalizedPreviousKey,
      )
    ) {
      showError(t('请先修正用户可选分组中的 JSON 错误'));
      return true;
    }

    setCards((previous) => {
      const nextGroupRows = previous.groupRatio.rows.map((row) =>
        row.id === rowId ? { ...row, key: nextKey } : row,
      );

      const nextUserUsableRows = syncUserUsableGroupRows(
        previous.userUsableGroups.rows,
        renameLinkedUserUsableGroup(
          stripId(previous.userUsableGroups.rows),
          normalizedPreviousKey,
          normalizedNextKey,
        ),
      );

      return {
        ...previous,
        groupRatio: buildUpdatedCardState(
          'groupRatio',
          previous.groupRatio,
          nextGroupRows,
          t,
        ),
        userUsableGroups: buildUpdatedCardState(
          'userUsableGroups',
          previous.userUsableGroups,
          nextUserUsableRows,
          t,
        ),
      };
    });

    if (normalizedNextKey) {
      linkedVisibleGroupKeyRef.current[rowId] = normalizedNextKey;
    }

    return true;
  };

  const updateRow = (cardKey, rowId, field, value) => {
    if (cardKey === 'groupRatio' && field === 'key') {
      const previousRow = cards.groupRatio.rows.find((row) => row.id === rowId);
      const rememberedKey = linkedVisibleGroupKeyRef.current[rowId];
      const sourceKey = String(rememberedKey ?? previousRow?.key ?? '').trim();

      if (
        sourceKey &&
        isGroupVisibleInUserUsableRows(
          stripId(cards.userUsableGroups.rows),
          sourceKey,
        )
      ) {
        linkedVisibleGroupKeyRef.current[rowId] = sourceKey;
      }

      if (
        previousRow &&
        renameGroupVisibilityLinkage({
          rowId,
          previousKey: sourceKey,
          nextKey: value,
        })
      ) {
        return;
      }
    }

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

  const setGroupVisibility = (rowId, visible) => {
    const targetRow = cards.groupRatio.rows.find((row) => row.id === rowId);
    const groupKey = String(targetRow?.key ?? '').trim();

    if (!groupKey) {
      return;
    }

    if (hasUserUsableGroupsJsonError) {
      showError(t('请先修正用户可选分组中的 JSON 错误'));
      return;
    }

    linkedVisibleGroupKeyRef.current[rowId] = groupKey;

    setCards((previous) => {
      const nextUserUsableRows = syncUserUsableGroupRows(
        previous.userUsableGroups.rows,
        setGroupVisibilityInUserUsableRows(
          stripId(previous.userUsableGroups.rows),
          groupKey,
          visible,
        ),
      );

      return {
        ...previous,
        userUsableGroups: buildUpdatedCardState(
          'userUsableGroups',
          previous.userUsableGroups,
          nextUserUsableRows,
          t,
        ),
      };
    });

    if (!visible) {
      delete linkedVisibleGroupKeyRef.current[rowId];
    }
  };

  const isGroupVisible = (groupKey) =>
    isGroupVisibleInUserUsableRows(stripId(cards.userUsableGroups.rows), groupKey);

  const deleteRow = (cardKey, rowId) => {
    if (cardKey === 'groupRatio') {
      delete linkedVisibleGroupKeyRef.current[rowId];
    }
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
    isGroupVisible,
    setGroupVisibility,
    hasUserUsableGroupsJsonError,
    rebuildCardRowsFromJson,
    submit,
  };
}
