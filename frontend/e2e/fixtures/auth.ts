import { type Page, type APIRequestContext, expect } from '@playwright/test';

export interface AuthSession {
  accessToken: string;
  userId: string;
  email: string;
  username: string;
  role: string;
}

export interface LoginResponse {
  accessToken: string;
  user: {
    id: string;
    username: string;
    email: string;
    role: string;
  };
}

export const E2E_TEST_USER = {
  email: 'e2e@test.com',
  password: 'E2etest123!',
  username: 'e2e_tester',
};

export const ADMIN_USER = {
  email: 'admin@1.com',
  password: 'admin123',
};

async function apiLogin(
  request: APIRequestContext,
  login: string,
  password: string,
): Promise<AuthSession> {
  const resp = await request.post('/ant.v1.AuthService/Login', {
    data: { login, password },
    headers: { 'Content-Type': 'application/json' },
  });
  expect(resp.ok(), `Login should succeed for ${login}`).toBe(true);
  const body = (await resp.json()) as LoginResponse;
  return {
    accessToken: body.accessToken,
    userId: body.user.id,
    email: body.user.email,
    username: body.user.username,
    role: body.user.role,
  };
}

export async function loginAsTestUser(request: APIRequestContext): Promise<AuthSession> {
  return apiLogin(request, E2E_TEST_USER.email, E2E_TEST_USER.password);
}

export async function loginAsAdmin(request: APIRequestContext): Promise<AuthSession> {
  return apiLogin(request, ADMIN_USER.email, ADMIN_USER.password);
}

export async function injectAuthState(page: Page, session: AuthSession): Promise<void> {
  await page.addInitScript((auth) => {
    const state = {
      state: {
        user: {
          id: auth.userId,
          email: auth.email,
          username: auth.username,
          role: auth.role,
        },
        accessToken: auth.accessToken,
        isAuthenticated: true,
        _hasHydrated: true,
        _rememberMe: false,
      },
      version: 0,
    };
    localStorage.setItem('auth-storage', JSON.stringify(state));
  }, session);
}

export async function registerTestUser(request: APIRequestContext): Promise<void> {
  const resp = await request.post('/ant.v1.AuthService/Register', {
    data: {
      email: E2E_TEST_USER.email,
      password: E2E_TEST_USER.password,
      username: E2E_TEST_USER.username,
    },
    headers: { 'Content-Type': 'application/json' },
  });
  if (resp.status() === 409 || resp.status() === 400) {
    return;
  }
  expect(resp.ok(), 'Register should succeed or user already exists').toBe(true);
}
