import { useState } from 'react';
import { Card, Form, Input, Button, message, Typography } from 'antd';
import { UserOutlined, LockOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';

const { Title } = Typography;

interface Props {
  onLogin: (token: string, userId: string) => void;
}

export default function LoginPage({ onLogin }: Props) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);

  async function handleSubmit(values: { email: string; password: string }) {
    setLoading(true);
    try {
      const userId = values.email;
      const token = btoa(`${userId}:${Date.now()}`);
      localStorage.setItem('auth_token', token);
      localStorage.setItem('userId', userId);
      message.success(t('auth.loginSuccess', 'Logged in'));
      onLogin(token, userId);
    } catch {
      message.error(t('auth.loginFailed', 'Login failed'));
    } finally {
      setLoading(false);
    }
  }

  return (
    <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '80vh' }}>
      <Card style={{ width: 400 }}>
        <Title level={3} style={{ textAlign: 'center' }}>{t('app.name', 'Ant v2')}</Title>
        <Form onFinish={handleSubmit} layout="vertical">
          <Form.Item name="email" rules={[{ required: true, message: t('auth.emailRequired', 'Enter your email') }]}>
            <Input prefix={<UserOutlined />} placeholder={t('auth.email', 'Email')} size="large" />
          </Form.Item>
          <Form.Item name="password" rules={[{ required: true, message: t('auth.passwordRequired', 'Enter your password') }]}>
            <Input.Password prefix={<LockOutlined />} placeholder={t('auth.password', 'Password')} size="large" />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit" loading={loading} block size="large">
              {t('auth.signIn', 'Sign In')}
            </Button>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
}
