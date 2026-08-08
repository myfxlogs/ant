import { test, expect } from '@playwright/test';
import { loginAsTestUser, injectAuthState, type AuthSession } from './fixtures/auth';

test('protected API without token returns 401', async ({ request }) => {
  const resp = await request.post('/ant.v1.MarketplaceService/ListSubscriptions', {
    data: {},
    headers: { 'Content-Type': 'application/json' },
  });
  expect(resp.ok(), 'API without token should fail').toBe(false);
  expect(resp.status()).toBe(401);
});

test.describe('Adversarial proof: Journey 3 purchase → live trading', () => {
  let session: AuthSession;

  test.beforeAll(async ({ request }) => {
    session = await loginAsTestUser(request);
  });

  test('API login with wrong password fails', async ({ request }) => {
    const resp = await request.post('/ant.v1.AuthService/Login', {
      data: { login: session.email, password: 'WRONG_PASSWORD' },
      headers: { 'Content-Type': 'application/json' },
    });
    expect(resp.ok(), 'Login with wrong password should fail').toBe(false);
    expect(resp.status()).toBe(401);
  });

  test('purchase with non-existent strategy fails', async ({ request }) => {
    const resp = await request.post('/ant.v1.MarketplaceService/PurchaseStrategy', {
      data: {
        userId: session.userId,
        strategyId: '00000000-0000-0000-0000-000000000000',
        publisherUserId: '00000000-0000-0000-0000-000000000000',
      },
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${session.accessToken}`,
      },
    });
    expect(resp.ok(), 'Purchase with non-existent strategy should fail').toBe(false);
  });

  test('injectAuthState with invalid token shows no authenticated content', async ({ page }) => {
    await injectAuthState(page, {
      accessToken: 'INVALID_TOKEN',
      userId: 'fake-user-id',
      email: 'fake@test.com',
      username: 'fake',
      role: 'user',
    });
    await page.goto('/strategy');
    await page.waitForTimeout(3000);

    const bodyText = await page.locator('body').innerText();
    const hasLandingOrLogin = /AI-Powered|Sign in|Email\/Account/i.test(bodyText);
    expect(hasLandingOrLogin, 'Invalid token should redirect to landing/login').toBe(true);
  });

  test('non-existent strategy detail page handles gracefully', async ({ page }) => {
    await injectAuthState(page, session);
    await page.goto('/strategy/view/non-existent-strategy-id');
    await page.waitForTimeout(3000);

    const content = page.locator('main, .ant-card, .ant-empty, .ant-typography');
    await expect(content.first()).toBeVisible({ timeout: 10_000 });
  });
});
