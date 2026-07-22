import { useCallback } from 'react';
import { Modal } from 'antd';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import { useAuthStore } from '@/stores/authStore';

/**
 * Returns a guard function. Call it before any auth-required action.
 * If authenticated, returns true. If not, shows a login/register prompt
 * and returns false.
 */
export function useAuthRequired(): () => boolean {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const isAuthenticated = useAuthStore(s => !!s.accessToken);

  return useCallback(() => {
    if (isAuthenticated) return true;
    Modal.confirm({
      title: t('common.loginRequired'),
      content: t('common.loginRequiredDesc'),
      okText: t('common.login'),
      cancelText: t('common.register'),
      onOk: () => navigate('/login'),
      onCancel: () => navigate('/register'),
    });
    return false;
  }, [isAuthenticated, t, navigate]);
}
