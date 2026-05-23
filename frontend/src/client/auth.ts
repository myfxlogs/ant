import { authClient } from './connect';

export type { User } from '../gen/api_pb';

export interface LoginResult {
  accessToken: string;
  refreshToken: string;
  expiresAt: bigint;
  user: any;
}

export interface RegisterResult {
  user: any;
}

export interface RefreshTokenResult {
  accessToken: string;
  refreshToken: string;
  expiresAt: bigint;
}



export const authApi = {
  login: async (email: string, password: string): Promise<LoginResult> => {
    const response: any = await authClient.login({ email, password });
    return {
      accessToken: response.accessToken,
      refreshToken: response.refreshToken,
      expiresAt: response.expiresAt,
      user: response.user,
    };
  },

  register: async (email: string, password: string, nickname: string): Promise<RegisterResult> => {
    const response: any = await authClient.register({ email, password, nickname });
    return {
      user: response.user,
    };
  },

  logout: async () => {
    await authClient.logout({});
  },

  getMe: async () => {
    const response: any = await authClient.getMe({});
    return response.user;
  },

  refreshToken: async (refreshToken: string): Promise<RefreshTokenResult> => {
    const response: any = await authClient.refreshToken({ refreshToken });
    return {
      accessToken: response.accessToken,
      refreshToken: response.refreshToken,
      expiresAt: response.expiresAt,
    };
  },
};
