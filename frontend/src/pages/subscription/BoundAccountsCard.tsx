import { Table, Button, Tag, Popconfirm, message, Progress } from 'antd';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { subscriptionApi, type BoundAccount } from '@/client/subscription';

export default function BoundAccountsCard() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();

  const { data: boundData, isLoading } = useQuery({
    queryKey: ['subscription', 'bound-accounts'],
    queryFn: () => subscriptionApi.listBoundAccounts(),
  });

  const unbindMutation = useMutation({
    mutationFn: (mtAccountId: string) => subscriptionApi.unbindAccount(mtAccountId),
    onSuccess: () => {
      message.success(t('subscription.unbindSuccess', { defaultValue: 'Account unbound successfully.' }));
      queryClient.invalidateQueries({ queryKey: ['subscription', 'bound-accounts'] });
    },
    onError: () => {
      message.error(t('subscription.unbindFailed', { defaultValue: 'Failed to unbind account.' }));
    },
  });

  const accounts = boundData?.accounts || [];
  const maxAccounts = boundData?.maxAccounts || 0;
  const boundCount = accounts.length;
  const isUnlimited = maxAccounts === 0;

  const columns = [
    {
      title: t('subscription.accountLogin', { defaultValue: 'Login' }),
      dataIndex: 'login',
      key: 'login',
    },
    {
      title: t('subscription.accountBroker', { defaultValue: 'Broker' }),
      dataIndex: 'broker',
      key: 'broker',
    },
    {
      title: t('subscription.accountServer', { defaultValue: 'Server' }),
      dataIndex: 'server',
      key: 'server',
    },
    {
      title: t('subscription.accountType', { defaultValue: 'Type' }),
      dataIndex: 'mtType',
      key: 'mtType',
      render: (v: string) => <Tag>{v?.toUpperCase()}</Tag>,
    },
    {
      title: t('subscription.accountStatus', { defaultValue: 'Status' }),
      dataIndex: 'accountStatus',
      key: 'accountStatus',
      render: (v: string) => <Tag color={v === 'active' ? 'green' : 'default'}>{v}</Tag>,
    },
    {
      title: t('subscription.boundAt', { defaultValue: 'Bound At' }),
      dataIndex: 'boundAt',
      key: 'boundAt',
      render: (v: string) => v ? new Date(v).toLocaleString() : '-',
    },
    {
      title: '',
      key: 'action',
      render: (_: unknown, record: BoundAccount) => (
        <Popconfirm
          title={t('subscription.unbindConfirm', { defaultValue: 'Unbind this account? Active schedules on it will be stopped.' })}
          onConfirm={() => unbindMutation.mutate(record.mtAccountId)}
          okText={t('common.confirm', { defaultValue: 'Confirm' })}
          cancelText={t('common.cancel', { defaultValue: 'Cancel' })}
        >
          <Button size="small" danger>
            {t('subscription.unbind', { defaultValue: 'Unbind' })}
          </Button>
        </Popconfirm>
      ),
    },
  ];

  return (
    <div>
      <div className="mb-4 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <span className="text-sm" style={{ color: 'var(--color-text-muted)' }}>
            {t('subscription.boundAccountsCount', { defaultValue: 'Bound Accounts' })}:
          </span>
          <Tag color={isUnlimited || boundCount < maxAccounts ? 'blue' : 'red'}>
            {boundCount} / {isUnlimited ? '∞' : maxAccounts}
          </Tag>
        </div>
        {!isUnlimited && maxAccounts > 0 && (
          <Progress
            percent={Math.min(100, (boundCount / maxAccounts) * 100)}
            size="small"
            style={{ width: 200 }}
            strokeColor={boundCount >= maxAccounts ? '#ff4d4f' : '#D4AF37'}
          />
        )}
      </div>
      <Table
        dataSource={accounts}
        columns={columns}
        rowKey="mtAccountId"
        loading={isLoading}
        size="small"
        pagination={false}
        locale={{ emptyText: t('subscription.noBoundAccounts', { defaultValue: 'No bound accounts yet. Schedule a strategy to auto-bind an account.' }) }}
      />
    </div>
  );
}
