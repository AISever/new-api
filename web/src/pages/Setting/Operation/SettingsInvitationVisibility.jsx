/* Copyright (C) 2025 QuantumNous — Licensed under AGPL-3.0-or-later. */
import React, { useEffect, useRef, useState } from 'react';
import { Banner, Button, Select, Space, Typography } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../../helpers';
import {
  parseInvitationVisibility,
  serializeInvitationVisibility,
} from '../../../components/topup/utils/invitationDetails';

const optionKey = 'InvitationAmountVisibility';
export default function SettingsInvitationVisibility({ options }) {
  const { t } = useTranslation();
  const [policy, setPolicy] = useState({ mode: 'off', user_ids: [] });
  const [choices, setChoices] = useState([]);
  const [saving, setSaving] = useState(false);
  const [searching, setSearching] = useState(false);
  const sequence = useRef(0);
  const raw = options[optionKey];
  useEffect(() => {
    let active = true;
    let value;
    try {
      value = parseInvitationVisibility(raw);
    } catch {
      value = { mode: 'off', user_ids: [] };
      showError(t('邀请金额配置无效，请重新保存'));
    }
    setPolicy(value);
    setSearching(false);
    const chunks = [];
    for (let i = 0; i < value.user_ids.length; i += 100)
      chunks.push(value.user_ids.slice(i, i + 100));
    Promise.all(
      chunks.map((ids) =>
        API.get('/api/option/invitation-users', {
          disableDuplicate: true,
          params: { ids: ids.join(',') },
        }),
      ),
    )
      .then((responses) => {
        if (!active) return;
        const rows = responses.flatMap((r) =>
          r.data.success ? r.data.data : [],
        );
        setChoices(
          rows.map((r) => ({ value: r.id, label: `${r.username} (#${r.id})` })),
        );
      })
      .catch(() => {
        if (active) showError(t('无法搜索用户，请重试'));
      });
    return () => {
      active = false;
      sequence.current += 1;
    };
  }, [raw, t]);
  const search = async (keyword) => {
    const ticket = ++sequence.current;
    setSearching(true);
    try {
      const response = await API.get('/api/option/invitation-users', {
        disableDuplicate: true,
        params: { keyword },
      });
      if (response.data.success !== true) throw new Error('Search failed');
      if (ticket !== sequence.current) return;
      setChoices((old) => {
        const merged = new Map(
          old
            .filter((c) => policy.user_ids.includes(c.value))
            .map((c) => [c.value, c]),
        );
        response.data.data.forEach((r) =>
          merged.set(r.id, { value: r.id, label: `${r.username} (#${r.id})` }),
        );
        return [...merged.values()];
      });
    } catch {
      if (ticket === sequence.current) showError(t('无法搜索用户，请重试'));
    } finally {
      if (ticket === sequence.current) setSearching(false);
    }
  };
  const save = async () => {
    setSaving(true);
    try {
      const response = await API.put('/api/option/', {
        key: optionKey,
        value: serializeInvitationVisibility(policy),
      });
      if (response.data.success !== true)
        throw new Error(response.data.message || 'Save failed');
      showSuccess(t('保存成功'));
    } catch {
      showError(t('保存失败，请重试'));
    } finally {
      setSaving(false);
    }
  };
  return (
    <Space vertical align='start' style={{ width: '100%' }}>
      <Typography.Title heading={6} id='invitation-visibility-mode-label'>
        {t('邀请金额可见范围')}
      </Typography.Title>
      <Banner
        type='info'
        closeIcon={null}
        description={t(
          '默认仅显示受邀用户名。金额权限授予邀请方，只允许查看其本人直接邀请的用户。',
        )}
      />
      <Select
        aria-labelledby='invitation-visibility-mode-label'
        value={policy.mode}
        disabled={saving}
        onChange={(mode) => setPolicy((p) => ({ ...p, mode }))}
        style={{ width: '100%', maxWidth: 480 }}
        optionList={[
          { value: 'off', label: t('关闭金额展示（默认）') },
          { value: 'all', label: t('全部邀请方') },
          { value: 'selected', label: t('仅指定用户') },
        ]}
      />
      {policy.mode === 'selected' && (
        <>
          <Typography.Text id='invitation-visibility-users-label'>
            {t('允许查看金额的邀请用户')}
          </Typography.Text>
          <Select
            multiple
            remote
            filter
            showClear
            disabled={saving}
            loading={searching}
            maxTagCount={5}
            aria-labelledby='invitation-visibility-users-label'
            placeholder={t('搜索并选择邀请方用户名')}
            value={policy.user_ids}
            optionList={choices}
            onSearch={search}
            onFocus={() => search('')}
            onChange={(user_ids) => setPolicy((p) => ({ ...p, user_ids }))}
            style={{ width: '100%', maxWidth: 680 }}
          />
        </>
      )}
      <Typography.Text type='tertiary'>
        {t(
          '指定用户列表为空时不向任何用户展示金额；关闭权限后下次请求立即生效。',
        )}
      </Typography.Text>
      <Button theme='solid' type='primary' loading={saving} onClick={save}>
        {t('保存邀请金额权限')}
      </Button>
    </Space>
  );
}
