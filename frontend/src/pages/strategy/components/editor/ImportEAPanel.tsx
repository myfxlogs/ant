import { useState } from 'react';
import { Button, Tag, Segmented, Typography, Spin, message, Radio, Alert, Input } from 'antd';
import { ImportOutlined, RobotOutlined, ThunderboltOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { codeAssistClient } from '@/client/connect';
import { strategyImportApi } from '@/client/strategy';
import { submitStrategy } from '@/client/agentGateway';
import { ImportAnalysisReport } from '@/components/strategy/ImportAnalysisReport';
import SemanticDiffCard from '@/components/strategy/SemanticDiffCard';
import type { AnalyzeImportCodeResponse } from '@/gen/ant/v1/strategy_runtime_pb';
import type { SubmitStrategyResponse } from '@/gen/ant/v1/agent_gateway_pb';

const { TextArea } = Input;
const { Text } = Typography;
type ImportMethod = 'migration' | 'ai' | 'bridge';

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
  const [importMethod, setImportMethod] = useState<ImportMethod>('migration');
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
    codeAssistClient.analyzeImportCode({ source: eaCode.trim() })
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
      .catch((e) => { message.error(String(e?.message || t('common.unknownError', 'Unknown error'))); })
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
      .catch((e) => { message.error(String(e?.message || t('common.unknownError', 'Unknown error'))); })
      .finally(() => { setBridging(false); });
  };

  const handleImportEA = () => {
    if (!eaCode.trim()) return;
    setEaTranslating(true);
    codeAssistClient.translateCode({ source: eaCode.trim() })
      .then((res) => { setEaResult(res.code || ''); if ((res as any).strategyId) onStrategyIdChange?.((res as any).strategyId); })
      .catch((e) => { message.error(String(e?.message || t('common.unknownError', 'Unknown error'))); })
      .finally(() => { setEaTranslating(false); });
  };

  const applyEaResult = () => { if (eaResult) onApplyCode(eaResult); };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: 420, overflow: 'auto' }}>
      <TextArea value={eaCode} onChange={e => setEaCode(e.target.value)} rows={8}
        placeholder={t('strategy.importEA.pastePlaceholder', { defaultValue: 'Paste MQL4/MQL5 EA code...' })}
        style={{ fontFamily: '"Fira Code", "Cascadia Code", "JetBrains Mono", monospace', fontSize: 13, lineHeight: 1.6, border: 'none', borderRadius: 0, resize: 'none', flex: 'none' }} />

      <div style={{ padding: '4px 14px', borderTop: '1px solid var(--color-border)', display: 'flex', gap: 8, alignItems: 'center', flexWrap: 'wrap' }}>
        <Radio.Group size="small" value={importMethod} onChange={e => { setImportMethod(e.target.value); setEaResult(''); setAnalysis(null); setBridgeResult(null); }}
          optionType="button" buttonStyle="solid">
          <Radio.Button value="migration"><ThunderboltOutlined /> {t('strategy.importEA.migration', { defaultValue: '策略导入' })}</Radio.Button>
          <Radio.Button value="ai"><RobotOutlined /> {t('strategy.importEA.aiTranslate', { defaultValue: 'AI 翻译' })}</Radio.Button>
          <Radio.Button value="bridge"><RobotOutlined /> {t('strategy.importEA.bridge', { defaultValue: '盲区桥接' })}</Radio.Button>
        </Radio.Group>
      </div>

      {/* Migration flow */}
      {importMethod === 'migration' && (
        <>
          <div style={{ padding: '6px 14px', borderBottom: '1px solid var(--color-border)', display: 'flex', gap: 8, alignItems: 'center' }}>
            {!analysis && (
              <Button type="primary" size="small" icon={<ThunderboltOutlined />} onClick={handleAnalyze} loading={analyzing}>
                {t('strategy.importEA.analyze', { defaultValue: '分析策略结构' })}</Button>
            )}
            {analysis && !eaResult && (
              <>
                <Button type="primary" size="small" icon={<ImportOutlined />} onClick={handleConfirmImport} loading={eaTranslating}>
                  {t('strategy.importEA.confirmImport', { defaultValue: '确认导入' })}</Button>
                {analysis.coverageScore < 0.7 && (analysis.blindSpots || []).some((b) =>
                  b.category !== 'Unsupported API Call' || (!b.description?.includes('ObjectCreate') && !b.description?.includes('ObjectDelete'))
                ) && (
                  <Button size="small" onClick={() => { setImportMethod('ai'); handleImportEA(); }}>
                    <RobotOutlined /> {t('strategy.importEA.tryAI', { defaultValue: 'AI 翻译补充' })}</Button>
                )}
              </>
            )}
            {eaResult && <Button size="small" onClick={applyEaResult}>{t('strategy.importEA.apply', { defaultValue: 'Apply to Editor' })}</Button>}
            {eaStrategyId && <Tag color="blue" style={{ marginLeft: 'auto' }}>ID: {eaStrategyId.slice(0, 8)}</Tag>}
          </div>
          <div style={{ flex: 1, overflow: 'auto', padding: '0 14px' }}>
            <ImportAnalysisReport analysis={analysis} loading={analyzing} />
            {eaResult && <Alert type="success" showIcon message={t('strategy.importEA.importSuccess', { defaultValue: 'MQL 源码已导入，点击「Apply to Editor」写入编辑器' })} style={{ margin: '8px 0' }} />}
            {!analysis && !eaResult && !analyzing && (
              <div style={{ textAlign: 'center', padding: 24 }}><Text type="secondary">{t('strategy.importEA.hint', { defaultValue: 'Paste MQL4/MQL5 code and click Analyze' })}</Text></div>
            )}
          </div>
        </>
      )}

      {/* AI translation flow */}
      {importMethod === 'ai' && (
        <>
          <div style={{ padding: '6px 14px', borderBottom: '1px solid var(--color-border)', display: 'flex', gap: 8, alignItems: 'center' }}>
            <Button type="primary" size="small" icon={<RobotOutlined />} onClick={handleImportEA} loading={eaTranslating}>
              {t('strategy.importEA.translate', { defaultValue: 'Translate to Go' })}</Button>
            {eaResult && <Button size="small" onClick={applyEaResult}>{t('strategy.importEA.apply', { defaultValue: 'Apply to Editor' })}</Button>}
          </div>
          <div style={{ flex: 1, overflow: 'auto', padding: '0 14px' }}>
            {eaResult ? (
              <pre style={{ margin: 0, padding: '10px 0', fontFamily: '"Fira Code", monospace', fontSize: 12, lineHeight: 1.5, whiteSpace: 'pre-wrap' }}>{eaResult}</pre>
            ) : (
              <div style={{ textAlign: 'center', padding: 24 }}>
                {eaTranslating ? <Spin tip={t('strategy.importEA.translating', { defaultValue: 'AI translating...' })} /> : <Text type="secondary">{t('strategy.importEA.hint', { defaultValue: 'Paste MQL4/MQL5 code and click Translate' })}</Text>}
              </div>
            )}
          </div>
        </>
      )}

      {/* Bridge flow */}
      {importMethod === 'bridge' && (
        <>
          <div style={{ padding: '6px 14px', borderBottom: '1px solid var(--color-border)', display: 'flex', gap: 8, alignItems: 'center', flexWrap: 'wrap' }}>
            <Button type="primary" size="small" icon={<RobotOutlined />} onClick={handleBridge} loading={bridging}>
              {t('strategy.importEA.bridgeBtn', { defaultValue: '盲区桥接翻译' })}</Button>
            <Segmented size="small" value={bridgeLanguage} onChange={(v) => setBridgeLanguage(v as 'mql4' | 'mql5')} options={[{ label: 'MQL4', value: 'mql4' }, { label: 'MQL5', value: 'mql5' }]} />
            <Input size="small" style={{ width: 90 }} value={bridgeSymbol} onChange={e => setBridgeSymbol(e.target.value)} placeholder="Symbol" />
            <Input size="small" style={{ width: 60 }} value={bridgeTimeframe} onChange={e => setBridgeTimeframe(e.target.value)} placeholder="TF" />
            {eaResult && <Button size="small" onClick={applyEaResult}>{t('strategy.importEA.apply', { defaultValue: 'Apply to Editor' })}</Button>}
            {bridgeResult && bridgeResult.bridgeStatus === 'success' && <Tag color="success" style={{ marginLeft: 'auto' }}>{t('strategy.importEA.bridgeSuccess', { defaultValue: '桥接成功' })}</Tag>}
            {bridgeResult && bridgeResult.bridgeStatus === 'bridge_failed' && <Tag color="error" style={{ marginLeft: 'auto' }}>{t('strategy.importEA.bridgeFailedTag', { defaultValue: '桥接失败' })}</Tag>}
          </div>
          <div style={{ flex: 1, overflow: 'auto', padding: '0 14px' }}>
            {bridging ? <div style={{ textAlign: 'center', padding: 24 }}><Spin tip={t('strategy.importEA.bridging', { defaultValue: 'AI bridging blind spots...' })} /></div>
              : bridgeResult ? <>
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
              </> : <div style={{ textAlign: 'center', padding: 24 }}><Text type="secondary">{t('strategy.importEA.bridgeHint', { defaultValue: 'Paste MQL4/MQL5 EA code, AI will bridge blind spots to platform bytecode' })}</Text></div>
            }
          </div>
        </>
      )}
    </div>
  );
}
