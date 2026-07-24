import { useState } from 'react';
import { Button, Tag, Segmented, Typography, Spin, message, Alert, Input } from 'antd';
import { ImportOutlined, RobotOutlined, ThunderboltOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { strategyImportApi } from '@/client/strategy';
import { submitStrategy } from '@/client/agentGateway';
import { ImportAnalysisReport } from '@/components/strategy/ImportAnalysisReport';
import SemanticDiffCard from '@/components/strategy/SemanticDiffCard';
import type { AnalyzeImportCodeResponse } from '@/gen/ant/v1/strategy_runtime_pb';
import type { SubmitStrategyResponse } from '@/gen/ant/v1/agent_gateway_pb';

const { TextArea } = Input;
const { Text } = Typography;

interface Props {
  onApplyCode: (code: string) => void;
  onStrategyIdChange?: (id: string | undefined) => void;
}

export default function ImportEAPanel({ onApplyCode, onStrategyIdChange }: Props) {
  const { t } = useTranslation();
  const [eaCode, setEaCode] = useState('');
  const [eaTranslating, setEaTranslating] = useState(false);
  const [eaResult, setEaResult] = useState('');
  const [eaStrategyId, setEaStrategyId] = useState('');
  const [analysis, setAnalysis] = useState<AnalyzeImportCodeResponse | null>(null);
  const [analyzing, setAnalyzing] = useState(false);
  const [bridgeResult, setBridgeResult] = useState<SubmitStrategyResponse | null>(null);
  const [bridging, setBridging] = useState(false);
  const [bridgeSymbol, setBridgeSymbol] = useState('EURUSD');
  const [bridgeTimeframe, setBridgeTimeframe] = useState('H1');
  const [bridgeLanguage, setBridgeLanguage] = useState<'mql4' | 'mql5'>('mql4');

  const handleAnalyze = () => {
    if (!eaCode.trim() || eaCode.trim().length < 20) {
      message.warning(t('strategy.importEA.codeTooShort', { defaultValue: 'Please paste complete EA/indicator source code.' }));
      return;
    }
    setAnalyzing(true);
    strategyImportApi.analyzeCode({ sourceCode: eaCode.trim(), sourceName: 'Imported EA', sourceLang: 'mql4' })
      .then((res) => { setAnalysis(res); })
      .catch((e) => { message.error(String(e?.message || t('common.unknownError', 'Unknown error'))); })
      .finally(() => { setAnalyzing(false); });
  };

  const handleConfirmImport = () => {
    if (!analysis) return;
    setEaTranslating(true);
    strategyImportApi.importStrategy({ sourceCode: eaCode.trim(), sourceName: analysis.strategyName || 'Imported EA', sourceLang: analysis.mqlVersion || 'mql4' })
      .then((res) => {
        setEaResult(res.goCode || '');
        if (res.strategyId) { setEaStrategyId(res.strategyId); onStrategyIdChange?.(res.strategyId); }
      })
      .catch((e: unknown) => { message.error(e instanceof Error ? e.message : t('common.unknownError', { defaultValue: 'Unknown error' })); })
      .finally(() => { setEaTranslating(false); });
  };

  const handleAITranslate = () => {
    if (!eaCode.trim()) return;
    setEaResult('');
    setEaTranslating(true);
    strategyImportApi.generateCode({ sourceCode: eaCode.trim(), sourceName: 'Imported EA', sourceLang: 'mql4' })
      .then((res) => { setEaResult(res.goCode || ''); if (res.strategyId) { setEaStrategyId(res.strategyId); onStrategyIdChange?.(res.strategyId); } })
      .catch((e: unknown) => { message.error(e instanceof Error ? e.message : t('common.unknownError', { defaultValue: 'Unknown error' })); })
      .finally(() => { setEaTranslating(false); });
  };

  const handleBridge = () => {
    if (!eaCode.trim()) return;
    setBridging(true);
    submitStrategy({ source: eaCode.trim(), language: bridgeLanguage, symbol: bridgeSymbol, timeframe: bridgeTimeframe })
      .then((res) => {
        setBridgeResult(res);
        if (res.generatedCode) setEaResult(res.generatedCode);
        if (res.strategyId) { setEaStrategyId(res.strategyId); onStrategyIdChange?.(res.strategyId); }
      })
      .catch((e: unknown) => { message.error(e instanceof Error ? e.message : t('common.unknownError', { defaultValue: 'Unknown error' })); })
      .finally(() => { setBridging(false); });
  };

  const applyEaResult = () => { if (eaResult) onApplyCode(eaResult); };

  const busy = analyzing || eaTranslating || bridging;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: 420, overflow: 'auto' }}>
      <TextArea value={eaCode} onChange={e => setEaCode(e.target.value)} rows={8}
        placeholder={t('strategy.importEA.pastePlaceholder', { defaultValue: 'Paste MQL4/MQL5 EA code...' })}
        style={{ fontFamily: '"Fira Code", "Cascadia Code", "JetBrains Mono", monospace', fontSize: 13, lineHeight: 1.6, border: 'none', borderRadius: 0, resize: 'none', flex: 'none' }} />

      <div style={{ padding: '4px 14px', borderTop: '1px solid var(--color-border)', display: 'flex', gap: 8, alignItems: 'center', flexWrap: 'wrap' }}>
        <Button size="small" type="primary" icon={<ThunderboltOutlined />} onClick={handleAnalyze} loading={analyzing} disabled={busy}>
          {t('strategy.importEA.analyze', { defaultValue: '分析策略结构' })}
        </Button>
        <Button size="small" icon={<RobotOutlined />} onClick={handleAITranslate} loading={eaTranslating} disabled={busy}>
          {t('strategy.importEA.aiTranslate', { defaultValue: 'AI 翻译' })}
        </Button>
        <Button size="small" icon={<RobotOutlined />} onClick={handleBridge} loading={bridging} disabled={busy}>
          {t('strategy.importEA.bridge', { defaultValue: '盲区桥接' })}
        </Button>
        <Segmented size="small" value={bridgeLanguage} onChange={(v) => setBridgeLanguage(v as 'mql4' | 'mql5')} options={[{ label: 'MQL4', value: 'mql4' }, { label: 'MQL5', value: 'mql5' }]} />
        <Input size="small" style={{ width: 90 }} value={bridgeSymbol} onChange={e => setBridgeSymbol(e.target.value)} placeholder="Symbol" />
        <Input size="small" style={{ width: 60 }} value={bridgeTimeframe} onChange={e => setBridgeTimeframe(e.target.value)} placeholder="TF" />
        {eaResult && <Button size="small" onClick={applyEaResult}>{t('strategy.importEA.apply', { defaultValue: 'Apply to Editor' })}</Button>}
        {eaStrategyId && <Tag color="blue" style={{ marginLeft: 'auto' }}>ID: {eaStrategyId.slice(0, 8)}</Tag>}
      </div>

      {analysis && !eaResult && analysis.coverageScore >= 0.4 && (
        <div style={{ padding: '4px 14px', borderBottom: '1px solid var(--color-border)' }}>
          <Button type="primary" size="small" icon={<ImportOutlined />} onClick={handleConfirmImport} loading={eaTranslating} disabled={busy}>
            {t('strategy.importEA.confirmImport', { defaultValue: '确认导入' })}
          </Button>
        </div>
      )}
      {analysis && !eaResult && analysis.coverageScore < 0.4 && (
        <div style={{ padding: '4px 14px', borderBottom: '1px solid var(--color-border)' }}>
          <Button type="primary" size="small" icon={<RobotOutlined />} onClick={handleAITranslate} loading={eaTranslating} disabled={busy}>
            {t('strategy.importEA.tryAI', { defaultValue: 'AI 翻译' })}
          </Button>
        </div>
      )}

      <div style={{ flex: 1, overflow: 'auto', padding: '0 14px' }}>
        {analyzing && <div style={{ textAlign: 'center', padding: 24 }}><Spin tip={t('importAnalysis.analyzing', { defaultValue: 'Analyzing strategy structure...' })} /></div>}
        {eaTranslating && !analyzing && <div style={{ textAlign: 'center', padding: 24 }}><Spin tip={t('strategy.importEA.translating', { defaultValue: 'AI translating...' })} /></div>}
        {bridging && !analyzing && !eaTranslating && <div style={{ textAlign: 'center', padding: 24 }}><Spin tip={t('strategy.importEA.bridging', { defaultValue: 'AI bridging blind spots...' })} /></div>}

        {!busy && analysis && <ImportAnalysisReport analysis={analysis} loading={false} />}

        {!busy && eaResult && !bridgeResult && (
          <>
            <Alert type="success" showIcon message={t('strategy.importEA.importSuccess', { defaultValue: 'MQL 源码已导入，点击「Apply to Editor」写入编辑器' })} style={{ margin: '8px 0' }} />
            <pre style={{ margin: 0, padding: '10px 0', fontFamily: '"Fira Code", monospace', fontSize: 12, lineHeight: 1.5, whiteSpace: 'pre-wrap' }}>{eaResult}</pre>
          </>
        )}

        {!busy && bridgeResult && (
          <>
            {bridgeResult.coverageScore !== undefined && bridgeResult.coverageScore < 1.0 && (
              <Alert type="info" showIcon style={{ margin: '8px 0' }}
                message={`覆盖率: ${(bridgeResult.coverageScore * 100).toFixed(0)}%`}
                description={bridgeResult.blindSpots && bridgeResult.blindSpots.length > 0
                  ? bridgeResult.blindSpots.map(bs => `${bs.builtin} (${bs.severity}, ×${bs.count})`).join(' · ') : undefined} />
            )}
            {bridgeResult.semanticDiff && <SemanticDiffCard diff={bridgeResult.semanticDiff} />}
            {bridgeResult.bridgeStatus === 'success' && eaResult ? (
              <pre style={{ margin: 0, padding: '10px 0', fontFamily: '"Fira Code", monospace', fontSize: 12, lineHeight: 1.5, whiteSpace: 'pre-wrap' }}>{eaResult}</pre>
            ) : bridgeResult.bridgeStatus === 'bridge_failed' ? (
              <Alert type="warning" showIcon message={t('strategy.importEA.bridgeFailedMsg', { defaultValue: 'Agent 无法自动桥接所有盲区' })} description={bridgeResult.bridgeCompileError || ''} style={{ margin: '8px 0' }} />
            ) : (
              <Alert type="info" showIcon message={t('strategy.importEA.noBridgeNeeded', { defaultValue: '覆盖率 100%，无需桥接' })} style={{ margin: '8px 0' }} />
            )}
          </>
        )}

        {!busy && !analysis && !eaResult && !bridgeResult && (
          <div style={{ textAlign: 'center', padding: 24 }}><Text type="secondary">{t('strategy.importEA.hint', { defaultValue: 'Paste MQL4/MQL5 code and click Analyze' })}</Text></div>
        )}
      </div>
    </div>
  );
}
