import React from 'react';
import { Progress, Tag, Alert, Descriptions, Spin, Typography, Space, Card } from 'antd';
import {
  CheckCircleOutlined,
  WarningOutlined,
  CloseCircleOutlined,
  InfoCircleOutlined,
  ThunderboltOutlined,
  SwapOutlined,
  DollarOutlined,
  SettingOutlined,
} from '@ant-design/icons';
import type { AnalyzeCodeResponse, BlindSpotItem, ParamField, ParamGroupInfo } from '../../gen/ant/v1/strategy_import_pb';

const { Text, Title } = Typography;

interface Props {
  analysis: AnalyzeCodeResponse | null;
  loading: boolean;
}

const severityColor = (s: string): string => {
  switch (s) {
    case '致命': return '#ff4d4f';
    case '警告': return '#faad14';
    case '信息': return '#1890ff';
    default: return '#999';
  }
};

const severityIcon = (s: string) => {
  switch (s) {
    case '致命': return <CloseCircleOutlined style={{ color: '#ff4d4f' }} />;
    case '警告': return <WarningOutlined style={{ color: '#faad14' }} />;
    case '信息': return <InfoCircleOutlined style={{ color: '#1890ff' }} />;
    default: return <InfoCircleOutlined />;
  }
};

const executionLabel = (kind: string): string => {
  switch (kind) {
    case 'on_bar': return 'Bar close 事件驱动';
    case 'on_tick': return 'Tick 驱动';
    case 'on_init_grid': return '初始化网格';
    default: return kind;
  }
};

const sizingLabel = (kind: string): string => {
  switch (kind) {
    case 'fixed': return '固定手数';
    case 'martingale': return '马丁格尔';
    case 'percent_balance': return '账户百分比';
    default: return kind;
  }
};

export const ImportAnalysisReport: React.FC<Props> = ({ analysis, loading }) => {
  if (loading) {
    return (
      <div style={{ textAlign: 'center', padding: 32 }}>
        <Spin tip="正在分析策略结构..." />
      </div>
    );
  }

  if (!analysis) return null;

  const coverage = Math.round(analysis.coverageScore * 100);
  const criticalBlindSpots = analysis.blindSpots.filter(b => b.severity === '致命');
  const warningBlindSpots = analysis.blindSpots.filter(b => b.severity === '警告');
  const infoBlindSpots = analysis.blindSpots.filter(b => b.severity === '信息');

  return (
    <div style={{ padding: '12px 0' }}>
      {/* ── Coverage ── */}
      <Card size="small" style={{ marginBottom: 12 }}>
        <Space direction="vertical" style={{ width: '100%' }} size="small">
          <Text strong>导入完整度</Text>
          <Progress
            percent={coverage}
            status={coverage >= 80 ? 'success' : coverage >= 60 ? 'active' : 'exception'}
            format={(p) => `已识别 ${p}% 策略逻辑`}
          />
          <Text type="secondary">
            共 {analysis.totalBlocks} 个逻辑块，{analysis.recognizedBlocks} 个已识别
          </Text>
        </Space>
      </Card>

      {/* ── Summary ── */}
      <Card size="small" style={{ marginBottom: 12 }}>
        <Space wrap size="small">
          <Tag icon={<ThunderboltOutlined />} color="blue">
            {analysis.entryRulesCount} 入场规则
          </Tag>
          <Tag icon={<SwapOutlined />} color="purple">
            {analysis.exitRulesCount} 出场规则
          </Tag>
          <Tag icon={<DollarOutlined />} color="green">
            {sizingLabel(analysis.sizingKind)}
          </Tag>
          <Tag icon={<SettingOutlined />} color="orange">
            {executionLabel(analysis.executionKind)}
          </Tag>
          {analysis.riskChecksCount > 0 && (
            <Tag color="red">{analysis.riskChecksCount} 风控检查</Tag>
          )}
          {analysis.indicatorNames.length > 0 && (
            <Tag color="cyan">{analysis.indicatorNames.join(', ')}</Tag>
          )}
        </Space>
      </Card>

      {/* ── Parameters ── */}
      {analysis.params.length > 0 && (
        <Card size="small" title={`策略参数 (${analysis.params.length})`} style={{ marginBottom: 12 }}>
          {analysis.groups.map((g: ParamGroupInfo) => {
            const groupParams = analysis.params.filter((p: ParamField) => p.group === g.name);
            if (!groupParams.length) return null;
            return (
              <div key={g.name} style={{ marginBottom: 8 }}>
                <Text type="secondary" style={{ fontSize: 12 }}>{g.name}</Text>
                <div style={{ marginTop: 4 }}>
                  {groupParams.map((p: ParamField) => (
                    <Tag key={p.name} style={{ marginBottom: 4 }}>
                      {p.label || p.name}: {p.defaultValue}
                    </Tag>
                  ))}
                </div>
              </div>
            );
          })}
        </Card>
      )}

      {/* ── Blind Spots ── */}
      {analysis.blindSpots.length > 0 && (
        <Card
          size="small"
          title={
            <Space>
              <WarningOutlined />
              <span>以下逻辑需要确认 ({analysis.blindSpots.length})</span>
            </Space>
          }
          style={{ marginBottom: 12, borderColor: criticalBlindSpots.length > 0 ? '#ff4d4f' : '#faad14' }}
        >
          {criticalBlindSpots.map((b: BlindSpotItem) => (
            <Alert
              key={b.id || b.description}
              type="error"
              showIcon
              icon={severityIcon(b.severity)}
              message={
                <Space direction="vertical" size={0}>
                  <Text strong style={{ color: '#ff4d4f' }}>{b.category}</Text>
                  <Text>{b.description}</Text>
                  <Text type="secondary" style={{ fontSize: 12 }}>
                    位置: {b.location} | 处理: {b.handling}
                    {b.userActionRequired && ' | ⚠️ 需要您的操作'}
                  </Text>
                </Space>
              }
              style={{ marginBottom: 8 }}
            />
          ))}
          {warningBlindSpots.map((b: BlindSpotItem) => (
            <Alert
              key={b.id || b.description}
              type="warning"
              showIcon
              icon={severityIcon(b.severity)}
              message={
                <Space direction="vertical" size={0}>
                  <Text>{b.category}</Text>
                  <Text type="secondary">{b.description}</Text>
                </Space>
              }
              style={{ marginBottom: 8 }}
            />
          ))}
        </Card>
      )}

      {/* ── All Clear ── */}
      {analysis.blindSpots.length === 0 && (
        <Alert
          type="success"
          showIcon
          icon={<CheckCircleOutlined />}
          message="未发现需要确认的逻辑"
          description="所有策略逻辑已自动识别，可以确认导入。"
          style={{ marginBottom: 12 }}
        />
      )}
    </div>
  );
};
