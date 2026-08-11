import { useTranslation } from 'react-i18next';
import { Button } from 'antd';
import { QUICK_TRADE_KEY } from '@/gen/ant/v1/i18n/strategy_workspace_keys';
import QuickTradePanel, { type AccountMeta, type PositionItem, type TradeItem } from '@/components/chart/QuickTradePanel';

interface Props {
  accountId: string;
  symbol: string;
  accountMeta?: AccountMeta | null;
  allPositions?: PositionItem[];
  positions?: PositionItem[];
  recentTrades?: TradeItem[];
  onClosePosition: (ticket: number) => void;
  onCollapse?: () => void;
}

export default function QuickTradeSidePanel({ accountId, symbol, accountMeta, allPositions, positions, recentTrades, onClosePosition, onCollapse }: Props) {
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
        <span style={{ flex: 1 }}>⚡ {t(QUICK_TRADE_KEY)}</span>
        {onCollapse && (
          <Button size="small" type="text" onClick={onCollapse}
            style={{ fontSize: 12, padding: '0 4px', lineHeight: 1 }}>✕</Button>
        )}
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
