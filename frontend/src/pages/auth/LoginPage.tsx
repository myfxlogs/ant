import { useState } from 'react';
import { Card, Form, Input, Button, message, Typography } from 'antd';
import { UserOutlined, LockOutlined } from '@ant-design/icons';

const { Title } = Typography;

interface Props {
  onLogin: (token: string, userId: string) => void;
}

export default function LoginPage({ onLogin }: Props) {
  const [loading, setLoading] = useState(false);

  async function handleSubmit(values: { email: string; password: string }) {
    setLoading(true);
    try {
      // For v2: fetch a JWT from the auth endpoint
      // Currently: accept any email and store it as user identity
      const userId = values.email;
      const token = btoa(`${userId}:${Date.now()}`);
      localStorage.setItem('auth_token', token);
      localStorage.setItem('userId', userId);
      message.success('Logged in');
      onLogin(token, userId);
    } catch {
      message.error('Login failed');
    } finally {
      setLoading(false);
    }
  }

  return (
    <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '80vh' }}>
      <Card style={{ width: 400 }}>
        <Title level={3} style={{ textAlign: 'center' }}>Ant v2</Title>
        <Form onFinish={handleSubmit} layout="vertical">
          <Form.Item name="email" rules={[{ required: true, message: 'Enter your email' }]}>
            <Input prefix={<UserOutlined />} placeholder="Email" size="large" />
          </Form.Item>
          <Form.Item name="password" rules={[{ required: true, message: 'Enter your password' }]}>
            <Input.Password prefix={<LockOutlined />} placeholder="Password" size="large" />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit" loading={loading} block size="large">
              Sign In
            </Button>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
}
