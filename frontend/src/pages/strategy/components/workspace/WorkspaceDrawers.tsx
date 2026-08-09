import { Drawer } from 'antd';
import { useTranslation } from 'react-i18next';
import { TITLE_KEY as INDICATOR_CATALOG_TITLE_KEY } from '@/gen/ant/v1/i18n/indicator_catalog_keys';
import VersionHistoryDrawer from './VersionHistoryDrawer';
import { SaveTemplateWrapper } from '../../WorkspaceLayout';
import { BacktestParamsModal, type BacktestModalResult } from './BacktestParamsModal';
import IndicatorCatalogContent from './IndicatorCatalogContent';
import { useWsCode, useWsAccount, useWsBacktest } from '../../WorkspaceContext';

interface Props {
  btModalOpen: boolean;
  setBtModalOpen: (v: boolean) => void;
  indicatorDrawerOpen: boolean;
  setIndicatorDrawerOpen: (v: boolean) => void;
  versionHistoryOpen: boolean;
  setVersionHistoryOpen: (v: boolean) => void;
}

export default function WorkspaceDrawers({ btModalOpen, setBtModalOpen, indicatorDrawerOpen, setIndicatorDrawerOpen, versionHistoryOpen, setVersionHistoryOpen }: Props) {
  const { t } = useTranslation();
  const code = useWsCode();
  const account = useWsAccount();
  const backtest = useWsBacktest();

  return (
    <>
      <BacktestParamsModal
        open={btModalOpen}
        onClose={() => setBtModalOpen(false)}
        code={code.code}
        symbol={account.symbol}
        timeframe={account.timeframe}
        onConfirm={(result: BacktestModalResult) => {
          const p = result.params;
          backtest.setInitialCapital(p.initialCapital);
          backtest.setLeverage(p.leverage);
          backtest.setCommission(p.commission);
          backtest.setSlippage(p.slippage);
          backtest.setTradeDirection(p.tradeDirection);
          backtest.setStrictMode(p.strictMode);
          backtest.setStartDate(result.startDate);
          backtest.setEndDate(result.endDate);
          if (result.strategyParams) {
            for (const [name, value] of Object.entries(result.strategyParams)) {
              backtest.runner.setParam(name, value);
            }
          }
          backtest.run({
            params: result.strategyParams,
            executionConfig: {
              commission: p.commission,
              slippage: p.slippage,
              leverage: p.leverage,
              tradeDirection: p.tradeDirection,
              strictMode: p.strictMode,
            },
          });
        }}
      />
      <SaveTemplateWrapper code={code} />
      <Drawer title={t(INDICATOR_CATALOG_TITLE_KEY)} open={indicatorDrawerOpen} onClose={() => setIndicatorDrawerOpen(false)} width={640} styles={{ body: { overflowY: 'auto' } }}>
        <IndicatorCatalogContent />
      </Drawer>
      <VersionHistoryDrawer
        open={versionHistoryOpen}
        strategyId={code.strategyId}
        onClose={() => setVersionHistoryOpen(false)}
        onRollback={(sourceCode) => { code.setCode(sourceCode); }}
      />
    </>
  );
}
