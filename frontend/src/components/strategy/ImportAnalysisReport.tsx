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
import { useTranslation } from 'react-i18next';
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

const executionLabel = (kind: string, t: (k: string, o?: any) => string): string => {
  switch (kind) {
    case 'on_bar': return t('importAnalysis.execution.onBar', { defaultValue: 'Bar close event-driven' });
    case 'on_tick': return t('importAnalysis.execution.onTick', { defaultValue: 'Tick-driven' });
    case 'on_init_grid': return t('importAnalysis.execution.onInitGrid', { defaultValue: 'Init grid' });
    default: return kind;
  }
};

const sizingLabel = (kind: string, t: (k: string, o?: any) => string): string => {
  switch (kind) {
    case 'fixed': return t('importAnalysis.sizing.fixed', { defaultValue: 'Fixed lots' });
    case 'martingale': return t('importAnalysis.sizing.martingale', { defaultValue: 'Martingale' });
    case 'percent_balance': return t('importAnalysis.sizing.percentBalance', { defaultValue: 'Percent of balance' });
    default: return kind;
  }
};

export const ImportAnalysisReport: React.FC<Props> = ({ analysis, loading }) => {
  const { t } = useTranslation();
  if (loading) {
    return (
      <div style={{ textAlign: 'center', padding: 32 }}>
        <Spin tip={t('importAnalysis.analyzing', { defaultValue: 'Analyzing strategy structure...' })} />
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
          message={t('importAnalysis.tradeLogicComplete', { defaultValue: 'Trading logic fully recognized' })}
          description={t('importAnalysis.guiNoiseDesc', { defaultValue: 'The following blind spots are chart display/button features that are skipped during server-side execution and do not affect trading results. Safe to import.' })}
          style={{ marginBottom: 12 }} />
      )}
      {hasRealGaps && triageLevel === 'block' && (
        <Alert type="error" showIcon icon={<CloseCircleOutlined />}
          message={t('importAnalysis.cannotImport', { defaultValue: 'Cannot auto-import' })}
          description={t('importAnalysis.cannotImportDesc', { coverage, defaultValue: `Core trading logic only ${coverage}% recognized. Unrecognized parts include entry/exit/risk logic. Try AI translation or simplify the EA and resubmit.` })}
          style={{ marginBottom: 12 }} />
      )}
      {hasRealGaps && triageLevel === 'warn' && (
        <Alert type="warning" showIcon icon={<WarningOutlined />}
          message={t('importAnalysis.incompleteCoverage', { defaultValue: 'Trading logic coverage incomplete' })}
          description={t('importAnalysis.incompleteCoverageDesc', { coverage, defaultValue: `${coverage}% code logic recognized (excluding GUI display). Unrecognized parts may affect trading behavior. Consider AI translation or manual review.` })}
          style={{ marginBottom: 12 }} />
      )}
      {!hasRealGaps && !isPureGuiNoise && triageLevel === 'pass' && (
        <Alert type="success" showIcon icon={<CheckCircleOutlined />}
          message={t('importAnalysis.goodCoverage', { defaultValue: 'Import coverage is good' })}
          description={t('importAnalysis.goodCoverageDesc', { defaultValue: 'Strategy main logic recognized. Safe to import. Check parameter list before use.' })}
          style={{ marginBottom: 12 }} />
      )}

      {/* ── Coverage ── */}
      <Card size="small" style={{ marginBottom: 12 }}>
        <Space direction="vertical" style={{ width: '100%' }} size="small">
          <Text strong>{t('importAnalysis.coverageTitle', { defaultValue: 'Import Coverage' })}</Text>
          <Progress
            percent={coverage}
            status={coverage >= 70 || isPureGuiNoise ? 'success' : coverage >= 40 ? 'active' : 'exception'}
            format={(p) => isPureGuiNoise ? t('importAnalysis.tradeLogicComplete', { defaultValue: 'Trading logic fully recognized' }) : t('importAnalysis.recognizedPct', { p, defaultValue: `${p}% strategy logic recognized` })}
          />
          <Text type="secondary">
            {t('importAnalysis.blocksSummary', { total: a.totalBlocks, recognized: a.recognizedBlocks, defaultValue: `${a.totalBlocks} logic blocks, ${a.recognizedBlocks} recognized` })}
          </Text>
        </Space>
      </Card>

      {/* ── Summary ── */}
      <Card size="small" style={{ marginBottom: 12 }}>
        <Space wrap size="small">
          <Tag icon={<ThunderboltOutlined />} color="blue">
            {t('importAnalysis.entryRules', { count: a.entryRulesCount, defaultValue: `${a.entryRulesCount} entry rules` })}
          </Tag>
          <Tag icon={<SwapOutlined />} color="purple">
            {t('importAnalysis.exitRules', { count: a.exitRulesCount, defaultValue: `${a.exitRulesCount} exit rules` })}
          </Tag>
          <Tag icon={<DollarOutlined />} color="green">
            {sizingLabel(a.sizingKind, t)}
          </Tag>
          <Tag icon={<SettingOutlined />} color="orange">
            {executionLabel(a.executionKind, t)}
          </Tag>
          {a.riskChecksCount > 0 && (
            <Tag color="red">{t('importAnalysis.riskChecks', { count: a.riskChecksCount, defaultValue: `${a.riskChecksCount} risk checks` })}</Tag>
          )}
          {a.indicatorNames.length > 0 && (
            <Tag color="cyan">{a.indicatorNames.join(', ')}</Tag>
          )}
        </Space>
      </Card>

      {/* ── Parameters ── */}
      {a.params.length > 0 && (
        <Card size="small" title={t('importAnalysis.params', { count: a.params.length, defaultValue: `Strategy Parameters (${a.params.length})` })} style={{ marginBottom: 12 }}>
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
              <span>{t('importAnalysis.needsConfirmation', { count: realBlindSpots.length, defaultValue: `Logic needs confirmation (${realBlindSpots.length})` })}</span>
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
                    {t('importAnalysis.location', { defaultValue: 'Location' })}: {b.location} | {t('importAnalysis.handling', { defaultValue: 'Handling' })}: {b.handling}
                    {b.userActionRequired && ` | ⚠️ ${t('importAnalysis.userActionRequired', { defaultValue: 'Your action required' })}`}
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

      {/* ── Info blind spots (by-design, GUI/event handlers) ── */}
      {infoBlindSpots.length > 0 && (
        <Card
          size="small"
          style={{ marginBottom: 12, borderColor: '#1890ff' }}
          title={
            <Space>
              <InfoCircleOutlined style={{ color: '#1890ff' }} />
              <span>{t('importAnalysis.skippedClientFeatures', { count: infoBlindSpots.length, defaultValue: `Skipped client features (${infoBlindSpots.length})` })}</span>
            </Space>
          }
        >
          <Space wrap size="small">
            {infoBlindSpots.map((b: BlindSpot) => (
              <Tag key={b.id || b.description} color="blue" style={{ marginBottom: 4 }}>
                {b.id || b.description}
              </Tag>
            ))}
          </Space>
        </Card>
      )}

      {/* ── All Clear ── */}
      {a.blindSpots.length === 0 && (
        <Alert
          type="success"
          showIcon
          icon={<CheckCircleOutlined />}
          message={t('importAnalysis.noBlindSpots', { defaultValue: 'No logic needs confirmation' })}
          description={t('importAnalysis.noBlindSpotsDesc', { defaultValue: 'All strategy logic auto-recognized. Safe to import.' })}
          style={{ marginBottom: 12 }}
        />
      )}
    </div>
  );
};
