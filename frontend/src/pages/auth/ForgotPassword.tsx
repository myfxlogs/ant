import { Result, Button } from 'antd';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { MailOutlined } from '@ant-design/icons';

export default function ForgotPassword() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  return (
    <div className="min-h-screen flex items-center justify-center" style={{ background: '#F5F7F9' }}>
      <Result
        icon={<MailOutlined style={{ color: '#D4AF37', fontSize: 64 }} />}
        title={t('auth.forgotPassword.title', 'Reset Password')}
        subTitle={t('auth.forgotPassword.hint', 'Please contact your administrator or support to reset your password.')}
        extra={
          <Button type="primary" onClick={() => navigate('/login')}>
            {t('auth.forgotPassword.backToLogin', 'Back to Login')}
          </Button>
        }
      />
    </div>
  );
}
