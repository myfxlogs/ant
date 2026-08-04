import { useTranslation } from 'react-i18next';
import { QUICK_TRADE_KEY } from '@/gen/ant/v1/i18n/strategy_workspace_keys';
import QuickTradePanel from '@/components/chart/QuickTradePanel';

interface Props {
  accountId: string;
  symbol: string;
  accountMeta?: { leverage?: number; balance?: number; currency?: string };
  allPositions: unknown[];
  positions: unknown[];
  recentTrades: unknown[];
  onClosePosition: (ticket: number) => void;
}

export default function QuickTradeSidePanel({ accountId, symbol, accountMeta, allPositions, positions, recentTrades, onClosePosition }: Props) {
  const { t } = useTranslation();
  return (
    <div style={{
      width: 420, flexShrink: 0,
      borderLeft: '1px solid var(--ant-color-border)',
      background: 'var(--ant-color-bg-elevated)',
      display: 'flex', flexDirection: 'column',
      maxHeight: 160, overflow: 'hidden',
    }}>
      <div style={{
        padding: '4px 10px', fontSize: 11, fontWeight: 700,
        borderBottom: '1px solid var(--ant-color-border)',
        background: 'var(--ant-color-bg-layout)',
        display: 'flex', alignItems: 'center', gap: 4, flexShrink: 0,
      }}>
        ⚡ {t(QUICK_TRADE_KEY)}
      </div>
      <div style={{ flex: 1, overflowY: 'auto', padding: '4px 10px' }}>
        <QuickTradePanel
          accountId={accountId}
          symbol={symbol}
          accountMeta={accountMeta}
          allPositions={allPositions}
          positions={positions}
          recentTrades={recentTrades}
          onClosePosition={onClosePosition}
          horizontal
        />
      </div>
    </div>
  );
}
