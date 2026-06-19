import { Col, Divider, Form, InputNumber, Row, Space, Tooltip } from 'antd';
import { useTranslation } from 'react-i18next'
import { SCHEDULE_LAUNCH_FORM_DEFAULT_VOLUME_KEY, SCHEDULE_LAUNCH_FORM_DEFAULT_VOLUME_TIP_KEY, SCHEDULE_LAUNCH_FORM_MAX_DRAWDOWN_PCT_KEY, SCHEDULE_LAUNCH_FORM_MAX_DRAWDOWN_PCT_TIP_KEY, SCHEDULE_LAUNCH_FORM_MAX_POSITIONS_KEY, SCHEDULE_LAUNCH_FORM_MAX_POSITIONS_TIP_KEY, SCHEDULE_LAUNCH_FORM_RISK_SECTION_KEY, SCHEDULE_LAUNCH_FORM_STOP_LOSS_OFFSET_KEY, SCHEDULE_LAUNCH_FORM_TAKE_PROFIT_OFFSET_KEY } from '@/gen/ant/v1/i18n/strategy_templates_keys';

;

export default function EditScheduleRiskFields() {
  const { t } = useTranslation();
  return (
    <>
      <Divider orientation="left" plain>
        {t(SCHEDULE_LAUNCH_FORM_RISK_SECTION_KEY, '风控参数（可选）')}
      </Divider>
      <Row gutter={12}>
        <Col span={8}>
          <Form.Item
            label={
              <Tooltip title={t(SCHEDULE_LAUNCH_FORM_DEFAULT_VOLUME_TIP_KEY, '策略信号里 volume=0 时默认下单量。手数单位。')}>
                <span>{t(SCHEDULE_LAUNCH_FORM_DEFAULT_TRADING_VOLUME_KEY, '默认手数')}</span>
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
              <Tooltip title={t(SCHEDULE_LAUNCH_FORM_MAX_POSITIONS_TIP_KEY, '同一品种上允许同时持有的最多持仓数；达到后本次信号跳过。')}>
                <span>{t(SCHEDULE_LAUNCH_FORM_MAX_TRADING_POSITIONS_KEY, '最大持仓数')}</span>
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
              <Tooltip title={t(SCHEDULE_LAUNCH_FORM_MAX_DRAWDOWN_PCT_TIP_KEY, '自峰值权益的最大回撤比例，0.2 = 20%；触发后调度自动停用。')}>
                <span>{t(SCHEDULE_LAUNCH_FORM_MAX_DRAWDOWN_PCT_KEY, '最大回撤比例（0~1）')}</span>
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
          label={t(SCHEDULE_LAUNCH_FORM_STOP_LOSS_OFFSET_KEY, '止损距离（价格）')}
          name="stopLossPriceOffset"
          style={{ flex: 1 }}
        >
          <InputNumber style={{ width: '100%' }} min={0} step={0.0001} placeholder="0.0020" />
        </Form.Item>
        <Form.Item
          label={t(SCHEDULE_LAUNCH_FORM_TAKE_PROFIT_OFFSET_KEY, '止盈距离（价格）')}
          name="takeProfitPriceOffset"
          style={{ flex: 1 }}
        >
          <InputNumber style={{ width: '100%' }} min={0} step={0.0001} placeholder="0.0040" />
        </Form.Item>
      </Space>
    </>
  );
}
