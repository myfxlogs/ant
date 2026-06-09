import { useTranslation } from 'react-i18next';
import { Card, Form, InputNumber, Button, Switch, Statistic, Row, Col, Typography, Spin } from 'antd';
import { ThunderboltOutlined, RiseOutlined, ReloadOutlined } from '@ant-design/icons';
import { useAutoTradingSettings } from './useAutoTradingSettings';
import TradingLogsTable from './TradingLogsTable';

const { Title } = Typography;

export default function AutoTradingSettingsPage() {
  const { t } = useTranslation();
  const {
    settings, status, logs,
    loading, saving, error,
    handleToggle, handleSave, handleRefresh,
  } = useAutoTradingSettings();

  const enabled = settings?.autoTradeEnabled ?? false;

  return (
    <div style={{ maxWidth: 900, margin: '0 auto', padding: '24px 16px' }}>
      {/* ── Header ── */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 20 }}>
        <Title level={3} style={{ margin: 0 }}>
          <ThunderboltOutlined style={{ marginRight: 8 }} />
          {t('autoTrading.title')}
        </Title>
        <Button icon={<ReloadOutlined />} onClick={handleRefresh} loading={loading} size="small">
          {t('common.refresh')}
        </Button>
      </div>

      {/* ── Error ── */}
      {error && (
        <div style={{ padding: '16px 0', color: '#ff4d4f', textAlign: 'center' }}>
          {error} <Button type="link" onClick={handleRefresh}>{t('common.retry')}</Button>
        </div>
      )}

      <Spin spinning={loading && !settings}>
        {/* ── Status Bar ── */}
        <Card size="small" style={{ marginBottom: 16, borderRadius: 8 }}>
          <Row align="middle" justify="space-between">
            <Col>
              <Switch
                checked={enabled}
                onChange={handleToggle}
                loading={loading}
                checkedChildren={t('autoTrading.status.enabled')}
                unCheckedChildren={t('autoTrading.status.disabled')}
              />
            </Col>
            <Col>
              <Row gutter={32}>
                <Col>
                  <Statistic
                    title={t('autoTrading.status.activeStrategies')}
                    value={status?.activeStrategies ?? 0}
                    valueStyle={{ fontSize: 18, fontWeight: 600 }}
                  />
                </Col>
                <Col>
                  <Statistic
                    title={t('autoTrading.status.todayExecutions')}
                    value={status?.todayExecutions ?? 0}
                    valueStyle={{ fontSize: 18, fontWeight: 600 }}
                  />
                </Col>
                <Col>
                  <Statistic
                    title={t('autoTrading.status.todayProfit')}
                    value={status?.todayProfit ?? 0}
                    precision={2}
                    prefix={<RiseOutlined />}
                    valueStyle={{
                      fontSize: 18,
                      fontWeight: 600,
                      color: (status?.todayProfit ?? 0) >= 0 ? '#26a69a' : '#ef5350',
                    }}
                  />
                </Col>
              </Row>
            </Col>
          </Row>
        </Card>

        {/* ── Global Settings ── */}
        <Card title={t('autoTrading.settings.title')} style={{ marginBottom: 16, borderRadius: 8 }}>
          <Form
            layout="vertical"
            size="small"
            initialValues={{
              maxRiskPercent: settings?.maxRiskPercent ?? 2,
              maxPositions: settings?.maxPositions ?? 5,
              maxLotSize: settings?.maxLotSize ?? 100,
              maxDailyLoss: settings?.maxDailyLoss ?? 0,
              maxDrawdownPercent: settings?.maxDrawdownPercent ?? 10,
            }}
            onFinish={handleSave}
          >
            <Row gutter={[16, 0]}>
              <Col xs={24} sm={12} md={8}>
                <Form.Item name="maxRiskPercent" label={t('autoTrading.settings.maxRiskPercent')}
                  tooltip={t('autoTrading.settings.maxRiskPercentHint')}>
                  <InputNumber min={0} max={100} step={0.1} style={{ width: '100%' }}
                    addonAfter="%" />
                </Form.Item>
              </Col>
              <Col xs={24} sm={12} md={8}>
                <Form.Item name="maxPositions" label={t('autoTrading.settings.maxPositions')}
                  tooltip={t('autoTrading.settings.maxPositionsHint')}>
                  <InputNumber min={1} max={100} style={{ width: '100%' }} />
                </Form.Item>
              </Col>
              <Col xs={24} sm={12} md={8}>
                <Form.Item name="maxLotSize" label={t('autoTrading.settings.maxLotSize')}
                  tooltip={t('autoTrading.settings.maxLotSizeHint')}>
                  <InputNumber min={0.01} max={1000} step={0.01} style={{ width: '100%' }} />
                </Form.Item>
              </Col>
              <Col xs={24} sm={12} md={8}>
                <Form.Item name="maxDailyLoss" label={t('autoTrading.settings.maxDailyLoss')}
                  tooltip={t('autoTrading.settings.maxDailyLossHint')}>
                  <InputNumber min={0} step={100} style={{ width: '100%' }} prefix="$" />
                </Form.Item>
              </Col>
              <Col xs={24} sm={12} md={8}>
                <Form.Item name="maxDrawdownPercent" label={t('autoTrading.settings.maxDrawdownPercent')}
                  tooltip={t('autoTrading.settings.maxDrawdownPercentHint')}>
                  <InputNumber min={0} max={100} step={0.1} style={{ width: '100%' }}
                    addonAfter="%" />
                </Form.Item>
              </Col>
            </Row>
            <Button type="primary" htmlType="submit" loading={saving}>
              {t('common.save')}
            </Button>
          </Form>
        </Card>

        {/* ── Trading Logs ── */}
        <Card title={t('autoTrading.logs.title')} style={{ borderRadius: 8 }}>
          <TradingLogsTable logs={logs} loading={loading} />
        </Card>
      </Spin>
    </div>
  );
}
