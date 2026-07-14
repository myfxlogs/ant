import { authClient } from './connect';
import type { User } from '@/types/auth';
import type { LoginResponse, RegisterResponse, GetMeResponse, VerifyEmailResponse, ResendVerificationResponse, RefreshTokenResponse } from '../gen/ant/v1/auth_pb';

export type { User };

export interface LoginResult {
  accessToken: string;
  refreshToken?: string;
  expiresAt: bigint;
  user: User;
}

export interface RegisterResult {
  user: User;
  emailVerificationSent: boolean;
}

export interface RefreshTokenResult {
  accessToken: string;
  refreshToken: string;
  expiresAt: bigint;
}

export const authApi = {
  login: async (login: string, password: string): Promise<LoginResult> => {
    const msg = await authClient.login({ login, password }) as LoginResponse;
    return {
      accessToken: msg.accessToken,
      expiresAt: msg.expiresAt,
      user: msg.user as unknown as User,
    };
  },

  register: async (email: string, password: string, username?: string): Promise<RegisterResult> => {
    const msg = await authClient.register({ email, password, username: username || email }) as RegisterResponse;
    return {
      user: msg.user as unknown as User,
      emailVerificationSent: msg.emailVerificationSent ?? false,
    };
  },

  logout: async () => {
    await authClient.logout({});
  },

  getMe: async () => {
    return (await authClient.getMe({}) as GetMeResponse).user as unknown as User;
  },

  verifyEmail: async (token: string): Promise<{ success: boolean; message: string }> => {
    const msg = await authClient.verifyEmail({ token }) as VerifyEmailResponse;
    return { success: msg.success, message: msg.message };
  },

  resendVerification: async (email: string): Promise<{ success: boolean; message: string }> => {
    const msg = await authClient.resendVerification({ email }) as ResendVerificationResponse;
    return { success: msg.success, message: msg.message };
  },

  refreshToken: async (refreshToken: string): Promise<RefreshTokenResult> => {
    const msg = await authClient.refreshToken({ refreshToken }) as RefreshTokenResponse;
    return {
      accessToken: msg.accessToken,
      refreshToken: msg.refreshToken,
      expiresAt: msg.expiresAt,
    };
  },
};
