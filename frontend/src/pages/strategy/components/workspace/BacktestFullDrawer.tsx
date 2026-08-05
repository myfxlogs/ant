import { Modal } from 'antd';
import { useTranslation } from 'react-i18next';
import { BACKTEST_KEY as WS_BACKTEST_KEY } from '@/gen/ant/v1/i18n/strategy_workspace_keys';
import BacktestPanel from '@/components/backtest/BacktestPanel';

interface Props {
  open: boolean;
  onClose: () => void;
  runner: ReturnType<typeof import('../../WorkspaceContext').useWsBacktest>['runner'];
  strategyCode: string;
  accountId: string;
  symbol: string;
  timeframe: string;
  templateId?: string;
  strategyId?: string;
  onOpenHistory: (templateId?: string) => void;
  onAIOptimize: () => void;
  onApplyTunedParams: (code: string) => void;
}

export default function BacktestFullDrawer({ open, onClose, runner, strategyCode, accountId, symbol, timeframe, templateId, strategyId, onOpenHistory, onAIOptimize, onApplyTunedParams }: Props) {
  const { t } = useTranslation();
  return (
    <Modal
      title={t(WS_BACKTEST_KEY)}
      open={open}
      onCancel={onClose}
      footer={null}
      width="95vw"
      style={{ top: 20 }}
      destroyOnClose
    >
      <div style={{ height: 'calc(95vh - 120px)', overflow: 'auto' }}>
        <BacktestPanel
          runner={runner}
          inputs={{ strategyCode, accountId, symbol, timeframe, templateId: templateId || undefined, strategyId: strategyId || undefined }}
          onOpenHistory={onOpenHistory}
          onAIOptimize={onAIOptimize}
          code={strategyCode}
          onApplyTunedParams={onApplyTunedParams}
        />
      </div>
    </Modal>
  );
}
