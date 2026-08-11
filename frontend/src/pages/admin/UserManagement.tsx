import { useState } from 'react';
import { Card, Table, Input, Select, Space, Tag, Popconfirm, Button, message, Typography } from 'antd';
import { useTranslation } from 'react-i18next';
import type { TableProps } from 'antd';
import { PlusOutlined, DeleteOutlined, UserDeleteOutlined, AuditOutlined, KeyOutlined } from '@ant-design/icons';
import { formatDateTime } from '@/utils/date';
import { StatusResult } from '@/components/common/StatusResult';
import GradientButton from '@/components/common/GradientButton';
import { useUserManagement } from './useUserManagement';
import UserCreateModal from './components/UserCreateModal';
import UserEditModal from './components/UserEditModal';
import UserPasswordModal from './components/UserPasswordModal';
import UserDetailDrawer from './components/UserDetailDrawer';
import type { UserWithAccounts } from '@/client/admin';

const { Search } = Input;
const { Text } = Typography;

export default function UserManagement() {
  const { t } = useTranslation();
  const ctx = useUserManagement();
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [batchDeleting, setBatchDeleting] = useState(false);

  const rowSelection: TableProps<UserWithAccounts>['rowSelection'] = {
    selectedRowKeys,
    onChange: (keys: React.Key[]) => setSelectedRowKeys(keys),
    selections: [Table.SELECTION_ALL, Table.SELECTION_INVERT, Table.SELECTION_NONE],
  };

  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) return;
    setBatchDeleting(true);
    try {
      await ctx.handleBatchDelete(selectedRowKeys.map(String));
      setSelectedRowKeys([]);
    } catch {
      message.error(t('admin.userManagement.messages.userDeleteFailed'));
    } finally {
      setBatchDeleting(false);
    }
  };

  const columns = [
    { title: t('admin.userManagement.table.email'), dataIndex: 'email', key: 'email', width: 200 },
    {
      title: t('admin.userManagement.form.accountNumber'), dataIndex: 'accountNumber', key: 'accountNumber', width: 100,
      render: (v: string) => v ? <Tag color="blue">{v}</Tag> : <Text type="secondary">—</Text>,
    },
    { title: t('admin.userManagement.table.nickname'), dataIndex: 'nickname', key: 'nickname', width: 120 },
    {
      title: t('admin.userManagement.table.role'), dataIndex: 'role', key: 'role', width: 100,
      render: (role: string) => {
        const m: Record<string, { label: string; color: string }> = {
          user: { label: t('admin.userManagement.roles.user'), color: 'default' },
          super_admin: { label: t('admin.userManagement.roles.superAdmin'), color: 'gold' },
          operation: { label: t('admin.userManagement.roles.operation'), color: 'blue' },
          customer_service: { label: t('admin.userManagement.roles.customerService'), color: 'green' },
          audit: { label: t('admin.userManagement.roles.audit'), color: 'purple' },
        };
        const c = m[role] || { label: role, color: 'default' };
        return <Tag color={c.color}>{c.label}</Tag>;
      },
    },
    { title: t('admin.userManagement.table.mtAccountCount'), dataIndex: 'mtAccountCount', key: 'mtAccountCount', width: 80 },
    {
      title: t('admin.userManagement.table.status'), dataIndex: 'status', key: 'status', width: 80,
      render: (status: string) => (
        <Tag color={status === 'active' ? 'success' : 'error'}>
          {status === 'active' ? t('admin.userManagement.status.active') : t('admin.userManagement.status.suspended')}
        </Tag>
      ),
    },
    {
      title: t('admin.userManagement.table.createdAt'), dataIndex: 'createdAt', key: 'createdAt', width: 180,
      render: (_: unknown, record: UserWithAccounts) => formatDateTime(record.createdAt),
    },
    {
      title: t('admin.userManagement.table.actions'), key: 'action', width: 280,
      render: (_: unknown, record: UserWithAccounts) => (
        <Space size="small">
          <Button type="link" size="small" onClick={() => ctx.showEditModal(record)}>{t('common.edit')}</Button>
          <Button type="link" size="small" onClick={() => ctx.showDetailDrawer(record)}>{t('admin.userManagement.actions.details')}</Button>
          <Button type="link" size="small" icon={<KeyOutlined size={14} />} onClick={() => ctx.showPasswordModal(record)}>
            {t('admin.userManagement.actions.changePassword')}
          </Button>
          <Button type="link" size="small"
            icon={record.status === 'active' ? <UserDeleteOutlined size={14} /> : <AuditOutlined size={14} />}
            onClick={() => ctx.handleToggleStatus(record)}>
            {record.status === 'active' ? t('admin.userManagement.actions.disable') : t('admin.userManagement.actions.enable')}
          </Button>
          <Popconfirm title={t('admin.userManagement.deleteConfirm.title')} onConfirm={() => ctx.handleDelete(record.id)} okText={t('common.confirm')} cancelText={t('common.cancel')}>
            <Button type="link" size="small" danger icon={<DeleteOutlined size={14} />}>{t('common.delete')}</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div className="space-y-4">
      <div className="flex justify-between items-center">
        <h1 className="text-2xl font-bold" style={{ color: 'var(--color-text)' }}>{t('admin.userManagement.title')}</h1>
        <Space>
          {selectedRowKeys.length > 0 && (
            <Popconfirm
              title={t('admin.userManagement.deleteConfirm.batchDeleteConfirm', { count: selectedRowKeys.length })}
              onConfirm={handleBatchDelete}
            >
              <Button danger icon={<DeleteOutlined />} loading={batchDeleting}>
                {t('common.deleteSelected', { count: selectedRowKeys.length })}
              </Button>
            </Popconfirm>
          )}
          <GradientButton icon={<PlusOutlined size={16} />} onClick={() => ctx.setCreateModalVisible(true)}>
            {t('admin.userManagement.addUser')}
          </GradientButton>
        </Space>
      </div>
      <Card>
        <div className="mb-4 flex gap-4 flex-wrap">
          <Search placeholder={t('admin.userManagement.filters.searchPlaceholder')} allowClear style={{ width: 250 }}
            onSearch={(value) => ctx.setParams({ ...ctx.params, search: value, page: 1 })} />
          <Select placeholder={t('admin.userManagement.filters.statusPlaceholder')} allowClear style={{ width: 120 }}
            onChange={(value) => ctx.setParams({ ...ctx.params, status: value, page: 1 })}
            options={[{ label: t('admin.userManagement.status.active'), value: 'active' }, { label: t('admin.userManagement.status.suspended'), value: 'suspended' }]} />
          <Select placeholder={t('admin.userManagement.filters.rolePlaceholder')} allowClear style={{ width: 140 }}
            onChange={(value) => ctx.setParams({ ...ctx.params, role: value, page: 1 })}
            options={[
              { label: t('admin.userManagement.roles.user'), value: 'user' },
              { label: t('admin.userManagement.roles.superAdmin'), value: 'super_admin' },
              { label: t('admin.userManagement.roles.operation'), value: 'operation' },
              { label: t('admin.userManagement.roles.customerService'), value: 'customer_service' },
              { label: t('admin.userManagement.roles.audit'), value: 'audit' },
            ]} />
        </div>
        <StatusResult error={ctx.error} onRetry={ctx.fetchUsers}>
          <Table rowSelection={rowSelection} scroll={{ x: 'max-content' }} columns={columns} dataSource={ctx.users} rowKey="id" loading={ctx.loading}
            pagination={{
              current: ctx.params.page, pageSize: ctx.params.pageSize, total: ctx.total, showSizeChanger: true,
              showTotal: (total) => t('admin.userManagement.pagination.total', { total }),
              onChange: (page, pageSize) => ctx.setParams({ ...ctx.params, page, pageSize }),
            }} />
        </StatusResult>
      </Card>

      <UserCreateModal visible={ctx.createModalVisible} form={ctx.createForm}
        onFinish={ctx.handleCreate} onCancel={() => ctx.setCreateModalVisible(false)} />
      <UserEditModal visible={ctx.editModalVisible} form={ctx.editForm}
        onFinish={ctx.handleUpdate} onCancel={() => ctx.setEditModalVisible(false)} />
      <UserPasswordModal visible={ctx.passwordModalVisible} form={ctx.passwordForm}
        userEmail={ctx.currentUser?.email || ''}
        onFinish={ctx.handleUpdatePassword} onCancel={() => ctx.setPasswordModalVisible(false)} />
      <UserDetailDrawer visible={ctx.detailDrawerVisible} user={ctx.currentUser}
        onClose={() => ctx.setDetailDrawerVisible(false)} />
    </div>
  );
}
