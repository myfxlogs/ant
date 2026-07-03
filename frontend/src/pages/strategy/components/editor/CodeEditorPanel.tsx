import { useState, useCallback } from 'react';
import { Button, Form, Tag, Segmented, Typography, Spin, message, Radio, Alert } from 'antd';
import { CodeOutlined, ImportOutlined, RobotOutlined, ThunderboltOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { EDIT_TEMPLATE_MODAL_FIELDS_CODE_KEY, EDIT_TEMPLATE_MODAL_PLACEHOLDERS_CODE_SAMPLE_KEY, EDIT_TEMPLATE_MODAL_VALIDATION_CODE_REQUIRED_KEY } from '@/gen/ant/v1/i18n/strategy_templates_keys';
import { codeAssistClient } from '@/client/connect';
import { strategyImportApi } from '@/client/strategy';
import { submitStrategy } from '@/client/agentGateway';
import { ImportAnalysisReport } from '@/components/strategy/ImportAnalysisReport';
import SemanticDiffCard from '@/components/strategy/SemanticDiffCard';
import type { AnalyzeImportCodeResponse } from '@/gen/ant/v1/strategy_runtime_pb';
import type { SubmitStrategyResponse } from '@/gen/ant/v1/agent_gateway_pb';
import type { FormInstance } from 'antd';
import { Input } from 'antd';

const { TextArea } = Input;
const { Text } = Typography;

type ImportMethod = 'migration' | 'ai' | 'bridge';

interface Props {
  form: FormInstance;
  code: string;
  onStrategyIdChange?: (id: string | undefined) => void;
}

export default function CodeEditorPanel({ form, code, onStrategyIdChange }: Props) {
  const { t } = useTranslation();
  const [mode, setMode] = useState<string>('write');
  const [eaCode, setEaCode] = useState('');
  const [eaTranslating, setEaTranslating] = useState(false);
  const [eaResult, setEaResult] = useState('');
  const [eaStrategyId, setEaStrategyId] = useState('');
  const [importMethod, setImportMethod] = useState<ImportMethod>('migration');
  // Migration engine state
  const [analysis, setAnalysis] = useState<AnalyzeImportCodeResponse | null>(null);
  const [analyzing, setAnalyzing] = useState(false);
  // Bridge state (ADR-0024 Phase 2)
  const [bridgeResult, setBridgeResult] = useState<SubmitStrategyResponse | null>(null);
  const [bridging, setBridging] = useState(false);

  // ── Migration: Analyze ──────────────────────────────────────────

  const handleAnalyze = useCallback(() => {
    if (!eaCode.trim() || eaCode.trim().length < 20) {
      message.warning(t('strategy.importEA.codeTooShort', { defaultValue: 'Please paste complete EA/indicator source code.' }));
      return;
    }
    setAnalyzing(true); setAnalysis(null); setEaResult('');
    (async () => {
      try {
        const resp = await strategyImportApi.analyzeCode({
          sourceCode: eaCode,
          sourceName: 'imported.mq4',
        });
        setAnalysis(resp);
      } catch (err: any) {
        if (err?.code == null) {
          message.error(t('strategy.importEA.analyzeFailed', { defaultValue: 'Analysis failed. Please try again.' }));
        }
      } finally { setAnalyzing(false); }
    })().catch(() => {});
  }, [eaCode, t]);

  // ── Migration: Confirm Import ────────────────────────────────────

  const handleConfirmImport = useCallback(() => {
    if (!eaCode.trim()) return;
    setEaTranslating(true);
    (async () => {
      try {
        const resp = await strategyImportApi.importStrategy({
          sourceCode: eaCode,
          sourceName: 'imported.mq4',
        });
        // MQL is the single source of truth — apply directly to editor.
        setEaResult(eaCode);
        setEaStrategyId(resp.strategyId || '');
        form.setFieldsValue({ code: eaCode, strategyId: resp.strategyId || '' });
        onStrategyIdChange?.(resp.strategyId || undefined);
      } catch (err: any) {
        if (err?.code == null) {
          message.error(t('strategy.importEA.importFailed', { defaultValue: 'Import failed. Please try again.' }));
        }
      } finally { setEaTranslating(false); }
    })().catch(() => {});
  }, [eaCode, t, form]);

  // ── Bridge (ADR-0024 Phase 2) ───────────────────────────────────

  const handleBridge = useCallback(() => {
    if (!eaCode.trim() || eaCode.trim().length < 20) {
      message.warning(t('strategy.importEA.codeTooShort', { defaultValue: 'Please paste complete EA/indicator source code.' }));
      return;
    }
    setBridging(true); setBridgeResult(null);
    (async () => {
      try {
        const resp = await submitStrategy({
          sourceCode: eaCode,
          language: 'mql4',
          backtestConfig: {
            symbol: 'EURUSD',
            timeframe: 'H1',
            startDateMs: Date.now() - 365 * 24 * 60 * 60 * 1000,
            endDateMs: Date.now(),
          },
        });
        setBridgeResult(resp);
        if (resp.bridgeStatus === 'success' && resp.bridgedPythonSource) {
          setEaResult(resp.bridgedPythonSource);
          setEaStrategyId(resp.strategyId || '');
        }
      } catch (err: any) {
        if (err?.code == null) {
          message.error(t('strategy.importEA.bridgeFailed', { defaultValue: 'Bridge translation failed. Please try again.' }));
        }
      } finally { setBridging(false); }
    })().catch(() => {});
  }, [eaCode, t]);

  // ── AI Translation ──────────────────────────────────────────────

  const handleImportEA = useCallback(() => {
    if (!eaCode.trim() || eaCode.trim().length < 20) {
      message.warning(t('strategy.importEA.codeTooShort', { defaultValue: 'Please paste complete EA/indicator source code.' }));
      return;
    }
    setEaTranslating(true); setEaResult(''); setAnalysis(null);
    (async () => {
      try {
        const resp = await codeAssistClient.transformCode({ sourceCode: eaCode, sourceLang: 'auto', targetLang: 'go' });
        setEaResult(resp.targetCode || '');
      } catch (err: any) {
        if (err?.code == null) {
          message.error(t('strategy.importEA.translateFailed', { defaultValue: 'Translation failed. Please try again.' }));
        }
      }
      finally { setEaTranslating(false); }
    })().catch(() => {});
  }, [eaCode, t]);

  const applyEaResult = useCallback(() => {
    if (eaResult) { form.setFieldsValue({ code: eaResult }); setMode('write'); }
  }, [eaResult, form]);

  return (
    <div style={{ border: '1px solid var(--color-border)', borderRadius: 10, overflow: 'hidden', display: 'flex', flexDirection: 'column' }}>
      <div style={{ padding: '4px 14px', background: 'var(--color-bg-tertiary)', borderBottom: '1px solid var(--color-border)', display: 'flex', alignItems: 'center', gap: 6 }}>
        <Segmented size="small" value={mode} onChange={v => setMode(v as string)}
          options={[
            { value: 'write', icon: <CodeOutlined />, label: t('strategy.importEA.writeTab', { defaultValue: 'Strategy Code' }) },
            { value: 'import', icon: <ImportOutlined />, label: t('strategy.importEA.importTab', { defaultValue: 'Import EA' }) },
          ]} />
        {mode === 'write' && code.trim() && <Tag style={{ marginLeft: 'auto' }}>{code.split('\n').length} lines</Tag>}
      </div>

      {mode === 'write' ? (
        <Form.Item name="code" rules={[{ required: true, message: t(EDIT_TEMPLATE_MODAL_VALIDATION_CODE_REQUIRED_KEY) }]} style={{ marginBottom: 0 }}>
          <TextArea rows={18} placeholder={t(EDIT_TEMPLATE_MODAL_PLACEHOLDERS_CODE_SAMPLE_KEY)}
            style={{ fontFamily: '"Fira Code", "Cascadia Code", "JetBrains Mono", monospace', fontSize: 13, lineHeight: 1.6, border: 'none', borderRadius: 0, resize: 'none', height: 420 }} />
        </Form.Item>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', height: 420, overflow: 'auto' }}>
          <TextArea value={eaCode} onChange={e => setEaCode(e.target.value)} rows={8}
            placeholder={t('strategy.importEA.pastePlaceholder', { defaultValue: 'Paste MQL4/MQL5 EA code...' })}
            style={{ fontFamily: '"Fira Code", "Cascadia Code", "JetBrains Mono", monospace', fontSize: 13, lineHeight: 1.6, border: 'none', borderRadius: 0, resize: 'none', flex: 'none' }} />

          {/* ── Method selector ── */}
          <div style={{ padding: '4px 14px', borderTop: '1px solid var(--color-border)', display: 'flex', gap: 8, alignItems: 'center', flexWrap: 'wrap' }}>
            <Radio.Group size="small" value={importMethod} onChange={e => { setImportMethod(e.target.value); setEaResult(''); setAnalysis(null); setBridgeResult(null); }}
              optionType="button" buttonStyle="solid">
              <Radio.Button value="migration">
                <ThunderboltOutlined /> {t('strategy.importEA.migration', { defaultValue: '策略导入' })}</Radio.Button>
              <Radio.Button value="ai">
                <RobotOutlined /> {t('strategy.importEA.aiTranslate', { defaultValue: 'AI 翻译' })}</Radio.Button>
              <Radio.Button value="bridge">
                <RobotOutlined /> {t('strategy.importEA.bridge', { defaultValue: '盲区桥接' })}</Radio.Button>
            </Radio.Group>
          </div>

          {/* ── Migration flow ── */}
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
                    {/* When coverage is low and has real gaps, suggest AI translate */}
                    {analysis.coverageScore < 0.7 && (analysis.blindSpots || []).some((b) =>
                      b.category !== '不支持的API调用' || (
                        !b.description?.includes('ObjectCreate') && !b.description?.includes('ObjectDelete')
                      )
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
                {eaResult && (
                  <Alert
                    type="success"
                    showIcon
                    message={t('strategy.importEA.importSuccess', { defaultValue: 'MQL 源码已导入，点击「Apply to Editor」写入编辑器' })}
                    style={{ margin: '8px 0' }}
                  />
                )}
                {!analysis && !eaResult && !analyzing && (
                  <div style={{ textAlign: 'center', padding: 24 }}>
                    <Text type="secondary">{t('strategy.importEA.hint', { defaultValue: 'Paste MQL4/MQL5 code and click Analyze' })}</Text>
                  </div>
                )}
              </div>
            </>
          )}

          {/* ── AI translation flow ── */}
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
                    {eaTranslating ? <Spin tip={t('strategy.importEA.translating', { defaultValue: 'AI translating...' })} />
                      : <Text type="secondary">{t('strategy.importEA.hint', { defaultValue: 'Paste MQL4/MQL5 code and click Translate' })}</Text>}
                  </div>
                )}
              </div>
            </>
          )}

          {/* ── Bridge flow (ADR-0024 Phase 2) ── */}
          {importMethod === 'bridge' && (
            <>
              <div style={{ padding: '6px 14px', borderBottom: '1px solid var(--color-border)', display: 'flex', gap: 8, alignItems: 'center' }}>
                <Button type="primary" size="small" icon={<RobotOutlined />} onClick={handleBridge} loading={bridging}>
                  {t('strategy.importEA.bridgeBtn', { defaultValue: '盲区桥接翻译' })}</Button>
                {eaResult && <Button size="small" onClick={applyEaResult}>{t('strategy.importEA.apply', { defaultValue: 'Apply to Editor' })}</Button>}
                {bridgeResult && bridgeResult.bridgeStatus === 'success' && (
                  <Tag color="success" style={{ marginLeft: 'auto' }}>
                    {t('strategy.importEA.bridgeSuccess', { defaultValue: '桥接成功' })}
                  </Tag>
                )}
                {bridgeResult && bridgeResult.bridgeStatus === 'bridge_failed' && (
                  <Tag color="error" style={{ marginLeft: 'auto' }}>
                    {t('strategy.importEA.bridgeFailedTag', { defaultValue: '桥接失败' })}
                  </Tag>
                )}
              </div>
              <div style={{ flex: 1, overflow: 'auto', padding: '0 14px' }}>
                {bridging ? (
                  <div style={{ textAlign: 'center', padding: 24 }}>
                    <Spin tip={t('strategy.importEA.bridging', { defaultValue: 'AI bridging blind spots...' })} />
                  </div>
                ) : bridgeResult ? (
                  <>
                    {bridgeResult.semanticDiff && (
                      <SemanticDiffCard diff={bridgeResult.semanticDiff} />
                    )}
                    {bridgeResult.bridgeStatus === 'success' && eaResult ? (
                      <pre style={{ margin: 0, padding: '10px 0', fontFamily: '"Fira Code", monospace', fontSize: 12, lineHeight: 1.5, whiteSpace: 'pre-wrap' }}>{eaResult}</pre>
                    ) : bridgeResult.bridgeStatus === 'bridge_failed' ? (
                      <Alert
                        type="warning"
                        showIcon
                        message={t('strategy.importEA.bridgeFailedMsg', { defaultValue: 'Agent 无法自动桥接所有盲区' })}
                        description={bridgeResult.bridgeCompileError || ''}
                        style={{ margin: '8px 0' }}
                      />
                    ) : (
                      <Alert
                        type="info"
                        showIcon
                        message={t('strategy.importEA.noBridgeNeeded', { defaultValue: '覆盖率 100%，无需桥接' })}
                        style={{ margin: '8px 0' }}
                      />
                    )}
                  </>
                ) : (
                  <div style={{ textAlign: 'center', padding: 24 }}>
                    <Text type="secondary">{t('strategy.importEA.bridgeHint', { defaultValue: '粘贴 MQL4/MQL5 EA 代码，AI 将自动翻译盲区为 Python 子集' })}</Text>
                  </div>
                )}
              </div>
            </>
          )}
        </div>
      )}
    </div>
  );
}
