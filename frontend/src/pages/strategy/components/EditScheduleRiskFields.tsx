import { Col, Divider, Form, InputNumber, Row, Space, Tooltip } from 'antd';
import { useTranslation } from 'react-i18next'
import { SCHEDULE_LAUNCH_FORM_DEFAULT_VOLUME_KEY, SCHEDULE_LAUNCH_FORM_DEFAULT_VOLUME_TIP_KEY, SCHEDULE_LAUNCH_FORM_MAX_DRAWDOWN_PCT_KEY, SCHEDULE_LAUNCH_FORM_MAX_DRAWDOWN_PCT_TIP_KEY, SCHEDULE_LAUNCH_FORM_MAX_POSITIONS_KEY, SCHEDULE_LAUNCH_FORM_MAX_POSITIONS_TIP_KEY, SCHEDULE_LAUNCH_FORM_RISK_SECTION_KEY, SCHEDULE_LAUNCH_FORM_STOP_LOSS_OFFSET_KEY, SCHEDULE_LAUNCH_FORM_TAKE_PROFIT_OFFSET_KEY } from '@/gen/ant/v1/i18n/strategy_templates_keys';

;

export default function EditScheduleRiskFields() {
  const { t } = useTranslation();
  return (
    <>
      <Divider titlePlacement="left" plain>
        {t(SCHEDULE_LAUNCH_FORM_RISK_SECTION_KEY, { defaultValue: 'Risk Parameters (Optional)' })}
      </Divider>
      <Row gutter={12}>
        <Col span={8}>
          <Form.Item
            label={
              <Tooltip title={t(SCHEDULE_LAUNCH_FORM_DEFAULT_VOLUME_TIP_KEY, { defaultValue: 'Default order volume when strategy signal has volume=0. In lots.' })}>
                <span>{t(SCHEDULE_LAUNCH_FORM_DEFAULT_VOLUME_KEY, { defaultValue: 'Default Volume' })}</span>
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
              <Tooltip title={t(SCHEDULE_LAUNCH_FORM_MAX_POSITIONS_TIP_KEY, { defaultValue: 'Max simultaneous positions per symbol; signal is skipped when reached.' })}>
                <span>{t(SCHEDULE_LAUNCH_FORM_MAX_POSITIONS_KEY, { defaultValue: 'Max Positions' })}</span>
              </Tooltip>
            }
            name="maxPositions"
          >
            <InputNumber style={{ width: '100%' }} min={1} step={1} placeholder={t('strategy.schedule.maxPositionsPlaceholder', { defaultValue: 'Unlimited' })} />
          </Form.Item>
        </Col>
        <Col span={8}>
          <Form.Item
            label={
              <Tooltip title={t(SCHEDULE_LAUNCH_FORM_MAX_DRAWDOWN_PCT_TIP_KEY, { defaultValue: 'Max drawdown ratio from peak equity, 0.2 = 20%; auto-disables schedule when triggered.' })}>
                <span>{t(SCHEDULE_LAUNCH_FORM_MAX_DRAWDOWN_PCT_KEY, { defaultValue: 'Max Drawdown (0~1)' })}</span>
              </Tooltip>
            }
            name="maxDrawdownPct"
            rules={[{ type: 'number', min: 0, max: 1 }]}
          >
            <InputNumber style={{ width: '100%' }} min={0} max={1} step={0.01} placeholder="0.2" />
          </Form.Item>
        </Col>
      </Row>
      <Space style={{ width: '100%' }} size="large">
        <Form.Item
          label={t(SCHEDULE_LAUNCH_FORM_STOP_LOSS_OFFSET_KEY, { defaultValue: 'Stop Loss Offset (price)' })}
          name="stopLossPriceOffset"
          style={{ flex: 1 }}
        >
          <InputNumber style={{ width: '100%' }} min={0} step={0.0001} placeholder="0.0020" />
        </Form.Item>
        <Form.Item
          label={t(SCHEDULE_LAUNCH_FORM_TAKE_PROFIT_OFFSET_KEY, { defaultValue: 'Take Profit Offset (price)' })}
          name="takeProfitPriceOffset"
          style={{ flex: 1 }}
        >
          <InputNumber style={{ width: '100%' }} min={0} step={0.0001} placeholder="0.0040" />
        </Form.Item>
      </Space>
    </>
  );
}
