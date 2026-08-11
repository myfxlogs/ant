import { useEffect, useState, useCallback } from 'react';
import { Card, Table, Button, Input, Select, Space, Tag, Drawer, Descriptions, message, Popconfirm } from 'antd';
import { timestampDate } from '@bufbuild/protobuf/wkt';
import { adminApi, type AccountWithUser, type AccountListParams } from '@/client/admin';
import { formatDateTime } from '@/utils/date';
import { getErrorMessage } from '@/utils/error';
import { StatusResult } from '@/components/common/StatusResult';
import { useTranslation } from 'react-i18next';
import { adminAccountClient } from '@/client/connect';

const { Search } = Input;

export default function AccountManagement() {
  const { t } = useTranslation();
  const [accounts, setAccounts] = useState<AccountWithUser[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [total, setTotal] = useState(0);
  const [params, setParams] = useState<AccountListParams>({ page: 1, pageSize: 20 });
  const [detailDrawerVisible, setDetailDrawerVisible] = useState(false);
  const [auditLogs, setAuditLogs] = useState<Array<{ id: string; action: string; detail: string; created_at: string }>>([]);
  const [auditLoading, setAuditLoading] = useState(false);

  const fetchAuditLogs = async (accountId: string) => {
    setAuditLoading(true);
    try {
      const resp = await adminAccountClient.getAccountAuditLogs({ accountId });
      setAuditLogs(resp.entries.map(e => ({
        id: e.id,
        action: e.action,
        detail: e.detail,
        created_at: e.createdAt ? timestampDate(e.createdAt).toISOString() : '',
      })));
    } catch { setAuditLogs([]); }
    finally { setAuditLoading(false); }
  };
  const [currentAccount, setCurrentAccount] = useState<AccountWithUser | null>(null);
  
  const fetchAccounts = useCallback(async (silent = false) => {
    if (!silent) setLoading(true);
    try {
      const result = await adminApi.listAccounts(params);
      setAccounts(result.accounts);
      setTotal(result.total);
      setError(null);
    } catch (err) {
      const msg = getErrorMessage(err, t('admin.account.errors.loadFailed', { defaultValue: 'Failed to load accounts' }));
      setError(msg);
      message.error(msg);
    } finally {
      setLoading(false);
    }
  }, [params, t]);

  useEffect(() => {
    fetchAccounts();
  }, [fetchAccounts]);


  const handleFreeze = async (account: AccountWithUser) => {
    try {
      await adminApi.freezeAccount(account.id);
      message.success(t('admin.account.frozen', { defaultValue: 'Account frozen' }));
      fetchAccounts(true);
    } catch (error) {
      message.error(getErrorMessage(error, t('admin.account.errors.freezeFailed', { defaultValue: 'Freeze failed' })));
    }
  };

  const handleUnfreeze = async (account: AccountWithUser) => {
    try {
      await adminApi.unfreezeAccount(account.id);
      message.success(t('admin.account.unfrozen', { defaultValue: 'Account unfrozen' }));
      fetchAccounts(true);
    } catch (error) {
      message.error(getErrorMessage(error, t('admin.account.errors.unfreezeFailed', { defaultValue: 'Unfreeze failed' })));
    }
  };

  const columns = [
    { title: t('admin.account.columns.id', { defaultValue: 'ID' }), dataIndex: 'id', key: 'id', width: 100, ellipsis: true },
    { title: t('admin.account.columns.user', { defaultValue: 'User' }), dataIndex: 'userEmail', key: 'userEmail', width: 150 },
    { title: t('admin.account.columns.login', { defaultValue: 'Login' }), dataIndex: 'login', key: 'login', width: 100 },
    { title: t('admin.account.columns.type', { defaultValue: 'Type' }), dataIndex: 'mtType', key: 'mtType', width: 80, render: (v: string) => <Tag color={v === 'MT5' ? 'blue' : 'green'}>{v}</Tag> },
    { title: t('admin.account.columns.broker', { defaultValue: 'Broker' }), dataIndex: 'brokerCompany', key: 'brokerCompany', width: 150 },
    { title: t('admin.account.columns.status', { defaultValue: 'Status' }), dataIndex: 'accountStatus', key: 'accountStatus', width: 100, render: (v: string) => {
      const color = v === 'online' ? 'success' : v === 'offline' ? 'error' : 'warning';
      return <Tag color={color}>{v}</Tag>;
    }},
    { title: t('admin.account.columns.balance', { defaultValue: 'Balance' }), dataIndex: 'balance', key: 'balance', width: 100, render: (v: number | string) => { const n = Number(v); return isNaN(n) ? '-' : n.toFixed(2); } },
    { title: t('admin.account.columns.createdAt', { defaultValue: 'Created At' }), dataIndex: 'createdAt', key: 'createdAt', width: 150, render: (_v: unknown, record: AccountWithUser) => formatDateTime(record.createdAt) },
    {
      title: t('admin.account.columns.action', { defaultValue: 'Action' }),
      key: 'action',
      width: 150,
      render: (_: unknown, record: AccountWithUser) => (
        <Space>
          <Button size="small" onClick={() => { setCurrentAccount(record); setDetailDrawerVisible(true); fetchAuditLogs(record.id); }}>
            {t('admin.account.detail', { defaultValue: 'Detail' })}
          </Button>
          {record.accountStatus === 'frozen' ? (
            <Button size="small" onClick={() => handleUnfreeze(record)}>{t('admin.account.unfreeze', { defaultValue: 'Unfreeze' })}</Button>
          ) : (
            <Popconfirm title={t('admin.account.confirmFreeze', { defaultValue: 'Freeze this account?' })} onConfirm={() => handleFreeze(record)}>
              <Button size="small" danger>{t('admin.account.freeze', { defaultValue: 'Freeze' })}</Button>
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ];

  return (
    <Card title={t('admin.account.title', { defaultValue: 'Account Management' })}>
      <div className="mb-4">
        <Space>
          <Search
            placeholder={t('admin.account.searchPlaceholder', { defaultValue: 'Search accounts' })}
            onSearch={(value) => setParams({ ...params, search: value, page: 1 })}
            style={{ width: 200 }}
          />
          <Select
            placeholder={t('admin.account.status', { defaultValue: 'Status' })}
            allowClear
            style={{ width: 120 }}
            onChange={(v) => setParams({ ...params, status: v, page: 1 })}
          >
            <Select.Option value="online">{t('admin.account.online', { defaultValue: 'Online' })}</Select.Option>
            <Select.Option value="offline">{t('admin.account.offline', { defaultValue: 'Offline' })}</Select.Option>
          </Select>
        </Space>
      </div>
      <StatusResult error={error} onRetry={() => fetchAccounts()}>
        <Table
          scroll={{ x: "max-content" }}
          columns={columns}
          dataSource={accounts}
          rowKey="id"
          loading={loading}
          pagination={{
            current: params.page,
            pageSize: params.pageSize,
            total,
            onChange: (page, pageSize) => setParams({ ...params, page, pageSize }),
          }}
        />
      </StatusResult>
      <Drawer
        title={t('admin.account.detail', { defaultValue: 'Account Detail' })}
        open={detailDrawerVisible}
        onClose={() => setDetailDrawerVisible(false)}
        width={500}
      >
        {currentAccount && (
          <Descriptions column={1}>
            <Descriptions.Item label="ID">{currentAccount.id}</Descriptions.Item>
            <Descriptions.Item label={t('admin.account.columns.user', { defaultValue: 'User' })}>{currentAccount.userEmail}</Descriptions.Item>
            <Descriptions.Item label={t('admin.account.columns.login', { defaultValue: 'Login' })}>{currentAccount.login}</Descriptions.Item>
            <Descriptions.Item label={t('admin.account.columns.type', { defaultValue: 'Type' })}>{currentAccount.mtType}</Descriptions.Item>
            <Descriptions.Item label={t('admin.account.columns.broker', { defaultValue: 'Broker' })}>{currentAccount.brokerCompany}</Descriptions.Item>
            <Descriptions.Item label={t('admin.account.columns.server', { defaultValue: 'Server' })}>{currentAccount.brokerServer}</Descriptions.Item>
            <Descriptions.Item label={t('admin.account.columns.status', { defaultValue: 'Status' })}>{currentAccount.accountStatus}</Descriptions.Item>
            <Descriptions.Item label={t('admin.account.columns.balance', { defaultValue: 'Balance' })}>{currentAccount.balance}</Descriptions.Item>
            <Descriptions.Item label={t('admin.account.columns.equity', { defaultValue: 'Equity' })}>{currentAccount.equity}</Descriptions.Item>
            <Descriptions.Item label={t('admin.account.columns.margin', { defaultValue: 'Margin' })}>{currentAccount.margin}</Descriptions.Item>
            <Descriptions.Item label={t('admin.account.columns.createdAt', { defaultValue: 'Created At' })}>{formatDateTime(currentAccount.createdAt)}</Descriptions.Item>
          </Descriptions>
        )}
        {currentAccount && (
          <div style={{ marginTop: 16 }}>
            <h4 style={{ marginBottom: 8 }}>{t('admin.account.auditLogs', { defaultValue: 'Audit Logs' })}</h4>
            <Table
              dataSource={auditLogs}
              loading={auditLoading}
              rowKey="id"
              size="small"
              pagination={false}
              columns={[
                { title: t('admin.account.columns.time', { defaultValue: 'Time' }), dataIndex: 'created_at', width: 160, render: (v: string) => v?.slice(0, 19).replace('T', ' ') },
                { title: t('admin.account.columns.action', { defaultValue: 'Action' }), dataIndex: 'action', width: 80 },
                { title: t('admin.account.columns.detail', { defaultValue: 'Detail' }), dataIndex: 'detail' },
              ]}
            />
          </div>
        )}
      </Drawer>
    </Card>
  );
}
