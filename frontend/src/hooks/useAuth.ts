import { useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuthStore } from '@/stores/authStore';
import { authApi } from '@/client/auth';
import { getErrorMessage } from '@/utils/error';
import { showError, showSuccess, showWarning } from '@/utils/message';
import i18n from '@/i18n';

export function useAuth() {
  const navigate = useNavigate();
  const { user, isAuthenticated, setTokens, logout: storeLogout } = useAuthStore();

  const login = useCallback(async (data: { email: string; password: string }) => {
    try {
      const response = await authApi.login(data.email, data.password);
      setTokens(response.accessToken, response.refreshToken || '', response.user as any);
      showSuccess(i18n.t('auth.messages.loginSuccess'));
      navigate('/');
      return true;
    } catch (error) {
      showError(getErrorMessage(error, i18n.t('auth.messages.loginFailed')));
      return false;
    }
  }, [setTokens, navigate]);

  const register = useCallback(async (data: { email: string; password: string; nickname?: string }) => {
    try {
      await authApi.register(data.email, data.password, data.nickname || data.email);
      showSuccess(i18n.t('auth.messages.registerSuccess'));
      navigate('/login');
      return true;
    } catch (error) {
      showError(getErrorMessage(error, i18n.t('auth.messages.registerFailed')));
      return false;
    }
  }, [navigate]);

  const logout = useCallback(async () => {
    try {
      await authApi.logout();
    } catch {
      // ignore
    }
    storeLogout();
    navigate('/login');
    showWarning(i18n.t('auth.messages.logoutSuccess'));
  }, [storeLogout, navigate]);

  const getMe = useCallback(async () => {
    try {
      const user = await authApi.getMe();
      return user;
    } catch (error) {
      showError(getErrorMessage(error, i18n.t('auth.messages.fetchMeFailed')));
      return null;
    }
  }, []);

  return {
    user,
    isAuthenticated,
    login,
    register,
    logout,
    getMe,
  };
}
