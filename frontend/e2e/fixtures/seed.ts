import { type APIRequestContext } from '@playwright/test';
import type { AuthSession } from './auth';

export interface PublishedStrategyInfo {
  strategyId: string;
  publisherUserId: string;
  title: string;
  price: string;
  priceModel: string;
}

export async function findCheapestStrategy(
  request: APIRequestContext,
  session: AuthSession,
): Promise<PublishedStrategyInfo | null> {
  const resp = await request.post('/ant.v1.MarketplaceService/ListPublished', {
    data: { limit: 50, offset: 0 },
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${session.accessToken}`,
    },
  });
  if (!resp.ok()) return null;
  const body = await resp.json();
  const strategies = body?.strategies || [];
  if (strategies.length === 0) return null;

  const sorted = strategies
    .map((s: { strategyId?: string; publisherUserId?: string; title?: string; strategyName?: string; price?: string; priceAmount?: string; priceModel?: string }) => ({
      strategyId: s.strategyId || '',
      publisherUserId: s.publisherUserId || '',
      title: s.title || s.strategyName || '',
      price: s.priceAmount || s.price || '0',
      priceModel: s.priceModel || 'free',
    }))
    .sort((a: PublishedStrategyInfo, b: PublishedStrategyInfo) => parseFloat(a.price) - parseFloat(b.price));

  return sorted[0] || null;
}

export async function purchaseStrategy(
  request: APIRequestContext,
  session: AuthSession,
  strategy: PublishedStrategyInfo,
): Promise<{ ok: boolean; error?: string }> {
  const resp = await request.post('/ant.v1.MarketplaceService/PurchaseStrategy', {
    data: {
      userId: session.userId,
      strategyId: strategy.strategyId,
      publisherUserId: strategy.publisherUserId,
    },
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${session.accessToken}`,
    },
  });
  if (resp.ok()) return { ok: true };
  const body = await resp.json().catch(() => ({}));
  const msg = body?.message || `HTTP ${resp.status()}`;
  if (msg.includes('already subscribed') || msg.includes('already_exists')) {
    return { ok: true };
  }
  return { ok: false, error: msg };
}

export async function listPurchasedStrategies(
  request: APIRequestContext,
  session: AuthSession,
): Promise<string[]> {
  const resp = await request.post('/ant.v1.MarketplaceService/ListSubscriptions', {
    data: {},
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${session.accessToken}`,
    },
  });
  if (!resp.ok()) return [];
  const body = await resp.json();
  const subs = body?.subscriptions || [];
  return subs.map((s: { strategyId?: string }) => s.strategyId).filter(Boolean);
}
