import { useState, useCallback } from 'react';
import { Button, Input, Select, message, InputNumber } from 'antd';
import { SendOutlined, RiseOutlined, FallOutlined } from '@ant-design/icons';
import { tradingApi } from '@/client/trading';

interface Props {
  accountId: string;
  symbol: string;
}

type OrderSide = 'buy' | 'sell';
type OrderKind = 'MARKET' | 'LIMIT' | 'STOP';

const ORDER_KINDS: { value: OrderKind; label: string }[] = [
  { value: 'MARKET', label: 'Market' },
  { value: 'LIMIT', label: 'Limit' },
  { value: 'STOP', label: 'Stop' },
];

export default function QuickTradePanel({ accountId, symbol }: Props) {
  const [side, setSide] = useState<OrderSide>('buy');
  const [orderKind, setOrderKind] = useState<OrderKind>('MARKET');
  const [volume, setVolume] = useState<number | null>(0.01);
  const [price, setPrice] = useState<number | null>(null);
  const [stopLoss, setStopLoss] = useState<number | null>(null);
  const [takeProfit, setTakeProfit] = useState<number | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const isLimitOrStop = orderKind === 'LIMIT' || orderKind === 'STOP';

  const handleSubmit = useCallback(async () => {
    if (!symbol || !accountId) {
      message.warning('Select a symbol first');
      return;
    }
    if (!volume || volume <= 0) {
      message.warning('Enter a valid volume');
      return;
    }
    if (isLimitOrStop && (!price || price <= 0)) {
      message.warning('Price is required for Limit/Stop orders');
      return;
    }
    setSubmitting(true);
    try {
      const typeStr = `${side}${isLimitOrStop ? `_${orderKind.toLowerCase()}` : ''}`;
      const result = await tradingApi.orderSend({
        accountId,
        symbol,
        type: typeStr,
        volume,
        price: isLimitOrStop ? price : undefined,
        stopLoss: stopLoss ?? undefined,
        takeProfit: takeProfit ?? undefined,
      });
      if (result.error && result.error !== '0' && result.error !== '') {
        message.error(result.message || result.error);
      } else {
        message.success(`${side === 'buy' ? 'Buy' : 'Sell'} order placed`);
      }
    } catch (e: any) {
      message.error(e?.message || 'Order failed');
    } finally {
      setSubmitting(false);
    }
  }, [accountId, symbol, side, orderKind, volume, price, stopLoss, takeProfit, isLimitOrStop]);

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 10, padding: '8px 0' }}>
      {/* Side toggle */}
      <div style={{ display: 'flex', gap: 0 }}>
        <Button
          block
          type={side === 'buy' ? 'primary' : 'default'}
          onClick={() => setSide('buy')}
          icon={<RiseOutlined />}
          style={{
            height: 38, fontWeight: 700, fontSize: 13,
            borderRadius: '8px 0 0 8px',
            background: side === 'buy' ? '#22c55e' : '#f1f5f9',
            borderColor: side === 'buy' ? '#22c55e' : '#d1d5db',
            color: side === 'buy' ? '#fff' : '#64748b',
          }}
        >
          Buy
        </Button>
        <Button
          block
          type={side === 'sell' ? 'primary' : 'default'}
          onClick={() => setSide('sell')}
          icon={<FallOutlined />}
          style={{
            height: 38, fontWeight: 700, fontSize: 13,
            borderRadius: '0 8px 8px 0', marginLeft: -1,
            background: side === 'sell' ? '#ef4444' : '#f1f5f9',
            borderColor: side === 'sell' ? '#ef4444' : '#d1d5db',
            color: side === 'sell' ? '#fff' : '#64748b',
          }}
        >
          Sell
        </Button>
      </div>

      {/* Order type */}
      <Select
        size="small"
        value={orderKind}
        onChange={setOrderKind}
        options={ORDER_KINDS}
        style={{ width: '100%' }}
      />

      {/* Volume */}
      <div>
        <div style={{ fontSize: 10, color: '#64748b', marginBottom: 2, fontWeight: 600 }}>
          Volume (lots)
        </div>
        <InputNumber
          size="small" style={{ width: '100%' }}
          min={0.01} step={0.01}
          value={volume} onChange={(v) => setVolume(v)}
          placeholder="0.01"
        />
      </div>

      {/* Price (Limit/Stop only) */}
      {isLimitOrStop && (
        <div>
          <div style={{ fontSize: 10, color: '#64748b', marginBottom: 2, fontWeight: 600 }}>
            Price
          </div>
          <InputNumber
            size="small" style={{ width: '100%' }}
            min={0} step={0.00001}
            value={price} onChange={(v) => setPrice(v)}
            placeholder="0.00000"
          />
        </div>
      )}

      {/* Stop Loss */}
      <div>
        <div style={{ fontSize: 10, color: '#64748b', marginBottom: 2, fontWeight: 600 }}>
          Stop Loss (optional)
        </div>
        <InputNumber
          size="small" style={{ width: '100%' }}
          min={0} step={0.00001}
          value={stopLoss} onChange={(v) => setStopLoss(v)}
          placeholder="—"
        />
      </div>

      {/* Take Profit */}
      <div>
        <div style={{ fontSize: 10, color: '#64748b', marginBottom: 2, fontWeight: 600 }}>
          Take Profit (optional)
        </div>
        <InputNumber
          size="small" style={{ width: '100%' }}
          min={0} step={0.00001}
          value={takeProfit} onChange={(v) => setTakeProfit(v)}
          placeholder="—"
        />
      </div>

      {/* Submit */}
      <Button
        type="primary" block loading={submitting}
        icon={<SendOutlined />}
        onClick={handleSubmit}
        disabled={!symbol}
        style={{
          height: 40, fontWeight: 700, fontSize: 14, marginTop: 4,
          borderRadius: 8,
          background: side === 'buy'
            ? 'linear-gradient(135deg, #22c55e 0%, #16a34a 100%)'
            : 'linear-gradient(135deg, #ef4444 0%, #dc2626 100%)',
          border: 'none',
          boxShadow: side === 'buy'
            ? '0 2px 8px rgba(34,197,94,0.35)'
            : '0 2px 8px rgba(239,68,68,0.35)',
        }}
      >
        {side === 'buy' ? 'Buy' : 'Sell'} {symbol || '—'}
      </Button>
    </div>
  );
}
