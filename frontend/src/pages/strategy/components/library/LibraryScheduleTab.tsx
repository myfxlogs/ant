import { Button } from 'antd';
import type { FormInstance } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next'
import { CREATE_SCHEDULE_KEY } from '@/gen/ant/v1/i18n/strategy_schedules_keys';
;
import ScheduleTable from '../ScheduleTable';
import type { ScheduleFormValues } from '../EditScheduleModal';
import type { ScheduleRow, ScheduleHealthSummary, TriggerResult, TriggerContext, TemplateOption, AccountRow } from '../../hooks/libraryTypes';

interface Props {
  schedules: ScheduleRow[];
  allSchedules: ScheduleRow[];
  loading: boolean;
  templates: TemplateOption[];
  accounts: AccountRow[];
  symbols: { value: string; label: string }[];
  symbolsLoading: boolean;
  formatTime: (v: unknown) => string;
  // Edit modal
  openEdit: boolean; setOpenEdit: (v: boolean) => void;
  editing: ScheduleRow | null; setEditing: (v: ScheduleRow | null) => void;
  form: FormInstance<ScheduleFormValues>; accountIdWatch: string | undefined;
  loadSymbols: (accountId: string, symbol?: string) => void;
  submitEdit: () => void;
  openUpdate: (row: ScheduleRow) => void;
  // Actions
  onToggleActive: (row: ScheduleRow, next: boolean) => void;
  onDelete: (row: ScheduleRow) => void;
  onManualTrigger: (row: ScheduleRow) => void;
  loadScheduleHealth: (row: ScheduleRow) => void;
  // Health
  healthOpen: boolean; setHealthOpen: (v: boolean) => void;
  healthLoading: boolean; healthTarget: ScheduleRow | null; setHealthTarget: (v: ScheduleRow | null) => void;
  healthSummary: ScheduleHealthSummary | null; setHealthSummary: (v: ScheduleHealthSummary | null) => void;
  // Trigger
  triggering: boolean; openTrigger: boolean; setOpenTrigger: (v: boolean) => void;
  triggerResult: TriggerResult | null; triggerContext: TriggerContext | null;
  setTriggerContext: (v: TriggerContext | null) => void; setTriggerResult: (v: TriggerResult | null) => void;
  doOrderSend: () => void;
  // Create
  openCreate: () => void;
}

export default function LibraryScheduleTab(props: Props) {
  const { t } = useTranslation();

  return (
    <div style={{ padding: '16px 0' }}>
      <div style={{ marginBottom: 12, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <span />
        <Button type="primary" icon={<PlusOutlined />} onClick={props.openCreate}>
          {t(CREATE_SCHEDULE_KEY, '新建运行')}
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
        onEdit={props.openUpdate}
        onToggleActive={props.onToggleActive}
        onHealthCheck={props.loadScheduleHealth}
        onManualTrigger={props.onManualTrigger}
        onDelete={props.onDelete}
      />

    </div>
  );
}
