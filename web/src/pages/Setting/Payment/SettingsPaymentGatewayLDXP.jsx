import React, { useEffect, useRef, useState } from 'react';
import { Banner, Button, Form, Typography, Spin } from '@douyinfe/semi-ui';
import {
  API,
  removeTrailingSlash,
  showError,
  showSuccess,
  verifyJSON,
} from '../../../helpers';
import { useTranslation } from 'react-i18next';

const { Text } = Typography;

export default function SettingsPaymentGatewayLDXP(props) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [inputs, setInputs] = useState({
    LdxpEnabled: false,
    LdxpBaseURL: 'https://pay.ldxp.cn',
    LdxpShopInput: '',
    LdxpShopToken: '',
    LdxpDefaultChannelId: 0,
    LdxpTopupProducts: '[]',
  });
  const [originInputs, setOriginInputs] = useState({});
  const [shopSnapshot, setShopSnapshot] = useState(null);
  const formApiRef = useRef(null);

  useEffect(() => {
    if (props.options && formApiRef.current) {
      const currentInputs = {
        LdxpEnabled:
          props.options.LdxpEnabled === 'true' || props.options.LdxpEnabled === true,
        LdxpBaseURL: props.options.LdxpBaseURL || 'https://pay.ldxp.cn',
        LdxpShopInput: '',
        LdxpShopToken: '',
        LdxpDefaultChannelId: Number(props.options.LdxpDefaultChannelId || 0),
        LdxpTopupProducts: props.options.LdxpTopupProducts || '[]',
      };
      try {
        currentInputs.LdxpTopupProducts = JSON.stringify(
          JSON.parse(currentInputs.LdxpTopupProducts),
          null,
          2,
        );
      } catch {}
      setInputs(currentInputs);
      setOriginInputs({ ...currentInputs });
      formApiRef.current.setValues(currentInputs);
    }
  }, [props.options]);

  const inspectLdxpShop = async () => {
    const shopInput = (inputs.LdxpShopInput || inputs.LdxpShopToken || '').trim();
    if (!shopInput) {
      showError(t('请输入 LDXP 店铺地址或 Token'));
      return;
    }

    setLoading(true);
    try {
      const res = await API.post('/api/option/ldxp/inspect', {
        base_url: removeTrailingSlash(inputs.LdxpBaseURL || 'https://pay.ldxp.cn'),
        shop_input: shopInput,
      });
      if (!res.data?.success) {
        showError(res.data?.message || t('读取 LDXP 店铺失败'));
        return;
      }

      const data = res.data.data || {};
      const nextInputs = {
        ...inputs,
        LdxpBaseURL: data.base_url || inputs.LdxpBaseURL,
        LdxpShopInput: shopInput,
        LdxpShopToken: data.shop_token || inputs.LdxpShopToken,
        LdxpDefaultChannelId:
          Number(data.default_channel_id || 0) || Number(inputs.LdxpDefaultChannelId || 0),
        LdxpTopupProducts: JSON.stringify(data.topup_products || [], null, 2),
      };
      setInputs(nextInputs);
      formApiRef.current?.setValues(nextInputs);
      setShopSnapshot(data);
      showSuccess(t('已读取 LDXP 店铺配置'));
    } catch {
      showError(t('读取 LDXP 店铺失败'));
    } finally {
      setLoading(false);
    }
  };

  const submitLdxpSettings = async () => {
    if (
      inputs.LdxpTopupProducts?.trim() &&
      !verifyJSON(inputs.LdxpTopupProducts)
    ) {
      showError(t('LDXP 充值档位不是合法的 JSON 数组'));
      return;
    }

    let parsedProducts = [];
    try {
      parsedProducts = JSON.parse(inputs.LdxpTopupProducts || '[]');
    } catch {
      parsedProducts = null;
    }
    if (!Array.isArray(parsedProducts)) {
      showError(t('LDXP 充值档位必须是 JSON 数组'));
      return;
    }

    setLoading(true);
    try {
      const options = [
        { key: 'LdxpEnabled', value: inputs.LdxpEnabled ? 'true' : 'false' },
        {
          key: 'LdxpBaseURL',
          value: removeTrailingSlash(inputs.LdxpBaseURL || 'https://pay.ldxp.cn'),
        },
        {
          key: 'LdxpDefaultChannelId',
          value: String(Number(inputs.LdxpDefaultChannelId || 0)),
        },
        {
          key: 'payment_setting.ldxp_topup_products',
          value: inputs.LdxpTopupProducts || '[]',
        },
      ];

      if (inputs.LdxpShopToken) {
        options.push({ key: 'LdxpShopToken', value: inputs.LdxpShopToken });
      }

      const results = await Promise.all(
        options.map((opt) =>
          API.put('/api/option/', {
            key: opt.key,
            value: opt.value,
          }),
        ),
      );

      const failed = results.find((res) => !res.data.success);
      if (failed) {
        showError(failed.data.message || t('更新失败'));
        return;
      }
      showSuccess(t('更新成功'));
      setOriginInputs({ ...inputs });
      props.refresh?.();
    } catch {
      showError(t('更新失败'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <Spin spinning={loading}>
      <Form
        initValues={inputs}
        onValueChange={(values) => setInputs(values)}
        getFormApi={(api) => (formApiRef.current = api)}
      >
        <Form.Section text={t('LDXP 设置')}>
          <Text type='tertiary'>
            {t(
              '将 LDXP 作为标准支付渠道接入。充值档位使用 JSON 数组配置 amount 与 goods_key 映射。',
            )}
          </Text>
          <Form.Switch field='LdxpEnabled' label={t('启用 LDXP')} />
          <Form.Input
            field='LdxpBaseURL'
            label={t('LDXP 地址')}
            placeholder='https://pay.ldxp.cn'
          />
          <Form.Input
            field='LdxpShopInput'
            label={t('LDXP 店铺地址或 Token')}
            placeholder='https://pay.ldxp.cn/shop/your-shop-token'
            extraText={t('支持直接粘贴店铺完整地址，系统会自动识别 Token。')}
          />
          <Button theme='light' onClick={inspectLdxpShop}>
            {t('从店铺读取')}
          </Button>
          <Form.Input
            field='LdxpShopToken'
            label={t('LDXP 店铺 Token')}
            placeholder={t('如需更新请重新输入')}
          />
          <Form.InputNumber
            field='LdxpDefaultChannelId'
            label={t('默认支付渠道 ID')}
            min={0}
            precision={0}
            style={{ width: '100%' }}
          />
          <Form.TextArea
            field='LdxpTopupProducts'
            label={t('LDXP 充值档位')}
            autosize={{ minRows: 8 }}
            placeholder={`[
  {
    "amount": 10,
    "goods_key": "goods_key_10",
    "label": "10",
    "enabled": true,
    "sort_order": 10
  }
]`}
            rules={[
              {
                validator: (_, value) =>
                  !value || verifyJSON(value)
                    ? Promise.resolve()
                    : Promise.reject(t('请输入合法 JSON')),
              },
            ]}
            extraText={t(
              '每个档位对应一个 LDXP 商品。enabled=false 会隐藏该档位。',
            )}
          />
          {shopSnapshot && (
            <Banner
              type='info'
              closeIcon={null}
              description={
                <div className='space-y-1'>
                  <div>
                    {t('已识别店铺')}：{shopSnapshot.shop_name || shopSnapshot.shop_token}
                  </div>
                  <div>
                    {t('推荐支付渠道 ID')}：{shopSnapshot.default_channel_id || '-'}
                  </div>
                  {Array.isArray(shopSnapshot.subscription_goods) &&
                    shopSnapshot.subscription_goods.length > 0 && (
                      <div>
                        {t('检测到的订阅商品')}：
                        {shopSnapshot.subscription_goods
                          .map(
                            (item) =>
                              `${item.name} (${item.goods_key}, ${item.price})`,
                          )
                          .join(' | ')}
                      </div>
                    )}
                </div>
              }
            />
          )}
          <Button onClick={submitLdxpSettings}>
            {t('更新 LDXP 设置')}
          </Button>
        </Form.Section>
      </Form>
    </Spin>
  );
}
