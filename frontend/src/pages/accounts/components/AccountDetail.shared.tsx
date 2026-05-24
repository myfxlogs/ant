import { Tag } from 'antd';
import React, { memo } from 'react';
import { PositionPrice } from '@/components/PositionPrice';
import { formatPrice } from '@/utils/price';
import { useTranslation } from 'react-i18next';

import { formatTimestamp } from './AccountDetail.utils';

export const StatCard = memo(
  ({
    icon,
    label,
    value,
    valueColor = '#141D22',
    background = '#F5F7F9',
  }: {
    icon: React.ReactNode;
    label: string;
    value: string;
    valueColor?: string;
    background?: string;
  }) => (
    <div className="p-2 rounded-lg" style={{ background }}>
      <div style={{ color: '#8A9AA5', fontSize: '10px' }}>{icon} {label}</div>
      <div className="text-base font-bold" style={{ color: valueColor }}>{value}</div>
    </div>
  ),
);

export const InfoCard = memo(
  ({ icon, label, value, loading }: { icon: React.ReactNode; label: string; value: string; loading?: boolean }) => {
    const { t } = useTranslation();
    return (
      <div className="rounded-2xl p-5" style={{ background: '#FFFFFF', boxShadow: '0 2px 8px rgba(0, 0, 0, 0.06)' }}>
        <div className="flex items-center gap-2 mb-3">
          {icon}
          <span style={{ color: '#8A9AA5', fontSize: '14px' }}>{label}</span>
        </div>
        {loading ? (
          <div className="text-lg" style={{ color: '#8A9AA5' }}>{t('common.loading')}</div>
        ) : (
          <div className="text-2xl font-bold" style={{ color: '#141D22' }}>{value}</div>
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
    valueColor = '#141D22',
  }: {
    icon: React.ReactNode;
    label: string;
    value: string;
    loading?: boolean;
    valueColor?: string;
  }) => {
    const { t } = useTranslation();
    return (
      <div className="rounded-xl p-4" style={{ background: '#FFFFFF', boxShadow: '0 2px 8px rgba(0, 0, 0, 0.06)' }}>
        <div className="flex items-center gap-2 mb-2">
          {icon}
          <span style={{ color: '#8A9AA5', fontSize: '13px' }}>{label}</span>
        </div>
        {loading ? (
          <div className="text-base" style={{ color: '#8A9AA5' }}>{t('common.loading')}</div>
        ) : (
          <div className="text-lg font-semibold" style={{ color: valueColor }}>{value}</div>
        )}
      </div>
    );
  },
);

export const PositionRow = memo(({ position }: { position: any }) => {
  const { t } = useTranslation();
  return (
    <tr className="border-b hover:bg-gray-50" style={{ borderColor: 'rgba(0, 0, 0, 0.06)' }}>
    <td className="p-3 font-medium" style={{ color: '#141D22' }}>{position.ticket}</td>
    <td className="p-3" style={{ color: '#141D22' }}>{position.symbol}</td>
    <td className="p-3">
      <Tag style={{ background: position.type === 'buy' ? 'rgba(0, 166, 81, 0.1)' : 'rgba(229, 57, 53, 0.1)', color: position.type === 'buy' ? '#00A651' : '#E53935', border: 'none', borderRadius: '4px' }}>
        {position.type === 'buy' ? t('trading.strategyExecute.confirm.buy') : t('trading.strategyExecute.confirm.sell')}
      </Tag>
    </td>
    <td className="text-right p-3" style={{ color: '#141D22' }}>{position.volume}</td>
    <td className="text-right p-3" style={{ color: '#141D22' }}>{formatPrice(position.openPrice, position.symbol)}</td>
    <td className="text-right p-3" style={{ color: '#141D22' }}>
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
    <td className="text-right p-3 font-medium" style={{ color: position.profit >= 0 ? '#00A651' : '#E53935' }}>
      {position.profit >= 0 ? '+' : ''}{position.profit.toFixed(2)}
    </td>
    <td className="p-3" style={{ color: '#8A9AA5', fontSize: '12px' }}>{formatTimestamp(position.openTime)}</td>
    </tr>
  );
});

export const PendingOrderRow = memo(({ order }: { order: any }) => {
  const { t } = useTranslation();
  return (
    <tr className="border-b hover:bg-gray-50" style={{ borderColor: 'rgba(0, 0, 0, 0.06)' }}>
    <td className="p-3 font-medium" style={{ color: '#141D22' }}>{order.ticket}</td>
    <td className="p-3" style={{ color: '#141D22' }}>{order.symbol}</td>
    <td className="p-3">
      <Tag style={{ background: order.type.includes('buy') ? 'rgba(0, 166, 81, 0.1)' : 'rgba(229, 57, 53, 0.1)', color: order.type.includes('buy') ? '#00A651' : '#E53935', border: 'none', borderRadius: '4px' }}>
        {order.type === 'buy_limit'
          ? t('accounts.detail.orderTypes.buyLimit')
          : order.type === 'sell_limit'
            ? t('accounts.detail.orderTypes.sellLimit')
            : order.type === 'buy_stop'
              ? t('accounts.detail.orderTypes.buyStop')
              : t('accounts.detail.orderTypes.sellStop')}
      </Tag>
    </td>
    <td className="text-right p-3" style={{ color: '#141D22' }}>{order.volume}</td>
    <td className="text-right p-3" style={{ color: '#141D22' }}>{formatPrice(order.openPrice, order.symbol)}</td>
    <td className="text-right p-3" style={{ color: '#141D22' }}>
      <PositionPrice
        symbol={order.symbol}
        defaultPrice={
          typeof order.currentPrice === 'number' && order.currentPrice > 0
            ? order.currentPrice
            : typeof order.closePrice === 'number' && order.closePrice > 0
              ? order.closePrice
              : undefined
        }
        orderType={order.type.includes('buy') ? 'buy' : 'sell'}
      />
    </td>
    <td className="p-3" style={{ color: '#8A9AA5', fontSize: '12px' }}>{formatTimestamp(order.openTime)}</td>
    </tr>
  );
});

export const HistoryTradeRow = memo(({ trade }: { trade: any }) => {
  const { t } = useTranslation();
  const rawType = trade.type || trade.orderType || trade.order_type || '';
  const orderType = rawType.replace(/^Op_Op_/, '').replace(/^Op_/, '').toLowerCase();
  const closePrice = trade.closePrice || trade.closePrice || 0;
  const closeTime = trade.closeTime || trade.closeTime || '';
  const openPrice = trade.openPrice || trade.openPrice || 0;
  const volume = trade.volume || trade.lots || 0;
  const isBalanceRecord = orderType === 'balance' || orderType === 'credit';
  const isDeposit = trade.profit >= 0;
  
  return (
    <tr className="border-b" style={{ borderColor: 'rgba(0, 0, 0, 0.06)', background: isBalanceRecord ? 'rgba(212, 175, 55, 0.05)' : 'transparent' }}>
      <td className="p-3 font-medium" style={{ color: '#141D22' }}>{trade.ticket}</td>
      <td className="p-3" style={{ color: '#141D22' }}>
        {isBalanceRecord ? (isDeposit ? t('accounts.detail.balanceRecord.depositIconText') : t('accounts.detail.balanceRecord.withdrawIconText')) : trade.symbol}
      </td>
      <td className="p-3">
        {isBalanceRecord ? (
          <Tag style={{ background: isDeposit ? 'rgba(212, 175, 55, 0.1)' : 'rgba(229, 57, 53, 0.1)', color: isDeposit ? '#D4AF37' : '#E53935', border: 'none', borderRadius: '4px' }}>
            {isDeposit ? t('accounts.detail.balanceRecord.deposit') : t('accounts.detail.balanceRecord.withdraw')}
          </Tag>
        ) : (
          <Tag style={{ background: orderType.includes('buy') ? 'rgba(0, 166, 81, 0.1)' : 'rgba(229, 57, 53, 0.1)', color: orderType.includes('buy') ? '#00A651' : '#E53935', border: 'none', borderRadius: '4px' }}>
            {orderType.includes('buy') ? t('trading.strategyExecute.confirm.buy') : t('trading.strategyExecute.confirm.sell')}
          </Tag>
        )}
      </td>
      <td className="text-right p-3" style={{ color: '#141D22' }}>{isBalanceRecord ? '-' : volume}</td>
      <td className="text-right p-3" style={{ color: '#141D22' }}>{isBalanceRecord ? '-' : formatPrice(openPrice, trade.symbol)}</td>
      <td className="text-right p-3" style={{ color: '#141D22' }}>{isBalanceRecord ? '-' : formatPrice(closePrice, trade.symbol)}</td>
      <td className="text-right p-3 font-medium" style={{ color: trade.profit >= 0 ? '#00A651' : '#E53935' }}>{trade.profit >= 0 ? '+' : ''}{trade.profit.toFixed(2)}</td>
      <td className="p-3" style={{ color: '#8A9AA5', fontSize: '12px' }}>{formatTimestamp(closeTime)}</td>
    </tr>
  );
});
