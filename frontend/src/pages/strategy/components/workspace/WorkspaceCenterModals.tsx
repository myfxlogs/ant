import { Modal, Drawer } from 'antd';
import { useTranslation } from 'react-i18next';
import { BACKTEST_KEY as WS_BACKTEST_KEY } from '@/gen/ant/v1/i18n/strategy_workspace_keys';
import BacktestPanel from '@/components/backtest/BacktestPanel';
import WorkspaceSidebar from './WorkspaceSidebar';
import { useWsAccount, useWsCode, useWsTemplates, useWsBacktest, useWsHistory, useWsAI } from '../../WorkspaceContext';

interface Props {
  btDrawerOpen: boolean;
  setBtDrawerOpen: (v: boolean) => void;
  sidebarDrawerOpen: boolean;
  setSidebarDrawerOpen: (v: boolean) => void;
  isMobile: boolean;
  onImport: () => void;
}

export default function WorkspaceCenterModals({
  btDrawerOpen, setBtDrawerOpen,
  sidebarDrawerOpen, setSidebarDrawerOpen,
  isMobile, onImport,
}: Props) {
  const { t } = useTranslation();
  const account = useWsAccount();
  const code = useWsCode();
  const templates = useWsTemplates();
  const backtest = useWsBacktest();
  const history = useWsHistory();
  const ai = useWsAI();

  return (
    <>
      <Modal
        title={t(WS_BACKTEST_KEY)}
        open={btDrawerOpen}
        onCancel={() => setBtDrawerOpen(false)}
        footer={null}
        width="95vw"
        style={{ top: 20 }}
        destroyOnClose
      >
        <div style={{ height: 'calc(95vh - 120px)', overflow: 'auto' }}>
          <BacktestPanel
            runner={backtest.runner}
            inputs={{
              strategyCode: code.code,
              accountId: account.accountId,
              symbol: account.symbol,
              timeframe: account.timeframe,
              templateId: templates.selectedId || undefined,
              strategyId: code.strategyId,
            }}
            onOpenHistory={(templateId?: string) => history.open(templateId)}
            onAIOptimize={() => ai.optimize()}
            code={code.code}
            onApplyTunedParams={code.setCode}
          />
        </div>
      </Modal>

      {isMobile && (
        <Drawer
          open={sidebarDrawerOpen}
          onClose={() => setSidebarDrawerOpen(false)}
          placement="left"
          width={280}
          styles={{ body: { padding: 0 } }}
        >
          <WorkspaceSidebar
            templates={templates.list}
            loading={templates.loading}
            selectedId={templates.selectedId || ''}
            onSelect={(id) => { templates.onSelect(id); setSidebarDrawerOpen(false); }}
            backtestRuns={(history.runs as Array<{ id: string; startedAt?: string; totalReturn?: number; totalTrades?: number; templateName?: string; templateId?: string }>) || []}
            runsLoading={history.loading}
            onOpenHistory={(tid) => { history.open(tid); setSidebarDrawerOpen(false); }}
            onImport={() => { onImport(); setSidebarDrawerOpen(false); }}
            onNew={() => { templates.onSelect(''); setSidebarDrawerOpen(false); }}
            collapsed={false}
            onToggle={() => setSidebarDrawerOpen(false)}
          />
        </Drawer>
      )}
    </>
  );
}
