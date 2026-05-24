import { useEffect, useState } from 'react';
import { Card, Table, Row, Col, Statistic, DatePicker } from 'antd';
import { IconTrendingUp, IconTrendingDown } from '@tabler/icons-react';
import { adminApi, type TradingSummary } from '@/client/admin';
import { getErrorMessage } from '@/utils/error';
import { showError } from '@/utils/message';

const { RangePicker } = DatePicker;

export default function TradingMonitor() {
  const [summary, setSummary] = useState<TradingSummary | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchSummary();
  }, []);

  const fetchSummary = async (startDate?: string, endDate?: string) => {
    setLoading(true);
    try {
      const result = await adminApi.getTradingSummary({
        startDate,
        endDate,
      });
      setSummary(result as TradingSummary);
    } catch (error) {
      showError(getErrorMessage(error, '加载交易统计失败'));
    } finally {
      setLoading(false);
    }
  };

  const handleDateChange = (dates: [Date | null, Date | null] | null) => {
    if (dates && dates[0] && dates[1]) {
      fetchSummary(
        dates[0].toISOString().split('T')[0],
        dates[1].toISOString().split('T')[0]
      );
    } else {
      fetchSummary();
    }
  };

  const platformColumns = [
    {
      title: '平台',
      dataIndex: 'platform',
      key: 'platform',
    },
    {
      title: '账户数',
      dataIndex: 'accounts',
      key: 'accounts',
    },
    {
      title: '订单数',
      dataIndex: 'orders',
      key: 'orders',
    },
    {
      title: '交易量',
      dataIndex: 'volume',
      key: 'volume',
      render: (value: number) => value?.toFixed(2) || '0.00',
    },
  ];

  const platformData = (summary as any)?.byPlatform
    ? Object.entries((summary as any).byPlatform).map(([platform, data]: [string, any]) => ({
        platform,
        ...data,
      }))
    : [];

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <h1 className="text-2xl font-bold" style={{ color: '#141D22' }}>交易监控</h1>
        <RangePicker onChange={(dates) => handleDateChange(dates as [Date | null, Date | null] | null)} />
      </div>

      <Row gutter={[16, 16]}>
        <Col xs={12} sm={6}>
          <Card loading={loading}>
            <Statistic
              title="总用户"
              value={(summary as any)?.overview?.totalUsers || 0}
            />
          </Card>
        </Col>
        <Col xs={12} sm={6}>
          <Card loading={loading}>
            <Statistic
              title="活跃用户"
              value={(summary as any)?.overview?.activeUsers || 0}
            />
          </Card>
        </Col>
        <Col xs={12} sm={6}>
          <Card loading={loading}>
            <Statistic
              title="总账户"
              value={(summary as any)?.overview?.totalAccounts || 0}
            />
          </Card>
        </Col>
        <Col xs={12} sm={6}>
          <Card loading={loading}>
            <Statistic
              title="已连接账户"
              value={(summary as any)?.overview?.connectedAccounts || 0}
            />
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]}>
        <Col xs={12} sm={6}>
          <Card loading={loading}>
            <Statistic
              title="总订单"
              value={(summary as any)?.trading?.totalOrders || 0}
            />
          </Card>
        </Col>
        <Col xs={12} sm={6}>
          <Card loading={loading}>
            <Statistic
              title="已平仓"
              value={(summary as any)?.trading?.closedOrders || 0}
            />
          </Card>
        </Col>
        <Col xs={12} sm={6}>
          <Card loading={loading}>
            <Statistic
              title="总交易量"
              value={(summary as any)?.trading?.totalVolume || 0}
              precision={2}
            />
          </Card>
        </Col>
        <Col xs={12} sm={6}>
          <Card loading={loading}>
            <Statistic
              title="净盈亏"
              value={(summary as any)?.trading?.netProfit || 0}
              precision={2}
              valueStyle={{ color: ((summary as any)?.trading?.netProfit || 0) >= 0 ? '#52c41a' : '#ff4d4f' }}
              prefix={((summary as any)?.trading?.netProfit || 0) >= 0 ? <IconTrendingUp size={16} /> : <IconTrendingDown size={16} />}
            />
          </Card>
        </Col>
      </Row>

      <Card title="按平台统计" loading={loading}>
        <Table
          scroll={{ x: "max-content" }}
          columns={platformColumns}
          dataSource={platformData}
          rowKey="platform"
          pagination={false}
        />
      </Card>

      <Card title="盈亏统计">
        <Row gutter={[16, 16]}>
          <Col xs={12} sm={8}>
            <Statistic
              title="总盈利"
              value={(summary as any)?.trading?.totalProfit || 0}
              precision={2}
              valueStyle={{ color: '#52c41a' }}
              prefix={<IconTrendingUp size={16} />}
            />
          </Col>
          <Col xs={12} sm={8}>
            <Statistic
              title="总亏损"
              value={Math.abs((summary as any)?.trading?.totalLoss || 0)}
              precision={2}
              valueStyle={{ color: '#ff4d4f' }}
              prefix={<IconTrendingDown size={16} />}
            />
          </Col>
          <Col xs={12} sm={8}>
            <Statistic
              title="挂单中"
              value={(summary as any)?.trading?.pendingOrders || 0}
            />
          </Col>
        </Row>
      </Card>
    </div>
  );
}
