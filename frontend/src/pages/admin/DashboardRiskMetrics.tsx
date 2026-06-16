import { Card, Row, Col, Statistic, Segmented, Table, Empty } from 'antd';

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
  const riskWindows: RiskWindow[] = ((metrics?.app?.riskWindows as RiskWindow[]) || []).map((item) => ({
    ...item, window: item?.window || `${item?.hours || 0}h`,
  }));
  const activeWindow = riskWindows.find((item) => item.window === selectedWindow)
    || riskWindows.find((item) => item.window === '24h')
    || riskWindows[0] || null;
  const topRejectCodes = activeWindow?.topRejectRiskCodes || [];

  return (
    <>
      <Card title="风控指标">
        <Row gutter={[16, 16]}>
          <Col xs={12} sm={8} lg={6}><Statistic title="校验总数" value={toNumber(metrics?.app?.riskValidateTotal)} /></Col>
          <Col xs={12} sm={8} lg={6}><Statistic title="校验通过" value={toNumber(metrics?.app?.riskValidatePass)} valueStyle={{ color: '#52c41a' }} /></Col>
          <Col xs={12} sm={8} lg={6}><Statistic title="校验拒绝" value={toNumber(metrics?.app?.riskValidateReject)} valueStyle={{ color: '#fa8c16' }} /></Col>
          <Col xs={12} sm={8} lg={6}><Statistic title="校验错误" value={toNumber(metrics?.app?.riskValidateError)} valueStyle={{ color: '#ff4d4f' }} /></Col>
          <Col xs={12} sm={8} lg={6}><Statistic title="下单成功" value={toNumber(metrics?.app?.orderSendSuccess)} /></Col>
          <Col xs={12} sm={8} lg={6}><Statistic title="下单失败" value={toNumber(metrics?.app?.orderSendFailed)} valueStyle={{ color: '#ff4d4f' }} /></Col>
          <Col xs={12} sm={8} lg={6}><Statistic title="平仓成功" value={toNumber(metrics?.app?.orderCloseSuccess)} /></Col>
          <Col xs={12} sm={8} lg={6}><Statistic title="平仓失败" value={toNumber(metrics?.app?.orderCloseFailed)} valueStyle={{ color: '#ff4d4f' }} /></Col>
        </Row>
      </Card>

      <Card title={`风险窗口指标 (${activeWindow?.window || '24h'})`}
        extra={<Segmented value={selectedWindow} onChange={(v) => onWindowChange(String(v))} options={['1h', '24h', '72h']} />}>
        {activeWindow ? (
          <Row gutter={[16, 16]}>
            <Col xs={12} sm={8} lg={6}><Statistic title={`校验总数 (${activeWindow.window})`} value={toNumber(activeWindow.riskValidateTotal)} /></Col>
            <Col xs={12} sm={8} lg={6}><Statistic title={`校验通过 (${activeWindow.window})`} value={toNumber(activeWindow.riskValidatePass)} valueStyle={{ color: '#52c41a' }} /></Col>
            <Col xs={12} sm={8} lg={6}><Statistic title={`校验拒绝 (${activeWindow.window})`} value={toNumber(activeWindow.riskValidateReject)} valueStyle={{ color: '#fa8c16' }} /></Col>
            <Col xs={12} sm={8} lg={6}><Statistic title={`校验错误 (${activeWindow.window})`} value={toNumber(activeWindow.riskValidateError)} valueStyle={{ color: '#ff4d4f' }} /></Col>
            <Col xs={12} sm={8} lg={6}><Statistic title={`下单成功 (${activeWindow.window})`} value={toNumber(activeWindow.orderSendSuccess)} /></Col>
            <Col xs={12} sm={8} lg={6}><Statistic title={`下单失败 (${activeWindow.window})`} value={toNumber(activeWindow.orderSendFailed)} valueStyle={{ color: '#ff4d4f' }} /></Col>
            <Col xs={12} sm={8} lg={6}><Statistic title={`平仓成功 (${activeWindow.window})`} value={toNumber(activeWindow.orderCloseSuccess)} /></Col>
            <Col xs={12} sm={8} lg={6}><Statistic title={`平仓失败 (${activeWindow.window})`} value={toNumber(activeWindow.orderCloseFailed)} valueStyle={{ color: '#ff4d4f' }} /></Col>
            <Col span={24}><Table scroll={{ x: "max-content" }} size="small" pagination={false} rowKey={(row) => row.riskCode}
              dataSource={topRejectCodes}
              locale={{ emptyText: <Empty description="暂无拒绝数据" image={Empty.PRESENTED_IMAGE_SIMPLE} /> }}
              columns={[
                { title: `拒绝风控码 (${activeWindow.window})`, dataIndex: 'riskCode', key: 'riskCode' },
                { title: '拒绝次数', dataIndex: 'count', key: 'count', width: 160, render: (v: unknown) => toNumber(v) },
              ]} /></Col>
          </Row>
        ) : <Empty description="暂无数据" image={Empty.PRESENTED_IMAGE_SIMPLE} />}
      </Card>
    </>
  );
}
