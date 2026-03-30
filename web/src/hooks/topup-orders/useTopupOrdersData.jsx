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

import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { API, showError } from '../../helpers';
import { ITEMS_PER_PAGE } from '../../constants';
import { useTableCompactMode } from '../common/useTableCompactMode';

export const useTopupOrdersData = () => {
  const { t } = useTranslation();
  const [compactMode, setCompactMode] = useTableCompactMode('topup-orders');
  const [topupOrders, setTopupOrders] = useState([]);
  const [loading, setLoading] = useState(true);
  const [searching, setSearching] = useState(false);
  const [activePage, setActivePage] = useState(1);
  const [pageSize, setPageSize] = useState(ITEMS_PER_PAGE);
  const [totalCount, setTotalCount] = useState(0);
  const [formApi, setFormApi] = useState(null);
  const [detailOrder, setDetailOrder] = useState(null);

  const formInitValues = {
    searchKeyword: '',
    status: undefined,
    paymentMethod: undefined,
  };

  const getFormValues = () => {
    const formValues = formApi ? formApi.getValues() : {};
    return {
      searchKeyword: formValues.searchKeyword || '',
      status: formValues.status || '',
      paymentMethod: formValues.paymentMethod || '',
    };
  };

  const buildQueryString = (page, size, filters) => {
    const params = new URLSearchParams();
    params.set('p', String(page));
    params.set('page_size', String(size));
    if (filters.searchKeyword) {
      params.set('keyword', filters.searchKeyword);
    }
    if (filters.status) {
      params.set('status', filters.status);
    }
    if (filters.paymentMethod) {
      params.set('payment_method', filters.paymentMethod);
    }
    return params.toString();
  };

  const loadTopupOrders = async (page = 1, size = pageSize) => {
    setLoading(true);
    try {
      const query = buildQueryString(page, size, getFormValues());
      const res = await API.get(`/api/user/topup?${query}`);
      const { success, message, data } = res.data;
      if (success) {
        setTopupOrders(data.items || []);
        setActivePage(data.page || 1);
        setTotalCount(data.total || 0);
      } else {
        showError(message || t('加载失败'));
      }
    } catch (error) {
      showError(error.message || t('请求失败'));
    } finally {
      setLoading(false);
    }
  };

  const searchTopupOrders = async (page = 1, size = pageSize) => {
    setSearching(true);
    try {
      await loadTopupOrders(page, size);
    } finally {
      setSearching(false);
    }
  };

  const refresh = async () => {
    await loadTopupOrders(activePage, pageSize);
  };

  const handlePageChange = (page) => {
    setActivePage(page);
    loadTopupOrders(page, pageSize);
  };

  const handlePageSizeChange = (size) => {
    localStorage.setItem('page-size', `${size}`);
    setPageSize(size);
    setActivePage(1);
    loadTopupOrders(1, size);
  };

  const openDetail = (order) => {
    setDetailOrder(order);
  };

  const closeDetail = () => {
    setDetailOrder(null);
  };

  useEffect(() => {
    loadTopupOrders(1, pageSize);
  }, []);

  return {
    topupOrders,
    loading,
    searching,
    compactMode,
    setCompactMode,
    activePage,
    pageSize,
    totalCount,
    formInitValues,
    setFormApi,
    loadTopupOrders,
    searchTopupOrders,
    refresh,
    handlePageChange,
    handlePageSizeChange,
    detailOrder,
    openDetail,
    closeDetail,
    t,
  };
};

