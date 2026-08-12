import BacktestResultsTab from '@/components/backtest/BacktestResultsTab';
import { useAIFix } from '@/components/backtest/useAIFix';
import { useWsAccount, useWsCode, useWsTemplates, useWsBacktest, useWsHistory, useWsAI } from '../../WorkspaceContext';

export default function MobileBacktestContent() {
  const account = useWsAccount();
  const code = useWsCode();
  const templates = useWsTemplates();
  const backtest = useWsBacktest();
  const history = useWsHistory();
  const ai = useWsAI();

  const aiFix = useAIFix({
    strategyId: code.strategyId,
    currentCode: code.code,
    onApplyCode: code.setCode,
    onRerunBacktest: () => backtest.runner.run({
      strategyCode: code.code, accountId: account.accountId, symbol: account.symbol,
      timeframe: account.timeframe, templateId: templates.selectedId || undefined, strategyId: code.strategyId,
    }),
  });

  return (
    <>
      <BacktestResultsTab
        status={backtest.runner.status}
        metrics={backtest.runner.metrics}
        executionAssumptions={backtest.runner.executionAssumptions}
        errorMsg={backtest.runner.errorMsg}
        onAIOptimize={() => ai.optimize()}
        onOpenHistory={() => history.open()}
        trades={backtest.runner.chartTrades}
        panelHeight={300}
        onCancel={backtest.runner.cancelRun}
        gateUpdate={backtest.runner.gateUpdate}
        gateResults={backtest.runner.gateResults}
        qualityPreview={backtest.runner.qualityPreview}
        blindSpots={backtest.runner.blindSpots}
        strategyId={code.strategyId}
        onAIFix={aiFix.handleAIFix}
        aiFixing={aiFix.aiFixing}
        runMeta={backtest.runner.runMeta}
      />
      {aiFix.diffModal}
    </>
  );
}
