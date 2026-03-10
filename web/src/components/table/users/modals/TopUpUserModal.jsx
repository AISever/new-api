/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import React, { useState, useEffect } from 'react';
import { Modal, InputNumber, Typography } from '@douyinfe/semi-ui';
import { renderQuota, getQuotaPerUnit } from '../../../../helpers';

const { Text } = Typography;

const TopUpUserModal = ({ visible, onCancel, onConfirm, user, loading, t }) => {
  const [amount, setAmount] = useState(0);
  const quotaPerUnit = getQuotaPerUnit();

  useEffect(() => {
    if (visible) {
      setAmount(quotaPerUnit); // 默认充值1单位
    }
  }, [visible, quotaPerUnit]);

  const handleConfirm = () => {
    if (amount > 0) {
      onConfirm(amount);
    }
  };

  return (
    <Modal
      title={t('充值额度')}
      visible={visible}
      onCancel={onCancel}
      onOk={handleConfirm}
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
        <Text>{t('当前额度')}: </Text>
        <Text strong>{renderQuota(user?.quota || 0)}</Text>
      </div>
      <div style={{ marginBottom: 8 }}>
        <Text>{t('充值额度')}: </Text>
      </div>
      <InputNumber
        value={amount}
        onChange={setAmount}
        min={quotaPerUnit}
        step={quotaPerUnit}
        style={{ width: '100%' }}
        suffix={<Text type="tertiary">{t('约')} {renderQuota(amount)}</Text>}
      />
      <div style={{ marginTop: 16 }}>
        <Text type="tertiary">{t('充值后额度')}: </Text>
        <Text strong type="success">{renderQuota((user?.quota || 0) + amount)}</Text>
      </div>
    </Modal>
  );
};

export default TopUpUserModal;
