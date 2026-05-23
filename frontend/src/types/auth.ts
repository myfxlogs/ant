export interface LoginRequest {
  email: string;
  password: string;
}

export interface RegisterRequest {
  email: string;
  password: string;
  nickname?: string;
}

export interface User {
  id: string;
  email: string;
  nickname: string;
  avatar: string;
  role: string;
  status: string;
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

export interface TokenPair {
  access_token: string;
  refresh_token: string;
  expires_in: number;
}
