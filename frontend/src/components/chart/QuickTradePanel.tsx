import { useState, useCallback, useRef, useEffect } from 'react';
import { Button, Select, InputNumber, Radio, message, Row, Col } from 'antd';
import { SendOutlined, RiseOutlined, FallOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next'
import { TRADING_BUY_KEY, TRADING_PRICE_KEY, TRADING_SELL_KEY, TRADING_STOP_LOSS_KEY, TRADING_TAKE_PROFIT_KEY } from '@/gen/ant/v1/i18n/trading_keys';
import { AMOUNT_LOTS_KEY, CROSS_KEY, ISOLATED_KEY, MARGIN_MODE_KEY, MT4_CROSS_ONLY_KEY, ORDER_FAILED_KEY, ORDER_PLACED_KEY, PRICE_REQUIRED_KEY, SELECT_SYMBOL_KEY, VALID_VOLUME_KEY } from '@/gen/ant/v1/i18n/strategy_quick_trade_section_keys';

;
import { tradingApi } from '@/client/trading';

export interface PositionItem {
  ticket: number; side: string; volume: number;
  openPrice: number; markPrice?: number; profit: number; leverage?: number;
}

export interface TradeItem {
  ticket: number; symbol: string; side: string;
  closePrice?: number; price?: number; profit: number;
  closeTime?: string; created_at?: string;
}

interface AccountMeta {
  brokerCompany: string;
  brokerServer: string;
  mtType: 'MT4' | 'MT5';
  leverage: number;
}

interface Props {
  accountId: string;
  symbol: string;
  accountMeta?: AccountMeta | null;
  allPositions?: PositionItem[];
  positions?: PositionItem[];
  recentTrades?: TradeItem[];
  onClosePosition?: (ticket: number, volume?: number) => void;
  onToggleAllPositions?: () => void;
  horizontal?: boolean;
}

type OrderSide = 'buy' | 'sell';
type OrderKind = 'MARKET' | 'LIMIT' | 'STOP';

const ORDER_KIND_KEYS: Record<OrderKind, string> = {
  MARKET: 'trading.market',
  LIMIT: 'trading.limit',
  STOP: 'trading.stop',
};

const _cardBox: React.CSSProperties = { background: 'var(--ant-color-bg-elevated)', border: '1px solid var(--ant-color-border)', borderRadius: 6, padding: '6px 10px' };
const labelSm: React.CSSProperties = { fontSize: 10, color: 'var(--ant-color-text-tertiary)', fontWeight: 600 };

export default function QuickTradePanel({ accountId, symbol, accountMeta, allPositions = [], _positions = [], _recentTrades = [], onClosePosition, _onToggleAllPositions, horizontal }: Props) {
  const { t } = useTranslation();
  const _totalLots = (allPositions || []).reduce((s, p) => s + (p.volume || 0), 0);
  const [side, setSide] = useState<OrderSide>('buy');
  const [orderKind, setOrderKind] = useState<OrderKind>('MARKET');
  const [volume, setVolume] = useState<number | null>(0.01);
  const [price, setPrice] = useState<number | null>(null);
  const [stopLoss, setStopLoss] = useState<number | null>(null);
  const [takeProfit, setTakeProfit] = useState<number | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [_closingTicket, setClosingTicket] = useState<number | null>(null);
  const [marginMode, setMarginMode] = useState<'cross' | 'isolated'>('cross');

  const isLimitOrStop = orderKind === 'LIMIT' || orderKind === 'STOP';
  const isMT5 = accountMeta?.mtType === 'MT5';

  const canSubmit = Boolean(symbol && accountId && (volume || 0) > 0 && !submitting);

  const handleSubmit = useCallback(async () => {
    if (submitting) return; // prevent double-click
    if (!symbol || !accountId) { message.warning(t(SELECT_SYMBOL_KEY)); return; }
    if (!volume || volume <= 0) { message.warning(t(VALID_VOLUME_KEY)); return; }
    if (isLimitOrStop && (!price || price <= 0)) { message.warning(t(PRICE_REQUIRED_KEY)); return; }
    setSubmitting(true);
    try {
      const typeStr = `${side}${isLimitOrStop ? `_${orderKind.toLowerCase()}` : ''}`;
      // Generate client-supplied idempotency key so the backend can dedup
      // accidental double-submissions (network retry, button double-click).
      const clientId = `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
      const result = await tradingApi.orderSend({
        accountId, symbol, type: typeStr, volume: volume, clientId,
        price: isLimitOrStop ? price : undefined,
        stopLoss: stopLoss ?? undefined, takeProfit: takeProfit ?? undefined,
        marginMode: isMT5 ? marginMode : undefined,
      });
      if (result.error && result.error !== '0' && result.error !== '') {
        message.error(result.message || result.error);
      } else { message.success(t(ORDER_PLACED_KEY, { side: side === 'buy' ? t('trading.buy') : t('trading.sell') })); }
    } catch (e: unknown) { message.error(e instanceof Error ? e.message : String(e) || t(ORDER_FAILED_KEY)); }
    finally { setSubmitting(false); }
  }, [accountId, symbol, side, orderKind, volume, price, stopLoss, takeProfit, isLimitOrStop, marginMode, isMT5, submitting, t]);

  const closeTimerRef = useRef<number | null>(null);
  useEffect(() => () => { if (closeTimerRef.current != null) window.clearTimeout(closeTimerRef.current); }, []);

  const _handleClosePos = useCallback(async (ticket: number, volume?: number) => {
    setClosingTicket(ticket);
    try {
      await onClosePosition?.(ticket, volume);
    } finally {
      if (closeTimerRef.current != null) window.clearTimeout(closeTimerRef.current);
      closeTimerRef.current = window.setTimeout(() => setClosingTicket(null), 5000);
    }
  }, [onClosePosition]);

  if (horizontal) {
    return (
      <div style={{ display: 'flex', flexDirection: 'column', height: '100%', gap: 30, padding: '8px 10px' }}>
        {symbol && (<>
        {/* Row 1: Buy/Sell | Volume | OrderType | MarginMode | Price */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
          <Button size="small" type={side === 'buy' ? 'primary' : 'default'}
            onClick={() => setSide('buy')} icon={<RiseOutlined />}
            style={{ height: 28, fontWeight: 700, fontSize: 11, borderRadius: '6px 0 0 6px',
              background: side === 'buy' ? '#22c55e' : 'var(--ant-color-bg-elevated)',
              borderColor: side === 'buy' ? '#22c55e' : 'var(--ant-color-border)',
              color: side === 'buy' ? '#fff' : 'var(--ant-color-text-secondary)',
            }}>{t(TRADING_BUY_KEY)}</Button>
          <Button size="small" type={side === 'sell' ? 'primary' : 'default'}
            onClick={() => setSide('sell')} icon={<FallOutlined />}
            style={{ height: 28, fontWeight: 700, fontSize: 11, borderRadius: '0 6px 6px 0', marginLeft: -1,
              background: side === 'sell' ? '#ef4444' : 'var(--ant-color-bg-elevated)',
              borderColor: side === 'sell' ? '#ef4444' : 'var(--ant-color-border)',
              color: side === 'sell' ? '#fff' : 'var(--ant-color-text-secondary)',
            }}>{t(TRADING_SELL_KEY)}</Button>
          <span style={labelSm}>{t(AMOUNT_LOTS_KEY)}</span>
          <InputNumber size="small" style={{ width: 64 }} min={0.01} step={0.01}
            value={volume} onChange={(v) => setVolume(v ?? 0.01)} placeholder="0.01" />
          <Select size="small" value={orderKind} onChange={setOrderKind}
            options={Object.entries(ORDER_KIND_KEYS).map(([value, key]) => ({ value, label: t(key) }))}
            style={{ width: 76 }} />
          {isMT5 && (
            <Radio.Group size="small" buttonStyle="solid"
              value={marginMode} onChange={e => setMarginMode(e.target.value)}>
              <Radio.Button value="cross">{t(CROSS_KEY)}</Radio.Button>
              <Radio.Button value="isolated">{t(ISOLATED_KEY)}</Radio.Button>
            </Radio.Group>
          )}
          {isLimitOrStop && (
            <InputNumber size="small" style={{ width: 76 }} min={0} step={0.00001}
              value={price} onChange={(v) => setPrice(v)} placeholder={t(TRADING_PRICE_KEY)} />
          )}
        </div>
        {/* Row 2: SL | TP | Submit */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
          <span style={labelSm}>{t(TRADING_STOP_LOSS_KEY)}</span>
          <InputNumber size="small" style={{ width: 80 }} min={0} step={0.00001}
            value={stopLoss} onChange={(v) => setStopLoss(v)} placeholder="0.00000" />
          <span style={labelSm}>{t(TRADING_TAKE_PROFIT_KEY)}</span>
          <InputNumber size="small" style={{ width: 80 }} min={0} step={0.00001}
            value={takeProfit} onChange={(v) => setTakeProfit(v)} placeholder="0.00000" />
          <Button type="primary" size="small" loading={submitting}
            icon={<SendOutlined />} onClick={handleSubmit} disabled={!canSubmit}
            style={{ height: 28, fontWeight: 700, fontSize: 12, borderRadius: 6,
              background: side === 'buy'
                ? 'linear-gradient(135deg, #22c55e 0%, #16a34a 100%)'
                : 'linear-gradient(135deg, #ef4444 0%, #dc2626 100%)',
              border: 'none',
            }}>
            {side === 'buy' ? t(TRADING_BUY_KEY) : t(TRADING_SELL_KEY)} {symbol}
          </Button>
        </div>
        </>)}
      </div>
    );
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 10, padding: '8px 0' }}>
      {/* Order form — only when symbol is selected */}
      {symbol && (<>
      {/* Side toggle */}
      <div style={{ display: 'flex', gap: 0 }}>
        <Button block type={side === 'buy' ? 'primary' : 'default'}
          onClick={() => setSide('buy')} icon={<RiseOutlined />}
          style={{
            height: 38, fontWeight: 700, fontSize: 13, borderRadius: '8px 0 0 8px',
            background: side === 'buy' ? '#22c55e' : 'var(--ant-color-bg-elevated)',
            borderColor: side === 'buy' ? '#22c55e' : 'var(--ant-color-border)',
            color: side === 'buy' ? '#fff' : 'var(--ant-color-text-secondary)',
          }}>{t(TRADING_BUY_KEY)}</Button>
        <Button block type={side === 'sell' ? 'primary' : 'default'}
          onClick={() => setSide('sell')} icon={<FallOutlined />}
          style={{
            height: 38, fontWeight: 700, fontSize: 13, borderRadius: '0 8px 8px 0', marginLeft: -1,
            background: side === 'sell' ? '#ef4444' : 'var(--ant-color-bg-elevated)',
            borderColor: side === 'sell' ? '#ef4444' : 'var(--ant-color-border)',
            color: side === 'sell' ? '#fff' : 'var(--ant-color-text-secondary)',
          }}>{t(TRADING_SELL_KEY)}</Button>
      </div>

      {/* Order type */}
      <Select size="small" value={orderKind} onChange={setOrderKind}
        options={Object.entries(ORDER_KIND_KEYS).map(([value, key]) => ({ value, label: t(key) }))}
        style={{ width: '100%' }} />

      {/* Volume */}
      <div>
        <div style={labelSm}>{t(AMOUNT_LOTS_KEY)}</div>
        <InputNumber size="small" style={{ width: '100%' }} min={0.01} step={0.01}
          value={volume} onChange={(v) => setVolume(v ?? 0.01)} placeholder="0.01" />
      </div>

      {/* Price (Limit/Stop only) */}
      {isLimitOrStop && (
        <div>
          <div style={labelSm}>{t(TRADING_PRICE_KEY)}</div>
          <InputNumber size="small" style={{ width: '100%' }} min={0} step={0.00001}
            value={price} onChange={(v) => setPrice(v)} placeholder="0.00000" />
        </div>
      )}

      {/* Margin Mode — MT5 supports cross/isolated; MT4 only cross */}
      <div>
        <div style={labelSm}>{t(MARGIN_MODE_KEY)}</div>
        <Radio.Group size="small" buttonStyle="solid"
          value={marginMode} onChange={e => setMarginMode(e.target.value)}
          disabled={!isMT5}>
          <Radio.Button value="cross">{t(CROSS_KEY)}</Radio.Button>
          <Radio.Button value="isolated">{t(ISOLATED_KEY)}</Radio.Button>
        </Radio.Group>
        {!isMT5 && (
          <div style={{ fontSize: 9, color: '#8c8c8c', marginTop: 2 }}>{t(MT4_CROSS_ONLY_KEY)}</div>
        )}
      </div>

      {/* SL / TP */}
      <Row gutter={8}>
        <Col span={12}>
          <div style={labelSm}>{t(TRADING_STOP_LOSS_KEY)}</div>
          <InputNumber size="small" style={{ width: '100%' }} min={0} step={0.00001}
            value={stopLoss} onChange={(v) => setStopLoss(v)} placeholder="SL" />
        </Col>
        <Col span={12}>
          <div style={labelSm}>{t(TRADING_TAKE_PROFIT_KEY)}</div>
          <InputNumber size="small" style={{ width: '100%' }} min={0} step={0.00001}
            value={takeProfit} onChange={(v) => setTakeProfit(v)} placeholder="TP" />
        </Col>
      </Row>

      {/* Submit */}
      <Button type="primary" block loading={submitting}
        icon={<SendOutlined />} onClick={handleSubmit} disabled={!canSubmit}
        style={{
          height: 40, fontWeight: 700, fontSize: 14, marginTop: 4, borderRadius: 8,
          background: side === 'buy'
            ? 'linear-gradient(135deg, #22c55e 0%, #16a34a 100%)'
            : 'linear-gradient(135deg, #ef4444 0%, #dc2626 100%)',
          border: 'none',
          boxShadow: side === 'buy'
            ? '0 2px 8px rgba(34,197,94,0.35)'
            : '0 2px 8px rgba(239,68,68,0.35)',
        }}>
        {side === 'buy' ? t(TRADING_BUY_KEY) : t(TRADING_SELL_KEY)} {symbol || '—'}
      </Button>
      </>)}
    </div>
  );
}
