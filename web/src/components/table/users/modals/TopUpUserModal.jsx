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

import React, { useEffect, useState } from 'react';
import { InputNumber, Modal, Typography } from '@douyinfe/semi-ui';
import { getQuotaPerUnit, renderQuota } from '../../../../helpers';

const { Text } = Typography;

const TopUpUserModal = ({ visible, onCancel, onConfirm, user, loading, t }) => {
  const quotaPerUnit = getQuotaPerUnit();
  const [amount, setAmount] = useState(quotaPerUnit);

  useEffect(() => {
    if (visible) {
      setAmount(quotaPerUnit);
    }
  }, [visible, quotaPerUnit]);

  return (
    <Modal
      title={t('充值额度')}
      visible={visible}
      onCancel={onCancel}
      onOk={() => onConfirm(amount)}
      okText={t('确定')}
      cancelText={t('取消')}
      confirmLoading={loading}
      okButtonProps={{ disabled: amount <= 0 }}
    >
      <div style={{ marginBottom: 16 }}>
        <Text>{t('用户')}: </Text>
        <Text strong>{user?.username}</Text>
      </div>
      <div style={{ marginBottom: 16 }}>
        <Text>{t('当前余额')}: </Text>
        <Text strong>{renderQuota(user?.quota || 0)}</Text>
      </div>
      <InputNumber
        value={amount}
        onChange={(value) => setAmount(value || quotaPerUnit)}
        min={quotaPerUnit}
        step={quotaPerUnit}
        precision={0}
        style={{ width: '100%' }}
        suffix={<Text type='tertiary'>{`${t('约')} ${renderQuota(amount || 0)}`}</Text>}
      />
      <div style={{ marginTop: 16 }}>
        <Text type='tertiary'>{t('新额度：')}</Text>
        <Text strong>{renderQuota((user?.quota || 0) + (amount || 0))}</Text>
      </div>
    </Modal>
  );
};

export default TopUpUserModal;
