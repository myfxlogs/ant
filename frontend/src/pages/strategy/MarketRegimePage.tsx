import { useState } from 'react';
import { Alert, Button, Card, Descriptions, Form, Input, InputNumber, Select, Space, Tag, Typography } from 'antd';
import { marketRegimeApi, type MarketRegime } from '@/client/marketRegime';
import { showError, showSuccess } from '@/utils/message';

const { Text, Title } = Typography;

export default function MarketRegimePage() {
  const [form] = Form.useForm();
  const [result, setResult] = useState<MarketRegime | null>(null);
  const [loading, setLoading] = useState(false);

  const detect = async (values: { accountId: string; symbol: string; timeframe: string; count: number }) => {
    setLoading(true);
    try {
      const row = await marketRegimeApi.detect(values);
      setResult(row);
      showSuccess('市场状态检测完成');
    } catch {
      showError('市场状态检测失败');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="space-y-4">
      <div>
        <Title level={3}>市场状态识别</Title>
        <Text type="secondary">后端基于 K 线计算趋势、波动与效率特征，前端只展示结果。</Text>
      </div>

      <Alert type="info" showIcon message="当前为规则版检测模型 rule-v1，K 线权威来源仍为后端 Market/Kline 服务。" />

      <Card title="检测参数">
        <Form form={form} layout="vertical" onFinish={detect} initialValues={{ timeframe: 'M15', count: 120 }}>
          <Form.Item name="accountId" label="账户 ID" rules={[{ required: true, message: '请输入账户 ID' }]}>
            <Input placeholder="MT 账户 UUID" />
          </Form.Item>
          <Space size="large" wrap>
            <Form.Item name="symbol" label="交易品种" rules={[{ required: true, message: '请输入交易品种' }]}>
              <Input style={{ width: 180 }} placeholder="EURUSD" />
            </Form.Item>
            <Form.Item name="timeframe" label="周期">
              <Select style={{ width: 140 }} options={['M1', 'M5', 'M15', 'M30', 'H1', 'H4', 'D1'].map(v => ({ value: v, label: v }))} />
            </Form.Item>
            <Form.Item name="count" label="K 线数量">
              <InputNumber min={20} max={500} />
            </Form.Item>
          </Space>
          <Form.Item>
            <Button type="primary" htmlType="submit" loading={loading}>开始检测</Button>
          </Form.Item>
        </Form>
      </Card>

      {result && (
        <Card title="检测结果">
          <Descriptions bordered column={1} size="small">
            <Descriptions.Item label="状态"><Tag color="blue">{result.regime}</Tag></Descriptions.Item>
            <Descriptions.Item label="置信度">{(result.confidence * 100).toFixed(1)}%</Descriptions.Item>
            <Descriptions.Item label="模型版本">{result.modelVersion}</Descriptions.Item>
            <Descriptions.Item label="策略族">{result.strategyFamilies.map(item => <Tag key={item}>{item}</Tag>)}</Descriptions.Item>
            <Descriptions.Item label="特征"><Text code>{JSON.stringify(result.features)}</Text></Descriptions.Item>
            <Descriptions.Item label="记录 ID"><Text copyable>{result.id}</Text></Descriptions.Item>
          </Descriptions>
        </Card>
      )}
    </div>
  );
}
