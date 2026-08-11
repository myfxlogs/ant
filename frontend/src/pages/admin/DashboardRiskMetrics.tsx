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
  metrics: Record<string, unknown> | null;
  selectedWindow: string;
  onWindowChange: (v: string) => void;
}

export function DashboardRiskMetrics({ metrics, selectedWindow, onWindowChange }: Props) {
  const { t } = useTranslation();
  const app = (metrics?.app ?? {}) as Record<string, unknown>;
  const riskWindows: RiskWindow[] = ((app.riskWindows as RiskWindow[]) || []).map((item) => ({
    ...item, window: item?.window || `${item?.hours || 0}h`,
  }));
  const activeWindow = riskWindows.find((item) => item.window === selectedWindow)
    || riskWindows.find((item) => item.window === '24h')
    || riskWindows[0] || null;
  const topRejectCodes = activeWindow?.topRejectRiskCodes || [];

  return (
    <>
      <Card title={t('admin.dashboard.riskMetrics.title')}>
        <Row gutter={[16, 16]}>
          <Col xs={12} sm={8} lg={6}><Statistic title={t('admin.dashboard.riskMetrics.riskValidateTotal')} value={toNumber(app.riskValidateTotal)} /></Col>
          <Col xs={12} sm={8} lg={6}><Statistic title={t('admin.dashboard.riskMetrics.riskValidatePass')} value={toNumber(app.riskValidatePass)} valueStyle={{ color: '#52c41a' }} /></Col>
          <Col xs={12} sm={8} lg={6}><Statistic title={t('admin.dashboard.riskMetrics.riskValidateReject')} value={toNumber(app.riskValidateReject)} valueStyle={{ color: '#fa8c16' }} /></Col>
          <Col xs={12} sm={8} lg={6}><Statistic title={t('admin.dashboard.riskMetrics.riskValidateError')} value={toNumber(app.riskValidateError)} valueStyle={{ color: '#ff4d4f' }} /></Col>
          <Col xs={12} sm={8} lg={6}><Statistic title={t('admin.dashboard.riskMetrics.orderSendSuccess')} value={toNumber(app.orderSendSuccess)} /></Col>
          <Col xs={12} sm={8} lg={6}><Statistic title={t('admin.dashboard.riskMetrics.orderSendFailed')} value={toNumber(app.orderSendFailed)} valueStyle={{ color: '#ff4d4f' }} /></Col>
          <Col xs={12} sm={8} lg={6}><Statistic title={t('admin.dashboard.riskMetrics.orderCloseSuccess')} value={toNumber(app.orderCloseSuccess)} /></Col>
          <Col xs={12} sm={8} lg={6}><Statistic title={t('admin.dashboard.riskMetrics.orderCloseFailed')} value={toNumber(app.orderCloseFailed)} valueStyle={{ color: '#ff4d4f' }} /></Col>
        </Row>
      </Card>

      <Card title={t('admin.dashboard.riskWindow.title')}
        extra={<Segmented value={selectedWindow} onChange={(v) => onWindowChange(String(v))} options={['1h', '24h', '72h']} />}>
        {activeWindow ? (
          <Row gutter={[16, 16]}>
            <Col xs={12} sm={8} lg={6}><Statistic title={t('admin.dashboard.riskWindow.validateTotal', { window: activeWindow.window })} value={toNumber(activeWindow.riskValidateTotal)} /></Col>
            <Col xs={12} sm={8} lg={6}><Statistic title={t('admin.dashboard.riskWindow.validatePass', { window: activeWindow.window })} value={toNumber(activeWindow.riskValidatePass)} valueStyle={{ color: '#52c41a' }} /></Col>
            <Col xs={12} sm={8} lg={6}><Statistic title={t('admin.dashboard.riskWindow.validateReject', { window: activeWindow.window })} value={toNumber(activeWindow.riskValidateReject)} valueStyle={{ color: '#fa8c16' }} /></Col>
            <Col xs={12} sm={8} lg={6}><Statistic title={t('admin.dashboard.riskWindow.validateError', { window: activeWindow.window })} value={toNumber(activeWindow.riskValidateError)} valueStyle={{ color: '#ff4d4f' }} /></Col>
            <Col xs={12} sm={8} lg={6}><Statistic title={t('admin.dashboard.riskWindow.orderSendSuccess', { window: activeWindow.window })} value={toNumber(activeWindow.orderSendSuccess)} /></Col>
            <Col xs={12} sm={8} lg={6}><Statistic title={t('admin.dashboard.riskWindow.orderSendFailed', { window: activeWindow.window })} value={toNumber(activeWindow.orderSendFailed)} valueStyle={{ color: '#ff4d4f' }} /></Col>
            <Col xs={12} sm={8} lg={6}><Statistic title={t('admin.dashboard.riskWindow.orderCloseSuccess', { window: activeWindow.window })} value={toNumber(activeWindow.orderCloseSuccess)} /></Col>
            <Col xs={12} sm={8} lg={6}><Statistic title={t('admin.dashboard.riskWindow.orderCloseFailed', { window: activeWindow.window })} value={toNumber(activeWindow.orderCloseFailed)} valueStyle={{ color: '#ff4d4f' }} /></Col>
            <Col span={24}><Table scroll={{ x: "max-content" }} size="small" pagination={false} rowKey={(row) => row.riskCode}
              dataSource={topRejectCodes}
              locale={{ emptyText: <Empty description={t('common.noData')} image={Empty.PRESENTED_IMAGE_SIMPLE} /> }}
              columns={[
                { title: t('admin.dashboard.riskWindow.rejectRiskCodesHeader', { window: activeWindow.window }), dataIndex: 'riskCode', key: 'riskCode' },
                { title: t('admin.dashboard.riskWindow.rejectCount'), dataIndex: 'count', key: 'count', width: 160, render: (v: unknown) => toNumber(v) },
              ]} /></Col>
          </Row>
        ) : <Empty description={t('common.noData')} image={Empty.PRESENTED_IMAGE_SIMPLE} />}
      </Card>
    </>
  );
}
