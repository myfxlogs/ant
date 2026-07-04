import { Select } from 'antd';
import { RiseOutlined, FallOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import {
  TRADING_BALANCE_KEY, TRADING_EQUITY_KEY, TRADING_PROFIT_KEY,
  TRADING_FREE_MARGIN_KEY, TRADING_POSITIONS_KEY,
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
  onTogglePositionsPanel?: () => void;
  mtError?: string | null;
  strategyName?: string;
  saveStatus?: 'modified' | 'saved' | 'none';
}

const groupStyle: React.CSSProperties = {
  padding: '6px 12px 8px', borderRadius: 10,
  background: 'rgba(255,255,255,0.72)', border: '1px solid rgba(0,0,0,0.05)',
  boxShadow: '0 1px 3px rgba(15,23,42,0.04)',
};

function fmtCompact(v: number | undefined | null): string {
  if (v == null) return '—';
  const abs = Math.abs(v);
  if (abs >= 1_000_000) return (v / 1_000_000).toFixed(2) + 'M';
  if (abs >= 1_000) return (v / 1_000).toFixed(1) + 'K';
  return v.toFixed(2);
}

function SummaryChip({ label, value, color, icon }: { label: string; value: string | number; color?: string; icon?: React.ReactNode }) {
  return (
    <div style={{
      display: 'flex', flexDirection: 'column', alignItems: 'center',
      padding: '6px 14px', borderRadius: 10,
      background: 'rgba(255,255,255,0.8)', border: '1px solid rgba(0,0,0,0.06)',
    }}>
      <span style={{ fontSize: 12, color: '#8c8c8c', fontWeight: 500 }}>{label}</span>
      <span style={{ fontSize: 14, fontWeight: 700, color: color || '#262626', display: 'flex', alignItems: 'center', gap: 4 }}>
        {icon}{value}
      </span>
    </div>
  );
}

export default function WorkspaceToolbar({
  accounts, accountId, onAccountChange, busy,
  symbol, onSymbolChange, accountInfo, positionCount,
  onTogglePositionsPanel,
  mtError, strategyName, saveStatus,
}: Props) {
  const { t } = useTranslation();
  const hasData = accountInfo != null;
  const profitColor = accountInfo && accountInfo.profit >= 0 ? '#26a69a' : '#ef5350';
  const selectedAccount = (accounts || []).find(a => a.id === accountId);

  return (
    <div style={{
      display: 'flex', alignItems: 'flex-end', gap: 10, flexWrap: 'wrap',
      padding: '8px 12px 10px', background: '#f8fafc',
      borderBottom: '1px solid #e8e8e8', flexShrink: 0,
    }}>
      {/* Account + Symbol selectors */}
      <div style={{ ...groupStyle, flex: '0 0 auto', display: 'flex', alignItems: 'center', gap: 8 }}>
        <Select style={{ minWidth: 120, width: 220, maxWidth: '36vw' }}
          value={accountId || undefined} onChange={onAccountChange} disabled={busy}
          placeholder={t(SELECT_ACCOUNT_KEY)} showSearch optionFilterProp="label"
          notFoundContent={t(NO_ACCOUNTS_KEY)}
          options={(accounts || []).map((a) => ({ value: a.id, label: `${a.brokerServer} · ${a.login}` }))} />
        <SymbolPicker accountId={accountId} value={symbol} onChange={onSymbolChange} style={{ width: 120 }} />
      </div>

      {/* Strategy name + save status */}
      {strategyName && (
        <div style={{ ...groupStyle, flex: '0 0 auto', display: 'flex', alignItems: 'center', gap: 4 }}>
          {saveStatus === 'modified' && <span style={{ color: '#f0a020', fontSize: 14 }}>●</span>}
          {saveStatus === 'saved' && <span style={{ color: '#3fb950', fontSize: 14 }}>✓</span>}
          <span style={{ fontSize: 14, fontWeight: 600, color: '#262626' }}>{strategyName}</span>
        </div>
      )}

      {/* Account Summary — SummaryChip cards */}
      {hasData && (
        <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', alignItems: 'flex-end' }}>
          <SummaryChip label={t(TRADING_BALANCE_KEY)} value={`$${fmtCompact(accountInfo!.balance)}`} />
          <SummaryChip label={t(TRADING_EQUITY_KEY)} value={`$${fmtCompact(accountInfo!.equity)}`} />
          <SummaryChip
            label={t(TRADING_PROFIT_KEY)}
            value={`$${fmtCompact(Math.abs(accountInfo!.profit))}`}
            color={profitColor}
            icon={accountInfo!.profit >= 0
              ? <RiseOutlined style={{ fontSize: 11 }} />
              : <FallOutlined style={{ fontSize: 11 }} />}
          />
          <SummaryChip label={t(TRADING_FREE_MARGIN_KEY)} value={`$${fmtCompact(accountInfo!.freeMargin)}`} />
          {accountInfo!.marginLevel > 0 && (
            <SummaryChip label={t('trading.marginLevel', 'Margin Lvl')} value={`${accountInfo!.marginLevel.toFixed(0)}%`} />
          )}
        </div>
      )}

      {/* Open Positions */}
      <div onClick={onTogglePositionsPanel} role="button" tabIndex={0}
        onKeyUp={e => e.key === 'Enter' && onTogglePositionsPanel?.()}
        style={{ cursor: 'pointer' }}>
        <SummaryChip label={t(TRADING_POSITIONS_KEY)} value={positionCount != null ? String(positionCount) : '0'}
          color={positionCount != null && positionCount > 0 ? '#1677ff' : undefined} />
      </div>

      {/* Account Metadata — SummaryChip cards for consistent rounded borders */}
      {selectedAccount && (
        <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', alignItems: 'flex-end' }}>
          <SummaryChip label={t('trading.platform', 'Platform')} value={selectedAccount.mtType}
            color={selectedAccount.mtType === 'MT5' ? '#1677ff' : '#fa8c16'} />
          <SummaryChip label={t('trading.broker', 'Broker')} value={selectedAccount.brokerCompany} />
          <SummaryChip label={t('trading.server', 'Server')} value={selectedAccount.brokerServer} />
          <SummaryChip label={t('trading.permission', 'Permission')}
            value={selectedAccount.isInvestor ? t('trading.investor', 'Investor') : t('trading.master', 'Master')}
            color={selectedAccount.isInvestor ? '#fa8c16' : '#52c41a'} />
          {selectedAccount.leverage && selectedAccount.leverage > 0 && (
            <SummaryChip label={t('trading.leverage', 'Leverage')} value={`1:${selectedAccount.leverage}`} />
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
