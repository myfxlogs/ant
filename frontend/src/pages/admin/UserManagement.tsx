import { useCallback, useEffect, useState } from 'react';
import { Card, Table, Input, Select, Space, Tag, Modal, Form, Popconfirm, Drawer, Descriptions, Button } from 'antd';
import { IconPlus, IconTrash, IconUserOff, IconUserCheck, IconKey } from '@tabler/icons-react';
import { adminApi, type UserWithAccounts, type UserListParams, type CreateUserRequest, type UpdateUserRequest } from '@/client/admin';
import { formatDateTime } from '@/utils/date';
import { useTranslation } from 'react-i18next';
import { showError, showSuccess } from '@/utils/message';
import { getErrorMessage } from '@/utils/error';
import GradientButton from '@/components/common/GradientButton';

const { Search } = Input;

export default function UserManagement() {
  const { t } = useTranslation();
  const [users, setUsers] = useState<UserWithAccounts[]>([]);
  const [loading, setLoading] = useState(true);
  const [total, setTotal] = useState(0);
  const [params, setParams] = useState<UserListParams>({ page: 1, pageSize: 20 });
  const [createModalVisible, setCreateModalVisible] = useState(false);
  const [editModalVisible, setEditModalVisible] = useState(false);
  const [detailDrawerVisible, setDetailDrawerVisible] = useState(false);
  const [passwordModalVisible, setPasswordModalVisible] = useState(false);
  const [currentUser, setCurrentUser] = useState<UserWithAccounts | null>(null);
  const [createForm] = Form.useForm();
  const [editForm] = Form.useForm();
  const [passwordForm] = Form.useForm();

  const fetchUsers = useCallback(async () => {
    setLoading(true);
    try {
      const result = await adminApi.listUsers(params);
      setUsers(result.users);
      setTotal(result.total);
    } catch (error) {
      showError(getErrorMessage(error, '加载用户列表失败'));
    } finally {
      setLoading(false);
    }
  }, [params]);

  useEffect(() => {
    fetchUsers();
  }, [fetchUsers]);

  const handleCreate = async (values: CreateUserRequest) => {
    try {
      await adminApi.createUser(values);
      showSuccess(t('admin.userManagement.messages.userCreatedSuccess'));
      setCreateModalVisible(false);
      createForm.resetFields();
      fetchUsers();
    } catch (_error) {
      showError(t('admin.userManagement.messages.userCreateFailed'));
    }
  };

  const handleUpdate = async (values: UpdateUserRequest) => {
    if (!currentUser) return;
    try {
      await adminApi.updateUser(currentUser.id, values);
      showSuccess(t('admin.userManagement.messages.userUpdatedSuccess'));
      setEditModalVisible(false);
      editForm.resetFields();
      fetchUsers();
    } catch (_error) {
      showError(t('admin.userManagement.messages.userUpdateFailed'));
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await adminApi.deleteUser(id);
      showSuccess(t('admin.userManagement.messages.userDeletedSuccess'));
      fetchUsers();
    } catch (_error) {
      showError(t('admin.userManagement.messages.userDeleteFailed'));
    }
  };

  const handleToggleStatus = async (user: UserWithAccounts) => {
    try {
      if (user.status === 'active') {
        await adminApi.disableUser(user.id);
        showSuccess(t('admin.userManagement.messages.userDisabled'));
      } else {
        await adminApi.enableUser(user.id);
        showSuccess(t('admin.userManagement.messages.userEnabled'));
      }
      fetchUsers();
    } catch (_error) {
      showError(t('common.operationFailed'));
    }
  };

  const showPasswordModal = (user: UserWithAccounts) => {
    setCurrentUser(user);
    passwordForm.resetFields();
    setPasswordModalVisible(true);
  };

  const handleUpdatePassword = async (_values: { newPassword: string }) => {
    if (!currentUser) return;
    try {
      await adminApi.resetUserPassword(currentUser.id);
      showSuccess(t('admin.userManagement.messages.passwordUpdatedSuccess'));
      setPasswordModalVisible(false);
      passwordForm.resetFields();
    } catch (_error) {
      showError(t('admin.userManagement.messages.passwordUpdateFailed'));
    }
  };

  const showEditModal = (user: UserWithAccounts) => {
    setCurrentUser(user);
    editForm.setFieldsValue({
      nickname: (user as any).nickname,
      role: user.role,
      status: user.status,
    });
    setEditModalVisible(true);
  };

  const showDetailDrawer = (user: UserWithAccounts) => {
    setCurrentUser(user);
    setDetailDrawerVisible(true);
  };

  const columns = [
    {
      title: t('admin.userManagement.table.id'),
      dataIndex: 'id',
      key: 'id',
      width: 100,
      ellipsis: true,
    },
    {
      title: t('admin.userManagement.table.email'),
      dataIndex: 'email',
      key: 'email',
      width: 200,
    },
    {
      title: t('admin.userManagement.table.nickname'),
      dataIndex: 'nickname',
      key: 'nickname',
      width: 120,
    },
    {
      title: t('admin.userManagement.table.role'),
      dataIndex: 'role',
      key: 'role',
      width: 100,
      render: (role: string) => {
        const roleMap: Record<string, { label: string; color: string }> = {
          user: { label: t('admin.userManagement.roles.user'), color: 'default' },
          super_admin: { label: t('admin.userManagement.roles.superAdmin'), color: 'gold' },
          operation: { label: t('admin.userManagement.roles.operation'), color: 'blue' },
          customer_service: { label: t('admin.userManagement.roles.customerService'), color: 'green' },
          audit: { label: t('admin.userManagement.roles.audit'), color: 'purple' },
        };
        const config = roleMap[role] || { label: role, color: 'default' };
        return <Tag color={config.color}>{config.label}</Tag>;
      },
    },
    {
      title: t('admin.userManagement.table.mtAccountCount'),
      dataIndex: 'mtAccountCount',
      key: 'mtAccountCount',
      width: 80,
    },
    {
      title: t('admin.userManagement.table.status'),
      dataIndex: 'status',
      key: 'status',
      width: 80,
      render: (status: string) => (
        <Tag color={status === 'active' ? 'success' : 'error'}>
          {status === 'active' ? t('admin.userManagement.status.active') : t('admin.userManagement.status.suspended')}
        </Tag>
      ),
    },
    {
      title: t('admin.userManagement.table.createdAt'),
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 180,
      render: (_text: any, record: UserWithAccounts) => formatDateTime(record.createdAt),
    },
    {
      title: t('admin.userManagement.table.actions'),
      key: 'action',
      width: 280,
      render: (_: unknown, record: UserWithAccounts) => (
        <Space size="small">
          <Button type="link" size="small" onClick={() => showEditModal(record)}>
            {t('common.edit')}
          </Button>
          <Button type="link" size="small" onClick={() => showDetailDrawer(record)}>
            {t('admin.userManagement.actions.details')}
          </Button>
          <Button
            type="link"
            size="small"
            icon={<IconKey size={14} />}
            onClick={() => showPasswordModal(record)}
          >
            {t('admin.userManagement.actions.changePassword')}
          </Button>
          <Button
            type="link"
            size="small"
            icon={record.status === 'active' ? <IconUserOff size={14} /> : <IconUserCheck size={14} />}
            onClick={() => handleToggleStatus(record)}
          >
            {record.status === 'active' ? t('admin.userManagement.actions.disable') : t('admin.userManagement.actions.enable')}
          </Button>
          <Popconfirm
            title={t('admin.userManagement.deleteConfirm.title')}
            onConfirm={() => handleDelete(record.id)}
            okText={t('common.confirm')}
            cancelText={t('common.cancel')}
          >
            <Button type="link" size="small" danger icon={<IconTrash size={14} />}>
              {t('common.delete')}
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div className="space-y-4">
      <div className="flex justify-between items-center">
        <h1 className="text-2xl font-bold" style={{ color: '#141D22' }}>{t('admin.userManagement.title')}</h1>
        <GradientButton
          icon={<IconPlus size={16} />}
          onClick={() => setCreateModalVisible(true)}
        >
          {t('admin.userManagement.addUser')}
        </GradientButton>
      </div>

      <Card>
        <div className="mb-4 flex gap-4 flex-wrap">
          <Search
            placeholder={t('admin.userManagement.filters.searchPlaceholder')}
            allowClear
            style={{ width: 250 }}
            onSearch={(value) => setParams({ ...params, search: value, page: 1 })}
          />
          <Select
            placeholder={t('admin.userManagement.filters.statusPlaceholder')}
            allowClear
            style={{ width: 120 }}
            onChange={(value) => setParams({ ...params, status: value, page: 1 })}
            options={[
              { label: t('admin.userManagement.status.active'), value: 'active' },
              { label: t('admin.userManagement.status.suspended'), value: 'suspended' },
            ]}
          />
          <Select
            placeholder={t('admin.userManagement.filters.rolePlaceholder')}
            allowClear
            style={{ width: 140 }}
            onChange={(value) => setParams({ ...params, role: value, page: 1 })}
            options={[
              { label: t('admin.userManagement.roles.user'), value: 'user' },
              { label: t('admin.userManagement.roles.superAdmin'), value: 'super_admin' },
              { label: t('admin.userManagement.roles.operation'), value: 'operation' },
              { label: t('admin.userManagement.roles.customerService'), value: 'customer_service' },
              { label: t('admin.userManagement.roles.audit'), value: 'audit' },
            ]}
          />
        </div>

        <Table

          scroll={{ x: "max-content" }}
          columns={columns}
          dataSource={users}
          rowKey="id"
          loading={loading}
          pagination={{
            current: params.page,
            pageSize: params.pageSize,
            total,
            showSizeChanger: true,
            showTotal: (total) => t('admin.userManagement.pagination.total', { total }),
            onChange: (page, pageSize) => setParams({ ...params, page, pageSize }),
          }}
        />
      </Card>

      <Modal
        title={t('admin.userManagement.modals.createTitle')}
        open={createModalVisible}
        onCancel={() => setCreateModalVisible(false)}
        footer={null}
      >
        <Form form={createForm} onFinish={handleCreate} layout="vertical">
          <Form.Item name="email" label={t('admin.userManagement.form.email')} rules={[{ required: true, type: 'email' }]}>
            <Input placeholder={t('admin.userManagement.form.placeholders.email')} />
          </Form.Item>
          <Form.Item name="password" label={t('admin.userManagement.form.password')} rules={[{ required: true, min: 8 }]}>
            <Input.Password placeholder={t('admin.userManagement.form.placeholders.password')} />
          </Form.Item>
          <Form.Item name="nickname" label={t('admin.userManagement.form.nickname')}>
            <Input placeholder={t('admin.userManagement.form.placeholders.nickname')} />
          </Form.Item>
          <Form.Item name="role" label={t('admin.userManagement.form.role')} initialValue="user">
            <Select options={[
              { label: t('admin.userManagement.roles.user'), value: 'user' },
              { label: t('admin.userManagement.roles.superAdmin'), value: 'super_admin' },
              { label: t('admin.userManagement.roles.operation'), value: 'operation' },
              { label: t('admin.userManagement.roles.customerService'), value: 'customer_service' },
              { label: t('admin.userManagement.roles.audit'), value: 'audit' },
            ]} />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">{t('common.create')}</Button>
              <Button onClick={() => setCreateModalVisible(false)}>{t('common.cancel')}</Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={t('admin.userManagement.modals.editTitle')}
        open={editModalVisible}
        onCancel={() => setEditModalVisible(false)}
        footer={null}
      >
        <Form form={editForm} onFinish={handleUpdate} layout="vertical">
          <Form.Item name="nickname" label={t('admin.userManagement.form.nickname')}>
            <Input placeholder={t('admin.userManagement.form.placeholders.nickname')} />
          </Form.Item>
          <Form.Item name="role" label={t('admin.userManagement.form.role')}>
            <Select options={[
              { label: t('admin.userManagement.roles.user'), value: 'user' },
              { label: t('admin.userManagement.roles.superAdmin'), value: 'super_admin' },
              { label: t('admin.userManagement.roles.operation'), value: 'operation' },
              { label: t('admin.userManagement.roles.customerService'), value: 'customer_service' },
              { label: t('admin.userManagement.roles.audit'), value: 'audit' },
            ]} />
          </Form.Item>
          <Form.Item name="status" label={t('admin.userManagement.form.status')}>
            <Select options={[
              { label: t('admin.userManagement.status.active'), value: 'active' },
              { label: t('admin.userManagement.status.suspended'), value: 'suspended' },
            ]} />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">{t('common.save')}</Button>
              <Button onClick={() => setEditModalVisible(false)}>{t('common.cancel')}</Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={t('admin.userManagement.modals.passwordTitle', { email: currentUser?.email || '' })}
        open={passwordModalVisible}
        onCancel={() => setPasswordModalVisible(false)}
        footer={null}
      >
        <Form form={passwordForm} onFinish={handleUpdatePassword} layout="vertical">
          <Form.Item 
            name="newPassword" 
            label={t('admin.userManagement.passwordForm.newPassword')}
            rules={[
              { required: true, message: t('admin.userManagement.passwordForm.validation.newPasswordRequired') },
              { min: 8, message: t('admin.userManagement.passwordForm.validation.passwordMin8') },
              { pattern: /^(?=.*[a-zA-Z])(?=.*\d).+$/, message: t('admin.userManagement.passwordForm.validation.passwordMustContainLettersAndNumbers') }
            ]}
          >
            <Input.Password placeholder={t('admin.userManagement.passwordForm.placeholders.newPassword')} />
          </Form.Item>
          <Form.Item 
            name="confirmPassword" 
            label={t('admin.userManagement.passwordForm.confirmPassword')}
            dependencies={['newPassword']}
            rules={[
              { required: true, message: t('admin.userManagement.passwordForm.validation.confirmPasswordRequired') },
              ({ getFieldValue }) => ({
                validator(_, value) {
                  if (!value || getFieldValue('newPassword') === value) {
                    return Promise.resolve();
                  }
                  return Promise.reject(new Error(t('admin.userManagement.passwordForm.validation.passwordMismatch')));
                },
              }),
            ]}
          >
            <Input.Password placeholder={t('admin.userManagement.passwordForm.placeholders.confirmPassword')} />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">{t('admin.userManagement.passwordForm.submit')}</Button>
              <Button onClick={() => setPasswordModalVisible(false)}>{t('common.cancel')}</Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      <Drawer
        title={t('admin.userManagement.drawer.title')}
        placement="right"
        width={500}
        onClose={() => setDetailDrawerVisible(false)}
        open={detailDrawerVisible}
      >
        {currentUser && (
          <Descriptions column={1} bordered>
            <Descriptions.Item label={t('admin.userManagement.drawer.labels.id')}>{currentUser.id}</Descriptions.Item>
            <Descriptions.Item label={t('admin.userManagement.drawer.labels.email')}>{currentUser.email}</Descriptions.Item>
            <Descriptions.Item label={t('admin.userManagement.drawer.labels.nickname')}>{(currentUser as any).nickname}</Descriptions.Item>
            <Descriptions.Item label={t('admin.userManagement.drawer.labels.role')}>{currentUser.role}</Descriptions.Item>
            <Descriptions.Item label={t('admin.userManagement.drawer.labels.status')}>{currentUser.status}</Descriptions.Item>
            <Descriptions.Item label={t('admin.userManagement.drawer.labels.mtAccountCount')}>{(currentUser as any).mtAccountCount}</Descriptions.Item>
            <Descriptions.Item label={t('admin.userManagement.drawer.labels.lastLogin')}>
              {formatDateTime((currentUser as any).lastLoginAt)}
            </Descriptions.Item>
            <Descriptions.Item label={t('admin.userManagement.drawer.labels.createdAt')}>
              {formatDateTime(currentUser.createdAt)}
            </Descriptions.Item>
          </Descriptions>
        )}
      </Drawer>
    </div>
  );
}
