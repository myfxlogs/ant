import { useCallback, useEffect, useState } from 'react';
import { Form, Modal, message } from 'antd';
import { adminApi, type UserWithAccounts, type UserListParams, type CreateUserRequest, type UpdateUserRequest } from '@/client/admin';
import { useTranslation } from 'react-i18next';
import { getErrorMessage } from '@/utils/error';

export function useUserManagement() {
  const { t } = useTranslation();
  const [users, setUsers] = useState<UserWithAccounts[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
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
    setError(null);
    try {
      const result = await adminApi.listUsers(params);
      setUsers(result.users);
      setTotal(result.total);
    } catch (err) {
      const msg = getErrorMessage(err, '加载用户列表失败');
      setError(msg);
      message.error(msg);
    } finally {
      setLoading(false);
    }
  }, [params]);

  useEffect(() => { fetchUsers(); }, [fetchUsers]);

  const handleCreate = async (values: CreateUserRequest) => {
    try {
      await adminApi.createUser(values);
      message.success(t('admin.userManagement.messages.userCreatedSuccess'));
      setCreateModalVisible(false);
      createForm.resetFields();
      fetchUsers();
    } catch { message.error(t('admin.userManagement.messages.userCreateFailed')); }
  };

  const handleUpdate = async (values: UpdateUserRequest) => {
    if (!currentUser) return;
    try {
      await adminApi.updateUser(currentUser.id, values);
      message.success(t('admin.userManagement.messages.userUpdatedSuccess'));
      setEditModalVisible(false);
      editForm.resetFields();
      fetchUsers();
    } catch { message.error(t('admin.userManagement.messages.userUpdateFailed')); }
  };

  const handleDelete = async (id: string) => {
    try {
      await adminApi.deleteUser(id);
      message.success(t('admin.userManagement.messages.userDeletedSuccess'));
      fetchUsers();
    } catch { message.error(t('admin.userManagement.messages.userDeleteFailed')); }
  };

  const handleBatchDelete = async (ids: string[]) => {
    try {
      const resp: any = await adminApi.deleteUsers(ids);
      if (resp?.failedCount > 0) {
        message.warning(
          t('admin.userManagement.messages.batchDeletePartial', {
            deleted: resp.deletedCount,
            failed: resp.failedCount,
          }),
        );
      } else {
        message.success(
          t('admin.userManagement.messages.batchDeleteSuccess', { count: resp?.deletedCount || ids.length }),
        );
      }
      fetchUsers();
    } catch {
      message.error(t('admin.userManagement.messages.userDeleteFailed'));
    }
  };

  const handleToggleStatus = async (user: UserWithAccounts) => {
    try {
      if (user.status === 'active') {
        await adminApi.disableUser(user.id);
        message.success(t('admin.userManagement.messages.userDisabled'));
      } else {
        await adminApi.enableUser(user.id);
        message.success(t('admin.userManagement.messages.userEnabled'));
      }
      fetchUsers();
    } catch { message.error(t('common.operationFailed')); }
  };

  const handleUpdatePassword = async (_values: { newPassword: string }) => {
    if (!currentUser) return;
    try {
      const result = await adminApi.resetUserPassword(currentUser.id);
      const newPass = (result as any)?.newPassword;
      if (newPass) {
        Modal.success({
          title: t('admin.userManagement.messages.passwordUpdatedSuccess'),
          content: t('admin.userManagement.messages.newPasswordIs', { password: newPass }),
        });
      } else {
        message.success(t('admin.userManagement.messages.passwordUpdatedSuccess'));
      }
      setPasswordModalVisible(false);
      passwordForm.resetFields();
    } catch { message.error(t('admin.userManagement.messages.passwordUpdateFailed')); }
  };

  const showEditModal = (user: UserWithAccounts) => {
    setCurrentUser(user);
    editForm.setFieldsValue({ nickname: user.nickname, role: user.role, status: user.status, accountNumber: user.accountNumber });
    setEditModalVisible(true);
  };

  const showDetailDrawer = (user: UserWithAccounts) => {
    setCurrentUser(user);
    setDetailDrawerVisible(true);
  };

  const showPasswordModal = (user: UserWithAccounts) => {
    setCurrentUser(user);
    passwordForm.resetFields();
    setPasswordModalVisible(true);
  };

  return {
    users, loading, error, total, params, setParams,
    createModalVisible, setCreateModalVisible, editModalVisible, setEditModalVisible,
    detailDrawerVisible, setDetailDrawerVisible, passwordModalVisible, setPasswordModalVisible,
    currentUser, createForm, editForm, passwordForm,
    fetchUsers, handleCreate, handleUpdate, handleDelete, handleBatchDelete, handleToggleStatus,
    handleUpdatePassword, showEditModal, showDetailDrawer, showPasswordModal,
  };
}
