import { Suspense, lazy } from 'react';
import { Collapse } from 'antd';
import { RobotOutlined, DoubleLeftOutlined, DoubleRightOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { useStrategyWorkspaceState } from './hooks/useStrategyWorkspaceState';
import WorkspaceCodePanel from './components/workspace/WorkspaceCodePanel';
import WorkspaceBacktestPanel from './components/workspace/WorkspaceBacktestPanel';
import WorkspaceTemplateManager from './components/workspace/WorkspaceTemplateManager';
import WorkspaceToolbar from './components/workspace/WorkspaceToolbar';
import BacktestParamsCard from './components/workspace/BacktestParamsCard';
import { AICodeReviseChat } from '@/components/strategy/CodeAssist';
import PriceChart from '@/components/chart/PriceChart';
import QuickTradePanel from '@/components/chart/QuickTradePanel';

const SaveTemplateModal = lazy(() => import('@/components/strategy/SaveTemplateModal'));

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
        quickTradeVisible={ws.quickTradeVisible} onToggleQuickTrade={() => ws.setQuickTradeVisible(!ws.quickTradeVisible)}
      />

      {/* ═══ THREE-COLUMN BODY ═══ */}
      <div style={{ display: 'flex', flex: '1 1 auto', overflow: 'hidden', minHeight: 0 }}>
        {/* ── LEFT: Code Panel ── */}
        {!ws.codePanelVisible ? (
          <div onClick={() => ws.setCodePanelVisible(true)} role="button" tabIndex={0}
            onKeyUp={(e) => e.key === 'Enter' && ws.setCodePanelVisible(true)}
            style={{
              width: 32, minWidth: 32, flex: '0 0 32px', display: 'flex', flexDirection: 'column',
              alignItems: 'center', justifyContent: 'center', gap: 8, cursor: 'pointer',
              background: 'linear-gradient(180deg, #f8fafc 0%, #eef2f7 100%)',
              borderRight: '1px solid #e2e8f0', boxShadow: '2px 0 8px rgba(15,23,42,0.04)',
              padding: '14px 0', transition: 'background 0.2s',
            }}>
            <DoubleRightOutlined style={{ fontSize: 14 }} />
            <span style={{ fontSize: 10, writingMode: 'vertical-rl', fontWeight: 500 }}>Code</span>
          </div>
        ) : (
          <div style={{
            width: '1%', minWidth: 280, flexShrink: 0,
            height: '100%', overflowY: 'auto',
            borderRight: '1px solid #eee', background: '#fcfcfd',
            padding: '12px 12px 12px 0', display: 'flex', flexDirection: 'column', gap: 12,
          }}>
            {/* Hide code handle */}
            <div onClick={() => ws.setCodePanelVisible(false)} role="button" tabIndex={0}
              onKeyUp={(e) => e.key === 'Enter' && ws.setCodePanelVisible(false)}
              style={{
                cursor: 'pointer', color: '#64748b', fontSize: 11, fontWeight: 600,
                display: 'flex', alignItems: 'center', gap: 4,
                background: 'linear-gradient(180deg, #f1f5f9 0%, #e8eef5 100%)',
                borderRadius: 6, padding: '6px 10px', marginBottom: 4,
              }}>
              <DoubleLeftOutlined /> {t('strategy.workspace.hideCode', 'Hide Code')}
            </div>

            <WorkspaceCodePanel
              code={ws.code} onCodeChange={ws.setCode}
              validating={ws.validating} onValidate={ws.handleValidate}
              validationResult={ws.validationResult}
              onRunBacktest={ws.handleRunBacktest} backtestSubmitting={ws.btSubmitting}
              canSave={ws.canSave} onSave={ws.handleSave} onCopy={ws.handleCopy}
              aiPrompt={ws.aiPrompt} onAiPromptChange={ws.setAiPrompt}
              aiGenerating={ws.aiGenerating} onGenerateCode={ws.handleGenerateCode}
            />

            <Collapse ghost size="small" style={{ background: 'transparent' }} items={[
              { key: 'ai', label: <span><RobotOutlined style={{ marginRight: 6 }} />{t('strategy.workspace.aiAssist', 'AI Assistant')}</span>, children: <AICodeReviseChat code={ws.code} onApply={ws.setCode} /> },
              { key: 'template', label: t('strategy.workspace.template.title', 'Template'), children: <WorkspaceTemplateManager templates={ws.templates} loading={ws.templatesLoading} loadedTemplate={ws.loadedTemplate} onLoad={ws.handleLoadTemplate} onSaveAs={ws.handleSaveAs} /> },
            ]} />
          </div>
        )}

        {/* ── MIDDLE: Chart + Backtest ── */}
        <div style={{ flex: '1 1 0', minWidth: 0, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
          {/* Chart area (PriceChart handles timeframe + indicator toolbar internally) */}
          {/* Chart area: flex column gives PriceChart 100% height via CSS */}
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

          {/* Backtest Section — content-height only; chart gets all remaining space */}
          <div style={{
            flexShrink: 0, borderTop: '1px solid #e8e8e8', overflowY: 'auto',
          }}>
            {/* Backtest Parameters Card */}
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

            {/* Backtest Results / Smart Tuning — collapsed until results arrive */}
            <div style={{
              borderTop: '1px solid #e8e8e8', background: '#fafbfc',
            }}>
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
                <span style={{ fontSize: 10, color: '#8c8c8c' }}>{ws.btResultsExpanded ? '▲' : '▼'}</span>
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
            borderLeft: '1px solid #e8e8e8', background: '#f8fafc',
            display: 'flex', flexDirection: 'column', overflow: 'hidden',
          }}>
            {/* Header */}
            <div style={{
              padding: '10px 14px', display: 'flex', justifyContent: 'space-between', alignItems: 'center',
              background: 'linear-gradient(180deg, #ffffff 0%, #f1f5f9 100%)',
              borderBottom: '1px solid #e8e8e8', flexShrink: 0,
            }}>
              <span style={{ fontSize: 13, fontWeight: 700, color: '#0f172a' }}>
                ⚡ Quick Trade
              </span>
              <span onClick={() => ws.setQuickTradeVisible(false)} role="button" tabIndex={0}
                onKeyUp={(e) => e.key === 'Enter' && ws.setQuickTradeVisible(false)}
                style={{ cursor: 'pointer', color: '#94a3b8', fontSize: 16, lineHeight: 1 }}>
                ✕
              </span>
            </div>
            {/* Body */}
            <div style={{ flex: 1, overflowY: 'auto', padding: 12 }}>
              {!ws.symbol ? (
                <div style={{ fontSize: 12, color: '#8c8c8c', textAlign: 'center', padding: '24px 0' }}>
                  Select a symbol first
                </div>
              ) : (
                <QuickTradePanel
                  accountId={ws.accountId} symbol={ws.symbol}
                  accountInfo={ws.accountInfo}
                  accountMeta={ws.selectedAccountMeta}
                  positions={ws.qtPositions}
                  recentTrades={ws.qtRecentTrades}
                  onClosePosition={ws.handleClosePosition}
                />
              )}
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
