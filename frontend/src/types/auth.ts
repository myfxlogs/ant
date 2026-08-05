export interface LoginRequest {
  login: string;
  password: string;
  rememberMe?: boolean;
}

export interface User {
  id: string;
  email: string;
  nickname: string;
  avatar: string;
  role: string;
  permissions: string[];
  capabilityTier: number;
  status: string;
  accountNumber: string;
  last_login_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface LoginResponse {
  user: User;
  access_token: string;
  refresh_token: string;
  expires_in: number;
}

export interface RegisterResponse {
  id: string;
  email: string;
  nickname: string;
  created_at: string;
}
