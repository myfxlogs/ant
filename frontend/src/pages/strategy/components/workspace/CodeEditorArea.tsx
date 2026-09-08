import { useState, useEffect, useRef, useCallback } from 'react';
import { Button, Tag, Tooltip, notification } from 'antd';
import { ImportOutlined, RobotOutlined, HistoryOutlined, CheckCircleOutlined, WarningOutlined, CloseCircleOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import StrategyCodeEditor, { type Diagnostic } from '@/components/strategy/StrategyCodeEditor';
import ImportEAPanel from '../editor/ImportEAPanel';
import { strategyVersionApi } from '@/client/strategy';
import type { BlindSpot } from '@/gen/ant/v1/strategy_runtime_pb';
import {
  AUDIT_CHECKING_KEY, AUDIT_COMPILE_FAILED_KEY, AUDIT_BLIND_SPOTS_KEY, AUDIT_ALL_CLEAR_KEY,
  BACK_TO_EDITOR_KEY, EMPTY_TITLE_KEY, EMPTY_DESC_KEY, IMPORT_MQL_KEY, AI_GENERATE_KEY, USE_TEMPLATE_KEY,
} from '@/gen/ant/v1/i18n/strategy_workspace_keys';

interface Props {
  code: string;
  importMode: boolean;
  isMobile: boolean;
  templateCount: number;
  onSetImportMode: (v: boolean) => void;
  onSetCode: (c: string) => void;
  onSetCenterTab: (tab: 'chat' | 'code') => void;
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
  const [compileError, setCompileError] = useState<string>('');
  const debounceTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const lastCheckedCode = useRef('');
  const prevStatusRef = useRef<'idle' | 'checking' | 'ok' | 'warn' | 'error'>('idle');

  const runCheck = useCallback(async (sourceCode: string) => {
    if (!sourceCode.trim() || sourceCode.trim().length < 20) {
      setDiagnostics([]);
      setCompileError('');
      setAuditStatus('idle');
      prevStatusRef.current = 'idle';
      return;
    }
    setAuditStatus('checking');
    try {
      const resp = await strategyVersionApi.checkCode(sourceCode);
      const errText = resp.compileError || '';
      const diags = blindSpotsToDiagnostics(resp.blindSpots, errText || undefined);
      setDiagnostics(diags);
      setCompileError(errText);
      let next: 'ok' | 'warn' | 'error';
      if (!resp.compileSuccess) {
        next = 'error';
        setAuditSummary(t(AUDIT_COMPILE_FAILED_KEY));
      } else if (resp.blindSpots.length > 0) {
        next = 'warn';
        setAuditSummary(t(AUDIT_BLIND_SPOTS_KEY, { count: resp.blindSpots.length, percent: (resp.coverageScore * 100).toFixed(0) }));
      } else {
        next = 'ok';
        setAuditSummary(t(AUDIT_ALL_CLEAR_KEY, { percent: (resp.coverageScore * 100).toFixed(0) }));
      }
      setAuditStatus(next);
      // 醒目提示只在「进入失败态」时弹一次——逐字编辑触发的重复检查不打扰。
      if (next === 'error' && prevStatusRef.current !== 'error') {
        notification.error({
          message: t(AUDIT_COMPILE_FAILED_KEY),
          description: (
            <div>
              <div style={{ whiteSpace: 'pre-wrap', maxHeight: 160, overflowY: 'auto' }}>{errText}</div>
              <div style={{ marginTop: 6, color: 'var(--ant-color-text-secondary)' }}>
                {t('ai.workspace.compileNotifyHint', { defaultValue: '打开 AI 助手即可修复——失败原因会自动作为上下文发给 AI。' })}
              </div>
            </div>
          ),
          placement: 'bottomRight',
          duration: 8,
        });
      }
      prevStatusRef.current = next;
    } catch {
      setAuditStatus('idle');
      prevStatusRef.current = 'idle';
    }
  }, [t]);

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
            ← {t(BACK_TO_EDITOR_KEY)}
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
          <div style={{ display: 'flex', alignItems: 'center', gap: 6, padding: '4px 12px', borderTop: '1px solid var(--ant-color-border)', fontSize: 12, color: 'var(--ant-color-text-secondary)', minWidth: 0 }}>
            {auditStatus === 'checking' && <Tag color="processing">{t(AUDIT_CHECKING_KEY)}</Tag>}
            {auditStatus === 'ok' && <Tooltip title={auditSummary}><CheckCircleOutlined style={{ color: 'var(--color-success)' }} /></Tooltip>}
            {auditStatus === 'warn' && <Tooltip title={auditSummary}><WarningOutlined style={{ color: 'var(--color-warning)' }} /></Tooltip>}
            {auditStatus === 'error' && (
              <Tooltip title={compileError || auditSummary}>
                <CloseCircleOutlined style={{ color: 'var(--color-danger)', flexShrink: 0 }} />
              </Tooltip>
            )}
            {auditStatus === 'error' ? (
              <Tooltip title={compileError || auditSummary}>
                <span style={{ color: 'var(--color-danger)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                  {auditSummary}{compileError ? `：${compileError.split('\n')[0].slice(0, 160)}` : ''}
                </span>
              </Tooltip>
            ) : auditStatus !== 'checking' && <span>{auditSummary}</span>}
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
          {t(EMPTY_TITLE_KEY)}
        </div>
        <div style={{ fontSize: 13, color: 'var(--ant-color-text-secondary)', marginBottom: 24, lineHeight: 1.6 }}>
          {t(EMPTY_DESC_KEY)}
        </div>
        <div style={{ display: 'flex', gap: 10, justifyContent: 'center', flexWrap: 'wrap' }}>
          <Button type="primary" icon={<ImportOutlined />} onClick={() => onSetImportMode(true)}>
            {t(IMPORT_MQL_KEY)}
          </Button>
          <Button icon={<RobotOutlined />} onClick={() => isMobile ? onSetCenterTab('chat') : onSetRightPanelTab('ai')}
            style={{ background: 'var(--color-ai-btn)', borderColor: 'var(--color-ai-btn)', color: '#fff' }}>
            {t(AI_GENERATE_KEY)}
          </Button>
          <Button icon={<HistoryOutlined />} onClick={onSelectFirstTemplate}
            disabled={templateCount === 0}>
            {t(USE_TEMPLATE_KEY)}
          </Button>
        </div>
      </div>
    </div>
  );
}
