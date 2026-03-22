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
import { parseSimpleMapOption, stringifySimpleMapOption } from '../utils/optionTransformers';

const createRowIdFactory = () => {
  let currentId = 0;
  return () => `model-ratio-row-${currentId++}`;
};

const OPTION_KEYS = [
  'ModelPrice',
  'ModelRatio',
  'CacheRatio',
  'CreateCacheRatio',
  'CompletionRatio',
  'ImageRatio',
  'AudioRatio',
  'AudioCompletionRatio',
];

const createEmptyCardState = () => ({
  mode: 'table',
  error: '',
  rows: [],
  rawJson: '{}',
});

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

const validateNumericRows = (rows, t) => {
  for (const row of rows) {
    if (!row.key?.trim() && !row.value?.trim()) {
      continue;
    }
    if (!row.key?.trim()) {
      return t('表格中存在缺少模型名称的行');
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

const stripId = (rows) => rows.map(({ id, ...row }) => row);

export default function useModelRatioOptionEditorState({ options, refresh, t }) {
  const createIdRef = useRef(createRowIdFactory());
  const [loading, setLoading] = useState(false);
  const [baseline, setBaseline] = useState({});
  const [cards, setCards] = useState(
    OPTION_KEYS.reduce((accumulator, key) => {
      accumulator[key] = createEmptyCardState();
      return accumulator;
    }, {}),
  );
  const [exposeRatioEnabled, setExposeRatioEnabled] = useState(false);

  useEffect(() => {
    const createId = createIdRef.current;
    const nextBaseline = OPTION_KEYS.reduce((accumulator, key) => {
      accumulator[key] = options[key] || '{}';
      return accumulator;
    }, {});
    nextBaseline.ExposeRatioEnabled = Boolean(options.ExposeRatioEnabled);

    setBaseline(nextBaseline);
    setExposeRatioEnabled(nextBaseline.ExposeRatioEnabled);
    setCards(
      OPTION_KEYS.reduce((accumulator, key) => {
        accumulator[key] = {
          mode: 'table',
          error: '',
          rawJson: nextBaseline[key],
          rows: withRowIds(parseSimpleMapOption(nextBaseline[key], 'number'), createId),
        };
        return accumulator;
      }, {}),
    );
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

  const setCardMode = (cardKey, mode) => updateCard(cardKey, { mode });

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
        parseSimpleMapOption(rawJson, 'number'),
        cards[cardKey].rows,
        createIdRef.current,
      ),
    });
  };

  const updateRow = (cardKey, rowId, field, value) => {
    updateCard(cardKey, (card) => ({
      ...card,
      error: validateNumericRows(
        card.rows.map((row) =>
          row.id === rowId ? { ...row, [field]: value } : row,
        ),
        t,
      ),
      rows: card.rows.map((row) =>
        row.id === rowId ? { ...row, [field]: value } : row,
      ),
    }));
  };

  const addRow = (cardKey) => {
    updateCard(cardKey, (card) => ({
      ...card,
      rows: [...card.rows, { id: createIdRef.current(), key: '', value: '' }],
    }));
  };

  const deleteRow = (cardKey, rowId) => {
    updateCard(cardKey, (card) => ({
      ...card,
      rows: card.rows.filter((row) => row.id !== rowId),
    }));
  };

  const rebuildRowsFromJson = (cardKey) => {
    const card = cards[cardKey];
    const error = parseJsonError(card.rawJson, t);
    if (error) {
      updateCard(cardKey, { error });
      return false;
    }

    updateCard(cardKey, {
      error: '',
      rows: withStableRowIds(
        parseSimpleMapOption(card.rawJson, 'number'),
        card.rows,
        createIdRef.current,
      ),
    });
    return true;
  };

  const serializedValues = useMemo(
    () =>
      OPTION_KEYS.reduce((accumulator, key) => {
        accumulator[key] = stringifySimpleMapOption(stripId(cards[key].rows), 'number');
        return accumulator;
      }, {}),
    [cards],
  );

  const hasPendingErrors = Object.values(cards).some((card) => Boolean(card.error));

  const hasChanges = useMemo(
    () =>
      OPTION_KEYS.some((key) => serializedValues[key] !== baseline[key]) ||
      exposeRatioEnabled !== baseline.ExposeRatioEnabled,
    [baseline, exposeRatioEnabled, serializedValues],
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

    setLoading(true);
    try {
      const payload = [
        ...OPTION_KEYS.map((key) => [key, serializedValues[key]]),
        ['ExposeRatioEnabled', String(exposeRatioEnabled)],
      ];

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

  const resetModelRatio = async () => {
    try {
      const result = await API.post('/api/option/rest_model_ratio');
      if (!result?.data?.success) {
        throw new Error(result?.data?.message || t('重置失败'));
      }
      showSuccess(result.data.message);
      await refresh();
    } catch (error) {
      showError(error.message || t('重置失败'));
    }
  };

  return {
    cards,
    loading,
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
  };
}
