import { useEffect } from 'react';
import {
  Alert,
  Button,
  Col,
  Collapse,
  Divider,
  Form,
  Input,
  InputNumber,
  Modal,
  Row,
  Select,
  Space,
  Switch,
  Tooltip,
} from 'antd';
import { ExclamationCircleOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { isTradingAccountEnabled } from '@/utils/accountStatus';

// ScheduleType 与 strategy/templates 页的"调度上线"表单保持一致：
//   interval     固定毫秒间隔
//   kline_close  按 K 线收盘稳定触发 (triggerMode=stable_kline)
//   hf_quote     逐笔报价高频触发 (triggerMode=hf_quote_stream)
//
// 后端 scheduleConfig 统一使用 intervalMs / hfCooldownMs (毫秒) + triggerMode。
// 这里不再暴露 cron / runPreset / intervalSeconds 等遗留字段。
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
  // 风控（可选）
  defaultVolume?: number;
  maxPositions?: number;
  stopLossPriceOffset?: number;
  takeProfitPriceOffset?: number;
  maxDrawdownPct?: number;
  // 工作流
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

function buildAccountOptions(accounts: any[]) {
  return (accounts || []).filter(isTradingAccountEnabled).map((a: any) => ({
    value: a.id,
    label: a.login ? `${a.login} (${a.mtType || ''})` : a.id,
  }));
}

export default function EditScheduleModal({
  editing,
  open,
  loading,
  form,
  templates,
  accounts,
  symbols,
  symbolsLoading,
  accountIdWatch,
  onCancel,
  onOk,
}: Props) {
  const { t } = useTranslation();
  const scheduleTypeWatch = Form.useWatch('scheduleType', form) as ScheduleType | undefined;
  const isCreate = !editing?.id;
  const accountOptions = buildAccountOptions(accounts);
  const noAccount = isCreate && accountOptions.length === 0;
  const symbolDisabled = isCreate && (!accountIdWatch || symbolsLoading);
  const symbolPlaceholder = symbolDisabled
    ? t(
        symbolsLoading
          ? 'strategy.schedules.editModal.placeholders.symbolsLoading'
          : 'strategy.schedules.editModal.placeholders.selectAccountFirst',
        symbolsLoading ? 'Loading trading symbols...' : 'Select an account first',
      )
    : t('strategy.templates.scheduleLaunch.form.symbolPlaceholder', '搜索品种，如 EURUSD');

  // 打开时兜底默认值，避免旧调度数据缺字段导致显示异常。
  useEffect(() => {
    if (!open) return;
    const cur = form.getFieldsValue(true) as Partial<ScheduleFormValues>;
    const patch: Partial<ScheduleFormValues> = {};
    if (!cur.scheduleType) patch.scheduleType = 'kline_close';
    if (cur.intervalMs === undefined) patch.intervalMs = 300_000;
    if (cur.hfCooldownMs === undefined) patch.hfCooldownMs = 1_000;
    if (Object.keys(patch).length > 0) {
      form.setFieldsValue(patch);
    }
  }, [open, form]);

  return (
    <Modal
      title={
        editing
          ? t('strategy.schedules.editModal.title.edit')
          : t('strategy.schedules.editModal.title.create')
      }
      open={open}
      onCancel={onCancel}
      onOk={onOk}
      confirmLoading={loading}
      okText={t('common.save')}
      cancelText={t('common.cancel')}
      width={720}
      destroyOnClose
    >
      {noAccount && (
        <Alert
          type="warning"
          showIcon
          icon={<ExclamationCircleOutlined />}
          className="mb-3"
          message={t('strategy.templates.scheduleLaunch.noAccountTitle', '还没有可用的交易账号')}
          description={
            <div>
              {t(
                'strategy.templates.scheduleLaunch.noAccountBody',
                '请先在"账户管理"中添加并绑定 MT4/MT5 账号，账号联机成功后才能上线调度。',
              )}
              <div className="mt-2">
                <Button
                  size="small"
                  type="primary"
                  onClick={() => window.open('/accounts/bind', '_blank')}
                >
                  {t('strategy.templates.scheduleLaunch.actions.addAccount', '去添加交易账号')}
                </Button>
              </div>
            </div>
          }
        />
      )}

      <Form form={form} layout="vertical" disabled={noAccount}>
        {isCreate && (
          <Row gutter={12}>
            <Col span={12}>
              <Form.Item
                label={t('strategy.schedules.editModal.fields.template')}
                name="templateId"
                rules={[
                  {
                    required: true,
                    message: t('strategy.schedules.editModal.validation.templateRequired'),
                  },
                ]}
                extra={t('strategy.schedules.editModal.fields.templateExtra')}
              >
                <Select
                  showSearch
                  optionFilterProp="label"
                  options={(templates || []).map((tpl: any) => ({
                    value: tpl.id,
                    label: tpl.name,
                  }))}
                />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                label={t('strategy.templates.scheduleLaunch.form.account', '交易账户')}
                name="accountId"
                rules={[{ required: true, message: t('common.required', '必填') }]}
              >
                <Select
                  showSearch
                  optionFilterProp="label"
                  options={accountOptions}
                />
              </Form.Item>
            </Col>
          </Row>
        )}

        <Form.Item
          label={t('strategy.templates.scheduleLaunch.form.scheduleName', '调度名称')}
          name="name"
          rules={[
            { required: true, message: t('strategy.schedules.editModal.validation.nameRequired') },
            { max: 100 },
          ]}
        >
          <Input
            placeholder={t(
              'strategy.templates.scheduleLaunch.form.scheduleNamePlaceholder',
              '可选，用于在调度列表中区分',
            )}
          />
        </Form.Item>

        <Row gutter={12}>
          <Col span={12}>
            <Form.Item
              label={t('strategy.templates.scheduleLaunch.form.symbol', '交易品种')}
              name="symbol"
              rules={[{ required: true, message: t('common.required', '必填') }]}
            >
              <Select
                showSearch
                allowClear
                loading={symbolsLoading}
                options={symbols}
                optionFilterProp="label"
                placeholder={symbolPlaceholder}
                disabled={symbolDisabled}
              />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item
              label={t('strategy.templates.scheduleLaunch.form.timeframe', '周期')}
              name="timeframe"
              rules={[{ required: true, message: t('common.required', '必填') }]}
            >
              <Select
                options={['M1', 'M5', 'M15', 'M30', 'H1', 'H4', 'D1'].map((tf) => ({
                  value: tf,
                  label: tf,
                }))}
              />
            </Form.Item>
          </Col>
        </Row>

        <Form.Item
          label={t('strategy.templates.scheduleLaunch.form.scheduleType', '调度类型')}
          name="scheduleType"
          rules={[{ required: true }]}
        >
          <Select
            options={[
              {
                value: 'interval',
                label: t(
                  'strategy.templates.scheduleLaunch.form.scheduleTypes.interval',
                  '固定间隔',
                ),
              },
              {
                value: 'kline_close',
                label: t(
                  'strategy.templates.scheduleLaunch.form.scheduleTypes.klineClose',
                  'K线收盘触发',
                ),
              },
              {
                value: 'hf_quote',
                label: t(
                  'strategy.templates.scheduleLaunch.form.scheduleTypes.hfQuote',
                  '逐笔报价（高频）',
                ),
              },
            ]}
          />
        </Form.Item>

        {scheduleTypeWatch === 'interval' && (
          <Form.Item
            label={
              <Tooltip
                title={t(
                  'strategy.templates.scheduleLaunch.form.intervalMsTip',
                  '策略重新评估的周期，单位 ms。默认 5 分钟 = 300000',
                )}
              >
                <span>
                  {t('strategy.templates.scheduleLaunch.form.intervalMs', '间隔（ms）')}
                </span>
              </Tooltip>
            }
            name="intervalMs"
            rules={[{ required: true, type: 'number', min: 1000, message: '>= 1000' }]}
          >
            <InputNumber style={{ width: '100%' }} min={1000} step={1000} />
          </Form.Item>
        )}

        {scheduleTypeWatch === 'hf_quote' && (
          <Form.Item
            label={
              <Tooltip
                title={t(
                  'strategy.templates.scheduleLaunch.form.hfCooldownMsTip',
                  '逐笔报价模式下连续两次 evaluate 的最短间隔，避免算力浪费。',
                )}
              >
                <span>
                  {t('strategy.templates.scheduleLaunch.form.hfCooldownMs', '冷却时间（ms）')}
                </span>
              </Tooltip>
            }
            name="hfCooldownMs"
            rules={[{ required: true, type: 'number', min: 100, message: '>= 100' }]}
          >
            <InputNumber style={{ width: '100%' }} min={100} step={100} />
          </Form.Item>
        )}

        <Divider orientation="left" plain>
          {t('strategy.templates.scheduleLaunch.form.riskSection', '风控参数（可选）')}
        </Divider>

        <Row gutter={12}>
          <Col span={8}>
            <Form.Item
              label={
                <Tooltip
                  title={t(
                    'strategy.templates.scheduleLaunch.form.defaultVolumeTip',
                    '策略信号里 volume=0 时默认下单量。手数单位。',
                  )}
                >
                  <span>
                    {t('strategy.templates.scheduleLaunch.form.defaultVolume', '默认手数')}
                  </span>
                </Tooltip>
              }
              name="defaultVolume"
            >
              <InputNumber style={{ width: '100%' }} min={0} step={0.01} placeholder="0.01" />
            </Form.Item>
          </Col>
          <Col span={8}>
            <Form.Item
              label={
                <Tooltip
                  title={t(
                    'strategy.templates.scheduleLaunch.form.maxPositionsTip',
                    '同一品种上允许同时持有的最多持仓数；达到后本次信号跳过。',
                  )}
                >
                  <span>
                    {t('strategy.templates.scheduleLaunch.form.maxPositions', '最大持仓数')}
                  </span>
                </Tooltip>
              }
              name="maxPositions"
            >
              <InputNumber style={{ width: '100%' }} min={1} step={1} placeholder="不限" />
            </Form.Item>
          </Col>
          <Col span={8}>
            <Form.Item
              label={
                <Tooltip
                  title={t(
                    'strategy.templates.scheduleLaunch.form.maxDrawdownPctTip',
                    '自峰值权益的最大回撤比例，0.2 = 20%；触发后调度自动停用。',
                  )}
                >
                  <span>
                    {t(
                      'strategy.templates.scheduleLaunch.form.maxDrawdownPct',
                      '最大回撤比例（0~1）',
                    )}
                  </span>
                </Tooltip>
              }
              name="maxDrawdownPct"
              rules={[{ type: 'number', min: 0, max: 1 }]}
            >
              <InputNumber
                style={{ width: '100%' }}
                min={0}
                max={1}
                step={0.01}
                placeholder="0.2"
              />
            </Form.Item>
          </Col>
        </Row>

        <Space style={{ width: '100%' }} size="large">
          <Form.Item
            label={t(
              'strategy.templates.scheduleLaunch.form.stopLossOffset',
              '止损距离（价格）',
            )}
            name="stopLossPriceOffset"
            style={{ flex: 1 }}
          >
            <InputNumber
              style={{ width: '100%' }}
              min={0}
              step={0.0001}
              placeholder="0.0020"
            />
          </Form.Item>
          <Form.Item
            label={t(
              'strategy.templates.scheduleLaunch.form.takeProfitOffset',
              '止盈距离（价格）',
            )}
            name="takeProfitPriceOffset"
            style={{ flex: 1 }}
          >
            <InputNumber
              style={{ width: '100%' }}
              min={0}
              step={0.0001}
              placeholder="0.0040"
            />
          </Form.Item>
        </Space>

        {isCreate && (
          <Form.Item
            label={t(
              'strategy.templates.scheduleLaunch.form.enableAfterCreate',
              '创建后立即启用',
            )}
            name="isActive"
            valuePropName="checked"
          >
            <Switch />
          </Form.Item>
        )}

        <Collapse
          items={[
            {
              key: 'advanced',
              label: t('strategy.schedules.editModal.advanced.title'),
              children: (
                <Form.Item
                  label={t('strategy.schedules.editModal.advanced.parametersJson')}
                  name="parametersJson"
                  extra={t('strategy.schedules.editModal.advanced.parametersJsonExtra')}
                >
                  <Input.TextArea
                    rows={7}
                    placeholder={`{\n  "fast": 10,\n  "slow": 20\n}`}
                  />
                </Form.Item>
              ),
            },
          ]}
        />
      </Form>
    </Modal>
  );
}

export type { ScheduleFormValues, ScheduleType };
