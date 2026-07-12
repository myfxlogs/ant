import { Card, Row, Col, Statistic, Segmented, Table, Empty } from 'antd';
import { useTranslation } from 'react-i18next';

function toNumber(value: unknown): number {
  if (typeof value === 'bigint') return Number(value);
  if (typeof value === 'number') return value;
  return Number(value || 0);
}

interface RiskWindow {
  window: string; hours?: number;
  riskValidateTotal?: number; riskValidatePass?: number;
  riskValidateReject?: number; riskValidateError?: number;
  orderSendSuccess?: number; orderSendFailed?: number;
  orderCloseSuccess?: number; orderCloseFailed?: number;
  topRejectRiskCodes?: Array<{ riskCode: string; count: number }>;
}

interface Props {
  metrics: Record<string, any> | null;
  selectedWindow: string;
  onWindowChange: (v: string) => void;
}

export function DashboardRiskMetrics({ metrics, selectedWindow, onWindowChange }: Props) {
  const { t } = useTranslation();
  const riskWindows: RiskWindow[] = ((metrics?.app?.riskWindows as RiskWindow[]) || []).map((item) => ({
    ...item, window: item?.window || `${item?.hours || 0}h`,
  }));
  const activeWindow = riskWindows.find((item) => item.window === selectedWindow)
    || riskWindows.find((item) => item.window === '24h')
    || riskWindows[0] || null;
  const topRejectCodes = activeWindow?.topRejectRiskCodes || [];

  return (
    <>
      <Card title={t('admin.dashboard.riskMetrics', { defaultValue: 'Risk Metrics' })}>
        <Row gutter={[16, 16]}>
          <Col xs={12} sm={8} lg={6}><Statistic title={t('admin.dashboard.validateTotal', { defaultValue: 'Validate Total' })} value={toNumber(metrics?.app?.riskValidateTotal)} /></Col>
          <Col xs={12} sm={8} lg={6}><Statistic title={t('admin.dashboard.validatePass', { defaultValue: 'Validate Pass' })} value={toNumber(metrics?.app?.riskValidatePass)} valueStyle={{ color: '#52c41a' }} /></Col>
          <Col xs={12} sm={8} lg={6}><Statistic title={t('admin.dashboard.validateReject', { defaultValue: 'Validate Reject' })} value={toNumber(metrics?.app?.riskValidateReject)} valueStyle={{ color: '#fa8c16' }} /></Col>
          <Col xs={12} sm={8} lg={6}><Statistic title={t('admin.dashboard.validateError', { defaultValue: 'Validate Error' })} value={toNumber(metrics?.app?.riskValidateError)} valueStyle={{ color: '#ff4d4f' }} /></Col>
          <Col xs={12} sm={8} lg={6}><Statistic title={t('admin.dashboard.orderSendSuccess', { defaultValue: 'Order Send Success' })} value={toNumber(metrics?.app?.orderSendSuccess)} /></Col>
          <Col xs={12} sm={8} lg={6}><Statistic title={t('admin.dashboard.orderSendFailed', { defaultValue: 'Order Send Failed' })} value={toNumber(metrics?.app?.orderSendFailed)} valueStyle={{ color: '#ff4d4f' }} /></Col>
          <Col xs={12} sm={8} lg={6}><Statistic title={t('admin.dashboard.orderCloseSuccess', { defaultValue: 'Order Close Success' })} value={toNumber(metrics?.app?.orderCloseSuccess)} /></Col>
          <Col xs={12} sm={8} lg={6}><Statistic title={t('admin.dashboard.orderCloseFailed', { defaultValue: 'Order Close Failed' })} value={toNumber(metrics?.app?.orderCloseFailed)} valueStyle={{ color: '#ff4d4f' }} /></Col>
        </Row>
      </Card>

      <Card title={t('admin.dashboard.windowMetrics', { window: activeWindow?.window || '24h', defaultValue: `Window Metrics (${activeWindow?.window || '24h'})` })}
        extra={<Segmented value={selectedWindow} onChange={(v) => onWindowChange(String(v))} options={['1h', '24h', '72h']} />}>
        {activeWindow ? (
          <Row gutter={[16, 16]}>
            <Col xs={12} sm={8} lg={6}><Statistic title={t('admin.dashboard.validateTotal', { defaultValue: 'Validate Total' }) + ` (${activeWindow.window})`} value={toNumber(activeWindow.riskValidateTotal)} /></Col>
            <Col xs={12} sm={8} lg={6}><Statistic title={t('admin.dashboard.validatePass', { defaultValue: 'Validate Pass' }) + ` (${activeWindow.window})`} value={toNumber(activeWindow.riskValidatePass)} valueStyle={{ color: '#52c41a' }} /></Col>
            <Col xs={12} sm={8} lg={6}><Statistic title={t('admin.dashboard.validateReject', { defaultValue: 'Validate Reject' }) + ` (${activeWindow.window})`} value={toNumber(activeWindow.riskValidateReject)} valueStyle={{ color: '#fa8c16' }} /></Col>
            <Col xs={12} sm={8} lg={6}><Statistic title={t('admin.dashboard.validateError', { defaultValue: 'Validate Error' }) + ` (${activeWindow.window})`} value={toNumber(activeWindow.riskValidateError)} valueStyle={{ color: '#ff4d4f' }} /></Col>
            <Col xs={12} sm={8} lg={6}><Statistic title={t('admin.dashboard.orderSendSuccess', { defaultValue: 'Order Send Success' }) + ` (${activeWindow.window})`} value={toNumber(activeWindow.orderSendSuccess)} /></Col>
            <Col xs={12} sm={8} lg={6}><Statistic title={t('admin.dashboard.orderSendFailed', { defaultValue: 'Order Send Failed' }) + ` (${activeWindow.window})`} value={toNumber(activeWindow.orderSendFailed)} valueStyle={{ color: '#ff4d4f' }} /></Col>
            <Col xs={12} sm={8} lg={6}><Statistic title={t('admin.dashboard.orderCloseSuccess', { defaultValue: 'Order Close Success' }) + ` (${activeWindow.window})`} value={toNumber(activeWindow.orderCloseSuccess)} /></Col>
            <Col xs={12} sm={8} lg={6}><Statistic title={t('admin.dashboard.orderCloseFailed', { defaultValue: 'Order Close Failed' }) + ` (${activeWindow.window})`} value={toNumber(activeWindow.orderCloseFailed)} valueStyle={{ color: '#ff4d4f' }} /></Col>
            <Col span={24}><Table scroll={{ x: "max-content" }} size="small" pagination={false} rowKey={(row) => row.riskCode}
              dataSource={topRejectCodes}
              locale={{ emptyText: <Empty description={t('common.noData', { defaultValue: 'No data' })} image={Empty.PRESENTED_IMAGE_SIMPLE} /> }}
              columns={[
                { title: t('admin.dashboard.rejectRiskCodes', { window: activeWindow.window, defaultValue: `Reject Risk Codes (${activeWindow.window})` }), dataIndex: 'riskCode', key: 'riskCode' },
                { title: t('admin.dashboard.rejectCount', { defaultValue: 'Reject Count' }), dataIndex: 'count', key: 'count', width: 160, render: (v: unknown) => toNumber(v) },
              ]} /></Col>
          </Row>
        ) : <Empty description={t('common.noData', { defaultValue: 'No data' })} image={Empty.PRESENTED_IMAGE_SIMPLE} />}
      </Card>
    </>
  );
}
