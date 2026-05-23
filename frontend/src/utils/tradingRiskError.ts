import i18n from '@/i18n';

const KNOWN_RISK_CODES = new Set<string>([
  'RISK_ACCOUNT_TRADE_DISABLED',
  'RISK_SYMBOL_TRADE_DISABLED',
  'RISK_MARKET_SESSION_CLOSED',
  'RISK_VOLUME_INVALID',
  'RISK_ORDER_TYPE_UNSUPPORTED',
  'RISK_STOP_DISTANCE_TOO_CLOSE',
  'RISK_ORDER_FROZEN_ZONE',
  'RISK_MARGIN_INSUFFICIENT',
  'RISK_MAX_OPEN_POSITIONS_EXCEEDED',
  'RISK_MAX_PENDING_ORDERS_EXCEEDED',
  'RISK_INTERNAL_RULE_UNAVAILABLE',
]);

function normalizeRiskCode(raw?: string): string {
  const trimmed = String(raw || '').trim().toUpperCase();
  return trimmed;
}

export function extractRiskErrorCode(...candidates: Array<string | undefined>): string | '' {
  for (const candidate of candidates) {
    const normalized = normalizeRiskCode(candidate);
    if (!normalized) continue;
    if (KNOWN_RISK_CODES.has(normalized)) return normalized;
    const matched = normalized.match(/RISK_[A-Z0-9_]+/);
    if (matched && KNOWN_RISK_CODES.has(matched[0])) {
      return matched[0];
    }
  }
  return '';
}

export function getTradingRiskToastMessage(params: {
  error?: string;
  message?: string;
  riskCode?: string;
  fallback: string;
}): string {
  const code = extractRiskErrorCode(params.riskCode, params.error, params.message);
  if (!code) return params.fallback;
  const titleKey = `trading.risk.errors.${code}.title`;
  const actionKey = `trading.risk.errors.${code}.action`;
  const title = i18n.t(titleKey);
  const action = i18n.t(actionKey);
  if (title && action && title !== titleKey && action !== actionKey) {
    return `${title} ${action}`;
  }
  return params.fallback;
}
