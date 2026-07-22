import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Card, Button, Table, Tag, Modal, Form, Input, InputNumber, message, Space, Popconfirm, Alert } from 'antd';
import { DollarOutlined, CloseCircleOutlined } from '@ant-design/icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { webauthnApi } from '@/client/webauthn';
import { formatAmount } from '@/utils/amount';
import { bufferToBase64url } from '@/utils/webauthn';

export function WithdrawalPanel({ balance, frozenBalance }: { balance: string; frozenBalance: string }) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [withdrawOpen, setWithdrawOpen] = useState(false);
  const [withdrawing, setWithdrawing] = useState(false);
  const [form] = Form.useForm();

  const { data: withdrawals } = useQuery({
    queryKey: ['webauthn', 'withdrawals'],
    queryFn: () => webauthnApi.listWithdrawals(1, 20),
  });

  const { data: whitelist } = useQuery({
    queryKey: ['webauthn', 'whitelist'],
    queryFn: () => webauthnApi.listWhitelistAddresses(),
  });

  const cancelMutation = useMutation({
    mutationFn: (withdrawalId: string) => webauthnApi.cancelWithdrawal(withdrawalId),
    onSuccess: () => {
      message.success(t('wallet.withdraw.cancelled', { defaultValue: 'Withdrawal cancelled, funds unfrozen' }));
      queryClient.invalidateQueries({ queryKey: ['webauthn', 'withdrawals'] });
      queryClient.invalidateQueries({ queryKey: ['wallet'] });
    },
    onError: (err: Error) => message.error(err.message),
  });

  const handleWithdraw = async () => {
    const values = await form.validateFields();
    setWithdrawing(true);
    try {
      const { challenge, withdrawalId } = await webauthnApi.beginWithdrawal(
        values.amount.toString(),
        values.destAddress,
      );

      const publicKey = {
        challenge: challenge,
        allowCredentials: [],
        timeout: 60000,
        userVerification: 'required' as UserVerificationRequirement,
      };

      const assertion = await navigator.credentials.get({ publicKey }) as PublicKeyCredential;
      const response = assertion.response as AuthenticatorAssertionResponse;

      const assertionPayload = JSON.stringify({
        id: assertion.id,
        rawId: bufferToBase64url(assertion.rawId),
        type: assertion.type,
        response: {
          authenticatorData: bufferToBase64url(new Uint8Array(response.authenticatorData)),
          clientDataJSON: bufferToBase64url(new Uint8Array(response.clientDataJSON)),
          signature: bufferToBase64url(new Uint8Array(response.signature)),
          userHandle: response.userHandle ? bufferToBase64url(new Uint8Array(response.userHandle)) : null,
        },
      });

      await webauthnApi.finishWithdrawal(
        withdrawalId,
        new TextEncoder().encode(assertionPayload),
        assertion.id,
      );

      message.success(t('wallet.withdraw.success', { defaultValue: 'Withdrawal submitted. Funds frozen, awaiting cold sign + broadcast.' }));
      setWithdrawOpen(false);
      form.resetFields();
      queryClient.invalidateQueries({ queryKey: ['webauthn', 'withdrawals'] });
      queryClient.invalidateQueries({ queryKey: ['wallet'] });
    } catch (err: any) {
      message.error(err.message || t('wallet.withdraw.failed', { defaultValue: 'Withdrawal failed' }));
    }
    setWithdrawing(false);
  };

  const statusColors: Record<string, string> = {
    PENDING: 'orange',
    SIGNED_WAITING_BUNDLE: 'blue',
    PENDING_SIGN: 'blue',
    BROADCASTING: 'cyan',
    DONE: 'green',
    FAILED: 'red',
    CANCELLED: 'default',
  };

  const columns = [
    {
      title: t('wallet.withdraw.amount', { defaultValue: 'Amount' }),
      dataIndex: 'amount',
      key: 'amount',
      width: 120,
      render: (v: string) => <span style={{ color: '#E53935', fontWeight: 500 }}>-{formatAmount(v)}</span>,
    },
    {
      title: t('wallet.withdraw.destAddress', { defaultValue: 'Destination' }),
      dataIndex: 'destAddress',
      key: 'destAddress',
      ellipsis: true,
      render: (v: string) => <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{v?.slice(0, 20)}...</span>,
    },
    {
      title: t('wallet.withdraw.status', { defaultValue: 'Status' }),
      dataIndex: 'status',
      key: 'status',
      width: 160,
      render: (v: string) => <Tag color={statusColors[v] || 'default'}>{v}</Tag>,
    },
    {
      title: t('wallet.withdraw.txHash', { defaultValue: 'Tx Hash' }),
      dataIndex: 'txHash',
      key: 'txHash',
      ellipsis: true,
      width: 180,
      render: (v: string) => v ? <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{v.slice(0, 20)}...</span> : '-',
    },
    {
      title: t('wallet.withdraw.time', { defaultValue: 'Time' }),
      dataIndex: 'createdAtTsMs',
      key: 'createdAtTsMs',
      width: 180,
      render: (v: string) => v ? new Date(Number(v)).toLocaleString() : '-',
    },
    {
      title: '',
      key: 'action',
      width: 80,
      render: (_: any, record: any) => {
        const cancellable = ['PENDING', 'SIGNED_WAITING_BUNDLE', 'PENDING_SIGN'].includes(record.status);
        if (!cancellable) return null;
        return (
          <Popconfirm
            title={t('wallet.withdraw.confirmCancel', { defaultValue: 'Cancel this withdrawal and unfreeze funds?' })}
            onConfirm={() => cancelMutation.mutate(record.id)}
          >
            <Button type="text" danger icon={<CloseCircleOutlined />} size="small" />
          </Popconfirm>
        );
      },
    },
  ];

  const availableBalance = parseFloat(balance || '0') - parseFloat(frozenBalance || '0');

  return (
    <Card
      size="small"
      style={{ marginBottom: 24 }}
      title={<span><DollarOutlined style={{ marginRight: 8, color: '#D4AF37' }} />{t('wallet.withdraw.title', { defaultValue: 'Withdraw USDT' })}</span>}
      extra={<Button type="primary" size="small" icon={<DollarOutlined />} disabled={availableBalance <= 0} onClick={() => setWithdrawOpen(true)}>{t('wallet.withdraw.new', { defaultValue: 'New Withdrawal' })}</Button>}
    >
      {availableBalance <= 0 && (
        <Alert
          type="info"
          message={t('wallet.withdraw.noBalance', { defaultValue: 'No available balance for withdrawal. Frozen funds are pending withdrawal completion.' })}
          style={{ marginBottom: 16 }}
          showIcon
        />
      )}
      <Table
        columns={columns}
        dataSource={withdrawals?.withdrawals || []}
        rowKey="id"
        size="small"
        pagination={{ pageSize: 10, size: 'small' }}
      />
      <Modal
        title={t('wallet.withdraw.new', { defaultValue: 'New Withdrawal' })}
        open={withdrawOpen}
        onCancel={() => setWithdrawOpen(false)}
        onOk={handleWithdraw}
        confirmLoading={withdrawing}
        okText={t('wallet.withdraw.submit', { defaultValue: 'Sign & Submit' })}
      >
        <Form form={form} layout="vertical">
          <Form.Item label={t('wallet.withdraw.available', { defaultValue: 'Available Balance' })}>
            <span style={{ fontWeight: 600 }}>{formatAmount(availableBalance.toString())} USDT</span>
          </Form.Item>
          <Form.Item
            name="amount"
            label={t('wallet.withdraw.amountLabel', { defaultValue: 'Amount (USDT)' })}
            rules={[{ required: true, message: t('wallet.withdraw.amountRequired', { defaultValue: 'Please enter amount' }) }]}
          >
            <InputNumber
              style={{ width: '100%' }}
              min={0.01}
              step={0.01}
              precision={6}
              placeholder="0.00"
            />
          </Form.Item>
          <Form.Item
            name="destAddress"
            label={t('wallet.withdraw.destLabel', { defaultValue: 'Destination Address (TRC20)' })}
            rules={[{ required: true, message: t('wallet.withdraw.destRequired', { defaultValue: 'Please enter destination address' }) }]}
          >
            <Input placeholder="T..." style={{ fontFamily: 'monospace' }} />
          </Form.Item>
          {whitelist && whitelist.length > 0 && (
            <Form.Item label={t('wallet.withdraw.whitelist', { defaultValue: 'Whitelist (click to fill)' })}>
              <Space wrap>
                {whitelist.filter(w => w.status === 'ACTIVE').map(w => (
                  <Tag
                    key={w.id}
                    style={{ cursor: 'pointer', fontFamily: 'monospace', fontSize: 12 }}
                    onClick={() => form.setFieldValue('destAddress', w.address)}
                  >
                    {w.label || w.address.slice(0, 12)}...
                  </Tag>
                ))}
              </Space>
            </Form.Item>
          )}
          <Alert
            type="warning"
            message={t('wallet.withdraw.warning', { defaultValue: 'You will be prompted to authenticate with your passkey. Funds will be frozen until the withdrawal is broadcast or cancelled.' })}
            showIcon
          />
        </Form>
      </Modal>
    </Card>
  );
}
