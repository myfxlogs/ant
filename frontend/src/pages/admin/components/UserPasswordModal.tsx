import { Modal, Form, Input, Space, Button } from 'antd';
import { useTranslation } from 'react-i18next';
import type { FormInstance } from 'antd';

interface Props {
  visible: boolean;
  userEmail: string;
  form: FormInstance;
  onFinish: (values: { newPassword: string }) => Promise<void>;
  onCancel: () => void;
}

export default function UserPasswordModal({ visible, userEmail, form, onFinish, onCancel }: Props) {
  const { t } = useTranslation();
  return (
    <Modal title={t('admin.userManagement.modals.passwordTitle', { email: userEmail })} open={visible} onCancel={onCancel} footer={null}>
      <Form form={form} onFinish={onFinish} layout="vertical">
        <Form.Item
          name="newPassword"
          label={t('admin.userManagement.passwordForm.newPassword')}
          rules={[
            { required: true, message: t('admin.userManagement.passwordForm.validation.newPasswordRequired') },
            { min: 8, message: t('admin.userManagement.passwordForm.validation.passwordMin8') },
            { pattern: /^(?=.*[a-zA-Z])(?=.*\d).+$/, message: t('admin.userManagement.passwordForm.validation.passwordMustContainLettersAndNumbers') },
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
                if (!value || getFieldValue('newPassword') === value) return Promise.resolve();
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
            <Button onClick={onCancel}>{t('common.cancel')}</Button>
          </Space>
        </Form.Item>
      </Form>
    </Modal>
  );
}
