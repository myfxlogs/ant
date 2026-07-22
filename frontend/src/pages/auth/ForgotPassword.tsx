import { useState } from 'react';
import { Button, Form, Input, Tabs, message } from 'antd';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { MailOutlined, LockOutlined, SafetyCertificateOutlined, CustomerServiceOutlined } from '@ant-design/icons';
import { authClient } from '@/client/connect';
import Seo from '@/components/common/Seo';

const EMAIL_REGEX = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
const isEmailValid = (email: string) => EMAIL_REGEX.test(email.trim());

export default function ForgotPassword() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const [emailLoading, setEmailLoading] = useState(false);
  const [mtLoading, setMtLoading] = useState(false);

  // MT verification state — email + login + password.
  // broker_host and platform are looked up from DB by the backend.
  const [mtEmail, setMtEmail] = useState('');
  const [mtLogin, setMtLogin] = useState('');
  const [mtPassword, setMtPassword] = useState('');

  const onEmailSubmit = async (values: { email: string }) => {
    setEmailLoading(true);
    try {
      await authClient.forgotPassword({ email: values.email });
      message.success(t('auth.forgotPassword.emailSent', 'If the email exists, a reset link has been sent.'));
    } catch {
      message.success(t('auth.forgotPassword.emailSent', 'If the email exists, a reset link has been sent.'));
    } finally {
      setEmailLoading(false);
    }
  };

  const onMTSubmit = async () => {
    if (!mtEmail.trim() || !mtLogin.trim() || !mtPassword.trim()) return;
    setMtLoading(true);
    try {
      const resp = await authClient.verifyMTIdentity({
        email: mtEmail.trim(),
        mtLogin: mtLogin.trim(),
        mtPassword: mtPassword,
      });
      if (resp.verified && resp.resetToken) {
        message.success(t('auth.forgotPassword.mtVerified', 'Identity verified. Redirecting to password reset.'));
        navigate(`/reset-password?token=${resp.resetToken}`);
      } else {
        message.error(resp.message || t('auth.forgotPassword.mtFailed', 'MT credential verification failed.'));
      }
    } catch {
      message.error(t('auth.forgotPassword.mtFailed', 'MT credential verification failed.'));
    } finally {
      setMtLoading(false);
    }
  };

  const tabItems = [
    {
      key: 'email',
      label: (
        <span className="flex items-center gap-1.5">
          <MailOutlined />
          {t('auth.forgotPassword.emailTab', 'Email')}
        </span>
      ),
      children: (
        <Form layout="vertical" onFinish={onEmailSubmit} requiredMark={false}>
          <Form.Item
            name="email"
            rules={[{ required: true, message: t('auth.validation.emailRequired', 'Please enter your email') }]}
          >
            <Input size="large" placeholder={t('auth.fields.email', 'Email')} style={{ borderRadius: '10px' }} />
          </Form.Item>
          <Form.Item>
            <Button
              type="primary"
              htmlType="submit"
              loading={emailLoading}
              className="w-full font-semibold"
              style={{ background: 'linear-gradient(135deg, #D4AF37 0%, #B8960B 100%)', borderRadius: '10px', height: '48px', color: '#FFFFFF', fontSize: '16px', border: 'none' }}
            >
              {t('auth.forgotPassword.sendResetLink', 'Send Reset Link')}
            </Button>
          </Form.Item>
        </Form>
      ),
    },
    {
      key: 'mt',
      label: (
        <span className="flex items-center gap-1.5">
          <SafetyCertificateOutlined />
          {t('auth.forgotPassword.mtTab', 'MT Verify')}
        </span>
      ),
      children: (
        <div className="space-y-4">
          <div>
            <label className="block mb-2 font-medium text-sm" style={{ color: 'var(--color-text)' }}>
              {t('auth.fields.email', 'Email')}
            </label>
            <Input
              value={mtEmail}
              onChange={(e) => setMtEmail(e.target.value)}
              placeholder={t('auth.fields.email', 'Email')}
              size="large"
              status={mtEmail && !isEmailValid(mtEmail) ? 'error' : ''}
              style={{ borderRadius: '10px' }}
            />
            {mtEmail && !isEmailValid(mtEmail) && (
              <p className="text-xs mt-1" style={{ color: 'var(--color-error)' }}>
                {t('auth.validation.emailInvalid', 'Please enter a valid email address')}
              </p>
            )}
          </div>
          <div>
            <label className="block mb-2 font-medium text-sm" style={{ color: 'var(--color-text)' }}>
              {t('auth.forgotPassword.mtLogin', 'MT Account Number')}
            </label>
            <Input
              value={mtLogin}
              onChange={(e) => setMtLogin(e.target.value)}
              placeholder={t('auth.forgotPassword.mtLoginPlaceholder', 'e.g. 12345678')}
              size="large"
              style={{ borderRadius: '10px' }}
            />
          </div>
          <div>
            <label className="block mb-2 font-medium text-sm" style={{ color: 'var(--color-text)' }}>
              {t('auth.forgotPassword.mtPassword', 'MT Password')}
            </label>
            <Input.Password
              value={mtPassword}
              onChange={(e) => setMtPassword(e.target.value)}
              placeholder={t('auth.forgotPassword.mtPasswordPlaceholder', 'MT trading password')}
              size="large"
              style={{ borderRadius: '10px' }}
            />
          </div>
          <p className="text-xs" style={{ color: 'var(--color-text-muted)' }}>
            {t('auth.forgotPassword.mtHint', 'Enter your bound MT account credentials to verify your identity. Server and platform are detected automatically.')}
          </p>
          <Button
            type="primary"
            onClick={onMTSubmit}
            loading={mtLoading}
            disabled={!mtEmail.trim() || !isEmailValid(mtEmail) || !mtLogin.trim() || !mtPassword.trim()}
            className="w-full font-semibold"
            style={{ background: 'linear-gradient(135deg, #D4AF37 0%, #B8960B 100%)', borderRadius: '10px', height: '48px', color: '#FFFFFF', fontSize: '16px', border: 'none' }}
          >
            {t('auth.forgotPassword.verifyAndReset', 'Verify & Reset Password')}
          </Button>
        </div>
      ),
    },
    {
      key: 'admin',
      label: (
        <span className="flex items-center gap-1.5">
          <CustomerServiceOutlined />
          {t('auth.forgotPassword.adminTab', 'Admin')}
        </span>
      ),
      children: (
        <div className="text-center py-6">
          <CustomerServiceOutlined style={{ fontSize: 40, color: '#D4AF37' }} />
          <p className="mt-4 text-sm" style={{ color: 'var(--color-text-secondary)' }}>
            {t('auth.forgotPassword.adminHint', 'Please contact your administrator or support to reset your password.')}
          </p>
        </div>
      ),
    },
  ];

  const initialTab = searchParams.get('tab') === 'mt' ? 'mt' : 'email';

  return (
    <>
      <Seo title="Forgot Password" description="Reset your AlphaForge account password." path="/forgot-password" />
      <div className="min-h-screen flex items-center justify-center p-4" style={{ background: 'var(--color-bg-secondary)' }}>
        <div className="w-full max-w-md rounded-2xl overflow-hidden" style={{ background: 'var(--color-bg-card)', boxShadow: '0 4px 24px rgba(0, 0, 0, 0.08)' }}>
          <div className="py-6 px-6 text-center" style={{ borderBottom: '1px solid rgba(0, 0, 0, 0.06)' }}>
            <div className="inline-flex items-center justify-center w-12 h-12 rounded-xl mb-3" style={{ background: 'linear-gradient(135deg, #D4AF37 0%, #B8960B 100%)' }}>
              <LockOutlined style={{ color: '#FFFFFF', fontSize: 22 }} />
            </div>
            <h1 className="text-xl font-bold" style={{ color: 'var(--color-text)' }}>
              {t('auth.forgotPassword.title', 'Reset Password')}
            </h1>
          </div>
          <div className="p-6">
            <Tabs items={tabItems} defaultActiveKey={initialTab} centered />
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
