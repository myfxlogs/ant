import React from 'react';
import { Card, Row, Col, Statistic, Typography, Empty, Tag, Space, Button } from 'antd';
import { CheckOutlined, CloseOutlined, EyeOutlined } from '@ant-design/icons';
import type { TFunction } from 'i18next';
import type { OptimizationTaskInfo, BacktestSnapshot } from '@/gen/ant/v1/marketplace_service_pb';

const { Text } = Typography;

export const STATUS_COLORS: Record<string, string> = {
  pending: 'default',
  generating: 'processing',
  completed: 'gold',
  rejected: 'red',
  published: 'green',
};

export function BacktestCompare({ original, optimized }: { original?: BacktestSnapshot; optimized?: BacktestSnapshot }) {
  const { t } = useTranslation();
  if (!original && !optimized) return <Empty description={t('marketplace.optimization.noBacktest')} />;
  const metrics: [string, string | undefined, string | undefined][] = [
    [t('marketplace.optimization.totalReturn'), original?.totalReturn, optimized?.totalReturn],
    [t('marketplace.optimization.annualReturn'), original?.annualReturn, optimized?.annualReturn],
    [t('marketplace.optimization.maxDrawdown'), original?.maxDrawdown, optimized?.maxDrawdown],
    [t('marketplace.optimization.sharpe'), original?.sharpeRatio, optimized?.sharpeRatio],
    [t('marketplace.optimization.winRate'), original?.winRate, optimized?.winRate],
    [t('marketplace.optimization.totalTrades'), original?.totalTrades?.toString(), optimized?.totalTrades?.toString()],
  ];
  return (
    <Row gutter={[8, 8]}>
      {metrics.map(([label, orig, opt]) => {
        const improved = orig && opt && Number(opt) > Number(orig);
        return (
          <Col xs={24} sm={12} md={8} key={label}>
            <Card size="small" style={{ borderRadius: 8 }}>
              <Statistic
                title={label}
                value={opt || '-'}
                prefix={improved ? <CheckOutlined style={{ color: '#52c41a' }} /> : undefined}
              />
              <Text type="secondary" style={{ fontSize: 12 }}>
                {t('marketplace.optimization.original')}: {orig || '-'}
              </Text>
            </Card>
          </Col>
        );
      })}
    </Row>
  );
}

export function buildTaskColumns(
  t: TFunction,
  handlePreview: (taskId: string) => void,
  handlePublish: (taskId: string) => void,
  handleReject: (taskId: string) => void,
  actionLoading: string | null,
) {
  return [
    { title: t('marketplace.optimization.strategyId'), dataIndex: 'strategyId', key: 'sid', width: 120, render: (v: string) => <Text copyable style={{ fontSize: 12 }}>{v?.slice(0, 8)}...</Text> },
    { title: t('marketplace.optimization.status'), dataIndex: 'status', key: 'status', width: 100, render: (s: string) => <Tag color={STATUS_COLORS[s] || 'default'}>{s}</Tag> },
    { title: t('marketplace.optimization.trigger'), dataIndex: 'triggerReason', key: 'trigger', width: 120, ellipsis: true },
    { title: t('marketplace.optimization.changeSummary'), dataIndex: 'changeSummary', key: 'summary', ellipsis: true },
    {
      title: t('marketplace.optimization.actions'), key: 'actions', width: 240,
      render: (_: unknown, row: OptimizationTaskInfo) => (
        <Space>
          {row.status === 'completed' && (
            <>
              <Button size="small" icon={<EyeOutlined />} onClick={() => handlePreview(row.id)}>
                {t('marketplace.optimization.preview')}
              </Button>
              <Button size="small" type="primary" icon={<CheckOutlined />}
                loading={actionLoading === row.id}
                onClick={() => handlePublish(row.id)}>
                {t('marketplace.optimization.publish')}
              </Button>
              <Button size="small" danger icon={<CloseOutlined />}
                loading={actionLoading === row.id}
                onClick={() => handleReject(row.id)}>
                {t('marketplace.optimization.reject')}
              </Button>
            </>
          )}
          {row.status === 'published' && (
            <Tag color="green">{t('marketplace.optimization.versionId')}: {row.publishedVersionId?.slice(0, 8)}</Tag>
          )}
        </Space>
      ),
    },
  ];
}
