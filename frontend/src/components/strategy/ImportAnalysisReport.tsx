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
import type { AnalyzeImportCodeResponse, BlindSpot, ParamField, ParamGroupInfo } from '@/gen/ant/v1/strategy_runtime_pb';

const { Text, Title } = Typography;

interface ParsedAnalysis {
  strategyName: string;
  mqlVersion: string;
  coverageScore: number;
  totalBlocks: number;
  recognizedBlocks: number;
  executionKind: string;
  entryRulesCount: number;
  exitRulesCount: number;
  sizingKind: string;
  riskChecksCount: number;
  indicatorNames: string[];
  params: ParamField[];
  groups: ParamGroupInfo[];
  blindSpots: BlindSpot[];
}

function parseAnalysis(resp: AnalyzeImportCodeResponse): ParsedAnalysis {
  return {
    strategyName: resp.strategyName,
    mqlVersion: resp.mqlVersion,
    coverageScore: resp.coverageScore,
    totalBlocks: resp.totalBlocks,
    recognizedBlocks: resp.recognizedBlocks,
    executionKind: resp.executionKind,
    entryRulesCount: resp.entryRulesCount,
    exitRulesCount: resp.exitRulesCount,
    sizingKind: resp.sizingKind,
    riskChecksCount: resp.riskChecksCount,
    indicatorNames: resp.indicatorNames || [],
    params: resp.params || [],
    groups: resp.groups || [],
    blindSpots: resp.blindSpots || [],
  };
}

interface Props {
  analysis: AnalyzeImportCodeResponse | null;
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

  const a = parseAnalysis(analysis);
  const coverage = Math.round(a.coverageScore * 100);
  const criticalBlindSpots = a.blindSpots.filter(b => b.severity === '致命');
  const warningBlindSpots = a.blindSpots.filter(b => b.severity === '警告');
  const infoBlindSpots = a.blindSpots.filter(b => b.severity === '信息');

  // Triage: backend marks GUI noise as severity=信息, real gaps as 警告/致命.
  const guiNoiseSpots = a.blindSpots.filter(b => b.severity === '信息');
  const realBlindSpots = a.blindSpots.filter(b => b.severity !== '信息');
  const isPureGuiNoise = realBlindSpots.length === 0 && guiNoiseSpots.length > 0;
  const hasRealGaps = realBlindSpots.length > 0;
  const triageLevel = hasRealGaps ? (coverage >= 70 ? 'pass' : coverage >= 40 ? 'warn' : 'block') : 'pass';

  return (
    <div style={{ padding: '12px 0' }}>
      {/* ── Triage Verdict ── */}
      {isPureGuiNoise && (
        <Alert type="success" showIcon icon={<CheckCircleOutlined />}
          message="交易逻辑已完整识别"
          description="以下盲区均为图表显示/按钮功能，服务端自动运行时会跳过，不影响交易结果。可以确认导入。"
          style={{ marginBottom: 12 }} />
      )}
      {hasRealGaps && triageLevel === 'block' && (
        <Alert type="error" showIcon icon={<CloseCircleOutlined />}
          message="无法自动导入"
          description={`核心交易逻辑仅识别 ${coverage}%，未识别部分包含入场/出场/风控等关键逻辑。建议使用 AI 翻译重试，或简化 EA 后重新提交。`}
          style={{ marginBottom: 12 }} />
      )}
      {hasRealGaps && triageLevel === 'warn' && (
        <Alert type="warning" showIcon icon={<WarningOutlined />}
          message="交易逻辑覆盖不完整"
          description={`已识别 ${coverage}% 代码逻辑（不含 GUI 显示功能）。未识别部分可能影响交易行为，建议 AI 翻译补充或人工审查。`}
          style={{ marginBottom: 12 }} />
      )}
      {!hasRealGaps && !isPureGuiNoise && triageLevel === 'pass' && (
        <Alert type="success" showIcon icon={<CheckCircleOutlined />}
          message="导入完整度良好"
          description="策略已识别主要逻辑，可确认导入。建议检查参数列表后使用。"
          style={{ marginBottom: 12 }} />
      )}

      {/* ── Coverage ── */}
      <Card size="small" style={{ marginBottom: 12 }}>
        <Space direction="vertical" style={{ width: '100%' }} size="small">
          <Text strong>导入完整度</Text>
          <Progress
            percent={coverage}
            status={coverage >= 70 || isPureGuiNoise ? 'success' : coverage >= 40 ? 'active' : 'exception'}
            format={(p) => isPureGuiNoise ? `交易逻辑已完整识别` : `已识别 ${p}% 策略逻辑`}
          />
          <Text type="secondary">
            共 {a.totalBlocks} 个逻辑块，{a.recognizedBlocks} 个已识别
          </Text>
        </Space>
      </Card>

      {/* ── Summary ── */}
      <Card size="small" style={{ marginBottom: 12 }}>
        <Space wrap size="small">
          <Tag icon={<ThunderboltOutlined />} color="blue">
            {a.entryRulesCount} 入场规则
          </Tag>
          <Tag icon={<SwapOutlined />} color="purple">
            {a.exitRulesCount} 出场规则
          </Tag>
          <Tag icon={<DollarOutlined />} color="green">
            {sizingLabel(a.sizingKind)}
          </Tag>
          <Tag icon={<SettingOutlined />} color="orange">
            {executionLabel(a.executionKind)}
          </Tag>
          {a.riskChecksCount > 0 && (
            <Tag color="red">{a.riskChecksCount} 风控检查</Tag>
          )}
          {a.indicatorNames.length > 0 && (
            <Tag color="cyan">{a.indicatorNames.join(', ')}</Tag>
          )}
        </Space>
      </Card>

      {/* ── Parameters ── */}
      {a.params.length > 0 && (
        <Card size="small" title={`策略参数 (${a.params.length})`} style={{ marginBottom: 12 }}>
          {a.groups.map((g: ParamGroupInfo) => {
            const groupParams = a.params.filter((p: ParamField) => p.group === g.name);
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

      {/* ── Blind Spots (real gaps only, not GUI noise) ── */}
      {realBlindSpots.length > 0 && (
        <Card
          size="small"
          title={
            <Space>
              <WarningOutlined />
              <span>以下逻辑需要确认 ({realBlindSpots.length})</span>
            </Space>
          }
          style={{ marginBottom: 12, borderColor: criticalBlindSpots.length > 0 ? '#ff4d4f' : '#faad14' }}
        >
          {criticalBlindSpots.filter(b => b.severity !== '信息').map((b: BlindSpot) => (
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
          {warningBlindSpots.filter(b => b.severity !== '信息').map((b: BlindSpot) => (
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
      {a.blindSpots.length === 0 && (
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
