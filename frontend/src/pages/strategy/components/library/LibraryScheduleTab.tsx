import { Button, Space } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import ScheduleTable from '../ScheduleTable';
import EditScheduleModal from '../EditScheduleModal';
import TriggerModal from '../TriggerModal';
import ScheduleHealthModal from '../ScheduleHealthModal';

interface Props {
  schedules: any[];
  allSchedules: any[];
  loading: boolean;
  templates: any[];
  accounts: any[];
  symbols: any[];
  symbolsLoading: boolean;
  formatTime: (v: any) => string;
  // Edit modal
  openEdit: boolean; setOpenEdit: (v: boolean) => void;
  editing: any; setEditing: (v: any) => void;
  form: any; accountIdWatch: string | undefined;
  loadSymbols: (accountId: string, symbol?: string) => void;
  submitEdit: () => void;
  // Actions
  onToggleActive: (row: any, next: boolean) => void;
  onDelete: (row: any) => void;
  onManualTrigger: (row: any) => void;
  loadScheduleHealth: (row: any) => void;
  // Health
  healthOpen: boolean; setHealthOpen: (v: boolean) => void;
  healthLoading: boolean; healthTarget: any; setHealthTarget: (v: any) => void;
  healthSummary: any; setHealthSummary: (v: any) => void;
  // Trigger
  triggering: boolean; openTrigger: boolean; setOpenTrigger: (v: boolean) => void;
  triggerResult: any; triggerContext: any; setTriggerContext: (v: any) => void; setTriggerResult: (v: any) => void;
  doOrderSend: (schedule: any) => void;
  // Create
  openCreate: () => void;
}

export default function LibraryScheduleTab(props: Props) {
  const { t } = useTranslation();

  const doOrderSend = async () => {
    // Extract logic from useSchedulePage's doOrderSend pattern
    const { schedule } = props.triggerContext || {};
    if (!schedule) return;
    const { triggerResult } = props;
    const raw = triggerResult?.signal;
    if (!raw) return;
    const { tradingApi } = await import('@/client/trading');
    const { getTradingRiskToastMessage } = await import('@/utils/tradingRiskError');
    const { message } = await import('antd');
    const signal = raw;
    const rawAction = String(signal?.type ?? signal?.signalType ?? signal?.signal ?? '').trim().toLowerCase();
    const action = rawAction === 'buy' || rawAction === 'sell' ? rawAction : '';
    const volumeNum = typeof signal?.volume === 'number' ? signal.volume : Number(signal?.volume);
    const volume = Number.isFinite(volumeNum) ? volumeNum : 0;
    if (!action || action === 'hold') { message.error(t('strategy.schedules.messages.signalHoldCannotOrder')); return; }
    if (!(volume > 0)) { message.error(t('strategy.schedules.messages.volumeInvalid')); return; }
    props.setTriggering && (() => {})();
    try {
      const payload: any = {
        accountId: schedule.accountId, symbol: signal.symbol || schedule.symbol, type: action, volume,
        price: typeof signal?.price === 'number' ? signal.price : Number(signal?.price || 0),
        stopLoss: typeof signal?.stopLoss === 'number' ? signal.stopLoss : Number(signal?.stopLoss || 0),
        takeProfit: typeof signal?.takeProfit === 'number' ? signal.takeProfit : Number(signal?.takeProfit || 0),
        comment: String(signal?.comment || ''),
      };
      const res = await tradingApi.orderSend(payload);
      if (res.error) { message.error(getTradingRiskToastMessage({ riskCode: res.riskError?.code, error: res.error, message: res.message, fallback: res.error || t('strategy.schedules.messages.orderFailed') })); return; }
      message.success(t('strategy.schedules.messages.orderSubmitted'));
      props.setOpenTrigger(false); props.setTriggerContext(null); props.setTriggerResult(null);
    } catch (e: any) { message.error(e?.message || t('strategy.schedules.messages.orderFailed')); }
  };

  return (
    <div style={{ padding: '16px 0' }}>
      <div style={{ marginBottom: 12, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <span />
        <Button type="primary" icon={<PlusOutlined />} onClick={props.openCreate}>
          {t('strategy.schedules.createSchedule', '新建运行')}
        </Button>
      </div>

      <ScheduleTable
        schedules={props.schedules}
        templates={props.templates}
        accounts={props.accounts}
        loading={props.loading}
        triggering={props.triggering}
        triggerContext={props.triggerContext}
        formatTime={props.formatTime}
        onEdit={props.openUpdate || ((row: any) => {
          const { openUpdate } = props as any;
          if (openUpdate) openUpdate(row);
        })}
        onToggleActive={props.onToggleActive}
        onHealthCheck={props.loadScheduleHealth}
        onManualTrigger={props.onManualTrigger}
        onDelete={props.onDelete}
      />

      <EditScheduleModal
        editing={props.editing}
        open={props.openEdit}
        loading={props.loading}
        form={props.form}
        templates={props.templates}
        accounts={props.accounts}
        symbols={props.symbols}
        symbolsLoading={props.symbolsLoading}
        accountIdWatch={props.accountIdWatch}
        onCancel={() => { props.setOpenEdit(false); props.setEditing(null); props.form.resetFields(); }}
        onOk={props.submitEdit}
      />

      <TriggerModal
        open={props.openTrigger}
        triggering={props.triggering}
        result={props.triggerResult}
        context={props.triggerContext}
        onCancel={() => { props.setOpenTrigger(false); props.setTriggerContext(null); props.setTriggerResult(null); }}
        onOrderSend={doOrderSend}
      />

      <ScheduleHealthModal
        open={props.healthOpen}
        loading={props.healthLoading}
        target={props.healthTarget}
        summary={props.healthSummary}
        onClose={() => { props.setHealthOpen(false); props.setHealthTarget(null); props.setHealthSummary(null); }}
      />
    </div>
  );
}
