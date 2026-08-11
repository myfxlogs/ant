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
