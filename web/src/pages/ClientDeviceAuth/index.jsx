import React, { useMemo, useState } from 'react';
import { Button, Card, Input, Typography } from '@douyinfe/semi-ui';
import { API, showError, showSuccess } from '../../helpers';
import { normalizeClientDeviceUserCode } from '../../helpers/clientDeviceAuth';

const { Title, Text, Paragraph } = Typography;

const ClientDeviceAuth = () => {
  const initialCode = useMemo(() => {
    const params = new URLSearchParams(window.location.search);
    return normalizeClientDeviceUserCode(params.get('userCode') || '');
  }, []);
  const [userCode, setUserCode] = useState(initialCode);
  const [submitting, setSubmitting] = useState(false);
  const [authorized, setAuthorized] = useState(false);

  const normalizedCode = normalizeClientDeviceUserCode(userCode);
  const canSubmit = normalizedCode.length === 9 && !submitting && !authorized;

  const handleAuthorize = async () => {
    if (!canSubmit) return;
    setSubmitting(true);
    try {
      const res = await API.post('/api/client/auth/device/authorize', {
        userCode: normalizedCode,
      });
      if (res.data?.success) {
        setAuthorized(true);
        showSuccess('桌面客户端授权成功');
        return;
      }
      showError(res.data?.message || '授权失败');
    } catch (error) {
      showError(error);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className='mt-[60px] flex justify-center px-4 py-8'>
      <Card className='w-full max-w-[560px]'>
        <div className='space-y-4'>
          <div>
            <Title heading={3}>授权桌面客户端</Title>
            <Paragraph type='secondary'>
              在 cc-switch 桌面客户端中发起登录后，将显示一组验证码。请确认验证码一致后授权。
            </Paragraph>
          </div>

          <div className='space-y-2'>
            <Text strong>验证码</Text>
            <Input
              size='large'
              value={userCode}
              placeholder='ABCD-EF12'
              onChange={(value) => setUserCode(normalizeClientDeviceUserCode(value))}
              disabled={authorized}
            />
            <Text type='tertiary'>验证码 10 分钟内有效，只授权你正在使用的桌面客户端。</Text>
          </div>

          {authorized ? (
            <div className='rounded-lg border border-green-200 bg-green-50 p-3 text-green-700'>
              授权成功，可以回到 cc-switch 继续使用。
            </div>
          ) : null}

          <Button
            theme='solid'
            type='primary'
            loading={submitting}
            disabled={!canSubmit}
            onClick={handleAuthorize}
          >
            确认授权
          </Button>
        </div>
      </Card>
    </div>
  );
};

export default ClientDeviceAuth;
