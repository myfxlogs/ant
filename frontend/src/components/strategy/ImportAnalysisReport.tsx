import React from 'react';
import type { TFunction } from 'i18next';
import { Progress, Tag, Alert, Spin, Typography, Space, Card } from 'antd';
import {
  WarningOutlined,
  InfoCircleOutlined,
  ThunderboltOutlined,
  SwapOutlined,
  DollarOutlined,
  SettingOutlined,
} from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import type { AnalyzeImportCodeResponse, BlindSpot, ParamField, ParamGroupInfo } from '@/gen/ant/v1/strategy_runtime_pb';

const { Text } = Typography;

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

const executionLabel = (kind: string, t: TFunction): string => {
  switch (kind) {
    case 'on_bar': return t('importAnalysis.execution.onBar', { defaultValue: 'Bar close event-driven' });
    case 'on_tick': return t('importAnalysis.execution.onTick', { defaultValue: 'Tick-driven' });
    case 'on_init_grid': return t('importAnalysis.execution.onInitGrid', { defaultValue: 'Init grid' });
    default: return kind;
  }
};

const sizingLabel = (kind: string, t: TFunction): string => {
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
  const realBlindSpots = a.blindSpots.filter(b => b.severity !== '信息');
  const infoBlindSpots = a.blindSpots.filter(b => b.severity === '信息');
  const isEmpty = a.totalBlocks === 0 && a.recognizedBlocks === 0 && a.coverageScore === 0;
  const isPureGuiNoise = realBlindSpots.length === 0 && infoBlindSpots.length > 0;

  // Single triage verdict
  const triage = isEmpty
    ? { type: 'error' as const, title: t('importAnalysis.cannotImport', { defaultValue: 'Cannot auto-import' }),
        desc: t('importAnalysis.emptyAnalysisDesc', { defaultValue: 'No strategy logic was recognized. The source code may be incomplete or use a different language.' }) }
    : isPureGuiNoise
    ? { type: 'success' as const, title: t('importAnalysis.tradeLogicComplete', { defaultValue: 'Trading logic fully recognized' }),
        desc: t('importAnalysis.guiNoiseDesc', { defaultValue: 'Chart display/button features are skipped during server-side execution. Safe to import.' }) }
    : realBlindSpots.length > 0 && coverage < 40
    ? { type: 'error' as const, title: t('importAnalysis.cannotImport', { defaultValue: 'Cannot auto-import' }),
        desc: t('importAnalysis.cannotImportDesc', { coverage, defaultValue: `Core trading logic only ${coverage}% recognized. Use 盲区桥接 to handle unrecognized functions.` }) }
    : realBlindSpots.length > 0 && coverage < 70
    ? { type: 'warning' as const, title: t('importAnalysis.incompleteCoverage', { defaultValue: 'Trading logic coverage incomplete' }),
        desc: t('importAnalysis.incompleteCoverageDesc', { coverage, defaultValue: `${coverage}% code logic recognized. Unrecognized parts may affect trading behavior. Consider 盲区桥接 or manual review.` }) }
    : { type: 'success' as const, title: t('importAnalysis.goodCoverage', { defaultValue: 'Import coverage is good' }),
        desc: t('importAnalysis.goodCoverageDesc', { defaultValue: 'Strategy main logic recognized. Safe to import. Check parameter list before use.' }) };

  return (
    <div style={{ padding: '12px 0' }}>
      <Alert type={triage.type} showIcon style={{ marginBottom: 12 }}
        message={triage.title} description={triage.desc} />

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

      {/* ── Blind Spots (real gaps only) ── */}
      {realBlindSpots.length > 0 && (
        <Card
          size="small"
          title={<span><WarningOutlined /> {t('importAnalysis.needsConfirmation', { count: realBlindSpots.length, defaultValue: `Logic needs confirmation (${realBlindSpots.length})` })}</span>}
          style={{ marginBottom: 12 }}
        >
          {realBlindSpots.map((b: BlindSpot) => (
            <Alert key={b.id || b.description}
              type={b.severity === '致命' ? 'error' : 'warning'} showIcon
              message={<span><Text strong>{b.category}</Text> — <Text type="secondary">{b.description}</Text></span>}
              style={{ marginBottom: 8 }} />
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
      {a.blindSpots.length === 0 && !isEmpty && (
        <Alert type="success" showIcon
          message={t('importAnalysis.noBlindSpots', { defaultValue: 'No logic needs confirmation' })}
          description={t('importAnalysis.noBlindSpotsDesc', { defaultValue: 'All strategy logic auto-recognized. Safe to import.' })}
          style={{ marginBottom: 12 }} />
      )}
    </div>
  );
};
