import { request } from '@playwright/test';
import { registerTestUser } from './fixtures/auth';

const BASE_URL = process.env.E2E_BASE_URL || 'http://localhost:8022';

export default async function globalSetup() {
  const ctx = await request.newContext({ baseURL: BASE_URL });
  try {
    await registerTestUser(ctx);
  } catch (err) {
    console.warn(`[E2E globalSetup] Backend at ${BASE_URL} is not reachable — skipping test user registration.\n`, err);
  } finally {
    await ctx.dispose();
  }
}
