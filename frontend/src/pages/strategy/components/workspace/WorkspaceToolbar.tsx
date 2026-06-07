import { Select, Space, Button, Tooltip, Tag } from 'antd';
import { ThunderboltOutlined, CodeOutlined, RiseOutlined, FallOutlined, BankOutlined, AimOutlined, KeyOutlined, EyeOutlined } from '@ant-design/icons';
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
  codePanelVisible: boolean; onToggleCodePanel: () => void;
  onCloseCodePanel?: () => void;
  positionsCount?: number; onTogglePositionsPanel?: () => void;
  quickTradeVisible: boolean; onToggleQuickTrade: () => void;
  mtError?: string | null;
}

const groupStyle: React.CSSProperties = {
  padding: '5px 10px 7px', borderRadius: 10,
  background: 'rgba(255,255,255,0.72)', border: '1px solid rgba(0,0,0,0.05)',
  boxShadow: '0 1px 3px rgba(15,23,42,0.04)',
};

const groupLabelStyle: React.CSSProperties = {
  fontSize: 10, fontWeight: 700, textTransform: 'uppercase' as const,
  color: '#64748b', marginBottom: 4, lineHeight: 1,
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
      background: 'rgba(255,255,255,0.8)', border: '1px solid rgba(0,0,0,0.06)',
    }}>
      <span style={{ fontSize: 9, color: '#8c8c8c', fontWeight: 500, textTransform: 'uppercase' }}>{label}</span>
      <span style={{ fontSize: 13, fontWeight: 700, color: color || '#262626', display: 'flex', alignItems: 'center', gap: 4 }}>
        {icon}{value}
      </span>
    </div>
  );
}

export default function WorkspaceToolbar({
  accounts, accountId, onAccountChange, busy,
  symbol, onSymbolChange, accountInfo, positionCount,
  codePanelVisible, onToggleCodePanel, onCloseCodePanel,
  positionsCount, onTogglePositionsPanel,
  quickTradeVisible, onToggleQuickTrade,
  mtError,
}: Props) {
  const hasData = accountInfo != null;
  const profitColor = accountInfo && accountInfo.profit >= 0 ? '#26a69a' : '#ef5350';
  const selectedAccount = (accounts || []).find(a => a.id === accountId);
  const maybeCloseCode = (open: boolean) => { if (open && codePanelVisible) onCloseCodePanel?.(); };

  return (
    <div style={{
      display: 'flex', alignItems: 'flex-end', gap: 10, flexWrap: 'wrap',
      padding: '8px 12px 10px', background: '#f8fafc',
      borderBottom: '1px solid #e8e8e8', flexShrink: 0,
    }}>
      {/* Watchlist group */}
      <div style={{ ...groupStyle, flex: '0 0 auto' }}>
        <div style={groupLabelStyle}>Watchlist</div>
        <Space size={4}>
          <Select size="small" style={{ minWidth: 120, width: 220, maxWidth: '36vw' }}
            value={accountId || undefined} onChange={onAccountChange} disabled={busy}
            onDropdownVisibleChange={maybeCloseCode}
            placeholder="Select account" showSearch optionFilterProp="label"
            notFoundContent="No accounts"
            options={(accounts || []).map((a) => ({ value: a.id, label: `${a.brokerServer} · ${a.login}` }))} />
          <SymbolPicker accountId={accountId} value={symbol} onChange={onSymbolChange}
            onDropdownVisibleChange={maybeCloseCode} style={{ width: 120 }} />
        </Space>
      </div>

      {/* Account Summary — shown inline in spare toolbar space */}
      {hasData && (
        <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', alignItems: 'flex-end' }}>
          <SummaryChip label="Balance" value={`$${fmtCompact(accountInfo!.balance)}`} />
          <SummaryChip label="Equity" value={`$${fmtCompact(accountInfo!.equity)}`} />
          <SummaryChip
            label="Profit"
            value={`$${fmtCompact(Math.abs(accountInfo!.profit))}`}
            color={profitColor}
            icon={accountInfo!.profit >= 0
              ? <RiseOutlined style={{ fontSize: 11 }} />
              : <FallOutlined style={{ fontSize: 11 }} />}
          />
          <SummaryChip label="Free Margin" value={`$${fmtCompact(accountInfo!.freeMargin)}`} />
          {accountInfo!.marginLevel > 0 && (
            <SummaryChip label="Margin Lvl" value={`${accountInfo!.marginLevel.toFixed(0)}%`} />
          )}
        </div>
      )}

      {/* Open Positions — always visible, opens overlay panel */}
      <div onClick={onTogglePositionsPanel} role="button" tabIndex={0}
        onKeyUp={e => e.key === 'Enter' && onTogglePositionsPanel?.()}
        style={{ cursor: 'pointer' }}>
        <SummaryChip label="Positions" value={positionCount != null ? String(positionCount) : '0'}
          color={positionCount != null && positionCount > 0 ? '#1677ff' : undefined} />
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
            {selectedAccount.isInvestor ? 'Investor (Read-only)' : 'Master (Trading)'}
          </Tag>
          {selectedAccount.leverage && selectedAccount.leverage > 0 && (
            <SummaryChip label="Leverage" value={`1:${selectedAccount.leverage}`} />
          )}
        </div>
      )}

      {/* Spacer */}
      <div style={{ flex: 1 }} />

      {/* Actions — chart toolbar handles timeframe + indicators internally */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexShrink: 0 }}>
        <Tooltip title={codePanelVisible ? 'Hide Code' : 'Show Code'}>
          <Button size="small" type={codePanelVisible ? 'primary' : 'default'}
            icon={<CodeOutlined />} onClick={onToggleCodePanel}
            style={{ width: 30, height: 30, borderRadius: 6, padding: 0,
              display: 'flex', alignItems: 'center', justifyContent: 'center' }} />
        </Tooltip>
        <Button size="small" type={quickTradeVisible ? 'primary' : 'default'}
          icon={<ThunderboltOutlined />} onClick={onToggleQuickTrade}
          style={{ height: 30, borderRadius: 6, fontWeight: 600, padding: '0 14px',
            background: quickTradeVisible ? '#1890ff' : undefined,
            borderColor: quickTradeVisible ? '#1890ff' : undefined,
            boxShadow: quickTradeVisible ? '0 2px 8px rgba(24,144,255,0.3)' : undefined }}>
          Quick Trade
        </Button>
      </div>

      {/* MT session error banner */}
      {mtError && (
        <div style={{
          marginTop: 8, padding: '6px 12px',
          background: '#fff2f0', border: '1px solid #ffccc7',
          borderRadius: 6, fontSize: 12, color: '#cf1322',
          display: 'flex', alignItems: 'center', gap: 8,
        }}>
          <span style={{ fontSize: 14 }}>⚠</span>
          <span>{mtError}</span>
        </div>
      )}
    </div>
  );
}
