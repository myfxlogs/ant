import { useState } from 'react';
import { Button, Tag, Typography, Spin, message, Alert, Input } from 'antd';
import { ThunderboltOutlined, RocketOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { strategyImportApi } from '@/client/strategy';
import { submitStrategy } from '@/client/agentGateway';
import { ImportAnalysisReport } from '@/components/strategy/ImportAnalysisReport';
import SemanticDiffInline from './SemanticDiffInline';
import type { AnalyzeImportCodeResponse } from '@/gen/ant/v1/strategy_runtime_pb';
import type { SubmitStrategyResponse } from '@/gen/ant/v1/agent_gateway_pb';

const { TextArea } = Input;
const { Text } = Typography;

// The backend's coverage.Version (from AnalyzeImportCode response) is authoritative.
// The `language` param sent to SubmitStrategy only distinguishes "python" from MQL;
// MQL4 vs MQL5 dispatch happens inside CompileMQLWithCoverage regardless.
function detectMQLVersion(code: string): string {
  if (/\binput\b/.test(code) || /\bCTrade\b/.test(code) ||
      /\bPositionsTotal\b/.test(code) || /\bPositionGetTicket\b/.test(code) ||
      /\bOnTradeTransaction\b/.test(code) || /\bOnBookEvent\b/.test(code)) {
    return 'mql5';
  }
  return 'mql4';
}

interface Props {
  onApplyCode: (code: string) => void;
  onStrategyIdChange?: (id: string | undefined) => void;
}

export default function ImportEAPanel({ onApplyCode, onStrategyIdChange }: Props) {
  const { t } = useTranslation();
  const [eaCode, setEaCode] = useState('');
  const [eaStrategyId, setEaStrategyId] = useState('');
  const [analysis, setAnalysis] = useState<AnalyzeImportCodeResponse | null>(null);
  const [analyzing, setAnalyzing] = useState(false);
  const [importResult, setImportResult] = useState<SubmitStrategyResponse | null>(null);
  const [importing, setImporting] = useState(false);

  const mqlVersion = analysis?.mqlVersion || detectMQLVersion(eaCode);

  const handleAnalyze = () => {
    if (!eaCode.trim() || eaCode.trim().length < 20) {
      message.warning(t('strategy.importEA.codeTooShort', { defaultValue: 'Please paste complete EA/indicator source code.' }));
      return;
    }
    setAnalyzing(true);
    setImportResult(null);
    strategyImportApi.analyzeCode({ sourceCode: eaCode.trim(), sourceName: 'Imported EA', sourceLang: detectMQLVersion(eaCode) })
      .then((res) => { setAnalysis(res); })
      .catch((e: unknown) => { message.error(e instanceof Error ? e.message : t('common.unknownError', { defaultValue: 'Unknown error' })); })
      .finally(() => { setAnalyzing(false); });
  };

  const handleImport = () => {
    if (!eaCode.trim()) return;
    setImporting(true);
    submitStrategy({ sourceCode: eaCode.trim(), language: mqlVersion })
      .then((res) => {
        setImportResult(res);
        if (res.compileSuccess === true) {
          onApplyCode(eaCode.trim());
        }
        if (res.strategyId) { setEaStrategyId(res.strategyId); onStrategyIdChange?.(res.strategyId); }
      })
      .catch((e: unknown) => { message.error(e instanceof Error ? e.message : t('common.unknownError', { defaultValue: 'Unknown error' })); })
      .finally(() => { setImporting(false); });
  };

  const busy = analyzing || importing;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', minHeight: 600, overflow: 'hidden' }}>
      <TextArea value={eaCode} onChange={(e: React.ChangeEvent<HTMLTextAreaElement>) => setEaCode(e.target.value)} rows={5}
        placeholder={t('strategy.importEA.pastePlaceholder', { defaultValue: 'Paste MQL4/MQL5 EA code...' })}
        style={{ fontFamily: '"Fira Code", "Cascadia Code", "JetBrains Mono", monospace', fontSize: 13, lineHeight: 1.6, border: 'none', borderRadius: 0, resize: 'none', flex: 'none' }} />

      <div style={{ padding: '4px 14px', borderTop: '1px solid var(--color-border)', display: 'flex', gap: 8, alignItems: 'center', flexWrap: 'wrap' }}>
        <Button size="small" icon={<ThunderboltOutlined />} onClick={handleAnalyze} loading={analyzing} disabled={busy}>
          {t('strategy.importEA.analyze')}
        </Button>
        <Button size="small" type="primary" icon={<RocketOutlined />} onClick={handleImport} loading={importing} disabled={busy || !eaCode.trim()}>
          {t('strategy.importEA.import')}
        </Button>
        {eaCode.trim() && <Tag color="blue">{mqlVersion.toUpperCase()}</Tag>}
        {eaStrategyId && <Tag color="green" style={{ marginLeft: 'auto' }}>ID: {eaStrategyId.slice(0, 8)}</Tag>}
      </div>

      <div style={{ flex: 1, overflow: 'auto', padding: '0 14px' }}>
        {analyzing && <div style={{ textAlign: 'center', padding: 24 }}><Spin tip={t('importAnalysis.analyzing', { defaultValue: 'Analyzing strategy structure...' })} /></div>}
        {importing && !analyzing && <div style={{ textAlign: 'center', padding: 24 }}><Spin tip={t('strategy.importEA.importing')} /></div>}

        {!busy && analysis && !importResult && <ImportAnalysisReport analysis={analysis} loading={false} />}

        {!busy && importResult && (
          <>
            {importResult.compileSuccess === false && (
              <Alert type="error" showIcon message={t('strategy.importEA.compileFailed')} description={importResult.compileError || ''} style={{ margin: '8px 0' }} />
            )}
            {importResult.compileSuccess === true && (
              <>
                <Alert type="success" showIcon style={{ margin: '8px 0' }}
                  message={t('strategy.importEA.importSuccess')}
                  description={eaStrategyId ? `Strategy ID: ${eaStrategyId.slice(0, 8)}` : ''} />

                {importResult.coverageScore !== undefined && importResult.coverageScore < 1.0 && (
                  <Alert type="info" showIcon style={{ margin: '8px 0' }}
                    message={t('strategy.importEA.coverage', { score: (importResult.coverageScore * 100).toFixed(0) })}
                    description={importResult.blindSpots && importResult.blindSpots.length > 0
                      ? importResult.blindSpots.map(bs => `${bs.builtin} (${bs.severity}, ×${bs.count})`).join(' · ') : undefined} />
                )}
                {importResult.semanticDiff && <SemanticDiffInline diff={importResult.semanticDiff} />}
                {importResult.bridgeStatus === 'success' && importResult.bridgedPythonSource ? (
                  <>
                    <div style={{ marginBottom: 4 }}><Text type="secondary" style={{ fontSize: 11 }}>{t('strategy.importEA.bridgeReference')}</Text></div>
                    <pre style={{ margin: 0, padding: '10px 0', fontFamily: '"Fira Code", monospace', fontSize: 12, lineHeight: 1.5, whiteSpace: 'pre-wrap' }}>{importResult.bridgedPythonSource}</pre>
                  </>
                ) : importResult.bridgeStatus === 'bridge_failed' ? (
                  <Alert type="warning" showIcon message={t('strategy.importEA.bridgeFailedMsg')} description={importResult.bridgeCompileError || ''} style={{ margin: '8px 0' }} />
                ) : importResult.bridgeStatus === 'not_attempted' ? null : null}
              </>
            )}
          </>
        )}

        {!busy && !analysis && !importResult && (
          <div style={{ textAlign: 'center', padding: 24 }}><Text type="secondary">{t('strategy.importEA.hint', { defaultValue: 'Paste MQL4/MQL5 code and click Analyze' })}</Text></div>
        )}
      </div>
    </div>
  );
}
