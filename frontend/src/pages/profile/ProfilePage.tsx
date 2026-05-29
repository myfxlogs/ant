import { Card, Descriptions, Avatar, Tag, Typography } from 'antd';
import { UserOutlined, MailOutlined, CalendarOutlined, SafetyCertificateOutlined, ClockCircleOutlined } from '@ant-design/icons';
import { useAuth } from '@/hooks/useAuth';
import { useTranslation } from 'react-i18next';

const { Title } = Typography;

export default function ProfilePage() {
  const { user } = useAuth();
  const { t } = useTranslation();

  if (!user) return null;

  const roleColors: Record<string, string> = {
    super_admin: 'red',
    admin: 'orange',
    operator: 'blue',
    user: 'green',
  };

  return (
    <div style={{ maxWidth: 720 }}>
      <Title level={4} style={{ marginBottom: 24 }}>{t('profile.title')}</Title>
      <Card>
        <div style={{ display: 'flex', alignItems: 'center', gap: 20, marginBottom: 24 }}>
          <Avatar size={72} icon={<UserOutlined />} style={{ background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)' }} />
          <div>
            <div style={{ fontSize: 20, fontWeight: 600 }}>{user.nickname || user.email}</div>
            <div style={{ color: '#8c8c8c' }}>{user.email}</div>
          </div>
        </div>
        <Descriptions column={1} bordered size="middle">
          <Descriptions.Item label={<><MailOutlined style={{ marginRight: 8 }} />{t('auth.fields.email')}</>}>
            {user.email}
          </Descriptions.Item>
          <Descriptions.Item label={<><UserOutlined style={{ marginRight: 8 }} />{t('profile.nickname')}</>}>
            {user.nickname || '-'}
          </Descriptions.Item>
          <Descriptions.Item label={<><SafetyCertificateOutlined style={{ marginRight: 8 }} />{t('profile.role')}</>}>
            <Tag color={roleColors[user.role] || 'default'}>{user.role}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label={<><ClockCircleOutlined style={{ marginRight: 8 }} />{t('profile.status')}</>}>
            <Tag color={user.status === 'active' ? 'green' : 'red'}>{user.status}</Tag>
          </Descriptions.Item>
          {user.last_login_at && (
            <Descriptions.Item label={<><CalendarOutlined style={{ marginRight: 8 }} />{t('profile.lastLogin')}</>}>
              {new Date(user.last_login_at).toLocaleString()}
            </Descriptions.Item>
          )}
          <Descriptions.Item label={<><CalendarOutlined style={{ marginRight: 8 }} />{t('profile.registered')}</>}>
            {new Date(user.created_at).toLocaleString()}
          </Descriptions.Item>
        </Descriptions>
      </Card>
    </div>
  );
}
