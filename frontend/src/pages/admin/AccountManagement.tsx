import { useEffect, useState, useCallback } from 'react';
import { Card, Table, Button, Input, Select, Space, Tag, Drawer, Descriptions, message, Popconfirm, Skeleton } from 'antd';
import { adminApi, type AccountWithUser, type AccountListParams } from '@/client/admin';
import { formatDateTime } from '@/utils/date';
import { useAdminAccountStore } from '@/stores/adminAccountStore';
import { getErrorMessage } from '@/utils/error';

const { Search } = Input;

const getParamsKey = (params: AccountListParams) => {
  return JSON.stringify({
    page: params.page || 1,
    pageSize: params.pageSize || 20,
    search: params.search || '',
    status: params.status || '',
    mtType: params.mtType || '',
    userId: params.userId || '',
  });
};

export default function AccountManagement() {
  const [accounts, setAccounts] = useState<AccountWithUser[]>([]);
  const [loading, setLoading] = useState(true);
  const [total, setTotal] = useState(0);
  const [params, setParams] = useState<AccountListParams>({ page: 1, pageSize: 20 });
  const [detailDrawerVisible, setDetailDrawerVisible] = useState(false);
  const [currentAccount, setCurrentAccount] = useState<AccountWithUser | null>(null);
  
  const getCachedData = useAdminAccountStore((state) => state.getCachedData);
  const setCachedData = useAdminAccountStore((state) => state.setCachedData);
  const setLoadingStore = useAdminAccountStore((state) => state.setLoading);

  const fetchAccounts = useCallback(async (silent = false) => {
    if (!silent) {
      setLoading(true);
      setLoadingStore(true);
    }
    
    const paramsKey = getParamsKey(params);
    const cached = getCachedData(paramsKey);
    
    if (cached) {
      setAccounts(cached.accounts);
      setTotal(cached.total);
      if (!silent) {
        setLoading(false);
        setLoadingStore(false);
      }
    }
    
    try {
      const result = await adminApi.listAccounts(params);
      setAccounts(result.accounts);
      setTotal(result.total);
      setCachedData(paramsKey, result.accounts, result.total);
    } catch (error) {
      if (!cached) {
        message.error(getErrorMessage(error, '加载账户列表失败'));
      }
    } finally {
      setLoading(false);
      setLoadingStore(false);
    }
  }, [params, getCachedData, setCachedData, setLoadingStore]);

  useEffect(() => {
    fetchAccounts();
  }, [fetchAccounts]);

  const invalidateCache = useAdminAccountStore((state) => state.invalidateCache);

  const handleFreeze = async (account: AccountWithUser) => {
    try {
      await adminApi.freezeAccount(account.id);
      message.success('账户已冻结');
      invalidateCache();
      fetchAccounts(true);
    } catch (error) {
      message.error(getErrorMessage(error, '冻结失败'));
    }
  };

  const handleUnfreeze = async (account: AccountWithUser) => {
    try {
      await adminApi.unfreezeAccount(account.id);
      message.success('账户已解冻');
      invalidateCache();
      fetchAccounts(true);
    } catch (error) {
      message.error(getErrorMessage(error, '解冻失败'));
    }
  };

  const columns = [
    { title: 'ID', dataIndex: 'id', key: 'id', width: 100, ellipsis: true },
    { title: '用户', dataIndex: 'userEmail', key: 'userEmail', width: 150 },
    { title: '账号', dataIndex: 'login', key: 'login', width: 100 },
    { title: '类型', dataIndex: 'mtType', key: 'mtType', width: 80, render: (v: string) => <Tag color={v === 'MT5' ? 'blue' : 'green'}>{v}</Tag> },
    { title: '经纪商', dataIndex: 'brokerCompany', key: 'brokerCompany', width: 150 },
    { title: '状态', dataIndex: 'accountStatus', key: 'accountStatus', width: 100, render: (v: string) => {
      const color = v === 'online' ? 'success' : v === 'offline' ? 'error' : 'warning';
      return <Tag color={color}>{v}</Tag>;
    }},
    { title: '余额', dataIndex: 'balance', key: 'balance', width: 100, render: (v: number) => v?.toFixed(2) },
    { title: '创建时间', dataIndex: 'createdAt', key: 'createdAt', width: 150, render: (_v: any, record: AccountWithUser) => formatDateTime(record.createdAt) },
    {
      title: '操作',
      key: 'action',
      width: 150,
      render: (_: any, record: AccountWithUser) => (
        <Space>
          <Button size="small" onClick={() => { setCurrentAccount(record); setDetailDrawerVisible(true); }}>
            详情
          </Button>
          {record.accountStatus === 'frozen' ? (
            <Button size="small" onClick={() => handleUnfreeze(record)}>解冻</Button>
          ) : (
            <Popconfirm title="确定冻结该账户？" onConfirm={() => handleFreeze(record)}>
              <Button size="small" danger>冻结</Button>
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ];

  return (
    <Card title="账户管理">
      <div className="mb-4">
        <Space>
          <Search
            placeholder="搜索账户"
            onSearch={(value) => setParams({ ...params, search: value, page: 1 })}
            style={{ width: 200 }}
          />
          <Select
            placeholder="状态"
            allowClear
            style={{ width: 120 }}
            onChange={(v) => setParams({ ...params, status: v, page: 1 })}
          >
            <Select.Option value="online">在线</Select.Option>
            <Select.Option value="offline">离线</Select.Option>
          </Select>
        </Space>
      </div>
      {loading && accounts.length === 0 ? (
        <Skeleton active paragraph={{ rows: 10 }} />
      ) : (
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
      )}
      <Drawer
        title="账户详情"
        open={detailDrawerVisible}
        onClose={() => setDetailDrawerVisible(false)}
        width={500}
      >
        {currentAccount && (
          <Descriptions column={1}>
            <Descriptions.Item label="ID">{currentAccount.id}</Descriptions.Item>
            <Descriptions.Item label="用户">{currentAccount.userEmail}</Descriptions.Item>
            <Descriptions.Item label="账号">{currentAccount.login}</Descriptions.Item>
            <Descriptions.Item label="类型">{currentAccount.mtType}</Descriptions.Item>
            <Descriptions.Item label="经纪商">{currentAccount.brokerCompany}</Descriptions.Item>
            <Descriptions.Item label="服务器">{currentAccount.brokerServer}</Descriptions.Item>
            <Descriptions.Item label="状态">{currentAccount.accountStatus}</Descriptions.Item>
            <Descriptions.Item label="余额">{currentAccount.balance}</Descriptions.Item>
            <Descriptions.Item label="净值">{currentAccount.equity}</Descriptions.Item>
            <Descriptions.Item label="保证金">{currentAccount.margin}</Descriptions.Item>
            <Descriptions.Item label="创建时间">{formatDateTime(currentAccount.createdAt)}</Descriptions.Item>
          </Descriptions>
        )}
      </Drawer>
    </Card>
  );
}
