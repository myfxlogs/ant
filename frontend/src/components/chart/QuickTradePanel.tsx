import { useState, useCallback } from 'react';
import { Button, Select, InputNumber, Radio, message, Row, Col, Tag } from 'antd';
import { SendOutlined, RiseOutlined, FallOutlined, BankOutlined } from '@ant-design/icons';
import { tradingApi } from '@/client/trading';
import PositionSection, { type PositionItem } from './PositionSection';
import TradeHistorySection, { type TradeItem } from './TradeHistorySection';
import type { AccountInfo } from '@/stores/tradingStore';

interface AccountMeta {
  brokerCompany: string;
  brokerServer: string;
  mtType: 'MT4' | 'MT5';
  leverage: number;
}

interface Props {
  accountId: string;
  symbol: string;
  accountInfo?: AccountInfo | null;
  accountMeta?: AccountMeta | null;
  positions?: PositionItem[];
  recentTrades?: TradeItem[];
  onClosePosition?: (ticket: number) => void;
}

type OrderSide = 'buy' | 'sell';
type OrderKind = 'MARKET' | 'LIMIT' | 'STOP';

const ORDER_KINDS: { value: OrderKind; label: string }[] = [
  { value: 'MARKET', label: 'Market' },
  { value: 'LIMIT', label: 'Limit' },
  { value: 'STOP', label: 'Stop' },
];

const cardBox: React.CSSProperties = { background: '#f6f9fc', border: '1px solid #e0e8f0', borderRadius: 6, padding: '6px 10px' };
const labelSm: React.CSSProperties = { fontSize: 10, color: '#64748b', fontWeight: 600 };

function fmtMoney(v: number | undefined | null): string {
  if (v == null) return '0.00';
  return v.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
}

function BalanceRow({ label, value }: { label: string; value: string }) {
  return <div style={{ display: 'flex', justifyContent: 'space-between' }}>
    <span style={{ fontSize: 10, color: '#64748b' }}>{label}</span>
    <span style={{ fontSize: 11, fontWeight: 700, color: '#262626' }}>{value}</span>
  </div>;
}

export default function QuickTradePanel({ accountId, symbol, accountInfo, accountMeta, positions = [], recentTrades = [], onClosePosition }: Props) {
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
  const freeMargin = accountInfo?.freeMargin ?? 0;
  const leverage = accountMeta?.leverage ?? 100;
  const isMT5 = accountMeta?.mtType === 'MT5';

  const canSubmit = Boolean(symbol && accountId && (volume || 0) > 0 && !submitting);

  const handleSubmit = useCallback(async () => {
    if (!symbol || !accountId) { message.warning('Select a symbol first'); return; }
    if (!volume || volume <= 0) { message.warning('Enter a valid volume'); return; }
    if (isLimitOrStop && (!price || price <= 0)) { message.warning('Price is required for Limit/Stop orders'); return; }
    setSubmitting(true);
    try {
      const typeStr = `${side}${isLimitOrStop ? `_${orderKind.toLowerCase()}` : ''}`;
      const result = await tradingApi.orderSend({
        accountId, symbol, type: typeStr, volume: volume,
        price: isLimitOrStop ? price : undefined,
        stopLoss: stopLoss ?? undefined, takeProfit: takeProfit ?? undefined,
      });
      if (result.error && result.error !== '0' && result.error !== '') {
        message.error(result.message || result.error);
      } else { message.success(`${side === 'buy' ? 'Buy' : 'Sell'} order placed`); }
    } catch (e: any) { message.error(e?.message || 'Order failed'); }
    finally { setSubmitting(false); }
  }, [accountId, symbol, side, orderKind, volume, price, stopLoss, takeProfit, isLimitOrStop]);

  const handleClosePos = useCallback((ticket: number) => {
    setClosingTicket(ticket);
    onClosePosition?.(ticket);
    setTimeout(() => setClosingTicket(null), 5000);
  }, [onClosePosition]);

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 10, padding: '8px 0' }}>
      {/* Account Balance */}
      {accountInfo && (
        <div style={{ ...cardBox, display: 'flex', flexDirection: 'column', gap: 2 }}>
          <BalanceRow label="Free Margin" value={`$${fmtMoney(freeMargin)}`} />
          {accountInfo.equity != null && <BalanceRow label="Equity" value={`$${fmtMoney(accountInfo.equity)}`} />}
          {accountInfo.balance != null && <BalanceRow label="Balance" value={`$${fmtMoney(accountInfo.balance)}`} />}
        </div>
      )}

      {/* Exchange / Broker info — account is selected in toolbar */}
      {accountMeta && (
        <div style={{
          background: '#f0f5ff', border: '1px solid #d6e4ff', borderRadius: 6,
          padding: '6px 10px', display: 'flex', alignItems: 'center', gap: 8,
        }}>
          <BankOutlined style={{ color: '#1890ff', fontSize: 14 }} />
          <div style={{ flex: 1 }}>
            <div style={{ fontSize: 11, fontWeight: 700, color: '#262626' }}>
              {accountMeta.brokerCompany}
            </div>
            <div style={{ fontSize: 9, color: '#8c8c8c' }}>
              {accountMeta.brokerServer} · {accountMeta.mtType}
            </div>
          </div>
          <Tag color="blue" style={{ fontSize: 9, margin: 0, lineHeight: '18px' }}>
            {accountMeta.mtType}
          </Tag>
        </div>
      )}

      {/* Side toggle */}
      <div style={{ display: 'flex', gap: 0 }}>
        <Button block type={side === 'buy' ? 'primary' : 'default'}
          onClick={() => setSide('buy')} icon={<RiseOutlined />}
          style={{
            height: 38, fontWeight: 700, fontSize: 13, borderRadius: '8px 0 0 8px',
            background: side === 'buy' ? '#22c55e' : '#f1f5f9',
            borderColor: side === 'buy' ? '#22c55e' : '#d1d5db',
            color: side === 'buy' ? '#fff' : '#64748b',
          }}>Buy</Button>
        <Button block type={side === 'sell' ? 'primary' : 'default'}
          onClick={() => setSide('sell')} icon={<FallOutlined />}
          style={{
            height: 38, fontWeight: 700, fontSize: 13, borderRadius: '0 8px 8px 0', marginLeft: -1,
            background: side === 'sell' ? '#ef4444' : '#f1f5f9',
            borderColor: side === 'sell' ? '#ef4444' : '#d1d5db',
            color: side === 'sell' ? '#fff' : '#64748b',
          }}>Sell</Button>
      </div>

      {/* Order type */}
      <Select size="small" value={orderKind} onChange={setOrderKind} options={ORDER_KINDS} style={{ width: '100%' }} />

      {/* Volume */}
      <div>
        <div style={labelSm}>Amount (lots)</div>
        <InputNumber size="small" style={{ width: '100%' }} min={0.01} step={0.01}
          value={volume} onChange={(v) => setVolume(v ?? 0.01)} placeholder="0.01" />
      </div>

      {/* Price (Limit/Stop only) */}
      {isLimitOrStop && (
        <div>
          <div style={labelSm}>Price</div>
          <InputNumber size="small" style={{ width: '100%' }} min={0} step={0.00001}
            value={price} onChange={(v) => setPrice(v)} placeholder="0.00000" />
        </div>
      )}

      {/* Leverage — read-only from broker account settings */}
      <div style={{ ...cardBox, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <span style={labelSm}>Leverage</span>
        <span style={{ fontSize: 13, fontWeight: 700, color: '#262626' }}>1:{leverage}</span>
      </div>

      {/* Margin Mode — MT5 supports cross/isolated; MT4 only cross */}
      <div>
        <div style={labelSm}>Margin Mode</div>
        <Radio.Group size="small" buttonStyle="solid"
          value={marginMode} onChange={e => setMarginMode(e.target.value)}
          disabled={!isMT5}>
          <Radio.Button value="cross">Cross</Radio.Button>
          <Radio.Button value="isolated">Isolated</Radio.Button>
        </Radio.Group>
        {!isMT5 && (
          <div style={{ fontSize: 9, color: '#8c8c8c', marginTop: 2 }}>MT4 supports Cross margin only</div>
        )}
      </div>

      {/* SL / TP */}
      <Row gutter={8}>
        <Col span={12}>
          <div style={labelSm}>Stop Loss</div>
          <InputNumber size="small" style={{ width: '100%' }} min={0} step={0.00001}
            value={stopLoss} onChange={(v) => setStopLoss(v)} placeholder="SL" />
        </Col>
        <Col span={12}>
          <div style={labelSm}>Take Profit</div>
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
        {side === 'buy' ? 'Buy' : 'Sell'} {symbol || '—'}
      </Button>

      {/* Position Section */}
      <PositionSection symbol={symbol} positions={positions}
        closingTicket={closingTicket} onClosePosition={handleClosePos} />

      {/* Trade History */}
      <TradeHistorySection trades={recentTrades} />
    </div>
  );
}
