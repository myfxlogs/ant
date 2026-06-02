import { Modal, Form, Input, Select, Space, Button } from 'antd';
import { useTranslation } from 'react-i18next';
import type { FormInstance } from 'antd';
import type { CreateUserRequest } from '@/client/admin';

interface Props {
  visible: boolean;
  form: FormInstance;
  onFinish: (values: CreateUserRequest) => Promise<void>;
  onCancel: () => void;
}

export default function UserCreateModal({ visible, form, onFinish, onCancel }: Props) {
  const { t } = useTranslation();
  return (
    <Modal title={t('admin.userManagement.modals.createTitle')} open={visible} onCancel={onCancel} footer={null}>
      <Form form={form} onFinish={onFinish} layout="vertical">
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
            <Button onClick={onCancel}>{t('common.cancel')}</Button>
          </Space>
        </Form.Item>
      </Form>
    </Modal>
  );
}
