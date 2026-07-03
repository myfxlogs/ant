import { Select, Space, Tag } from 'antd';
import { RiseOutlined, FallOutlined, BankOutlined, AimOutlined, KeyOutlined, EyeOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next'
import { TRADING_BALANCE_KEY, TRADING_EQUITY_KEY, TRADING_FREE_MARGIN_KEY, TRADING_MARGIN_LEVEL_KEY, TRADING_POSITIONS_KEY, TRADING_POSITION_LEVERAGE_KEY, TRADING_PROFIT_KEY } from '@/gen/ant/v1/i18n/trading_keys';
import { NO_ACCOUNTS_KEY } from '@/gen/ant/v1/i18n/dashboard_keys';
import { INVESTOR_READ_ONLY_KEY, MASTER_TRADING_KEY, SELECT_ACCOUNT_KEY } from '@/gen/ant/v1/i18n/strategy_workspace_keys';

;
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
}

const groupStyle: React.CSSProperties = {
  padding: '5px 10px 7px', borderRadius: 10,
  background: 'var(--ant-color-bg-elevated)', border: '1px solid var(--ant-color-border)',
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
      padding: '4px 12px', borderRadius: 8,
      background: 'var(--ant-color-bg-elevated)', border: '1px solid var(--ant-color-border)',
    }}>
      <span style={{ fontSize: 9, color: 'var(--ant-color-text-tertiary)', fontWeight: 500, textTransform: 'uppercase' }}>{label}</span>
      <span style={{ fontSize: 13, fontWeight: 700, color: color || 'var(--ant-color-text)', display: 'flex', alignItems: 'center', gap: 4 }}>
        {icon}{value}
      </span>
    </div>
  );
}

export default function WorkspaceToolbar({
  accounts, accountId, onAccountChange, busy,
  symbol, onSymbolChange, accountInfo, positionCount,
  positionsCount, onTogglePositionsPanel,
  mtError,
}: Props) {
  const { t } = useTranslation();
  const hasData = accountInfo != null;
  const profitColor = accountInfo && accountInfo.profit >= 0 ? '#3fb950' : '#f85149';
  const selectedAccount = (accounts || []).find(a => a.id === accountId);

  return (
    <div style={{
      display: 'flex', alignItems: 'flex-end', gap: 10, flexWrap: 'wrap',
      padding: '8px 12px 10px', background: 'var(--ant-color-bg-container)',
      borderBottom: '1px solid var(--ant-color-border)', flexShrink: 0,
    }}>
      {/* Account & Symbol selector */}
      <div style={{ ...groupStyle, flex: '0 0 auto' }}>
        <Space size={4}>
          <Select size="small" style={{ minWidth: 120, width: 220, maxWidth: '36vw' }}
            value={accountId || undefined} onChange={onAccountChange} disabled={busy}
            placeholder={t(SELECT_ACCOUNT_KEY)} showSearch optionFilterProp="label"
            notFoundContent={t(NO_ACCOUNTS_KEY)}
            options={(accounts || []).map((a) => ({ value: a.id, label: `${a.brokerServer} · ${a.login}` }))} />
          <SymbolPicker accountId={accountId} value={symbol} onChange={onSymbolChange}
            style={{ width: 120 }} />
        </Space>
      </div>

      {/* Account Summary — shown inline in spare toolbar space */}
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
            <SummaryChip label={t(TRADING_MARGIN_LEVEL_KEY)} value={`${accountInfo!.marginLevel.toFixed(0)}%`} />
          )}
        </div>
      )}

      {/* Open Positions — always visible, opens overlay panel */}
      <div onClick={onTogglePositionsPanel} role="button" tabIndex={0}
        onKeyUp={e => e.key === 'Enter' && onTogglePositionsPanel?.()}
        style={{ cursor: 'pointer' }}>
        <SummaryChip label={t(TRADING_POSITIONS_KEY)} value={positionCount != null ? String(positionCount) : '0'}
          color={positionCount != null && positionCount > 0 ? '#58a6ff' : undefined} />
      </div>

      {/* Account Metadata — platform, broker, server, mode, investor, leverage */}
      {selectedAccount && (
        <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap', alignItems: 'center' }}>
          <Tag color={selectedAccount.mtType === 'MT5' ? 'blue' : 'orange'} style={{ fontSize: 10, margin: 0, lineHeight: '18px' }}>
            {selectedAccount.mtType}
          </Tag>
          <Tag icon={<BankOutlined style={{ fontSize: 10 }} />} color="default" style={{ fontSize: 10, margin: 0, lineHeight: '18px' }}>
            {selectedAccount.brokerCompany}
          </Tag>
          <Tag icon={<AimOutlined style={{ fontSize: 10 }} />} color="default" style={{ fontSize: 10, margin: 0, lineHeight: '18px' }}>
            {selectedAccount.brokerServer}
          </Tag>
          {/* Connection method: Master (full trading) vs Investor (read-only) */}
          <Tag icon={selectedAccount.isInvestor
              ? <EyeOutlined style={{ fontSize: 10 }} />
              : <KeyOutlined style={{ fontSize: 10 }} />}
            color={selectedAccount.isInvestor ? 'orange' : 'green'}
            style={{ fontSize: 10, margin: 0, lineHeight: '18px' }}>
            {selectedAccount.isInvestor ? t(INVESTOR_READ_ONLY_KEY) : t(MASTER_TRADING_KEY)}
          </Tag>
          {selectedAccount.leverage && selectedAccount.leverage > 0 && (
            <SummaryChip label={t(TRADING_POSITION_LEVERAGE_KEY)} value={`1:${selectedAccount.leverage}`} />
          )}
        </div>
      )}

      {/* MT session error banner */}
      {mtError && (
        <div style={{
          marginTop: 8, padding: '6px 12px',
          background: 'rgba(248,81,73,0.1)', border: '1px solid var(--ant-color-error)',
          borderRadius: 6, fontSize: 12, color: 'var(--ant-color-error)',
          display: 'flex', alignItems: 'center', gap: 8,
        }}>
          <span style={{ fontSize: 14 }}>⚠</span>
          <span>{mtError}</span>
        </div>
      )}
    </div>
  );
}
