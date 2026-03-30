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
import CardPro from '../../common/ui/CardPro';
import TopupOrdersActions from './TopupOrdersActions';
import TopupOrdersDescription from './TopupOrdersDescription';
import TopupOrdersFilters from './TopupOrdersFilters';
import TopupOrdersTable from './TopupOrdersTable';
import TopupOrderDetailModal from './modals/TopupOrderDetailModal';
import { useTopupOrdersData } from '../../../hooks/topup-orders/useTopupOrdersData';
import { useIsMobile } from '../../../hooks/common/useIsMobile';
import { createCardProPagination } from '../../../helpers/utils';

const TopupOrdersPage = () => {
  const topupOrdersData = useTopupOrdersData();
  const isMobile = useIsMobile();

  const {
    compactMode,
    setCompactMode,
    refresh,
    loading,
    searching,
    formInitValues,
    setFormApi,
    searchTopupOrders,
    loadTopupOrders,
    pageSize,
    activePage,
    totalCount,
    handlePageChange,
    handlePageSizeChange,
    detailOrder,
    closeDetail,
    t,
  } = topupOrdersData;

  return (
    <>
      <TopupOrderDetailModal
        visible={Boolean(detailOrder)}
        onCancel={closeDetail}
        order={detailOrder}
        t={t}
      />

      <CardPro
        type='type1'
        descriptionArea={
          <TopupOrdersDescription
            compactMode={compactMode}
            setCompactMode={setCompactMode}
            t={t}
          />
        }
        actionsArea={[
          <TopupOrdersActions refresh={refresh} loading={loading} t={t} />,
          <TopupOrdersFilters
            formInitValues={formInitValues}
            setFormApi={setFormApi}
            searchTopupOrders={searchTopupOrders}
            loadTopupOrders={loadTopupOrders}
            pageSize={pageSize}
            loading={loading}
            searching={searching}
            t={t}
          />,
        ]}
        paginationArea={createCardProPagination({
          currentPage: activePage,
          pageSize,
          total: totalCount,
          onPageChange: handlePageChange,
          onPageSizeChange: handlePageSizeChange,
          isMobile,
          t,
        })}
        t={t}
      >
        <TopupOrdersTable {...topupOrdersData} />
      </CardPro>
    </>
  );
};

export default TopupOrdersPage;

