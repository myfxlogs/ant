import { useState, useEffect } from 'react';
import { useSearchParams, Link, useNavigate } from 'react-router-dom';
import { Button, Input, Spin, message } from 'antd';
import { CheckCircleOutlined, CloseCircleOutlined, MailOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { authApi } from '@/client/auth';
import { PRIMARY_GRADIENT } from '@/components/common/GradientButton';
import Seo from '@/components/common/Seo';

export default function VerifyEmail() {
  const { t } = useTranslation();
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const token = searchParams.get('token');
  const email = searchParams.get('email') || '';
  const [status, setStatus] = useState<'verifying' | 'success' | 'failed' | 'pending'>('pending');
  const [resultMessage, setResultMessage] = useState('');
  const [resending, setResending] = useState(false);

  useEffect(() => {
    if (token) {
      setStatus('verifying');
      authApi.verifyEmail(token).then((res) => {
        if (res.success) {
          setStatus('success');
        } else {
          setStatus('failed');
        }
        setResultMessage(res.message);
      }).catch(() => {
        setStatus('failed');
        setResultMessage(t('auth.verify.failed'));
      });
    }
  }, [token, t]);

  const handleResend = async () => {
    if (!email) return;
    setResending(true);
    try {
      const res = await authApi.resendVerification(email);
      if (res.success) {
        message.success(res.message);
      } else {
        message.error(res.message);
      }
    } catch {
      message.error(t('auth.verify.resendFailed'));
    } finally {
      setResending(false);
    }
  };

  const renderContent = () => {
    if (status === 'verifying') {
      return (
        <div className="text-center py-8">
          <Spin size="large" />
          <p className="mt-4 text-sm" style={{ color: 'var(--color-text-muted)' }}>
            {t('auth.verify.verifying')}
          </p>
        </div>
      );
    }

    if (status === 'success') {
      return (
        <div className="text-center py-8">
          <CheckCircleOutlined style={{ fontSize: 48, color: '#52c41a' }} />
          <h2 className="mt-4 text-xl font-bold" style={{ color: 'var(--color-text)' }}>
            {t('auth.verify.successTitle')}
          </h2>
          <p className="mt-2 text-sm" style={{ color: 'var(--color-text-muted)' }}>
            {resultMessage || t('auth.verify.successDesc')}
          </p>
          <Button
            type="primary"
            className="mt-6"
            onClick={() => navigate('/login')}
            style={{ background: PRIMARY_GRADIENT, border: 'none', borderRadius: '10px', height: '44px' }}
          >
            {t('auth.verify.goLogin')}
          </Button>
        </div>
      );
    }

    if (status === 'failed') {
      return (
        <div className="text-center py-8">
          <CloseCircleOutlined style={{ fontSize: 48, color: '#ff4d4f' }} />
          <h2 className="mt-4 text-xl font-bold" style={{ color: 'var(--color-text)' }}>
            {t('auth.verify.failedTitle')}
          </h2>
          <p className="mt-2 text-sm" style={{ color: 'var(--color-text-muted)' }}>
            {resultMessage || t('auth.verify.failedDesc')}
          </p>
          {email && (
            <Button
              className="mt-6"
 onClick={handleResend}
              loading={resending}
              style={{ borderRadius: '10px', height: '44px' }}
            >
              {t('auth.verify.resend')}
            </Button>
          )}
          <div className="mt-4">
            <Link to="/login" style={{ color: '#D4AF37', fontSize: '14px' }}>
              {t('auth.verify.goLogin')}
            </Link>
          </div>
        </div>
      );
    }

    // pending — show "check your email" prompt
    return (
      <div className="text-center py-8">
        <MailOutlined style={{ fontSize: 48, color: '#D4AF37' }} />
        <h2 className="mt-4 text-xl font-bold" style={{ color: 'var(--color-text)' }}>
          {t('auth.verify.pendingTitle')}
        </h2>
        <p className="mt-2 text-sm" style={{ color: 'var(--color-text-muted)' }}>
          {t('auth.verify.pendingDesc')}
        </p>
        {email && (
          <div className="mt-6">
            <p className="text-sm mb-3" style={{ color: 'var(--color-text-muted)' }}>
              {t('auth.verify.didntReceive')}
            </p>
            <Button
              onClick={handleResend}
              loading={resending}
              style={{ borderRadius: '10px', height: '44px' }}
            >
              {t('auth.verify.resend')}
            </Button>
          </div>
        )}
        <div className="mt-4">
          <Link to="/login" style={{ color: '#D4AF37', fontSize: '14px' }}>
            {t('auth.verify.goLogin')}
          </Link>
        </div>
      </div>
    );
  };

  return (
    <>
      <Seo title="Verify Email" description="Verify your email address" path="/verify-email" />
      <div
        className="min-h-screen flex flex-col items-center justify-center p-4"
        style={{ background: 'var(--color-bg-secondary)' }}
      >
        <div
          className="w-full max-w-md rounded-2xl overflow-hidden"
          style={{
            background: 'var(--color-bg-card)',
            boxShadow: '0 4px 24px rgba(0, 0, 0, 0.08)',
          }}
        >
          <div className="text-center py-6 px-6" style={{ borderBottom: '1px solid rgba(0, 0, 0, 0.06)' }}>
            <div
              className="inline-flex items-center justify-center w-14 h-14 rounded-xl mb-4"
              style={{ background: PRIMARY_GRADIENT }}
            >
              <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="#FFFFFF" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <path d="M13 7h8m0 0v8m0-8l-8 8-4-4-6 6" />
              </svg>
            </div>
            <h1 className="text-2xl font-bold" style={{ fontFamily: 'Poppins, sans-serif', color: 'var(--color-text)' }}>
              {t('app.name')}
            </h1>
          </div>
          <div className="p-6">
            {renderContent()}
          </div>
        </div>
      </div>
    </>
  );
}
