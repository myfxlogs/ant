import { Tag } from 'antd';
import React, { memo } from 'react';
import { PositionPrice } from '@/components/PositionPrice';
import { formatPrice } from '@/utils/price';
import { useTranslation } from 'react-i18next'
import { TRADING_STRATEGY_EXECUTE_CONFIRM_BUY_KEY, TRADING_STRATEGY_EXECUTE_CONFIRM_SELL_KEY } from '@/gen/ant/v1/i18n/trading_keys';
import { DETAIL_BALANCE_RECORD_DEPOSIT_ICON_TEXT_KEY, DETAIL_BALANCE_RECORD_DEPOSIT_KEY, DETAIL_BALANCE_RECORD_WITHDRAW_ICON_TEXT_KEY, DETAIL_BALANCE_RECORD_WITHDRAW_KEY, DETAIL_ORDER_TYPES_BUY_LIMIT_KEY, DETAIL_ORDER_TYPES_BUY_STOP_KEY, DETAIL_ORDER_TYPES_SELL_LIMIT_KEY, DETAIL_ORDER_TYPES_SELL_STOP_KEY } from '@/gen/ant/v1/i18n/accounts_keys';

;
import type { Position } from '@/types/trading';
import type { TradeRecordItem } from '@/client/analyticsTypes';

import { formatTimestamp } from './AccountDetail.utils';

export const SmallInfoCard = memo(
  ({
    icon,
    label,
    value,
    loading,
    valueColor = 'var(--color-text)',
  }: {
    icon: React.ReactNode;
    label: string;
    value: string;
    loading?: boolean;
    valueColor?: string;
  }) => {
    const { t } = useTranslation();
    return (
      <div className="rounded-xl p-4" style={{ background: 'var(--color-bg-card)', boxShadow: '0 2px 8px var(--color-border)' }}>
        <div className="flex items-center gap-2 mb-2">
          {icon}
          <span style={{ color: 'var(--color-text-muted)', fontSize: '13px' }}>{label}</span>
        </div>
        {loading ? (
          <div className="text-base" style={{ color: 'var(--color-text-muted)' }}>{t('common.loading')}</div>
        ) : (
          <div className="text-lg font-semibold" style={{ color: valueColor }}>{value}</div>
        )}
      </div>
    );
  },
);

export const PositionRow = memo(({ position }: { position: Position }) => {
  const { t } = useTranslation();
  return (
    <tr className="border-b hover:bg-gray-50" style={{ borderColor: 'var(--color-border)' }}>
    <td className="p-3 font-medium" style={{ color: 'var(--color-text)' }}>{position.ticket}</td>
    <td className="p-3" style={{ color: 'var(--color-text)' }}>{position.symbol}</td>
    <td className="p-3">
      <Tag style={{ background: position.type === 'buy' ? 'var(--color-success-bg)' : 'var(--color-danger-bg)', color: position.type === 'buy' ? 'var(--color-success)' : 'var(--color-danger)', border: 'none', borderRadius: '4px' }}>
        {position.type === 'buy' ? t(TRADING_STRATEGY_EXECUTE_CONFIRM_BUY_KEY) : t(TRADING_STRATEGY_EXECUTE_CONFIRM_SELL_KEY)}
      </Tag>
    </td>
    <td className="text-right p-3" style={{ color: 'var(--color-text)' }}>{position.volume}</td>
    <td className="text-right p-3" style={{ color: 'var(--color-text)' }}>{formatPrice(position.openPrice, position.symbol)}</td>
    <td className="text-right p-3" style={{ color: 'var(--color-text)' }}>
      <PositionPrice
        symbol={position.symbol}
        defaultPrice={
          typeof position.currentPrice === 'number' && position.currentPrice > 0
            ? position.currentPrice
            : typeof position.closePrice === 'number' && position.closePrice > 0
              ? position.closePrice
              : undefined
        }
      />
    </td>
    <td className="text-right p-3 font-medium" style={{ color: (Number(position.profit) || 0) >= 0 ? 'var(--color-success)' : 'var(--color-danger)' }}>
      {(Number(position.profit) || 0) >= 0 ? '+' : ''}{(Number(position.profit) || 0).toFixed(2)}
    </td>
    <td className="p-3" style={{ color: 'var(--color-text-muted)', fontSize: '12px' }}>{formatTimestamp(position.openTime)}</td>
    <td className="p-3" style={{ color: 'var(--color-text-muted)', fontSize: '12px' }}>{position.magicNumber ? position.magicNumber : '-'}</td>
    </tr>
  );
});

export const PendingOrderRow = memo(({ order }: { order: Position }) => {
  const { t } = useTranslation();
  return (
    <tr className="border-b hover:bg-gray-50" style={{ borderColor: 'var(--color-border)' }}>
    <td className="p-3 font-medium" style={{ color: 'var(--color-text)' }}>{order.ticket}</td>
    <td className="p-3" style={{ color: 'var(--color-text)' }}>{order.symbol}</td>
    <td className="p-3">
      <Tag style={{ background: (typeof order.type === 'string' && order.type.includes('buy')) ? 'var(--color-success-bg)' : 'var(--color-danger-bg)', color: (typeof order.type === 'string' && order.type.includes('buy')) ? 'var(--color-success)' : 'var(--color-danger)', border: 'none', borderRadius: '4px' }}>
        {order.type === 'buy_limit'
          ? t(DETAIL_ORDER_TYPES_BUY_LIMIT_KEY)
          : order.type === 'sell_limit'
            ? t(DETAIL_ORDER_TYPES_SELL_LIMIT_KEY)
            : order.type === 'buy_stop'
              ? t(DETAIL_ORDER_TYPES_BUY_STOP_KEY)
              : t(DETAIL_ORDER_TYPES_SELL_STOP_KEY)}
      </Tag>
    </td>
    <td className="text-right p-3" style={{ color: 'var(--color-text)' }}>{order.volume}</td>
    <td className="text-right p-3" style={{ color: 'var(--color-text)' }}>{formatPrice(order.openPrice, order.symbol)}</td>
    <td className="text-right p-3" style={{ color: 'var(--color-text)' }}>
      <PositionPrice
        symbol={order.symbol}
        defaultPrice={
          typeof order.currentPrice === 'number' && order.currentPrice > 0
            ? order.currentPrice
            : typeof order.closePrice === 'number' && order.closePrice > 0
              ? order.closePrice
              : undefined
        }
      />
    </td>
    <td className="p-3" style={{ color: 'var(--color-text-muted)', fontSize: '12px' }}>{formatTimestamp(order.openTime)}</td>
    <td className="p-3" style={{ color: 'var(--color-text-muted)', fontSize: '12px' }}>{order.magicNumber ? order.magicNumber : '-'}</td>
    </tr>
  );
});

export const HistoryTradeRow = memo(({ trade }: { trade: TradeRecordItem }) => {
  const { t } = useTranslation();
  const rawType = trade.type || '';
  const orderType = rawType.replace(/^Op_Op_/, '').replace(/^Op_/, '').toLowerCase();
  const closePrice = trade.closePrice ?? 0;
  const closeTime = trade.closeTime ?? '';
  const openPrice = trade.openPrice ?? 0;
  const volume = trade.volume || 0;
  const symbol = trade.symbol || '';
  // Balance/credit records may have empty symbol and/or balance/credit order type.
  const isBalanceRecord = !symbol || orderType === 'balance' || orderType === 'credit';
  const isDeposit = isBalanceRecord
    ? (orderType === 'credit' || trade.profit > 0 || (trade.profit === 0 && orderType === 'balance'))
    : trade.profit >= 0;
  
  return (
    <tr className="border-b" style={{ borderColor: 'var(--color-border)', background: isBalanceRecord ? 'var(--color-gold-bg-hover)' : 'transparent' }}>
      <td className="p-3 font-medium" style={{ color: 'var(--color-text)' }}>{trade.ticket}</td>
      <td className="p-3" style={{ color: 'var(--color-text)' }}>
        {isBalanceRecord ? (isDeposit ? t(DETAIL_BALANCE_RECORD_DEPOSIT_ICON_TEXT_KEY) : t(DETAIL_BALANCE_RECORD_WITHDRAW_ICON_TEXT_KEY)) : (trade.symbol || '--')}
      </td>
      <td className="p-3">
        {isBalanceRecord ? (
          <Tag style={{ background: isDeposit ? 'var(--color-gold-bg)' : 'var(--color-danger-bg)', color: isDeposit ? 'var(--color-primary)' : 'var(--color-danger)', border: 'none', borderRadius: '4px' }}>
            {isDeposit ? t(DETAIL_BALANCE_RECORD_DEPOSIT_KEY) : t(DETAIL_BALANCE_RECORD_WITHDRAW_KEY)}
          </Tag>
        ) : (
          <Tag style={{ background: orderType.includes('buy') ? 'var(--color-success-bg)' : 'var(--color-danger-bg)', color: orderType.includes('buy') ? 'var(--color-success)' : 'var(--color-danger)', border: 'none', borderRadius: '4px' }}>
            {orderType.includes('buy') ? t(TRADING_STRATEGY_EXECUTE_CONFIRM_BUY_KEY) : t(TRADING_STRATEGY_EXECUTE_CONFIRM_SELL_KEY)}
          </Tag>
        )}
      </td>
      <td className="text-right p-3" style={{ color: 'var(--color-text)' }}>{isBalanceRecord ? '-' : volume}</td>
      <td className="text-right p-3" style={{ color: 'var(--color-text)' }}>{isBalanceRecord ? '-' : formatPrice(openPrice, trade.symbol)}</td>
      <td className="text-right p-3" style={{ color: 'var(--color-text)' }}>{isBalanceRecord ? '-' : formatPrice(closePrice, trade.symbol)}</td>
      <td className="text-right p-3 font-medium" style={{ color: (Number(trade.profit) || 0) >= 0 ? 'var(--color-success)' : 'var(--color-danger)' }}>{(Number(trade.profit) || 0) >= 0 ? '+' : ''}{(Number(trade.profit) || 0).toFixed(2)}</td>
      <td className="p-3" style={{ color: 'var(--color-text-muted)', fontSize: '12px' }}>{formatTimestamp(closeTime)}</td>
      <td className="p-3" style={{ color: 'var(--color-text-muted)', fontSize: '12px' }}>{trade.magicNumber ? trade.magicNumber : '-'}</td>
    </tr>
  );
});
