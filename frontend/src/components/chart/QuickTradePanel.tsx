import { useState, useCallback, useRef, useEffect } from 'react';
import { Button, Select, InputNumber, Radio, message, Row, Col } from 'antd';
import { SendOutlined, RiseOutlined, FallOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next'
import { BUY_KEY, OPEN_POSITIONS_TITLE_KEY, PRICE_KEY, SELL_KEY, STOP_LOSS_KEY, TAKE_PROFIT_KEY } from '@/gen/ant/v1/i18n/trading_keys';
import { AMOUNT_LOTS_KEY, CROSS_KEY, ISOLATED_KEY, MARGIN_MODE_KEY, MT4_CROSS_ONLY_KEY, ORDER_FAILED_KEY, ORDER_PLACED_KEY, PRICE_REQUIRED_KEY, SELECT_SYMBOL_KEY, VALID_VOLUME_KEY } from '@/gen/ant/v1/i18n/strategy_quick_trade_section_keys';

;
import { tradingApi } from '@/client/trading';
import PositionSection, { type PositionItem } from './PositionSection';
import TradeHistorySection, { type TradeItem } from './TradeHistorySection';
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
}

type OrderSide = 'buy' | 'sell';
type OrderKind = 'MARKET' | 'LIMIT' | 'STOP';

const ORDER_KIND_KEYS: Record<OrderKind, string> = {
  MARKET: 'trading.market',
  LIMIT: 'trading.limit',
  STOP: 'trading.stop',
};

const cardBox: React.CSSProperties = { background: '#f6f9fc', border: '1px solid #e0e8f0', borderRadius: 6, padding: '6px 10px' };
const labelSm: React.CSSProperties = { fontSize: 10, color: '#64748b', fontWeight: 600 };

export default function QuickTradePanel({ accountId, symbol, accountMeta, allPositions = [], positions = [], recentTrades = [], onClosePosition, onToggleAllPositions }: Props) {
  const { t } = useTranslation();
  const totalLots = (allPositions || []).reduce((s, p) => s + (p.volume || 0), 0);
  const [side, setSide] = useState<OrderSide>('buy');
  const [orderKind, setOrderKind] = useState<OrderKind>('MARKET');
  const [volume, setVolume] = useState<number | null>(0.01);
  const [price, setPrice] = useState<number | null>(null);
  const [stopLoss, setStopLoss] = useState<number | null>(null);
  const [takeProfit, setTakeProfit] = useState<number | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [closingTicket, setClosingTicket] = useState<number | null>(null);
  const [marginMode, setMarginMode] = useState<'cross' | 'isolated'>('cross');

  const isLimitOrStop = orderKind === 'LIMIT' || orderKind === 'STOP';
  const isMT5 = accountMeta?.mtType === 'MT5';

  const canSubmit = Boolean(symbol && accountId && (volume || 0) > 0 && !submitting);

  const handleSubmit = useCallback(async () => {
    if (submitting) return; // prevent double-click
    if (!symbol || !accountId) { message.warning(t(SELECT_TRADING_SYMBOL_KEY)); return; }
    if (!volume || volume <= 0) { message.warning(t(VALID_TRADING_VOLUME_KEY)); return; }
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
      });
      if (result.error && result.error !== '0' && result.error !== '') {
        message.error(result.message || result.error);
      } else { message.success(t(ORDER_PLACED_KEY, { side: side === 'buy' ? t('trading.buy') : t('trading.sell') })); }
    } catch (e: any) { message.error(e?.message || t(ORDER_FAILED_KEY)); }
    finally { setSubmitting(false); }
  }, [accountId, symbol, side, orderKind, volume, price, stopLoss, takeProfit, isLimitOrStop]);

  const closeTimerRef = useRef<number | null>(null);
  useEffect(() => () => { if (closeTimerRef.current != null) window.clearTimeout(closeTimerRef.current); }, []);

  const handleClosePos = useCallback(async (ticket: number, volume?: number) => {
    setClosingTicket(ticket);
    try {
      await onClosePosition?.(ticket, volume);
    } finally {
      if (closeTimerRef.current != null) window.clearTimeout(closeTimerRef.current);
      closeTimerRef.current = window.setTimeout(() => setClosingTicket(null), 5000);
    }
  }, [onClosePosition]);

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
            background: side === 'buy' ? '#22c55e' : '#f1f5f9',
            borderColor: side === 'buy' ? '#22c55e' : '#d1d5db',
            color: side === 'buy' ? '#fff' : '#64748b',
          }}>{t(BUY_KEY)}</Button>
        <Button block type={side === 'sell' ? 'primary' : 'default'}
          onClick={() => setSide('sell')} icon={<FallOutlined />}
          style={{
            height: 38, fontWeight: 700, fontSize: 13, borderRadius: '0 8px 8px 0', marginLeft: -1,
            background: side === 'sell' ? '#ef4444' : '#f1f5f9',
            borderColor: side === 'sell' ? '#ef4444' : '#d1d5db',
            color: side === 'sell' ? '#fff' : '#64748b',
          }}>{t(SELL_KEY)}</Button>
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
          <div style={labelSm}>{t(PRICE_KEY)}</div>
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
          <div style={labelSm}>{t(STOP_LOSS_KEY)}</div>
          <InputNumber size="small" style={{ width: '100%' }} min={0} step={0.00001}
            value={stopLoss} onChange={(v) => setStopLoss(v)} placeholder="SL" />
        </Col>
        <Col span={12}>
          <div style={labelSm}>{t(TAKE_PROFIT_KEY)}</div>
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
        {side === 'buy' ? t(BUY_KEY) : t(SELL_KEY)} {symbol || '—'}
      </Button>
      </>)}

      {/* Position summary — click to expand all positions overlay */}
      <div onClick={onToggleAllPositions} role="button" tabIndex={0}
        onKeyUp={e => e.key === 'Enter' && onToggleAllPositions?.()}
        style={{ ...cardBox, display: 'flex', justifyContent: 'space-between', alignItems: 'center', cursor: 'pointer' }}>
        <span style={labelSm}>{t(OPEN_POSITIONS_TITLE_KEY)}</span>
        <span style={{ fontSize: 13, fontWeight: 700, color: '#262626' }}>
          {totalLots.toFixed(2)} lots · {allPositions.length} &gt;
        </span>
      </div>

      {/* Position Section */}
      <PositionSection symbol={symbol} positions={positions}
        closingTicket={closingTicket} onClosePosition={handleClosePos} />

      {/* Trade History */}
      <TradeHistorySection trades={recentTrades} />
    </div>
  );
}
