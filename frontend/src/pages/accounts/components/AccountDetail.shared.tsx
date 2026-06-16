import { Tag } from 'antd';
import React, { memo } from 'react';
import { PositionPrice } from '@/components/PositionPrice';
import { formatPrice } from '@/utils/price';
import { useTranslation } from 'react-i18next';
import type { Position } from '@/types/trading';
import type { TradeRecordItem } from '@/client/analyticsTypes';

import { formatTimestamp } from './AccountDetail.utils';

export const StatCard = memo(
  ({
    icon,
    label,
    value,
    valueColor = 'var(--color-text)',
    background = 'var(--color-bg-secondary)',
  }: {
    icon: React.ReactNode;
    label: string;
    value: string;
    valueColor?: string;
    background?: string;
  }) => (
    <div className="p-2 rounded-lg" style={{ background }}>
      <div style={{ color: 'var(--color-text-muted)', fontSize: '10px' }}>{icon} {label}</div>
      <div className="text-base font-bold" style={{ color: valueColor }}>{value}</div>
    </div>
  ),
);

export const InfoCard = memo(
  ({ icon, label, value, loading }: { icon: React.ReactNode; label: string; value: string; loading?: boolean }) => {
    const { t } = useTranslation();
    return (
      <div className="rounded-2xl p-5" style={{ background: 'var(--color-bg-card)', boxShadow: '0 2px 8px var(--color-border)' }}>
        <div className="flex items-center gap-2 mb-3">
          {icon}
          <span style={{ color: 'var(--color-text-muted)', fontSize: '14px' }}>{label}</span>
        </div>
        {loading ? (
          <div className="text-lg" style={{ color: 'var(--color-text-muted)' }}>{t('common.loading')}</div>
        ) : (
          <div className="text-2xl font-bold" style={{ color: 'var(--color-text)' }}>{value}</div>
        )}
      </div>
    );
  },
);

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
        {position.type === 'buy' ? t('trading.strategyExecute.confirm.buy') : t('trading.strategyExecute.confirm.sell')}
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
        orderType={position.type}
      />
    </td>
    <td className="text-right p-3 font-medium" style={{ color: (Number(position.profit) || 0) >= 0 ? 'var(--color-success)' : 'var(--color-danger)' }}>
      {(Number(position.profit) || 0) >= 0 ? '+' : ''}{(Number(position.profit) || 0).toFixed(2)}
    </td>
    <td className="p-3" style={{ color: 'var(--color-text-muted)', fontSize: '12px' }}>{formatTimestamp(position.openTime)}</td>
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
          ? t('accounts.detail.orderTypes.buyLimit')
          : order.type === 'sell_limit'
            ? t('accounts.detail.orderTypes.sellLimit')
            : order.type === 'buy_stop'
              ? t('accounts.detail.orderTypes.buyStop')
              : t('accounts.detail.orderTypes.sellStop')}
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
        orderType={(typeof order.type === 'string' && order.type.includes('buy')) ? 'buy' : 'sell'}
      />
    </td>
    <td className="p-3" style={{ color: 'var(--color-text-muted)', fontSize: '12px' }}>{formatTimestamp(order.openTime)}</td>
    </tr>
  );
});

export const HistoryTradeRow = memo(({ trade }: { trade: TradeRecordItem }) => {
  const { t } = useTranslation();
  const rawType = trade.type || trade.orderType || trade.order_type || '';
  const orderType = rawType.replace(/^Op_Op_/, '').replace(/^Op_/, '').toLowerCase();
  const closePrice = trade.closePrice ?? trade.close_price ?? 0;
  const closeTime = trade.closeTime ?? trade.close_time ?? '';
  const openPrice = trade.openPrice ?? trade.open_price ?? 0;
  const volume = trade.volume || trade.lots || 0;
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
        {isBalanceRecord ? (isDeposit ? t('accounts.detail.balanceRecord.depositIconText') : t('accounts.detail.balanceRecord.withdrawIconText')) : (trade.symbol || '--')}
      </td>
      <td className="p-3">
        {isBalanceRecord ? (
          <Tag style={{ background: isDeposit ? 'var(--color-gold-bg)' : 'var(--color-danger-bg)', color: isDeposit ? 'var(--color-primary)' : 'var(--color-danger)', border: 'none', borderRadius: '4px' }}>
            {isDeposit ? t('accounts.detail.balanceRecord.deposit') : t('accounts.detail.balanceRecord.withdraw')}
          </Tag>
        ) : (
          <Tag style={{ background: orderType.includes('buy') ? 'var(--color-success-bg)' : 'var(--color-danger-bg)', color: orderType.includes('buy') ? 'var(--color-success)' : 'var(--color-danger)', border: 'none', borderRadius: '4px' }}>
            {orderType.includes('buy') ? t('trading.strategyExecute.confirm.buy') : t('trading.strategyExecute.confirm.sell')}
          </Tag>
        )}
      </td>
      <td className="text-right p-3" style={{ color: 'var(--color-text)' }}>{isBalanceRecord ? '-' : volume}</td>
      <td className="text-right p-3" style={{ color: 'var(--color-text)' }}>{isBalanceRecord ? '-' : formatPrice(openPrice, trade.symbol)}</td>
      <td className="text-right p-3" style={{ color: 'var(--color-text)' }}>{isBalanceRecord ? '-' : formatPrice(closePrice, trade.symbol)}</td>
      <td className="text-right p-3 font-medium" style={{ color: (Number(trade.profit) || 0) >= 0 ? 'var(--color-success)' : 'var(--color-danger)' }}>{(Number(trade.profit) || 0) >= 0 ? '+' : ''}{(Number(trade.profit) || 0).toFixed(2)}</td>
      <td className="p-3" style={{ color: 'var(--color-text-muted)', fontSize: '12px' }}>{formatTimestamp(closeTime)}</td>
    </tr>
  );
});
