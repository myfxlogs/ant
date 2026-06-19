import { useEffect } from 'react';
import { Alert, Button, Collapse, Form, Input, Modal } from 'antd';
import { ExclamationCircleOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next'
import { SCHEDULE_LAUNCH_ACTIONS_ADD_ACCOUNT_KEY, SCHEDULE_LAUNCH_NO_ACCOUNT_BODY_KEY, SCHEDULE_LAUNCH_NO_ACCOUNT_TITLE_KEY } from '@/gen/ant/v1/i18n/strategy_templates_keys';

;
import { buildAccountOptions } from './EditScheduleBasicFields';
import EditScheduleBasicFields from './EditScheduleBasicFields';
import EditScheduleRiskFields from './EditScheduleRiskFields';

// ScheduleType 与 strategy/templates 页的"调度上线"表单保持一致：
//   interval     固定毫秒间隔
//   kline_close  按 K 线收盘稳定触发 (triggerMode=stable_kline)
//   hf_quote     逐笔报价高频触发 (triggerMode=hf_quote_stream)
//
// 后端 scheduleConfig 统一使用 intervalMs / hfCooldownMs (毫秒) + triggerMode。
type ScheduleType = 'interval' | 'kline_close' | 'hf_quote';

type ScheduleFormValues = {
  id?: string;
  templateId: string;
  accountId: string;
  name: string;
  symbol: string;
  timeframe: string;
  scheduleType: ScheduleType;
  intervalMs?: number;
  hfCooldownMs?: number;
  defaultVolume?: number;
  maxPositions?: number;
  stopLossPriceOffset?: number;
  takeProfitPriceOffset?: number;
  maxDrawdownPct?: number;
  isActive?: boolean;
  parametersJson?: string;
};

type Props = {
  editing: any | null;
  open: boolean;
  loading: boolean;
  form: any;
  templates: any[];
  accounts: any[];
  symbols: { value: string; label: string }[];
  symbolsLoading: boolean;
  accountIdWatch: string | undefined;
  onCancel: () => void;
  onOk: () => void;
};

export default function EditScheduleModal({
  editing, open, loading, form, templates, accounts,
  symbols, symbolsLoading, accountIdWatch, onCancel, onOk,
}: Props) {
  const { t } = useTranslation();
  const scheduleTypeWatch = Form.useWatch('scheduleType', form) as ScheduleType | undefined;
  const isCreate = !editing?.id;
  const accountOptions = buildAccountOptions(accounts);
  const noAccount = isCreate && accountOptions.length === 0;

  useEffect(() => {
    if (!open) return;
    const cur = form.getFieldsValue(true) as Partial<ScheduleFormValues>;
    const patch: Partial<ScheduleFormValues> = {};
    if (!cur.scheduleType) patch.scheduleType = 'kline_close';
    if (cur.intervalMs === undefined) patch.intervalMs = 300_000;
    if (cur.hfCooldownMs === undefined) patch.hfCooldownMs = 1_000;
    if (Object.keys(patch).length > 0) form.setFieldsValue(patch);
  }, [open, form]);

  return (
    <Modal
      title={editing ? t(EDIT_MODAL_TITLE_EDIT_KEY) : t(EDIT_MODAL_TITLE_CREATE_KEY)}
      open={open} onCancel={onCancel} onOk={onOk} confirmLoading={loading}
      okText={t('common.save')} cancelText={t('common.cancel')} width={720} destroyOnClose
    >
      {noAccount && (
        <Alert type="warning" showIcon icon={<ExclamationCircleOutlined />} className="mb-3"
          message={t(SCHEDULE_LAUNCH_NO_ACCOUNT_TITLE_KEY, '还没有可用的交易账号')}
          description={
            <div>
              {t(SCHEDULE_LAUNCH_NO_ACCOUNT_BODY_KEY, '请先在"账户管理"中添加并绑定 MT4/MT5 账号，账号联机成功后才能上线调度。')}
              <div className="mt-2">
                <Button size="small" type="primary" onClick={() => window.open('/accounts/bind', '_blank')}>
                  {t(SCHEDULE_LAUNCH_ACTIONS_ADD_ACCOUNT_KEY, '去添加交易账号')}
                </Button>
              </div>
            </div>
          }
        />
      )}

      <Form form={form} layout="vertical" disabled={noAccount}>
        <EditScheduleBasicFields
          isCreate={isCreate}
          accountIdWatch={accountIdWatch}
          symbols={symbols}
          symbolsLoading={symbolsLoading}
          scheduleTypeWatch={scheduleTypeWatch}
          templates={templates}
          accounts={accounts}
        />
        <EditScheduleRiskFields />
        <Collapse items={[{
          key: 'advanced',
          label: t(EDIT_MODAL_ADVANCED_TITLE_KEY),
          children: (
            <Form.Item
              label={t(EDIT_MODAL_ADVANCED_PARAMETERS_JSON_KEY)}
              name="parametersJson"
              extra={t(EDIT_MODAL_ADVANCED_PARAMETERS_JSON_EXTRA_KEY)}
            >
              <Input.TextArea rows={7} placeholder={`{\n  "fast": 10,\n  "slow": 20\n}`} />
            </Form.Item>
          ),
        }]} />
      </Form>
    </Modal>
  );
}

export type { ScheduleFormValues, ScheduleType };
