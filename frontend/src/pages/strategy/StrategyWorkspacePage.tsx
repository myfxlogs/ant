import { Suspense, lazy } from 'react';
import { Collapse } from 'antd';
import { DoubleRightOutlined, DoubleLeftOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { useStrategyWorkspaceState } from './hooks/useStrategyWorkspaceState';
import WorkspaceCodePanel from './components/workspace/WorkspaceCodePanel';
import WorkspaceBacktestPanel from './components/workspace/WorkspaceBacktestPanel';
import WorkspaceTemplateManager from './components/workspace/WorkspaceTemplateManager';
import WorkspaceToolbar from './components/workspace/WorkspaceToolbar';
import BacktestParamsCard from './components/workspace/BacktestParamsCard';
import MiniPositionsTable from './components/workspace/MiniPositionsTable';
import { AICodeReviseChat } from '@/components/strategy/CodeAssist';
import StrategyGenChat from '@/components/strategy/StrategyGenChat';
import PriceChart from '@/components/chart/PriceChart';
import QuickTradePanel from '@/components/chart/QuickTradePanel';

const SaveTemplateModal = lazy(() => import('@/components/strategy/SaveTemplateModal'));

const CODE_PANEL_WIDTH = 750;
const POSITIONS_PANEL_WIDTH = 520;
const C = { border: "#e8e8e8", bg: "#f8fafc", bgAlt: "#f1f5f9", muted: "#8c8c8c", accent: "#1677ff" };

export default function StrategyWorkspacePage() {
  const { t } = useTranslation();
  const ws = useStrategyWorkspaceState();

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: 'calc(100vh - 112px)', background: '#fff' }}>
      {/* Title bar */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '8px 0 12px' }}>
        <h2 style={{ margin: 0, fontSize: 16, fontWeight: 600 }}>{t('strategy.workspace.title', 'Strategy Workspace')}</h2>
      </div>

      {/* ═══ TOP TOOLBAR ═══ */}
      <WorkspaceToolbar
        accounts={ws.activeAccounts} accountId={ws.accountId} onAccountChange={ws.handleAccountChange}
        symbol={ws.symbol} onSymbolChange={ws.setSymbol}
        accountInfo={ws.accountInfo} positionCount={ws.positionCount}
        codePanelVisible={ws.codePanelVisible} onToggleCodePanel={() => ws.setCodePanelVisible(!ws.codePanelVisible)}
        onCloseCodePanel={() => ws.setCodePanelVisible(false)}
        quickTradeVisible={ws.quickTradeVisible} onToggleQuickTrade={() => ws.setQuickTradeVisible(!ws.quickTradeVisible)}
      />

      {/* ═══ BODY: Chart + Backtest (code overlays chart when open) ═══ */}
      <div style={{ display: 'flex', flex: '1 1 auto', overflow: 'hidden', minHeight: 0, position: 'relative' }}>
        {/* ── Code toggle strip (always in flow) ── */}
        <div onClick={() => ws.setCodePanelVisible(!ws.codePanelVisible)} role="button" tabIndex={0}
          onKeyUp={(e) => e.key === 'Enter' && ws.setCodePanelVisible(!ws.codePanelVisible)}
          style={{
            width: 32, minWidth: 32, flex: '0 0 32px', display: 'flex', flexDirection: 'column',
            alignItems: 'center', justifyContent: 'center', gap: 8, cursor: 'pointer', zIndex: 10,
            background: ws.codePanelVisible
              ? 'linear-gradient(180deg, #1677ff 0%, #0958d9 100%)'
              : 'linear-gradient(180deg, #f8fafc 0%, #eef2f7 100%)',
            borderRight: '1px solid #e2e8f0',
            padding: '14px 0', transition: 'background 0.2s',
          }}>
          {ws.codePanelVisible
            ? <DoubleLeftOutlined style={{ fontSize: 14, color: '#fff' }} />
            : <DoubleRightOutlined style={{ fontSize: 14 }} />
          }
          <span style={{ fontSize: 10, writingMode: 'vertical-rl', fontWeight: 500,
            color: ws.codePanelVisible ? '#fff' : 'inherit' }}>Code</span>
        </div>

        {/* ── Expanded code panel (overlays chart, stays within workspace) ── */}
        {ws.codePanelVisible && (
          <div style={{
            position: 'absolute', left: 32, top: 0, bottom: 0, zIndex: 100,
            width: CODE_PANEL_WIDTH, overflowY: 'auto',
            background: '#fcfdfd', borderRight: '1px solid ' + C.border,
            boxShadow: '4px 0 24px rgba(0,0,0,0.1)',
            padding: '12px 14px', display: 'flex', flexDirection: 'column', gap: 12,
          }}>
            <WorkspaceCodePanel
              code={ws.code} onCodeChange={ws.setCode}
              validating={ws.validating} onValidate={ws.handleValidate}
              validationResult={ws.validationResult}
              onRunBacktest={ws.handleRunBacktest} backtestSubmitting={ws.btSubmitting}
              canSave={ws.canSave} onSave={ws.handleSave} onCopy={ws.handleCopy}
            />
            <StrategyGenChat symbol={ws.symbol} timeframe={ws.timeframe} onApply={ws.setCode} />
            <AICodeReviseChat code={ws.code} onApply={ws.setCode} />
            <Collapse ghost size="small" style={{ background: 'transparent' }} items={[
              { key: 'template', label: t('strategy.workspace.template.title', 'Template'), children: <WorkspaceTemplateManager templates={ws.templates} loading={ws.templatesLoading} loadedTemplate={ws.loadedTemplate} onLoad={ws.handleLoadTemplate} onSaveAs={ws.handleSaveAs} /> },
            ]} />
          </div>
        )}

        {/* ── Positions overlay panel (right side, next to Quick Trade) ── */}
        {ws.positionsPanelVisible && (
          <div style={{
            position: 'absolute', right: ws.quickTradeVisible ? 310 : 0, top: 0, bottom: 0, zIndex: 100,
            width: POSITIONS_PANEL_WIDTH, overflowY: 'auto',
            background: '#fcfdfd', borderLeft: '1px solid ' + C.border,
            boxShadow: '-4px 0 24px rgba(0,0,0,0.1)',
            display: 'flex', flexDirection: 'column',
          }}>
            <div style={{
              display: 'flex', justifyContent: 'space-between', alignItems: 'center',
              padding: '8px 14px', flexShrink: 0,
              background: 'linear-gradient(180deg, #fff 0%, #f1f5f9 100%)',
              borderBottom: '1px solid ' + C.border,
            }}>
              <span style={{ fontSize: 13, fontWeight: 700, color: '#0f172a' }}>
                Open Positions ({ws.allPositions.length})
              </span>
              <span onClick={() => ws.setPositionsPanelVisible(false)} role="button" tabIndex={0}
                onKeyUp={e => e.key === 'Enter' && ws.setPositionsPanelVisible(false)}
                style={{ cursor: 'pointer', color: '#94a3b8', fontSize: 16, lineHeight: 1 }}>✕</span>
            </div>
            <div style={{ flex: 1, overflowY: 'auto', padding: '8px 14px' }}>
              {ws.allPositions.length > 0 ? (
                <MiniPositionsTable positions={ws.allPositions} onClosePosition={ws.handleClosePosition} />
              ) : (
                <div style={{ textAlign: 'center', padding: 40, color: '#8c8c8c', fontSize: 13 }}>
                  No open positions for this account
                </div>
              )}
            </div>
          </div>
        )}

        {/* ── MIDDLE: Chart + Backtest ── */}
        <div style={{ flex: '1 1 0', minWidth: 0, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
          <div style={{ flex: '1 1 0', minHeight: 0, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
            {ws.symbol ? (
              <PriceChart
                symbol={ws.symbol} timeframe={ws.timeframe} onTimeframeChange={ws.setTimeframe}
                accountId={ws.accountId}
              />
            ) : (
              <div style={{
                display: 'flex', alignItems: 'center', justifyContent: 'center',
                height: '100%', color: '#6b7280',
                border: '1px dashed rgba(0,0,0,0.12)', borderRadius: 8, margin: 12,
              }}>
                {t('strategy.workspace.selectSymbolHint', 'Select a trading account and symbol to view chart')}
              </div>
            )}
          </div>

          <div style={{ flexShrink: 0, borderTop: '1px solid #e8e8e8', overflowY: 'auto' }}>
            <div style={{ marginBottom: 6 }}>
            <BacktestParamsCard
              initialCapital={ws.btInitialCapital} onInitialCapitalChange={ws.setBtInitialCapital}
              leverage={ws.btLeverage} onLeverageChange={ws.setBtLeverage}
              commission={ws.btCommission} onCommissionChange={ws.setBtCommission}
              slippage={ws.btSlippage} onSlippageChange={ws.setBtSlippage}
              startDate={ws.btStartDate} onStartDateChange={ws.setBtStartDate}
              endDate={ws.btEndDate} onEndDateChange={ws.setBtEndDate}
              tradeDirection={ws.btTradeDirection} onTradeDirectionChange={ws.setBtTradeDirection}
              highPrecision={ws.btHighPrecision} onHighPrecisionChange={ws.setBtHighPrecision}
              canRun={Boolean(ws.code && ws.symbol)}
              running={ws.btSubmitting} onRunBacktest={ws.handleRunBacktest}
              datePresets={ws.DATE_PRESETS} datePresetKey={ws.btDatePreset}
              onApplyDatePreset={ws.applyDatePreset}
              expanded={ws.btParamsExpanded} onExpandedChange={ws.setBtParamsExpanded}
            />
            </div>

            <div style={{ borderTop: '1px solid #e8e8e8', background: '#fafbfc' }}>
              <div onClick={() => ws.setBtResultsExpanded(!ws.btResultsExpanded)} role="button" tabIndex={0}
                onKeyUp={e => e.key === 'Enter' && ws.setBtResultsExpanded(!ws.btResultsExpanded)}
                style={{
                  display: 'flex', alignItems: 'center', justifyContent: 'space-between',
                  padding: '8px 14px', cursor: 'pointer', userSelect: 'none',
                  background: 'linear-gradient(180deg, #f8fafc 0%, #f1f5f9 100%)',
                }}>
                <span style={{ fontSize: 12, fontWeight: 700, color: '#262626' }}>
                  {ws.backtestSubTab === 'tuning' ? 'Smart Tuning' : 'Backtest Results'}
                  {ws.btStatus === 'running' && <span style={{ color: '#1890ff', marginLeft: 8, fontSize: 11 }}>Running...</span>}
                  {ws.btStatus === 'completed' && <span style={{ color: '#26a69a', marginLeft: 8, fontSize: 11 }}>Completed</span>}
                </span>
                <span style={{ fontSize: 10, color: C.muted }}>{ws.btResultsExpanded ? '▲' : '▼'}</span>
              </div>
              {ws.btResultsExpanded && (
                <div style={{ padding: '8px 14px' }}>
                  <WorkspaceBacktestPanel
                    status={ws.btStatus} metrics={ws.btMetrics}
                    errorMessage={ws.btError}
                    subTab={ws.backtestSubTab} onSubTabChange={ws.setBacktestSubTab}
                    tuneMethod={ws.tuneMethod} onTuneMethodChange={ws.setTuneMethod}
                    sweepDimensions={ws.sweepDimensions} onToggleDimension={ws.toggleDimension}
                    enabledSweepDims={ws.enabledSweepDims} cartesianSize={ws.cartesianSize}
                    tuningRunning={ws.tuningRunning} canRunTuning={Boolean(ws.code && ws.symbol)}
                    onRunTuning={ws.handleRunTuning}
                    gateLoading={ws.gateLoading} gateGates={ws.gateGates}
                    gateSummary={ws.gateSummary} gateError={ws.gateError}
                    onRunGate={ws.handleRunGate}
                  />
                </div>
              )}
            </div>
          </div>
        </div>

        {/* ── RIGHT: Quick Trade ── */}
        {ws.quickTradeVisible && (
          <div style={{
            width: '1%', minWidth: 300, flexShrink: 0,
            borderLeft: '1px solid #e8e8e8', background: C.bg,
            display: 'flex', flexDirection: 'column', overflow: 'hidden',
          }}>
            <div style={{
              padding: '10px 14px', display: 'flex', justifyContent: 'space-between', alignItems: 'center',
              background: 'linear-gradient(180deg, #ffffff 0%, #f1f5f9 100%)',
              borderBottom: '1px solid #e8e8e8', flexShrink: 0,
            }}>
              <span style={{ fontSize: 13, fontWeight: 700, color: '#0f172a' }}>⚡ Quick Trade</span>
              <span onClick={() => ws.setQuickTradeVisible(false)} role="button" tabIndex={0}
                onKeyUp={(e) => e.key === 'Enter' && ws.setQuickTradeVisible(false)}
                style={{ cursor: 'pointer', color: '#94a3b8', fontSize: 16, lineHeight: 1 }}>✕</span>
            </div>
            <div style={{ flex: 1, overflowY: 'auto', padding: 12 }}>
              <QuickTradePanel
                accountId={ws.accountId} symbol={ws.symbol}
                accountMeta={ws.selectedAccountMeta}
                allPositions={ws.allPositions}
                positions={ws.qtPositions}
                recentTrades={ws.qtRecentTrades}
                onClosePosition={ws.handleClosePosition}
                onToggleAllPositions={() => ws.setPositionsPanelVisible(!ws.positionsPanelVisible)}
              />
            </div>
          </div>
        )}
      </div>

      <Suspense fallback={null}>
        <SaveTemplateModal open={ws.saveModalOpen} confirmLoading={ws.saveLoading} form={ws.saveForm}
          onCancel={() => ws.setSaveModalOpen(false)} onOk={ws.handleSaveModalOk} />
      </Suspense>
    </div>
  );
}
