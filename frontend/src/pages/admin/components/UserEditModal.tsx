import { Modal, Form, Input, Select, Space, Button } from 'antd';
import { useTranslation } from 'react-i18next';
import type { FormInstance } from 'antd';
import type { UpdateUserRequest } from '@/client/admin';

interface Props {
  visible: boolean;
  form: FormInstance;
  onFinish: (values: UpdateUserRequest) => Promise<void>;
  onCancel: () => void;
}

export default function UserEditModal({ visible, form, onFinish, onCancel }: Props) {
  const { t } = useTranslation();
  return (
    <Modal title={t('admin.userManagement.modals.editTitle')} open={visible} onCancel={onCancel} footer={null}>
      <Form form={form} onFinish={onFinish} layout="vertical">
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
            <Button onClick={onCancel}>{t('common.cancel')}</Button>
          </Space>
        </Form.Item>
      </Form>
    </Modal>
  );
}
