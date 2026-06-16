import { useState } from 'react';
import { Card, Table, Input, Select, Space, Tag, Popconfirm, Button, message, Typography } from 'antd';
import type { TableRowSelection } from 'antd/es/table';
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
  const ctx = useUserManagement();
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [batchDeleting, setBatchDeleting] = useState(false);

  const rowSelection: TableRowSelection<UserWithAccounts> = {
    selectedRowKeys,
    onChange: (keys) => setSelectedRowKeys(keys),
    selections: [Table.SELECTION_ALL, Table.SELECTION_INVERT, Table.SELECTION_NONE],
  };

  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) return;
    setBatchDeleting(true);
    try {
      await ctx.handleBatchDelete(selectedRowKeys.map(String));
      setSelectedRowKeys([]);
    } catch {
      message.error('删除失败');
    } finally {
      setBatchDeleting(false);
    }
  };

  const columns = [
    { title: 'ID', dataIndex: 'id', key: 'id', width: 100, ellipsis: true },
    { title: '邮箱', dataIndex: 'email', key: 'email', width: 200 },
    {
      title: '钱包号', dataIndex: 'accountNumber', key: 'accountNumber', width: 100,
      render: (v: string) => v ? <Tag color="blue">{v}</Tag> : <Text type="secondary">—</Text>,
    },
    { title: '昵称', dataIndex: 'nickname', key: 'nickname', width: 120 },
    {
      title: '角色', dataIndex: 'role', key: 'role', width: 100,
      render: (role: string) => {
        const m: Record<string, { label: string; color: string }> = {
          user: { label: '用户', color: 'default' },
          super_admin: { label: '超级管理员', color: 'gold' },
          operation: { label: '运营', color: 'blue' },
          customer_service: { label: '客服', color: 'green' },
          audit: { label: '审计', color: 'purple' },
        };
        const c = m[role] || { label: role, color: 'default' };
        return <Tag color={c.color}>{c.label}</Tag>;
      },
    },
    { title: 'MT账户', dataIndex: 'mtAccountCount', key: 'mtAccountCount', width: 80 },
    {
      title: '状态', dataIndex: 'status', key: 'status', width: 80,
      render: (status: string) => (
        <Tag color={status === 'active' ? 'success' : 'error'}>
          {status === 'active' ? '活跃' : '已停用'}
        </Tag>
      ),
    },
    {
      title: '注册时间', dataIndex: 'createdAt', key: 'createdAt', width: 180,
      render: (_: unknown, record: UserWithAccounts) => formatDateTime(record.createdAt),
    },
    {
      title: '操作', key: 'action', width: 280,
      render: (_: unknown, record: UserWithAccounts) => (
        <Space size="small">
          <Button type="link" size="small" onClick={() => ctx.showEditModal(record)}>编辑</Button>
          <Button type="link" size="small" onClick={() => ctx.showDetailDrawer(record)}>详情</Button>
          <Button type="link" size="small" icon={<KeyOutlined size={14} />} onClick={() => ctx.showPasswordModal(record)}>
            改密
          </Button>
          <Button type="link" size="small"
            icon={record.status === 'active' ? <UserDeleteOutlined size={14} /> : <AuditOutlined size={14} />}
            onClick={() => ctx.handleToggleStatus(record)}>
            {record.status === 'active' ? '停用' : '启用'}
          </Button>
          <Popconfirm title="确认删除此用户？" onConfirm={() => ctx.handleDelete(record.id)} okText="确认" cancelText="取消">
            <Button type="link" size="small" danger icon={<DeleteOutlined size={14} />}>删除</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div className="space-y-4">
      <div className="flex justify-between items-center">
        <h1 className="text-2xl font-bold" style={{ color: 'var(--color-text)' }}>用户管理</h1>
        <Space>
          {selectedRowKeys.length > 0 && (
            <Popconfirm
              title={`确认删除 ${selectedRowKeys.length} 个用户？`}
              onConfirm={handleBatchDelete}
            >
              <Button danger icon={<DeleteOutlined />} loading={batchDeleting}>
                删除 {selectedRowKeys.length} 个
              </Button>
            </Popconfirm>
          )}
          <GradientButton icon={<PlusOutlined size={16} />} onClick={() => ctx.setCreateModalVisible(true)}>
            添加用户
          </GradientButton>
        </Space>
      </div>
      <Card>
        <div className="mb-4 flex gap-4 flex-wrap">
          <Search placeholder="搜索邮箱 / 昵称 / 钱包号" allowClear style={{ width: 250 }}
            onSearch={(value) => ctx.setParams({ ...ctx.params, search: value, page: 1 })} />
          <Select placeholder="状态筛选" allowClear style={{ width: 120 }}
            onChange={(value) => ctx.setParams({ ...ctx.params, status: value, page: 1 })}
            options={[{ label: '活跃', value: 'active' }, { label: '已停用', value: 'suspended' }]} />
          <Select placeholder="角色筛选" allowClear style={{ width: 140 }}
            onChange={(value) => ctx.setParams({ ...ctx.params, role: value, page: 1 })}
            options={[
              { label: '用户', value: 'user' },
              { label: '超级管理员', value: 'super_admin' },
              { label: '运营', value: 'operation' },
              { label: '客服', value: 'customer_service' },
              { label: '审计', value: 'audit' },
            ]} />
        </div>
        <StatusResult error={ctx.error} onRetry={ctx.fetchUsers}>
          <Table rowSelection={rowSelection} scroll={{ x: 'max-content' }} columns={columns} dataSource={ctx.users} rowKey="id" loading={ctx.loading}
            pagination={{
              current: ctx.params.page, pageSize: ctx.params.pageSize, total: ctx.total, showSizeChanger: true,
              showTotal: (total) => `共 ${total} 个用户`,
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
