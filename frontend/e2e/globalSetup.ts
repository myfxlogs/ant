import { request } from '@playwright/test';
import { registerTestUser } from './fixtures/auth';

const BASE_URL = process.env.E2E_BASE_URL || 'http://localhost:8022';

export default async function globalSetup() {
  const ctx = await request.newContext({ baseURL: BASE_URL });
  try {
    await registerTestUser(ctx);
  } finally {
    await ctx.dispose();
  }
}
