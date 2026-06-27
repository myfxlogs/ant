import { useState } from 'react';
import { Card, Form, Button, InputNumber, Radio, Space } from 'antd';
import { SendOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next'
import { TRADING_BUY_KEY, TRADING_LIMIT_KEY, TRADING_MARKET_KEY, TRADING_PLACE_ORDER_KEY, TRADING_PRICE_KEY, TRADING_SELL_KEY, TRADING_STOP_KEY, TRADING_STOP_LOSS_KEY, TRADING_SYMBOL_KEY, TRADING_TAKE_PROFIT_KEY, TRADING_VOLUME_KEY } from '@/gen/ant/v1/i18n/trading_keys';
;
import { useTradingStore } from '@/stores/tradingStore';
import { useTrading } from '@/hooks/useTrading';
import SymbolPicker from '@/components/chart/SymbolPicker';

interface PlaceOrderFormProps {
  onSymbolChange?: (symbol: string) => void;
}

export default function PlaceOrderForm({ onSymbolChange }: PlaceOrderFormProps) {
  const { t } = useTranslation();
  const currentAccountId = useTradingStore((s) => s.currentAccountId);
  const loading = useTradingStore((s) => s.loading);
  const { sendOrder } = useTrading();
  const [side, setSide] = useState<'buy' | 'sell'>('buy');
  const [orderType, setOrderType] = useState<'market' | 'limit' | 'stop'>('market');
  const [form] = Form.useForm();

  const handleSubmit = async () => {
    if (!currentAccountId) return;
    try {
      const values = await form.validateFields();
      const type = orderType === 'market'
        ? side === 'buy' ? 'buy' : 'sell'
        : side === 'buy' ? `buy_${orderType}` : `sell_${orderType}`;
      await sendOrder({
        accountId: currentAccountId,
        symbol: values.symbol,
        type,
        volume: values.volume,
        price: values.price || undefined,
        stopLoss: values.stopLoss || undefined,
        takeProfit: values.takeProfit || undefined,
      });
      form.resetFields(['price', 'stopLoss', 'takeProfit', 'volume']);
    } catch {
      // error already shown by useTrading.sendOrder
    }
  };

  return (
    <Card
      title={
        <span>
          <SendOutlined style={{ marginRight: 8 }} />
          {t(TRADING_PLACE_ORDER_KEY, 'Place Order')}
        </span>
      }
      style={{ marginTop: 16 }}
    >
      <Form form={form} layout="vertical" size="small">
        <Space style={{ marginBottom: 12 }}>
          <Radio.Group value={side} onChange={(e) => setSide(e.target.value)} buttonStyle="solid">
            <Radio.Button value="buy" style={{ color: side === 'buy' ? '#52c41a' : undefined }}>
              {t(TRADING_BUY_KEY, 'Buy')}
            </Radio.Button>
            <Radio.Button value="sell" style={{ color: side === 'sell' ? '#ff4d4f' : undefined }}>
              {t(TRADING_SELL_KEY, 'Sell')}
            </Radio.Button>
          </Radio.Group>
          <Radio.Group value={orderType} onChange={(e) => setOrderType(e.target.value)} buttonStyle="outline">
            <Radio.Button value="market">{t(TRADING_MARKET_KEY, 'Market')}</Radio.Button>
            <Radio.Button value="limit">{t(TRADING_LIMIT_KEY, 'Limit')}</Radio.Button>
            <Radio.Button value="stop">{t(TRADING_STOP_KEY, 'Stop')}</Radio.Button>
          </Radio.Group>
        </Space>

        <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap' }}>
          <Form.Item name="symbol" label={t(TRADING_SYMBOL_KEY, 'Symbol')} rules={[{ required: true }]} style={{ width: 180 }}>
            <SymbolPicker
              accountId={currentAccountId || ''}
              onChange={(sym) => onSymbolChange?.(sym)}
              placeholder="EURUSD"
            />
          </Form.Item>
          <Form.Item name="volume" label={t(TRADING_VOLUME_KEY, 'Volume')} rules={[{ required: true, type: 'number', min: 0.01 }]} style={{ width: 100 }}>
            <InputNumber min={0.01} step={0.01} placeholder="0.01" />
          </Form.Item>
          {orderType !== 'market' && (
            <Form.Item name="price" label={t(TRADING_PRICE_KEY, 'Price')} rules={[{ required: true, type: 'number' }]} style={{ width: 120 }}>
              <InputNumber style={{ width: '100%' }} placeholder="1.0000" />
            </Form.Item>
          )}
          <Form.Item name="stopLoss" label={t(TRADING_STOP_LOSS_KEY, 'SL')} style={{ width: 110 }}>
            <InputNumber style={{ width: '100%' }} placeholder="0" />
          </Form.Item>
          <Form.Item name="takeProfit" label={t(TRADING_TAKE_PROFIT_KEY, 'TP')} style={{ width: 110 }}>
            <InputNumber style={{ width: '100%' }} placeholder="0" />
          </Form.Item>
          <Form.Item label=" " style={{ width: 80 }}>
            <Button
              type="primary"
              icon={<SendOutlined />}
              onClick={handleSubmit}
              loading={loading}
              disabled={!currentAccountId}
              danger={side === 'sell'}
            >
              {side === 'buy' ? t(TRADING_BUY_KEY, 'Buy') : t(TRADING_SELL_KEY, 'Sell')}
            </Button>
          </Form.Item>
        </div>
      </Form>
    </Card>
  );
}
