import { useState } from 'react';
import { Select, Button, Radio, Tooltip, Space } from 'antd';
import { ThunderboltOutlined, CodeOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import SymbolPicker from '@/components/chart/SymbolPicker';
import PriceChart from '@/components/chart/PriceChart';
import type { Account } from '@/types/account';

const TIMEFRAMES = [
  { label: '1m', value: '1m' }, { label: '5m', value: '5m' },
  { label: '15m', value: '15m' }, { label: '30m', value: '30m' },
  { label: '1h', value: '1h' }, { label: '4h', value: '4h' },
  { label: '1d', value: '1d' }, { label: '1w', value: '1w' },
];

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
  const [quickTradeVisible, setQuickTradeVisible] = useState(false);

  return (
    <div style={{ position: 'relative' }}>
      {/* Chart toolbar — matches QuantDinger chart-panel-toolbar */}
      <div style={{ marginBottom: 8 }}>
        {/* Top row: title + action buttons */}
        <div style={{
          display: 'flex', justifyContent: 'space-between', alignItems: 'center',
          marginBottom: 6,
        }}>
          <span style={{ fontSize: 13, fontWeight: 500, color: '#595959' }}>
            {t('strategy.workspace.chartWindow', 'Chart')}
          </span>
          <Space size="small">
            <Tooltip title={codePanelVisible
              ? t('strategy.workspace.hideCode', 'Hide Code')
              : t('strategy.workspace.showCode', 'Show Code')}>
              <Button
                size="small"
                type={codePanelVisible ? 'primary' : 'default'}
                icon={<CodeOutlined />}
                onClick={onToggleCodePanel}
              />
            </Tooltip>
            <Tooltip title={t('strategy.workspace.quickTrade', 'Quick Trade')}>
              <Button
                size="small"
                type={quickTradeVisible ? 'primary' : 'default'}
                icon={<ThunderboltOutlined />}
                onClick={() => setQuickTradeVisible(!quickTradeVisible)}
              >
                {t('strategy.workspace.quickTrade', 'Trade')}
              </Button>
            </Tooltip>
          </Space>
        </div>

        {/* Control row: Account → Symbol → Timeframe */}
        <Space size="small" wrap>
          <span style={{ fontSize: 12, color: '#8c8c8c' }}>
            {t('strategy.workspace.account', 'Account')}
          </span>
          <Select
            size="small"
            style={{ minWidth: 180 }}
            value={accountId || undefined}
            onChange={onAccountChange}
            placeholder={t('strategy.workspace.accountPlaceholder', 'Select account')}
            showSearch
            optionFilterProp="label"
            notFoundContent={t('strategy.workspace.noAccounts', 'No available accounts')}
            options={accounts.map((a) => ({
              value: a.id,
              label: a.alias || `${a.brokerCompany} · ${a.login}`,
            }))}
          />
          <span style={{ fontSize: 12, color: '#8c8c8c' }}>
            {t('strategy.workspace.selectSymbol', 'Symbol')}
          </span>
          <SymbolPicker
            accountId={accountId}
            value={symbol}
            onChange={onSymbolChange}
            style={{ width: 130 }}
          />
          <Radio.Group
            value={timeframe}
            onChange={(e) => onTimeframeChange(e.target.value)}
            size="small"
            optionType="button"
            buttonStyle="solid"
          >
            {TIMEFRAMES.map((tf) => (
              <Radio.Button key={tf.value} value={tf.value}>{tf.label}</Radio.Button>
            ))}
          </Radio.Group>
        </Space>
      </div>

      {/* Chart + Quick-trade side-by-side */}
      <div style={{ display: 'flex', gap: 8 }}>
        <div style={{ flex: 1, minWidth: 0 }}>
          {symbol ? (
            <PriceChart
              symbol={symbol}
              timeframe={timeframe}
              height={450}
              accountId={accountId}
            />
          ) : (
            <div style={{
              display: 'flex', alignItems: 'center', justifyContent: 'center',
              height: 450, color: '#6b7280',
              border: '1px dashed rgba(0,0,0,0.12)', borderRadius: 8,
            }}>
              {t('strategy.workspace.selectSymbolHint', 'Select a trading account and symbol to view chart')}
            </div>
          )}
        </div>

        {/* Quick-trade panel */}
        {quickTradeVisible && (
          <div style={{
            width: 240, minWidth: 240,
            border: '1px solid rgba(0,0,0,0.12)', borderRadius: 8,
            padding: 12, background: '#fafafa',
          }}>
            <div style={{ fontSize: 13, fontWeight: 500, marginBottom: 8, color: '#595959' }}>
              <ThunderboltOutlined style={{ marginRight: 4 }} />
              {t('strategy.workspace.quickTrade', 'Quick Trade')}
            </div>
            {!symbol ? (
              <div style={{ fontSize: 12, color: '#8c8c8c' }}>
                {t('strategy.workspace.quickTradeHint', 'Select a symbol first')}
              </div>
            ) : (
              <div style={{ fontSize: 12, color: '#8c8c8c', textAlign: 'center', padding: '24px 0' }}>
                {t('strategy.workspace.tradePanelPlaceholder', 'Trade panel — coming soon')}
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
