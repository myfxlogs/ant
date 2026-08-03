import { Drawer } from 'antd';
import { ImportOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { TITLE_KEY as INDICATOR_CATALOG_TITLE_KEY } from '@/gen/ant/v1/i18n/indicator_catalog_keys';
import { useWorkspaceStore } from '@/stores/workspaceStore';
import BacktestHistoryDrawer from './BacktestHistoryDrawer';
import VersionHistoryDrawer from './VersionHistoryDrawer';
import { SaveTemplateWrapper } from '../../WorkspaceLayout';
import { BacktestParamsModal, type BacktestModalResult } from './BacktestParamsModal';
import IndicatorCatalogContent from './IndicatorCatalogContent';
import ImportEAPanel from '../editor/ImportEAPanel';
import { useWsCode, useWsAccount, useWsBacktest, useWsHistory } from '../../WorkspaceContext';

interface Props {
  btModalOpen: boolean;
  setBtModalOpen: (v: boolean) => void;
  indicatorDrawerOpen: boolean;
  setIndicatorDrawerOpen: (v: boolean) => void;
  importDrawerOpen: boolean;
  setImportDrawerOpen: (v: boolean) => void;
  versionHistoryOpen: boolean;
  setVersionHistoryOpen: (v: boolean) => void;
}

export default function WorkspaceDrawers({ btModalOpen, setBtModalOpen, indicatorDrawerOpen, setIndicatorDrawerOpen, importDrawerOpen, setImportDrawerOpen, versionHistoryOpen, setVersionHistoryOpen }: Props) {
  const { t } = useTranslation();
  const setCenterTab = useWorkspaceStore(s => s.setCenterTab);
  const code = useWsCode();
  const account = useWsAccount();
  const backtest = useWsBacktest();
  const history = useWsHistory();

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
          backtest.run();
          setCenterTab('backtest');
        }}
      />
      <SaveTemplateWrapper code={code} />
      <Drawer title={t(INDICATOR_CATALOG_TITLE_KEY)} open={indicatorDrawerOpen} onClose={() => setIndicatorDrawerOpen(false)} width={640} styles={{ body: { overflowY: 'auto' } }}>
        <IndicatorCatalogContent />
      </Drawer>
      <Drawer
        title={<span><ImportOutlined style={{ marginRight: 8 }} />{t('strategy.importEA.title', { defaultValue: 'Import MQL Strategy' })}</span>}
        open={importDrawerOpen}
        onClose={() => setImportDrawerOpen(false)}
        width={680}
        destroyOnClose
      >
        <ImportEAPanel
          onApplyCode={(c) => { code.setCode(c); setImportDrawerOpen(false); setCenterTab('code'); }}
          onStrategyIdChange={(id) => { if (id) code.setStrategyId(id); }}
        />
      </Drawer>
      <BacktestHistoryDrawer
        open={history.modalOpen || history.drawerOpen}
        runs={history.runs}
        loading={history.loading}
        page={history.page}
        pageSize={history.pageSize}
        total={history.total}
        selectedRowKeys={history.selectedRowKeys}
        deleting={history.deleting}
        onPageChange={history.onPageChange}
        onSelectionChange={history.setSelectedRowKeys}
        onViewRun={history.onViewRun}
        onDeleteRun={history.onDeleteRun}
        onBatchDelete={history.onBatchDelete}
        onRefresh={history.onRefresh}
        onClose={history.runId ? history.close : history.closeModal}
        runId={history.runId}
      />
      <VersionHistoryDrawer
        open={versionHistoryOpen}
        strategyId={code.strategyId}
        onClose={() => setVersionHistoryOpen(false)}
        onRollback={(sourceCode) => { code.setCode(sourceCode); }}
      />
    </>
  );
}
