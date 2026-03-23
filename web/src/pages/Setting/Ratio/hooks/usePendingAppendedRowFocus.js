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

import { useCallback, useEffect, useRef, useState } from 'react';
import { resolveAppendedRowFocus } from '../utils/focusTracking';

export default function usePendingAppendedRowFocus(rows) {
  const containerRef = useRef(null);
  const previousLengthRef = useRef(rows.length);
  const focusNextAddedRowRef = useRef(false);
  const [focusRowId, setFocusRowId] = useState(null);

  const requestFocusOnNextAddedRow = useCallback(() => {
    focusNextAddedRowRef.current = true;
  }, []);

  useEffect(() => {
    const nextFocusRowId = resolveAppendedRowFocus({
      shouldFocusNewRow: focusNextAddedRowRef.current,
      previousLength: previousLengthRef.current,
      rows,
    });

    if (nextFocusRowId) {
      setFocusRowId(nextFocusRowId);
      focusNextAddedRowRef.current = false;
    }

    previousLengthRef.current = rows.length;
  }, [rows]);

  useEffect(() => {
    if (!focusRowId || !containerRef.current) {
      return;
    }

    const focusTarget = containerRef.current.querySelector(
      `[data-row-focus-id="${focusRowId}"] input, ` +
        `[data-row-focus-id="${focusRowId}"] textarea, ` +
        `[data-row-focus-id="${focusRowId}"] [role="combobox"]`,
    );

    if (!focusTarget) {
      return;
    }

    focusTarget.focus();
    focusTarget.scrollIntoView({
      block: 'nearest',
      inline: 'nearest',
    });
  }, [focusRowId]);

  return {
    containerRef,
    focusRowId,
    requestFocusOnNextAddedRow,
  };
}
