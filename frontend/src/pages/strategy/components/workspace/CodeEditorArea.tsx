import { useState, useEffect, useRef, useCallback } from 'react';
import { Button, Tag, Tooltip } from 'antd';
import { ImportOutlined, RobotOutlined, HistoryOutlined, CheckCircleOutlined, WarningOutlined, CloseCircleOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import StrategyCodeEditor, { type Diagnostic } from '@/components/strategy/StrategyCodeEditor';
import ImportEAPanel from '../editor/ImportEAPanel';
import { strategyVersionApi } from '@/client/strategy';
import type { BlindSpot } from '@/gen/ant/v1/strategy_runtime_pb';

interface Props {
  code: string;
  importMode: boolean;
  isMobile: boolean;
  templateCount: number;
  onSetImportMode: (v: boolean) => void;
  onSetCode: (c: string) => void;
  onSetCenterTab: (tab: string) => void;
  onSetRightPanelTab: (tab: 'ai') => void;
  onSelectFirstTemplate: () => void;
  onStrategyIdChange?: (id: string | undefined) => void;
}

function blindSpotsToDiagnostics(blindSpots: BlindSpot[], compileError?: string): Diagnostic[] {
  const diags: Diagnostic[] = [];
  if (compileError) {
    diags.push({ message: compileError, severity: 'error' });
  }
  for (const bs of blindSpots) {
    const severity = bs.severity === '致命' ? 'error' : bs.severity === '警告' ? 'warning' : 'info';
    diags.push({ message: `${bs.id}: ${bs.description}`, severity });
  }
  return diags;
}

export default function CodeEditorArea({ code, importMode, isMobile, templateCount, onSetImportMode, onSetCode, onSetCenterTab, onSetRightPanelTab, onSelectFirstTemplate, onStrategyIdChange }: Props) {
  const { t } = useTranslation();
  const [diagnostics, setDiagnostics] = useState<Diagnostic[]>([]);
  const [auditStatus, setAuditStatus] = useState<'idle' | 'checking' | 'ok' | 'warn' | 'error'>('idle');
  const [auditSummary, setAuditSummary] = useState<string>('');
  const debounceTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const lastCheckedCode = useRef('');

  const runCheck = useCallback(async (sourceCode: string) => {
    if (!sourceCode.trim() || sourceCode.trim().length < 20) {
      setDiagnostics([]);
      setAuditStatus('idle');
      return;
    }
    setAuditStatus('checking');
    try {
      const resp = await strategyVersionApi.checkCode(sourceCode);
      const diags = blindSpotsToDiagnostics(resp.blindSpots, resp.compileError || undefined);
      setDiagnostics(diags);
      if (!resp.compileSuccess) {
        setAuditStatus('error');
        setAuditSummary(resp.compileError || 'Compile failed');
      } else if (resp.blindSpots.length > 0) {
        setAuditStatus('warn');
        setAuditSummary(`${resp.blindSpots.length} blind spot(s), coverage ${(resp.coverageScore * 100).toFixed(0)}%`);
      } else {
        setAuditStatus('ok');
        setAuditSummary(`All clear, coverage ${(resp.coverageScore * 100).toFixed(0)}%`);
      }
    } catch {
      setAuditStatus('idle');
    }
  }, []);

  useEffect(() => {
    if (debounceTimer.current) clearTimeout(debounceTimer.current);
    if (!code || code === lastCheckedCode.current) return;
    debounceTimer.current = setTimeout(() => {
      lastCheckedCode.current = code;
      runCheck(code);
    }, 800);
    return () => { if (debounceTimer.current) clearTimeout(debounceTimer.current); };
  }, [code, runCheck]);

  if (importMode) {
    return (
      <div style={{ flex: 1, overflow: 'auto', padding: '12px 16px' }}>
        <div style={{ display: 'flex', alignItems: 'center', marginBottom: 8 }}>
          <Button size="small" type="text" onClick={() => onSetImportMode(false)}>
            ← {t('strategy.workspace.backToEditor', { defaultValue: 'Back' })}
          </Button>
        </div>
        <ImportEAPanel
          onApplyCode={(c) => { onSetCode(c); onSetCenterTab('code'); onSetImportMode(false); }}
          onStrategyIdChange={onStrategyIdChange}
        />
      </div>
    );
  }

  if (code) {
    return (
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minHeight: 0 }}>
        <StrategyCodeEditor
          value={code}
          onChange={onSetCode}
          diagnostics={diagnostics}
          style={{ flex: 1, borderRadius: 0, border: 'none', minHeight: 0 }}
        />
        {auditStatus !== 'idle' && (
          <div style={{ display: 'flex', alignItems: 'center', gap: 6, padding: '4px 12px', borderTop: '1px solid var(--ant-color-border)', fontSize: 12, color: 'var(--ant-color-text-secondary)' }}>
            {auditStatus === 'checking' && <Tag color="processing">checking...</Tag>}
            {auditStatus === 'ok' && <Tooltip title={auditSummary}><CheckCircleOutlined style={{ color: '#52c41a' }} /></Tooltip>}
            {auditStatus === 'warn' && <Tooltip title={auditSummary}><WarningOutlined style={{ color: '#faad14' }} /></Tooltip>}
            {auditStatus === 'error' && <Tooltip title={auditSummary}><CloseCircleOutlined style={{ color: '#ff4d4f' }} /></Tooltip>}
            {auditStatus !== 'checking' && <span>{auditSummary}</span>}
          </div>
        )}
      </div>
    );
  }

  return (
    <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
      <div style={{ textAlign: 'center', maxWidth: 420, padding: 40 }}>
        <div style={{ fontSize: 48, marginBottom: 16 }}>📝</div>
        <div style={{ fontSize: 16, fontWeight: 600, marginBottom: 8, color: 'var(--ant-color-text)' }}>
          {t('strategy.workspace.emptyTitle', { defaultValue: 'Start building your strategy' })}
        </div>
        <div style={{ fontSize: 13, color: 'var(--ant-color-text-secondary)', marginBottom: 24, lineHeight: 1.6 }}>
          {t('strategy.workspace.emptyDesc', { defaultValue: 'Import an existing MQL EA, pick a template, or let AI generate one for you. All backtesting and deployment happens right here.' })}
        </div>
        <div style={{ display: 'flex', gap: 10, justifyContent: 'center', flexWrap: 'wrap' }}>
          <Button type="primary" icon={<ImportOutlined />} onClick={() => onSetImportMode(true)}>
            {t('strategy.workspace.importMql', { defaultValue: 'Import MQL EA' })}
          </Button>
          <Button icon={<RobotOutlined />} onClick={() => isMobile ? onSetCenterTab('chat') : onSetRightPanelTab('ai')}
            style={{ background: '#722ed1', borderColor: '#722ed1', color: '#fff' }}>
            {t('strategy.workspace.aiGenerate', { defaultValue: 'AI Generate' })}
          </Button>
          <Button icon={<HistoryOutlined />} onClick={onSelectFirstTemplate}
            disabled={templateCount === 0}>
            {t('strategy.workspace.useTemplate', { defaultValue: 'Use Template' })}
          </Button>
        </div>
      </div>
    </div>
  );
}
