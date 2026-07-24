import { useState } from 'react';
import { Button, Form, Input, message } from 'antd';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { LockOutlined } from '@ant-design/icons';
import { authClient } from '@/client/connect';
import Seo from '@/components/common/Seo';

export default function ResetPassword() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const [loading, setLoading] = useState(false);
  const token = searchParams.get('token') || '';

  const onFinish = async (values: { newPassword: string; confirmPassword: string }) => {
    if (values.newPassword !== values.confirmPassword) {
      message.error(t('auth.resetPassword.mismatch', 'Passwords do not match.'));
      return;
    }
    if (!token) {
      message.error(t('auth.resetPassword.invalidToken', 'Invalid or missing reset token.'));
      return;
    }
    setLoading(true);
    try {
      const resp = await authClient.resetPassword({ token, newPassword: values.newPassword });
      if (resp.success) {
        message.success(t('auth.resetPassword.success', 'Password has been reset. Please log in with your new password.'));
        navigate('/login');
      } else {
        message.error(resp.message || t('auth.resetPassword.failed', 'Failed to reset password.'));
      }
    } catch {
      message.error(t('auth.resetPassword.failed', 'Failed to reset password.'));
    } finally {
      setLoading(false);
    }
  };

  if (!token) {
    return (
      <>
        <Seo title={t('auth.seo.resetPassword.title')} description={t('auth.seo.resetPassword.description')} path="/reset-password" />
        <div className="min-h-screen flex items-center justify-center p-4" style={{ background: 'var(--color-bg-secondary)' }}>
          <div className="w-full max-w-md rounded-2xl overflow-hidden" style={{ background: 'var(--color-bg-card)', boxShadow: '0 4px 24px rgba(0, 0, 0, 0.08)' }}>
            <div className="py-6 px-6 text-center">
              <p className="text-sm" style={{ color: 'var(--color-text-secondary)' }}>
                {t('auth.resetPassword.invalidToken', 'Invalid or missing reset token.')}
              </p>
              <Link to="/forgot-password" style={{ color: '#D4AF37', fontSize: '14px' }}>
                {t('auth.forgotPassword.title', 'Reset Password')}
              </Link>
            </div>
          </div>
        </div>
      </>
    );
  }

  return (
    <>
      <Seo title={t('auth.seo.resetPassword.title')} description={t('auth.seo.resetPassword.description')} path="/reset-password" />
      <div className="min-h-screen flex items-center justify-center p-4" style={{ background: 'var(--color-bg-secondary)' }}>
        <div className="w-full max-w-md rounded-2xl overflow-hidden" style={{ background: 'var(--color-bg-card)', boxShadow: '0 4px 24px rgba(0, 0, 0, 0.08)' }}>
          <div className="py-6 px-6 text-center" style={{ borderBottom: '1px solid rgba(0, 0, 0, 0.06)' }}>
            <div className="inline-flex items-center justify-center w-12 h-12 rounded-xl mb-3" style={{ background: 'linear-gradient(135deg, #D4AF37 0%, #B8960B 100%)' }}>
              <LockOutlined style={{ color: '#FFFFFF', fontSize: 22 }} />
            </div>
            <h1 className="text-xl font-bold" style={{ color: 'var(--color-text)' }}>
              {t('auth.resetPassword.title', 'Set New Password')}
            </h1>
          </div>
          <div className="p-6">
            <Form layout="vertical" onFinish={onFinish} requiredMark={false}>
              <Form.Item
                name="newPassword"
                rules={[
                  { required: true, message: t('auth.validation.passwordRequired', 'Please enter your password') },
                  { min: 8, message: t('auth.validation.passwordMin8', 'Password must be at least 8 characters') },
                ]}
              >
                <Input.Password size="large" placeholder={t('auth.resetPassword.newPassword', 'New Password')} style={{ borderRadius: '10px' }} />
              </Form.Item>
              <Form.Item
                name="confirmPassword"
                rules={[
                  { required: true, message: t('auth.resetPassword.confirmRequired', 'Please confirm your password') },
                ]}
              >
                <Input.Password size="large" placeholder={t('auth.resetPassword.confirmPassword', 'Confirm Password')} style={{ borderRadius: '10px' }} />
              </Form.Item>
              <Form.Item>
                <Button
                  type="primary"
                  htmlType="submit"
                  loading={loading}
                  className="w-full font-semibold"
                  style={{ background: 'linear-gradient(135deg, #D4AF37 0%, #B8960B 100%)', borderRadius: '10px', height: '48px', color: '#FFFFFF', fontSize: '16px', border: 'none' }}
                >
                  {t('auth.resetPassword.submit', 'Reset Password')}
                </Button>
              </Form.Item>
            </Form>
            <div className="text-center pt-4" style={{ borderTop: '1px solid rgba(0, 0, 0, 0.06)' }}>
              <Link to="/login" style={{ color: '#D4AF37', fontSize: '14px' }}>
                {t('auth.forgotPassword.backToLogin', 'Back to Login')}
              </Link>
            </div>
          </div>
        </div>
      </div>
    </>
  );
}
