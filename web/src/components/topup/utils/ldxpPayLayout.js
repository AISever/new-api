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
      summaryActionsStacked: false,
      mobileUsesExternalPayFlow: true,
      mobileKeepsCurrentPage: true,
      summaryActionsClassName: 'grid grid-cols-1 gap-2',
      iframeActionsClassName: 'grid grid-cols-1 gap-2',
      iframeHeight: 620,
      summaryGridClassName: 'grid grid-cols-1 gap-2',
      heroCardClassName: '!rounded-2xl shadow-sm border-0',
      iframeWrapperClassName: 'rounded-2xl overflow-hidden border bg-[var(--semi-color-bg-0)]',
      placeholderClassName:
        'min-h-[220px] rounded-2xl border border-dashed border-[var(--semi-color-border)] bg-[var(--semi-color-fill-0)] px-4 py-6 flex items-center justify-center text-center',
    };
  }

  return {
    pageClassName: 'p-4 md:p-6',
    shellClassName: 'w-full max-w-[1600px] mx-auto',
    contentGridClassName:
      'grid grid-cols-1 gap-4 2xl:grid-cols-[320px_minmax(0,1fr)] items-start',
    summaryActionsStacked: true,
    mobileUsesExternalPayFlow: false,
    mobileKeepsCurrentPage: false,
    summaryActionsClassName: 'flex flex-col gap-2',
    iframeActionsClassName: 'flex flex-wrap gap-2',
    iframeHeight: 860,
    summaryGridClassName: 'grid gap-2 [grid-template-columns:repeat(auto-fit,minmax(220px,1fr))]',
    heroCardClassName: '!rounded-2xl shadow-sm border-0',
    iframeWrapperClassName: 'rounded-2xl overflow-hidden border bg-[var(--semi-color-bg-0)]',
    placeholderClassName:
      'min-h-[320px] rounded-2xl border border-dashed border-[var(--semi-color-border)] bg-[var(--semi-color-fill-0)] px-6 py-8 flex items-center justify-center text-center',
  };
}
