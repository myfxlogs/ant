import { useState } from 'react';
import { Select, Button, Space, Tooltip } from 'antd';
import { ThunderboltOutlined, CodeOutlined, FullscreenOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import SymbolPicker from '@/components/chart/SymbolPicker';
import PriceChart from '@/components/chart/PriceChart';
import IndicatorPicker from '@/components/chart/IndicatorPicker';
import QuickTradePanel from '@/components/chart/QuickTradePanel';
import type { Account } from '@/types/account';

// Shared toolbar group style — matches QuantDinger .ide-toolbar-group
const groupStyle: React.CSSProperties = {
  padding: '6px 10px 8px', borderRadius: 10,
  background: 'rgba(255,255,255,0.72)', border: '1px solid rgba(0,0,0,0.05)',
  boxShadow: '0 1px 3px rgba(15,23,42,0.04)',
};

const groupLabelStyle: React.CSSProperties = {
  fontSize: 10, fontWeight: 700, textTransform: 'uppercase' as const,
  color: '#64748b', marginBottom: 4, lineHeight: 1,
};

interface Props {
  accounts: Account[];
  accountId: string;
  onAccountChange: (id: string) => void;
  symbol: string;
  onSymbolChange: (s: string) => void;
  timeframe: string;
  onTimeframeChange: (tf: string) => void;
  codePanelVisible: boolean;
  onToggleCodePanel: () => void;
}

export default function WorkspaceChartTab({
  accounts, accountId, onAccountChange,
  symbol, onSymbolChange, timeframe, onTimeframeChange,
  codePanelVisible, onToggleCodePanel,
}: Props) {
  const { t } = useTranslation();
  const [quickTradeVisible, setQuickTradeVisible] = useState(true);

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      {/* Chart panel toolbar — matches QuantDinger .chart-panel-toolbar */}
      <div style={{ marginBottom: 8 }}>
        {/* Top row — title + action buttons */}
        <div style={{
          display: 'flex', justifyContent: 'space-between', alignItems: 'center',
          marginBottom: 10, paddingBottom: 8,
          borderBottom: '1px solid #f0f0f0',
        }}>
          <span style={{ fontSize: 12, fontWeight: 600, color: '#595959' }}>
            {t('strategy.workspace.chartWindow', 'Chart Window')}
          </span>
          <Space size={6}>
            {/* Code toggle — matches .chart-panel-icon-btn */}
            <Tooltip title={codePanelVisible ? 'Hide Code' : 'Show Code'}>
              <Button size="small" type={codePanelVisible ? 'primary' : 'default'}
                icon={<CodeOutlined />} onClick={onToggleCodePanel}
                style={{ width: 28, height: 28, borderRadius: 8, padding: 0, display: 'flex', alignItems: 'center', justifyContent: 'center' }}
              />
            </Tooltip>
            {/* Quick Trade toggle — matches .chart-panel-qt-btn */}
            <Button size="small" type={quickTradeVisible ? 'primary' : 'default'}
              icon={<ThunderboltOutlined />} onClick={() => setQuickTradeVisible(!quickTradeVisible)}
              style={{ height: 28, borderRadius: 8, fontWeight: 600, padding: '0 10px' }}
            >
              Quick Trade
            </Button>
            {/* Fullscreen */}
            <Tooltip title="Fullscreen">
              <Button size="small" icon={<FullscreenOutlined />}
                style={{ width: 28, height: 28, borderRadius: 8, padding: 0, display: 'flex', alignItems: 'center', justifyContent: 'center' }}
              />
            </Tooltip>
          </Space>
        </div>

        {/* Control row — 3 grouped sections: Watchlist / Timeframe / Indicators */}
        <div style={{ display: 'flex', gap: 10, flexWrap: 'wrap', alignItems: 'flex-end' }}>
          {/* Watchlist group — matches .ide-toolbar-group--watchlist */}
          <div style={{ ...groupStyle, flex: '0 0 auto' }}>
            <div style={groupLabelStyle}>Watchlist</div>
            <Space size={4}>
              <Select
                size="small" style={{ minWidth: 120, width: 220, maxWidth: '36vw' }}
                value={accountId || undefined} onChange={onAccountChange}
                placeholder="Select account" showSearch optionFilterProp="label"
                notFoundContent="No accounts"
                options={accounts.map((a) => ({ value: a.id, label: a.alias || `${a.brokerCompany} · ${a.login}` }))}
              />
              <SymbolPicker accountId={accountId} value={symbol} onChange={onSymbolChange} style={{ width: 120 }} />
            </Space>
          </div>

          {/* Indicator group — matches .ide-toolbar-group--indicator */}
          <div style={{ ...groupStyle, flex: '1 1 240px' }}>
            <div style={groupLabelStyle}>Indicators</div>
            <IndicatorPicker />
          </div>
        </div>
      </div>

      {/* Chart + Quick Trade side-by-side — matches .ide-chart-fs-row */}
      <div style={{ display: 'flex', flex: 1, gap: quickTradeVisible ? 0 : 0, overflow: 'hidden' }}>
        {/* Chart area — matches .chart-panel */}
        <div style={{ flex: 1, minWidth: 0, overflow: 'hidden', background: '#fff' }}>
          {symbol ? (
            <PriceChart
              symbol={symbol} timeframe={timeframe} onTimeframeChange={onTimeframeChange}
              height={Math.max(400, window.innerHeight - 440)}
              accountId={accountId}
            />
          ) : (
            <div style={{
              display: 'flex', alignItems: 'center', justifyContent: 'center',
              height: 400, color: '#6b7280',
              border: '1px dashed rgba(0,0,0,0.12)', borderRadius: 8,
            }}>
              {t('strategy.workspace.selectSymbolHint', 'Select a trading account and symbol to view chart')}
            </div>
          )}
        </div>

        {/* Quick Trade side panel — matches .ide-quick-right--chart-fs */}
        {quickTradeVisible && (
          <div style={{
            width: '30%', minWidth: 260, maxWidth: 400, flex: '0 0 auto',
            borderLeft: '1px solid #e8e8e8', background: '#f8fafc',
            display: 'flex', flexDirection: 'column', overflow: 'hidden',
          }}>
            {/* Head — matches .ide-quick-panel-head */}
            <div style={{
              padding: '10px 14px', display: 'flex', justifyContent: 'space-between', alignItems: 'center',
              background: 'linear-gradient(180deg, #ffffff 0%, #f1f5f9 100%)',
              borderBottom: '1px solid #e8e8e8',
            }}>
              <span style={{ fontSize: 13, fontWeight: 700, color: '#0f172a' }}>
                <ThunderboltOutlined style={{ marginRight: 6 }} />
                Quick Trade
              </span>
              <Button type="text" size="small" onClick={() => setQuickTradeVisible(false)}
                style={{ color: '#94a3b8', padding: 0, minWidth: 20 }}>
                ✕
              </Button>
            </div>
            {/* Body — matches .ide-quick-panel-body */}
            <div style={{ flex: 1, overflowY: 'auto', padding: 12 }}>
              {!symbol ? (
                <div style={{ fontSize: 12, color: '#8c8c8c', textAlign: 'center', padding: '24px 0' }}>
                  Select a symbol first
                </div>
              ) : (
                <QuickTradePanel accountId={accountId} symbol={symbol} />
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
