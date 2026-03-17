import React, { useEffect, useRef, useState } from 'react';
import {
  Button,
  Form,
  Input,
  Space,
  Switch,
  Typography,
} from '@douyinfe/semi-ui';
import {
  API,
  removeTrailingSlash,
  showError,
  showSuccess,
} from '../../../helpers';
import { useTranslation } from 'react-i18next';

const { Text } = Typography;

export default function SettingsGPTTeamPlan(props) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [inputs, setInputs] = useState({
    gptteamplan_enabled: false,
    gptteamplan_base_url: 'http://gptteamplan.tech',
  });
  const formApiRef = useRef(null);

  useEffect(() => {
    if (!props.options || !formApiRef.current) {
      return;
    }
    const nextValues = {
      gptteamplan_enabled: props.options['gptteamplan.enabled'] === 'true',
      gptteamplan_base_url:
        props.options['gptteamplan.base_url'] || 'http://gptteamplan.tech',
    };
    setInputs(nextValues);
    formApiRef.current.setValues(nextValues);
  }, [props.options]);

  const handleSubmit = async () => {
    setLoading(true);
    try {
      const options = [
        {
          key: 'gptteamplan.enabled',
          value: inputs.gptteamplan_enabled ? 'true' : 'false',
        },
        {
          key: 'gptteamplan.base_url',
          value: removeTrailingSlash(
            inputs.gptteamplan_base_url || 'http://gptteamplan.tech',
          ),
        },
      ];
      const results = await Promise.all(
        options.map((option) =>
          API.put('/api/option/', {
            key: option.key,
            value: option.value,
          }),
        ),
      );
      const failed = results.find((res) => !res.data.success);
      if (failed) {
        showError(failed.data.message || t('更新失败'));
        return;
      }
      showSuccess(t('更新成功'));
      props.refresh?.();
    } catch {
      showError(t('更新失败'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <Space vertical align='start' style={{ width: '100%' }}>
      <Text strong>{t('GPT Team 兑换')}</Text>
      <Text type='tertiary'>
        {t(
          '控制商品商城内 GPT Team 兑换页签的显示，并配置后端代调的基础地址。',
        )}
      </Text>
      <Form
        getFormApi={(api) => {
          formApiRef.current = api;
        }}
        initValues={inputs}
        onValueChange={(values) => setInputs(values)}
        style={{ width: '100%' }}
      >
        <Form.Switch
          field='gptteamplan_enabled'
          label={t('启用 GPT Team 兑换')}
          checkedText={t('开启')}
          uncheckedText={t('关闭')}
        />
        <Form.Input
          field='gptteamplan_base_url'
          label={t('基础地址')}
          placeholder='http://gptteamplan.tech'
        />
      </Form>
      <Button
        onClick={handleSubmit}
        loading={loading}
        theme='solid'
        type='primary'
      >
        {t('保存设置')}
      </Button>
    </Space>
  );
}
