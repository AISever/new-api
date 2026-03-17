import React, { useMemo, useState } from 'react';
import {
  Button,
  Card,
  Empty,
  Input,
  Space,
  Table,
  Typography,
} from '@douyinfe/semi-ui';
import { API, showError, showSuccess } from '../../helpers';
import { useTranslation } from 'react-i18next';
import {
  formatGPTTeamPlanDateTime,
  formatGPTTeamTeamStatus,
  formatGPTTeamWarrantyExpiry,
  getGPTTeamWarrantyRecordExpiry,
  getGPTTeamWarrantyRecordTeamName,
  getMaskedGPTTeamCode,
  getMaskedGPTTeamEmail,
} from './gptTeamPlanUtils';

const { Text } = Typography;

function DetailRow({ label, value }) {
  const displayValue =
    value === null || value === undefined || value === '' ? '-' : value;
  return (
    <div className='flex flex-col gap-1 md:flex-row md:items-start'>
      <Text type='tertiary' style={{ minWidth: 120 }}>
        {label}
      </Text>
      <Text>{displayValue}</Text>
    </div>
  );
}

export default function GptTeamPlanTab({ remainingSeats: initialRemainingSeats = null }) {
  const { t } = useTranslation();
  const [redeemEmail, setRedeemEmail] = useState('');
  const [redeemCode, setRedeemCode] = useState('');
  const [warrantyCode, setWarrantyCode] = useState('');
  const [redeemLoading, setRedeemLoading] = useState(false);
  const [warrantyLoading, setWarrantyLoading] = useState(false);
  const [redeemResult, setRedeemResult] = useState(null);
  const [warrantyResult, setWarrantyResult] = useState(null);
  const [revealedWarrantyCodes, setRevealedWarrantyCodes] = useState({});
  const [revealedWarrantyEmails, setRevealedWarrantyEmails] = useState({});
  const [showOriginalCode, setShowOriginalCode] = useState(false);

  const warrantyColumns = useMemo(
    () => [
      {
        title: t('兑换码'),
        dataIndex: 'code',
        render: (value, record) => {
          const revealed = Boolean(revealedWarrantyCodes[record.code]);
          return (
            <Space>
              <Text code>
                {revealed ? value || '-' : getMaskedGPTTeamCode(value)}
              </Text>
              {value ? (
                <Button
                  size='small'
                  theme='borderless'
                  type='tertiary'
                  onClick={() =>
                    setRevealedWarrantyCodes((current) => ({
                      ...current,
                      [record.code]: !revealed,
                    }))
                  }
                >
                  {revealed ? t('隐藏') : t('完整显示')}
                </Button>
              ) : null}
            </Space>
          );
        },
      },
      {
        title: t('邮箱'),
        dataIndex: 'email',
        render: (value, record) => {
          const key = `${record.code || ''}:${record.email || ''}`;
          const revealed = Boolean(revealedWarrantyEmails[key]);
          return (
            <Space>
              <Text>{revealed ? value || '-' : getMaskedGPTTeamEmail(value)}</Text>
              {value ? (
                <Button
                  size='small'
                  theme='borderless'
                  type='tertiary'
                  onClick={() =>
                    setRevealedWarrantyEmails((current) => ({
                      ...current,
                      [key]: !revealed,
                    }))
                  }
                >
                  {revealed ? t('隐藏') : t('完整显示')}
                </Button>
              ) : null}
            </Space>
          );
        },
      },
      {
        title: t('Team'),
        dataIndex: 'team_name',
        render: (value) => getGPTTeamWarrantyRecordTeamName(value),
      },
      {
        title: t('状态'),
        dataIndex: 'team_status',
        render: (value) => formatGPTTeamTeamStatus(value),
      },
      {
        title: t('兑换时间'),
        dataIndex: 'used_at',
        render: (value) => formatGPTTeamPlanDateTime(value),
      },
      {
        title: t('到期时间'),
        dataIndex: 'team_expires_at',
        render: (_, record) => getGPTTeamWarrantyRecordExpiry(record),
      },
    ],
    [revealedWarrantyCodes, revealedWarrantyEmails, t],
  );

  const redeem = async () => {
    if (!redeemEmail.trim() || !redeemCode.trim()) {
      showError(t('请输入邮箱和兑换码'));
      return;
    }
    setRedeemLoading(true);
    try {
      const res = await API.post(
        '/api/gptteamplan/redeem',
        {
          email: redeemEmail.trim(),
          code: redeemCode.trim(),
        },
        { skipErrorHandler: true },
      );
      if (!res.data.success) {
        setRedeemResult({
          error: true,
          message: res.data.message || t('兑换失败'),
        });
        return;
      }
      setRedeemResult(res.data.data || null);
      showSuccess(t('兑换成功'));
    } catch (error) {
      setRedeemResult({
        error: true,
        message: error.message || t('兑换失败'),
      });
    } finally {
      setRedeemLoading(false);
    }
  };

  const checkWarranty = async () => {
    if (!warrantyCode.trim()) {
      showError(t('请输入兑换码'));
      return;
    }
    setWarrantyLoading(true);
    try {
      const res = await API.post(
        '/api/gptteamplan/warranty/check',
        {
          code: warrantyCode.trim(),
        },
        { skipErrorHandler: true },
      );
      if (!res.data.success) {
        setWarrantyResult({
          error: true,
          message: res.data.message || t('质保查询失败'),
        });
        setShowOriginalCode(false);
        return;
      }
      setWarrantyResult(res.data.data || null);
      setShowOriginalCode(false);
    } catch (error) {
      setWarrantyResult({
        error: true,
        message: error.message || t('质保查询失败'),
      });
      setShowOriginalCode(false);
    } finally {
      setWarrantyLoading(false);
    }
  };

  const currentRemainingSeats =
    redeemResult?.error || redeemResult?.remaining_seats === undefined
      ? initialRemainingSeats
      : redeemResult.remaining_seats;

  const remainingSeatsDisplay =
    currentRemainingSeats === null ||
    currentRemainingSeats === undefined ||
    Number.isNaN(Number(currentRemainingSeats))
      ? '-'
      : String(currentRemainingSeats);

  return (
    <div className='space-y-6'>
      <div className='grid grid-cols-1 gap-4 xl:grid-cols-2'>
        <Card
          className='!rounded-2xl shadow-sm border-0 h-full'
          title={
            <div className='flex flex-wrap items-center justify-between gap-3'>
              <span>{t('兑换账号')}</span>
              <div className='flex items-center gap-2 rounded-full bg-blue-50 px-3 py-1 text-sm'>
                <Text type='tertiary'>{t('剩余车位')}</Text>
                <Text strong>{remainingSeatsDisplay}</Text>
              </div>
            </div>
          }
        >
          <Space vertical align='start' style={{ width: '100%' }}>
            <Input
              value={redeemEmail}
              onChange={setRedeemEmail}
              placeholder={t('请输入兑换邮箱')}
            />
            <Input
              value={redeemCode}
              onChange={setRedeemCode}
              placeholder={t('请输入待兑换的兑换码')}
            />
            <Button
              loading={redeemLoading}
              onClick={redeem}
              theme='solid'
              type='primary'
            >
              {t('提交兑换')}
            </Button>
            {redeemResult ? (
              <Card
                shadows='hover'
                style={{ width: '100%' }}
                bodyStyle={{ paddingTop: 12 }}
              >
                <Space vertical align='start' style={{ width: '100%' }}>
                  {redeemResult.error ? (
                    <Text strong type='danger'>{redeemResult.message}</Text>
                  ) : (
                    <>
                      {redeemResult.message ? (
                        <Text strong>{redeemResult.message}</Text>
                      ) : null}
                      <DetailRow label={t('团队名称')} value={redeemResult.team_name} />
                      <DetailRow
                        label={t('订阅计划')}
                        value={redeemResult.subscription_plan}
                      />
                      <DetailRow
                        label={t('剩余车位')}
                        value={redeemResult.remaining_seats}
                      />
                      <DetailRow
                        label={
                          redeemResult.has_warranty
                            ? t('质保到期时间')
                            : t('账号到期时间')
                        }
                        value={formatGPTTeamPlanDateTime(redeemResult.expires_at)}
                      />
                      <DetailRow
                        label={t('团队到期时间')}
                        value={formatGPTTeamPlanDateTime(redeemResult.team_expires_at)}
                      />
                      <DetailRow
                        label={t('质保状态')}
                        value={redeemResult.has_warranty ? t('已启用') : t('未启用')}
                      />
                      <DetailRow
                        label={t('质保有效期')}
                        value={formatGPTTeamPlanDateTime(
                          redeemResult.warranty_expires_at || redeemResult.expires_at,
                        )}
                      />
                    </>
                  )}
                </Space>
              </Card>
            ) : null}
          </Space>
        </Card>

        <Card className='!rounded-2xl shadow-sm border-0 h-full' title={t('质保查询')}>
          <Space vertical align='start' style={{ width: '100%' }}>
            <Input
              value={warrantyCode}
              onChange={setWarrantyCode}
              placeholder={t('请输入需要查询的兑换码')}
            />
            <Button loading={warrantyLoading} onClick={checkWarranty}>
              {t('查询质保')}
            </Button>
            {warrantyResult ? (
              <Card
                shadows='hover'
                style={{ width: '100%' }}
                bodyStyle={{ paddingTop: 12 }}
              >
                <Space vertical align='start' style={{ width: '100%' }}>
                  {warrantyResult.error ? (
                    <Text strong type='danger'>{warrantyResult.message}</Text>
                  ) : (
                    <>
                      {warrantyResult.message &&
                      (!warrantyResult.records ||
                        warrantyResult.records.length === 0) ? (
                        <Text strong>{warrantyResult.message}</Text>
                      ) : null}
                      <DetailRow
                        label={t('质保状态')}
                        value={
                          warrantyResult.has_warranty
                            ? warrantyResult.warranty_valid
                              ? t('有效')
                              : t('已失效')
                            : t('无质保')
                        }
                      />
                      <DetailRow
                        label={t('质保到期时间')}
                        value={formatGPTTeamWarrantyExpiry(
                          warrantyResult.warranty_expires_at,
                          warrantyResult.has_warranty,
                          warrantyResult.warranty_valid,
                        )}
                      />
                      {warrantyResult.can_reuse ? (
                        <>
                          <DetailRow
                            label={t('是否可复用')}
                            value={t('可以')}
                          />
                          <Space>
                            <Text type='tertiary'>{t('原始兑换码')}</Text>
                            <Text code>
                              {showOriginalCode
                                ? warrantyResult.original_code || '-'
                                : getMaskedGPTTeamCode(warrantyResult.original_code)}
                            </Text>
                            {warrantyResult.original_code ? (
                              <Button
                                size='small'
                                theme='borderless'
                                type='tertiary'
                                onClick={() => setShowOriginalCode((current) => !current)}
                              >
                                {showOriginalCode ? t('隐藏') : t('完整显示')}
                              </Button>
                            ) : null}
                          </Space>
                        </>
                      ) : null}
                      <div style={{ width: '100%' }}>
                        <Text strong>{t('质保记录')}</Text>
                        <Table
                          style={{ marginTop: 12 }}
                          rowKey={(record) =>
                            `${record.code || 'no-code'}-${record.email || 'no-email'}`
                          }
                          pagination={false}
                          columns={warrantyColumns}
                          dataSource={warrantyResult.records || []}
                          empty={
                            <Empty description={t('暂无质保记录')} image={<Empty.PRESENTED_IMAGE_SIMPLE />} />
                          }
                        />
                      </div>
                    </>
                  )}
                </Space>
              </Card>
            ) : null}
          </Space>
        </Card>
      </div>
    </div>
  );
}
