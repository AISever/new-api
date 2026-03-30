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

export function getLdxpPayLayout(isMobile) {
  if (isMobile) {
    return {
      pageClassName: 'p-3',
      shellClassName: 'w-full max-w-full mx-auto',
      contentGridClassName: 'grid grid-cols-1 gap-3',
      mobileUsesExternalPayFlow: true,
      sidebarWidth: null,
      placeholderHeight: 220,
      summaryActionsClassName: 'grid grid-cols-1 gap-2',
      iframeActionsClassName: 'grid grid-cols-1 gap-2',
      iframeHeight: 620,
      summaryGridClassName:
        'overflow-hidden rounded-2xl border border-[var(--semi-color-border)] bg-[var(--semi-color-fill-0)] divide-y divide-[var(--semi-color-border)]',
      summaryRowClassName: 'px-4 py-3 space-y-1',
      summaryValueClassName:
        'text-sm font-medium break-all text-[var(--semi-color-text-0)]',
      sideCardClassName: '!rounded-2xl shadow-sm border-0',
      mainCardClassName: '!rounded-2xl shadow-sm border-0',
      workspaceSplitClassName: 'grid grid-cols-1 gap-3',
      iframeWrapperClassName: 'rounded-2xl overflow-hidden border bg-[var(--semi-color-bg-0)]',
      placeholderClassName:
        'rounded-2xl border border-dashed border-[var(--semi-color-border)] bg-[var(--semi-color-fill-0)] px-4 py-6 flex items-center justify-center text-left',
      workspaceAsideClassName:
        'rounded-2xl border border-[var(--semi-color-border)] bg-[var(--semi-color-fill-0)] px-4 py-4',
    };
  }

  return {
    pageClassName: 'px-4 py-5 md:px-6 md:py-6',
    shellClassName: 'w-full max-w-[1480px] mx-auto',
    contentGridClassName:
      'grid grid-cols-1 gap-5 xl:grid-cols-[minmax(360px,380px)_minmax(0,1fr)] items-start',
    mobileUsesExternalPayFlow: false,
    sidebarWidth: 380,
    placeholderHeight: 260,
    summaryActionsClassName: 'flex flex-col gap-2 pt-1',
    iframeActionsClassName: 'flex flex-wrap gap-2',
    iframeHeight: 780,
    summaryGridClassName:
      'overflow-hidden rounded-2xl border border-[var(--semi-color-border)] bg-[var(--semi-color-fill-0)] divide-y divide-[var(--semi-color-border)]',
    summaryRowClassName: 'flex items-start justify-between gap-4 px-4 py-3',
    summaryValueClassName:
      'max-w-[68%] text-right text-sm font-medium break-all text-[var(--semi-color-text-0)]',
    sideCardClassName: '!rounded-2xl shadow-sm border-0 xl:sticky xl:top-24',
    mainCardClassName: '!rounded-2xl shadow-sm border-0',
    workspaceSplitClassName:
      'grid grid-cols-1 gap-4 2xl:grid-cols-[minmax(0,1.35fr)_minmax(280px,0.65fr)]',
    iframeWrapperClassName: 'rounded-2xl overflow-hidden border bg-[var(--semi-color-bg-0)]',
    placeholderClassName:
      'rounded-2xl border border-dashed border-[var(--semi-color-border)] bg-[var(--semi-color-fill-0)] px-6 py-7 flex items-center justify-center text-left',
    workspaceAsideClassName:
      'rounded-2xl border border-[var(--semi-color-border)] bg-[var(--semi-color-fill-0)] px-5 py-5',
  };
}
