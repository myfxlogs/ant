import { Select } from 'antd';
import { RiseOutlined, FallOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import {
  TRADING_BALANCE_KEY, TRADING_EQUITY_KEY, TRADING_PROFIT_KEY,
  TRADING_FREE_MARGIN_KEY, TRADING_POSITIONS_KEY,
  TRADING_MARGIN_LEVEL_KEY, TRADING_PLATFORM_KEY, TRADING_BROKER_KEY,
  TRADING_SERVER_KEY, TRADING_PERMISSION_KEY, TRADING_INVESTOR_KEY,
  TRADING_MASTER_KEY, TRADING_LEVERAGE_KEY,
} from '@/gen/ant/v1/i18n/trading_keys';
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
  onToggleBottomPanel?: () => void;
  mtError?: string | null;
  onMTErrorChange?: (hasError: boolean) => void;
}

const groupStyle: React.CSSProperties = {
  padding: '6px 12px 8px', borderRadius: 10,
  background: 'var(--color-chip-bg)', border: '1px solid var(--color-chip-border)',
  boxShadow: '0 1px 3px var(--color-shadow)',
};

function fmtFull(v: number | undefined | null): string {
  if (v == null) return '—';
  return v.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 });
}

function SummaryChip({ label, value, color, icon }: { label: string; value: string | number; color?: string; icon?: React.ReactNode }) {
  return (
    <div style={{
      display: 'flex', flexDirection: 'column', alignItems: 'center',
      padding: '6px 14px', borderRadius: 10,
      background: 'var(--color-chip-bg)', border: '1px solid var(--color-chip-border)',
    }}>
      <span style={{ fontSize: 12, color: 'var(--color-text-muted)', fontWeight: 500 }}>{label}</span>
      <span style={{ fontSize: 14, fontWeight: 700, color: color || 'var(--color-text)', display: 'flex', alignItems: 'center', gap: 4 }}>
        {icon}{value}
      </span>
    </div>
  );
}

export default function WorkspaceToolbar({
  accounts, accountId, onAccountChange, busy,
  symbol, onSymbolChange, accountInfo, positionCount,
  onToggleBottomPanel,
  mtError, onMTErrorChange,
}: Props) {
  const { t } = useTranslation();
  const hasData = accountInfo != null;
  const profitColor = accountInfo && accountInfo.profit >= 0 ? 'var(--color-success)' : 'var(--color-danger)';
  const selectedAccount = (accounts || []).find(a => a.id === accountId);

  return (
    <div style={{
      display: 'flex', alignItems: 'flex-end', gap: 10, flexWrap: 'wrap',
      padding: '8px 12px 10px', background: 'var(--color-bg-secondary)',
      borderBottom: '1px solid var(--color-border)', flexShrink: 0,
    }}>
      {/* Account + Symbol selectors */}
      <div style={{ ...groupStyle, flex: '0 0 auto', display: 'flex', alignItems: 'center', gap: 8 }}>
        <Select style={{ minWidth: 120, width: 220, maxWidth: '36vw' }}
          value={accountId || undefined} onChange={onAccountChange} disabled={busy}
          placeholder={t(SELECT_ACCOUNT_KEY)} showSearch optionFilterProp="label"
          notFoundContent={t(NO_ACCOUNTS_KEY)}
          options={(accounts || []).map((a) => ({ value: a.id, label: `${a.brokerServer} · ${a.login}` }))} />
        <SymbolPicker accountId={selectedAccount ? accountId : ''} value={symbol} onChange={onSymbolChange} onMTErrorChange={onMTErrorChange} style={{ width: 120 }} />
      </div>


      {/* Account Summary — SummaryChip cards */}
      {hasData && (
        <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', alignItems: 'flex-end' }}>
          <SummaryChip label={t(TRADING_BALANCE_KEY)} value={`$${fmtFull(accountInfo!.balance)}`} />
          <SummaryChip label={t(TRADING_EQUITY_KEY)} value={`$${fmtFull(accountInfo!.equity)}`} />
          <SummaryChip
            label={t(TRADING_PROFIT_KEY)}
            value={`$${fmtFull(Math.abs(accountInfo!.profit))}`}
            color={profitColor}
            icon={accountInfo!.profit >= 0
              ? <RiseOutlined style={{ fontSize: 11 }} />
              : <FallOutlined style={{ fontSize: 11 }} />}
          />
          <SummaryChip label={t(TRADING_FREE_MARGIN_KEY)} value={`$${fmtFull(accountInfo!.freeMargin)}`} />
          {accountInfo!.marginLevel > 0 && (
            <SummaryChip label={t(TRADING_MARGIN_LEVEL_KEY)} value={`${accountInfo!.marginLevel.toFixed(0)}%`} />
          )}
        </div>
      )}

      {/* Open Positions */}
      <div onClick={onToggleBottomPanel} role="button" tabIndex={0}
        onKeyUp={e => e.key === 'Enter' && onToggleBottomPanel?.()}
        style={{ cursor: 'pointer' }}>
        <SummaryChip label={t(TRADING_POSITIONS_KEY)} value={positionCount != null ? String(positionCount) : '0'}
          color={positionCount != null && positionCount > 0 ? 'var(--color-info)' : undefined} />
      </div>

      {/* Account Metadata — SummaryChip cards for consistent rounded borders */}
      {selectedAccount && (
        <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', alignItems: 'flex-end' }}>
          <SummaryChip label={t(TRADING_PLATFORM_KEY)} value={selectedAccount.mtType}
            color={selectedAccount.mtType === 'MT5' ? 'var(--color-info)' : 'var(--color-warning)'} />
          <SummaryChip label={t(TRADING_BROKER_KEY)} value={selectedAccount.brokerCompany} />
          <SummaryChip label={t(TRADING_SERVER_KEY)} value={selectedAccount.brokerServer} />
          <SummaryChip label={t(TRADING_PERMISSION_KEY)}
            value={selectedAccount.isInvestor ? t(TRADING_INVESTOR_KEY) : t(TRADING_MASTER_KEY)}
            color={selectedAccount.isInvestor ? 'var(--color-warning)' : 'var(--color-success)'} />
          {selectedAccount.leverage && selectedAccount.leverage > 0 && (
            <SummaryChip label={t(TRADING_LEVERAGE_KEY)} value={`1:${selectedAccount.leverage}`} />
          )}
        </div>
      )}

      {/* Spacer */}
      <div style={{ flex: 1 }} />

      {mtError && (
        <span style={{ fontSize: 14, color: 'var(--ant-color-error)' }}>⚠ {mtError}</span>
      )}
    </div>
  );
}
