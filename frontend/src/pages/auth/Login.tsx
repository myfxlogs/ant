import { useState } from 'react';
import { Button, Dropdown, Form, Input } from 'antd';
import { Link, useNavigate } from 'react-router-dom';
import { useAuth } from '@/hooks/useAuth';
import type { LoginRequest } from '@/types/auth';
import { useTranslation } from 'react-i18next';
import { PRIMARY_GRADIENT } from '@/components/common/GradientButton';
import i18n, { normalizeLanguage, setLanguage, type SupportedLanguage } from '@/i18n';
import Seo from '@/components/common/Seo';

export default function Login() {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();
  const { login } = useAuth();

  const languages: { key: SupportedLanguage; labelKey: string }[] = [
    { key: 'zh-cn', labelKey: 'language.simplifiedChinese' },
    { key: 'zh-tw', labelKey: 'language.traditionalChinese' },
    { key: 'en', labelKey: 'language.english' },
    { key: 'ja', labelKey: 'language.japanese' },
    { key: 'vi', labelKey: 'language.vietnamese' },
  ];

  const currentLang = normalizeLanguage(i18n.language);
  const languageMenu = {
    items: languages.map((lang) => ({
      key: lang.key,
      label: t(lang.labelKey),
    })),
    onClick: ({ key }: { key: string }) => setLanguage(normalizeLanguage(key)),
    selectedKeys: [currentLang],
  };

  const onFinish = async (values: LoginRequest) => {
    setLoading(true);
    try {
      const success = await login(values);
      if (success) {
        navigate('/');
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <>
    <Seo title="Login" description="Log in to AlphaForge to manage your MT4/MT5 trading strategies." path="/login" />
    <div
      className="min-h-screen flex flex-col items-center justify-center p-4"
      style={{ background: 'var(--color-bg-secondary)' }}
    >
      <div className="w-full max-w-md flex justify-end mb-3">
        <Dropdown menu={languageMenu} placement="bottomRight" trigger={['click']}>
          <button
            type="button"
            className="px-3 py-2 rounded-lg text-sm"
            style={{ background: 'var(--color-bg-card)', border: '1px solid rgba(0, 0, 0, 0.08)', color: 'var(--color-text-secondary)' }}
          >
            {t(languages.find((l) => l.key === currentLang)?.labelKey || 'language.english')}
          </button>
        </Dropdown>
      </div>
      <div 
        className="w-full max-w-md rounded-2xl overflow-hidden"
        style={{ 
          background: 'var(--color-bg-card)',
          boxShadow: '0 4px 24px rgba(0, 0, 0, 0.08)',
        }}
      >
        <div className="text中心 py-8 px-6" style={{ borderBottom: '1px solid rgba(0, 0, 0, 0.06)' }}>
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
          <p className="mt-1 text-sm" style={{ color: 'var(--color-text-muted)' }}>
            {t('auth.login.subtitle')}
          </p>
        </div>

        <div className="p-6">
          <Form
            name="login"
            onFinish={onFinish}
            autoComplete="off"
            layout="vertical"
            requiredMark={false}
          >
            <Form.Item
              name="login"
              rules={[
                { required: true, message: t('auth.validation.loginRequired', { defaultValue: 'Please enter your email or account number' }) },
              ]}
            >
              <Input
                placeholder={t('auth.fields.login', { defaultValue: '邮箱/账号' })}
                size="large"
                style={{
                  borderRadius: '10px',
                  padding: '14px 16px',
                }}
              />
            </Form.Item>

            <Form.Item
              name="password"
              rules={[
                { required: true, message: t('auth.validation.passwordRequired') },
                { min: 8, message: t('auth.validation.passwordMin8') },
              ]}
            >
              <Input 
                placeholder={t('auth.fields.password')}
                size="large"
                style={{
                  borderRadius: '10px',
                  padding: '14px 16px',
                }}
              />
            </Form.Item>

            <div className="flex items-center justify-between mb-4">
              <label className="flex items-center gap-2 cursor-pointer">
                <input 
                  type="checkbox" 
                  className="w-4 h-4 rounded"
                  style={{ accentColor: '#D4AF37' }}
                />
                <span style={{ color: 'var(--color-text-secondary)', fontSize: '14px' }}>{t('auth.login.rememberMe')}</span>
              </label>
              <Link 
                to="/forgot-password" 
                style={{ color: '#D4AF37', fontSize: '14px' }}
              >
                {t('auth.login.forgotPassword')}
              </Link>
            </div>

            <Form.Item>
              <Button
                type="primary"
                htmlType="submit"
                loading={loading}
                className="w-full font-semibold transition-all"
                style={{
                  background: 'linear-gradient(135deg, #D4AF37 0%, #B8960B 100%)',
                  borderRadius: '10px',
                  height: '48px',
                  color: '#FFFFFF',
                  fontSize: '16px',
                  border: 'none',
                }}
              >
                {loading ? t('auth.login.signingIn') : t('auth.login.login')}
              </Button>
            </Form.Item>
          </Form>

          <div className="text-center pt-4" style={{ borderTop: '1px solid rgba(0, 0, 0, 0.06)' }}>
            <span style={{ color: 'var(--color-text-muted)', fontSize: '14px' }}>{t('auth.login.noAccount')}</span>
            <Link 
              to="/register" 
              className="ml-2 font-medium"
              style={{ color: '#D4AF37', fontSize: '14px' }}
            >
              {t('auth.login.registerNow')}
            </Link>
          </div>
        </div>
      </div>
    </div>
    </>
  );
}
