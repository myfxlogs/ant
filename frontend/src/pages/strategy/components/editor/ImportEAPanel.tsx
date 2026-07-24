import { useState } from 'react';
import { Button, Tag, Typography, Spin, message, Alert, Input, Card } from 'antd';
import { ImportOutlined, RobotOutlined, ThunderboltOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { strategyImportApi } from '@/client/strategy';
import { submitStrategy } from '@/client/agentGateway';
import { ImportAnalysisReport } from '@/components/strategy/ImportAnalysisReport';
import type { AnalyzeImportCodeResponse } from '@/gen/ant/v1/strategy_runtime_pb';
import type { SubmitStrategyResponse, SemanticDiff } from '@/gen/ant/v1/agent_gateway_pb';

const { TextArea } = Input;
const { Text } = Typography;

function detectMQLVersion(code: string): string {
  if (/\binput\b/.test(code) || /\bCTrade\b/.test(code) ||
      /\bPositionsTotal\b/.test(code) || /\bPositionGetTicket\b/.test(code) ||
      /\bOnTradeTransaction\b/.test(code) || /\bOnBookEvent\b/.test(code)) {
    return 'mql5';
  }
  return 'mql4';
}

const diffIcon: Record<string, string> = { added: '+', modified: '~', removed: '-', remaining: '=' };
const diffColor: Record<string, string> = { added: '#52c41a', modified: '#1677ff', removed: '#cf1322', remaining: '#faad14' };

function SemanticDiffInline({ diff }: { diff: SemanticDiff | null }) {
  if (!diff || (!diff.changes?.length && !diff.effectSummary)) return null;
  return (
    <Card size="small" style={{ marginBottom: 8 }}>
      {diff.changes?.map((c, i) => (
        <div key={i} style={{ display: 'flex', gap: 6, marginBottom: 2, fontSize: 12 }}>
          <span style={{ color: diffColor[c.kind] || '#999' }}>{diffIcon[c.kind] || '~'}</span>
          <Text style={{ fontSize: 12, color: '#595959' }}>{c.description}</Text>
        </div>
      ))}
      {diff.effectSummary && (
        <div style={{ borderTop: '1px solid #f0f0f0', paddingTop: 4, marginTop: 4 }}>
          <Text type="secondary" style={{ fontSize: 11 }}>{diff.effectSummary}</Text>
        </div>
      )}
    </Card>
  );
}

interface Props {
  onApplyCode: (code: string) => void;
  onStrategyIdChange?: (id: string | undefined) => void;
}

export default function ImportEAPanel({ onApplyCode, onStrategyIdChange }: Props) {
  const { t } = useTranslation();
  const [eaCode, setEaCode] = useState('');
  const [importing, setImporting] = useState(false);
  const [applied, setApplied] = useState(false);
  const [eaStrategyId, setEaStrategyId] = useState('');
  const [analysis, setAnalysis] = useState<AnalyzeImportCodeResponse | null>(null);
  const [analyzing, setAnalyzing] = useState(false);
  const [bridgeResult, setBridgeResult] = useState<SubmitStrategyResponse | null>(null);
  const [bridging, setBridging] = useState(false);
  const [bridgeSymbol, setBridgeSymbol] = useState('EURUSD');
  const [bridgeTimeframe, setBridgeTimeframe] = useState('H1');
  const [showBridgeParams, setShowBridgeParams] = useState(false);

  const mqlVersion = analysis?.mqlVersion || detectMQLVersion(eaCode);

  const handleAnalyze = () => {
    if (!eaCode.trim() || eaCode.trim().length < 20) {
      message.warning(t('strategy.importEA.codeTooShort', { defaultValue: 'Please paste complete EA/indicator source code.' }));
      return;
    }
    setAnalyzing(true);
    setApplied(false);
    setBridgeResult(null);
    setShowBridgeParams(false);
    strategyImportApi.analyzeCode({ sourceCode: eaCode.trim(), sourceName: 'Imported EA', sourceLang: detectMQLVersion(eaCode) })
      .then((res) => { setAnalysis(res); })
      .catch((e) => { message.error(String(e?.message || t('common.unknownError', 'Unknown error'))); })
      .finally(() => { setAnalyzing(false); });
  };

  const handleConfirmImport = () => {
    if (!analysis) return;
    setImporting(true);
    strategyImportApi.importStrategy({ sourceCode: eaCode.trim(), sourceName: analysis.strategyName || 'Imported EA', sourceLang: analysis.mqlVersion || 'mql4' })
      .then((res) => {
        setApplied(true);
        if (res.strategyId) { setEaStrategyId(res.strategyId); onStrategyIdChange?.(res.strategyId); }
        if (res.goCode) onApplyCode(res.goCode);
      })
      .catch((e: unknown) => { message.error(e instanceof Error ? e.message : t('common.unknownError', { defaultValue: 'Unknown error' })); })
      .finally(() => { setImporting(false); });
  };

  const handleBridgeClick = () => {
    if (!eaCode.trim()) return;
    setShowBridgeParams(true);
  };

  const handleBridgeExecute = () => {
    if (!eaCode.trim()) return;
    setBridging(true);
    submitStrategy({ sourceCode: eaCode.trim(), language: mqlVersion, symbol: bridgeSymbol, timeframe: bridgeTimeframe })
      .then((res) => {
        setBridgeResult(res);
        if (res.bridgedPythonSource) onApplyCode(res.bridgedPythonSource);
        if (res.strategyId) { setEaStrategyId(res.strategyId); onStrategyIdChange?.(res.strategyId); }
      })
      .catch((e: unknown) => { message.error(e instanceof Error ? e.message : t('common.unknownError', { defaultValue: 'Unknown error' })); })
      .finally(() => { setBridging(false); });
  };

  const busy = analyzing || importing || bridging;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', minHeight: 600, overflow: 'hidden' }}>
      <TextArea value={eaCode} onChange={(e: React.ChangeEvent<HTMLTextAreaElement>) => setEaCode(e.target.value)} rows={5}
        placeholder={t('strategy.importEA.pastePlaceholder', { defaultValue: 'Paste MQL4/MQL5 EA code...' })}
        style={{ fontFamily: '"Fira Code", "Cascadia Code", "JetBrains Mono", monospace', fontSize: 13, lineHeight: 1.6, border: 'none', borderRadius: 0, resize: 'none', flex: 'none' }} />

      <div style={{ padding: '4px 14px', borderTop: '1px solid var(--color-border)', display: 'flex', gap: 8, alignItems: 'center', flexWrap: 'wrap' }}>
        <Button size="small" type="primary" icon={<ThunderboltOutlined />} onClick={handleAnalyze} loading={analyzing} disabled={busy}>
          {t('strategy.importEA.analyze', { defaultValue: '分析策略结构' })}
        </Button>
        <Button size="small" icon={<RobotOutlined />} onClick={handleBridgeClick} disabled={busy}>
          {t('strategy.importEA.bridge', { defaultValue: '盲区桥接' })}
        </Button>
        {eaStrategyId && <Tag color="blue" style={{ marginLeft: 'auto' }}>ID: {eaStrategyId.slice(0, 8)}</Tag>}
      </div>

      {showBridgeParams && !bridgeResult && (
        <div style={{ padding: '4px 14px', borderBottom: '1px solid var(--color-border)', display: 'flex', gap: 8, alignItems: 'center', flexWrap: 'wrap' }}>
          <Tag color="blue">{mqlVersion.toUpperCase()}</Tag>
          <Input size="small" style={{ width: 90 }} value={bridgeSymbol} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setBridgeSymbol(e.target.value)} placeholder="Symbol" />
          <Input size="small" style={{ width: 60 }} value={bridgeTimeframe} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setBridgeTimeframe(e.target.value)} placeholder="TF" />
          <Button size="small" type="primary" icon={<RobotOutlined />} onClick={handleBridgeExecute} loading={bridging} disabled={busy}>
            {t('strategy.importEA.bridgeBtn', { defaultValue: '执行桥接' })}
          </Button>
        </div>
      )}

      {analysis && !applied && analysis.coverageScore >= 0.4 && (
        <div style={{ padding: '4px 14px', borderBottom: '1px solid var(--color-border)' }}>
          <Button type="primary" size="small" icon={<ImportOutlined />} onClick={handleConfirmImport} loading={importing} disabled={busy}>
            {t('strategy.importEA.confirmImport', { defaultValue: '确认导入' })}
          </Button>
        </div>
      )}
      {analysis && !applied && analysis.coverageScore < 0.4 && (
        <div style={{ padding: '4px 14px', borderBottom: '1px solid var(--color-border)' }}>
          <Text type="warning" style={{ fontSize: 12 }}>
            {t('strategy.importEA.lowCoverageHint', { defaultValue: '覆盖率较低，建议使用盲区桥接处理无法识别的函数' })}
          </Text>
        </div>
      )}

      <div style={{ flex: 1, overflow: 'auto', padding: '0 14px' }}>
        {analyzing && <div style={{ textAlign: 'center', padding: 24 }}><Spin tip={t('importAnalysis.analyzing', { defaultValue: 'Analyzing strategy structure...' })} /></div>}
        {importing && !analyzing && <div style={{ textAlign: 'center', padding: 24 }}><Spin tip={t('strategy.importEA.importing', { defaultValue: 'Importing...' })} /></div>}
        {bridging && !analyzing && !importing && <div style={{ textAlign: 'center', padding: 24 }}><Spin tip={t('strategy.importEA.bridging', { defaultValue: 'AI bridging blind spots...' })} /></div>}

        {!busy && analysis && <ImportAnalysisReport analysis={analysis} loading={false} />}

        {!busy && applied && !bridgeResult && (
          <Alert type="success" showIcon message={t('strategy.importEA.importSuccess', { defaultValue: '代码已应用到编辑器' })} style={{ margin: '8px 0' }} />
        )}

        {!busy && bridgeResult && (
          <>
            {bridgeResult.coverageScore !== undefined && bridgeResult.coverageScore < 1.0 && (
              <Alert type="info" showIcon style={{ margin: '8px 0' }}
                message={`覆盖率: ${(bridgeResult.coverageScore * 100).toFixed(0)}%`}
                description={bridgeResult.blindSpots && bridgeResult.blindSpots.length > 0
                  ? bridgeResult.blindSpots.map(bs => `${bs.builtin} (${bs.severity}, ×${bs.count})`).join(' · ') : undefined} />
            )}
            {bridgeResult.semanticDiff && <SemanticDiffInline diff={bridgeResult.semanticDiff} />}
            {bridgeResult.bridgeStatus === 'success' && bridgeResult.bridgedPythonSource ? (
              <pre style={{ margin: 0, padding: '10px 0', fontFamily: '"Fira Code", monospace', fontSize: 12, lineHeight: 1.5, whiteSpace: 'pre-wrap' }}>{bridgeResult.bridgedPythonSource}</pre>
            ) : bridgeResult.bridgeStatus === 'bridge_failed' ? (
              <Alert type="warning" showIcon message={t('strategy.importEA.bridgeFailedMsg', { defaultValue: 'Agent 无法自动桥接所有盲区' })} description={bridgeResult.bridgeCompileError || ''} style={{ margin: '8px 0' }} />
            ) : (
              <Alert type="info" showIcon message={t('strategy.importEA.noBridgeNeeded', { defaultValue: '覆盖率 100%，无需桥接' })} style={{ margin: '8px 0' }} />
            )}
          </>
        )}

        {!busy && !analysis && !applied && !bridgeResult && (
          <div style={{ textAlign: 'center', padding: 24 }}><Text type="secondary">{t('strategy.importEA.hint', { defaultValue: 'Paste MQL4/MQL5 code and click Analyze' })}</Text></div>
        )}
      </div>
    </div>
  );
}
