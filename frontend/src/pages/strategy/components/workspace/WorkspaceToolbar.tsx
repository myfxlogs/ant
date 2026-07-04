import { Select } from 'antd';
import { useTranslation } from 'react-i18next';
import { TRADING_BALANCE_KEY, TRADING_EQUITY_KEY, TRADING_PROFIT_KEY, TRADING_POSITIONS_KEY } from '@/gen/ant/v1/i18n/trading_keys';
import { NO_ACCOUNTS_KEY } from '@/gen/ant/v1/i18n/dashboard_keys';
import { SELECT_ACCOUNT_KEY } from '@/gen/ant/v1/i18n/strategy_workspace_keys';
import SymbolPicker from '@/components/chart/SymbolPicker';
import type { AccountInfo } from '@/stores/tradingStore';
import type { Account } from '@/types/account';

interface Props {
  accounts: Account[];
  accountId: string; onAccountChange: (id: string) => void;
  symbol: string; onSymbolChange: (s: string) => void;
  accountInfo?: AccountInfo | null;
  positionCount?: number;
  busy?: boolean;
  positionsCount?: number; onTogglePositionsPanel?: () => void;
  mtError?: string | null;
  strategyName?: string;
  saveStatus?: 'modified' | 'saved' | 'none';
}

function fmtCompact(v: number | undefined | null): string {
  if (v == null) return '—';
  const abs = Math.abs(v);
  if (abs >= 1_000_000) return (v / 1_000_000).toFixed(2) + 'M';
  if (abs >= 1_000) return (v / 1_000).toFixed(1) + 'K';
  return v.toFixed(2);
}

function Stat({ label, value, color }: { label: string; value: string; color?: string }) {
  return (
    <span style={{ fontSize: 11, color: 'var(--ant-color-text-secondary)', whiteSpace: 'nowrap' }}>
      {label} <span style={{ fontWeight: 600, fontSize: 12, color: color || 'var(--ant-color-text)' }}>{value}</span>
    </span>
  );
}

export default function WorkspaceToolbar({
  accounts, accountId, onAccountChange, busy,
  symbol, onSymbolChange, accountInfo, positionCount,
  positionsCount, onTogglePositionsPanel,
  mtError, strategyName, saveStatus,
}: Props) {
  const { t } = useTranslation();
  const hasData = accountInfo != null;
  const profitColor = accountInfo && accountInfo.profit >= 0 ? '#3fb950' : '#f85149';

  return (
    <div style={{
      display: 'flex', alignItems: 'center', gap: 8,
      height: 44, flexShrink: 0, padding: '0 10px',
      background: 'var(--ant-color-bg-container)',
      borderBottom: '1px solid var(--ant-color-border)',
    }}>
      <span style={{ fontWeight: 700, color: '#58a6ff', fontSize: 14 }}>Ant</span>
      <span style={{ color: 'var(--ant-color-text-secondary)', fontSize: 12 }}>Strategy Workspace</span>

      <Select size="small" style={{ minWidth: 160, width: 220, maxWidth: '30vw' }}
        value={accountId || undefined} onChange={onAccountChange} disabled={busy}
        placeholder={t(SELECT_ACCOUNT_KEY)} showSearch optionFilterProp="label"
        notFoundContent={t(NO_ACCOUNTS_KEY)}
        options={(accounts || []).map((a) => ({ value: a.id, label: `${a.brokerServer} · ${a.login}` }))} />

      <SymbolPicker accountId={accountId} value={symbol} onChange={onSymbolChange} style={{ width: 120 }} />

      {strategyName && (
        <span style={{ fontSize: 11, color: 'var(--ant-color-text-secondary)', whiteSpace: 'nowrap', maxWidth: 160, overflow: 'hidden', textOverflow: 'ellipsis' }}>
          {strategyName}
        </span>
      )}
      {saveStatus === 'modified' && <span style={{ fontSize: 10, color: '#f0a020' }}>●</span>}
      {saveStatus === 'saved' && <span style={{ fontSize: 10, color: '#3fb950' }}>✓</span>}

      <div style={{ flex: 1 }} />

      {hasData && (
        <>
          <Stat label={t(TRADING_BALANCE_KEY)} value={`$${fmtCompact(accountInfo!.balance)}`} />
          <Stat label={t(TRADING_EQUITY_KEY)} value={`$${fmtCompact(accountInfo!.equity)}`}
            color={accountInfo!.equity >= accountInfo!.balance ? '#3fb950' : undefined} />
          <Stat label={t(TRADING_PROFIT_KEY)}
            value={`${accountInfo!.profit >= 0 ? '+' : ''}$${fmtCompact(Math.abs(accountInfo!.profit))}`}
            color={profitColor} />
        </>
      )}

      <div onClick={onTogglePositionsPanel} role="button" tabIndex={0}
        onKeyUp={e => e.key === 'Enter' && onTogglePositionsPanel?.()}
        style={{ cursor: 'pointer' }}>
        <Stat label={t(TRADING_POSITIONS_KEY)} value={positionCount != null ? String(positionCount) : '0'}
          color={positionCount != null && positionCount > 0 ? '#58a6ff' : undefined} />
      </div>

      {mtError && (
        <span style={{ fontSize: 11, color: 'var(--ant-color-error)', marginLeft: 8 }}>⚠ {mtError}</span>
      )}
    </div>
  );
}
