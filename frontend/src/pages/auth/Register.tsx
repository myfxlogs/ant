import { useState } from 'react';
import { Button, Dropdown, Form, Input } from 'antd';
import { Link, useNavigate } from 'react-router-dom';
import { useAuth } from '@/hooks/useAuth';
import { useTranslation } from 'react-i18next';
import { PRIMARY_GRADIENT } from '@/components/common/GradientButton';
import i18n, { normalizeLanguage, setLanguage, type SupportedLanguage } from '@/i18n';

export default function Register() {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();
  const { register } = useAuth();

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

  const onFinish = async (values: any) => {
    setLoading(true);
    try {
      const success = await register({
        password: values.password,
        email: values.email,
      });
      if (success) {
        navigate('/login');
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <div 
      className="min-h-screen flex flex-col items-center justify-center p-4"
      style={{ background: '#F5F7F9' }}
    >
      <div className="w-full max-w-md flex justify-end mb-3">
        <Dropdown menu={languageMenu} placement="bottomRight" trigger={['click']}>
          <button
            type="button"
            className="px-3 py-2 rounded-lg text-sm"
            style={{ background: '#FFFFFF', border: '1px solid rgba(0, 0, 0, 0.08)', color: '#5A6B75' }}
          >
            {t(languages.find((l) => l.key === currentLang)?.labelKey || 'language.english')}
          </button>
        </Dropdown>
      </div>
      <div 
        className="w-full max-w-md rounded-2xl overflow-hidden"
        style={{ 
          background: '#FFFFFF',
          boxShadow: '0 4px 24px rgba(0, 0, 0, 0.08)',
        }}
      >
        {/* Logo 区域 */}
        <div className="text-center py-8 px-6" style={{ borderBottom: '1px solid rgba(0, 0, 0, 0.06)' }}>
          <div 
            className="inline-flex items-center justify-center w-14 h-14 rounded-xl mb-4"
            style={{ background: PRIMARY_GRADIENT }}
          >
            <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="#FFFFFF" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M13 7h8m0 0v8m0-8l-8 8-4-4-6 6" />
            </svg>
          </div>
          <h1 className="text-2xl font-bold" style={{ fontFamily: 'Poppins, sans-serif', color: '#141D22' }}>
            {t('app.name')}
          </h1>
          <p className="mt-1 text-sm" style={{ color: '#8A9AA5' }}>
            {t('auth.register.subtitle')}
          </p>
        </div>

        {/* 表单区域 */}
        <div className="p-6">
          <Form
            name="register"
            onFinish={onFinish}
            autoComplete="off"
            layout="vertical"
            requiredMark={false}
          >
            <Form.Item
              name="email"
              rules={[
                { required: true, message: t('auth.validation.emailRequired') },
                { type: 'email', message: t('auth.validation.emailInvalid') },
              ]}
            >
              <div className="relative">
                <Input
                  type="email"
                  placeholder={t('auth.fields.email')}
                  className="w-full outline-none transition-all"
                  style={{
                    background: '#FFFFFF',
                    border: '1px solid rgba(185, 201, 223, 0.4)',
                    borderRadius: '10px',
                    padding: '14px 16px',
                    fontSize: '16px',
                    color: '#141D22',
                  }}
                  onFocus={(e) => {
                    e.target.style.borderColor = '#D4AF37';
                  }}
                  onBlur={(e) => {
                    e.target.style.borderColor = 'rgba(185, 201, 223, 0.4)';
                  }}
                />
              </div>
            </Form.Item>

            <Form.Item
              name="password"
              rules={[
                { required: true, message: t('auth.validation.passwordRequired') },
                { min: 8, message: t('auth.validation.passwordMin8') },
              ]}
            >
              <div className="relative">
                <Input
                  type="text"
                  placeholder={t('auth.fields.password')}
                  className="w-full outline-none transition-all pr-12"
                  style={{
                    background: '#FFFFFF',
                    border: '1px solid rgba(185, 201, 223, 0.4)',
                    borderRadius: '10px',
                    padding: '14px 16px',
                    fontSize: '16px',
                    color: '#141D22',
                  }}
                  onFocus={(e) => {
                    (e.target as HTMLInputElement).style.borderColor = '#D4AF37';
                  }}
                  onBlur={(e) => {
                    (e.target as HTMLInputElement).style.borderColor = 'rgba(185, 201, 223, 0.4)';
                  }}
                />
              </div>
            </Form.Item>

            <Form.Item
              name="confirmPassword"
              dependencies={['password']}
              rules={[
                { required: true, message: t('auth.validation.confirmPasswordRequired') },
                ({ getFieldValue }) => ({
                  validator(_, value) {
                    if (!value || getFieldValue('password') === value) {
                      return Promise.resolve();
                    }
                    return Promise.reject(new Error(t('auth.validation.passwordMismatch')));
                  },
                }),
              ]}
            >
              <div className="relative">
                <Input
                  type="text"
                  placeholder={t('auth.fields.confirmPassword')}
                  className="w-full outline-none transition-all pr-12"
                  style={{
                    background: '#FFFFFF',
                    border: '1px solid rgba(185, 201, 223, 0.4)',
                    borderRadius: '10px',
                    padding: '14px 16px',
                    fontSize: '16px',
                    color: '#141D22',
                  }}
                  onFocus={(e) => {
                    (e.target as HTMLInputElement).style.borderColor = '#D4AF37';
                  }}
                  onBlur={(e) => {
                    (e.target as HTMLInputElement).style.borderColor = 'rgba(185, 201, 223, 0.4)';
                  }}
                />
              </div>
            </Form.Item>

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
                {loading ? t('auth.register.signingUp') : t('auth.register.register')}
              </Button>
            </Form.Item>
          </Form>

          <div className="text-center pt-4" style={{ borderTop: '1px solid rgba(0, 0, 0, 0.06)' }}>
            <span style={{ color: '#8A9AA5', fontSize: '14px' }}>{t('auth.register.haveAccount')}</span>
            <Link 
              to="/login" 
              className="ml-2 font-medium"
              style={{ color: '#D4AF37', fontSize: '14px' }}
            >
              {t('auth.register.loginNow')}
            </Link>
          </div>
        </div>
      </div>

      {/* 底部信息 */}
      <div className="text-center mt-6" style={{ color: '#8A9AA5', fontSize: '12px' }}>
        <p>
          {t('auth.register.agreePrefix')}
          <Link to="/terms" style={{ color: '#D4AF37' }}> {t('auth.register.terms')} </Link>
          {t('auth.register.and')}
          <Link to="/privacy" style={{ color: '#D4AF37' }}> {t('auth.register.privacy')} </Link>
        </p>
      </div>
    </div>
  );
}
