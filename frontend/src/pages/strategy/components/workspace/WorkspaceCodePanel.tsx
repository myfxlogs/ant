import { useState, useEffect, useMemo } from 'react';
import { Button, Space, Tooltip, Select, message } from 'antd';
import {
  CheckCircleOutlined, PlayCircleOutlined, CopyOutlined,
  SaveOutlined, SettingOutlined, RobotOutlined,
} from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { useSystemAIConfigsQuery } from '@/queries/useSystemAIConfigsQuery';
import { aiApi } from '@/client/ai';
import { discoverSystemAIModels } from '@/pages/ai/systemai/api';
import type { ValidateExtendedResult } from '@/client/codeAssist';
import StrategyCodeEditor from '@/components/strategy/StrategyCodeEditor';
import ValidationResultAlert from './ValidationResultAlert';
import AISettingsModal from './AISettingsModal';

interface Props {
  code: string;
  onCodeChange: (code: string) => void;
  validating: boolean; onValidate: () => void;
  validationResult: ValidateExtendedResult | null;
  onRunBacktest: () => void; backtestSubmitting: boolean;
  canSave: boolean; onSave: () => void; onCopy: () => void;
  onAskAI?: () => void;
  onAutoFix?: () => void;
  autoFixing?: boolean;
}

const btnStyle: React.CSSProperties = { width: 30, height: 30, borderRadius: 6, padding: 0,
  display: 'flex', alignItems: 'center', justifyContent: 'center' };

function decodeModel(value: string): { providerId: string; model: string } {
  if (!value) return { providerId: '', model: '' };
  const idx = value.indexOf('|');
  if (idx < 0) return { providerId: value, model: '' };
  return { providerId: value.slice(0, idx), model: value.slice(idx + 1) };
}

export default function WorkspaceCodePanel({
  code, onCodeChange,
  validating, onValidate, validationResult,
  onRunBacktest, backtestSubmitting, canSave, onSave, onCopy,
  onAskAI, onAutoFix, autoFixing,
}: Props) {
  const { t } = useTranslation();
  const [settingsOpen, setSettingsOpen] = useState(false);
  const { data, refetch } = useSystemAIConfigsQuery();
  const configs = data?.items ?? [];

  // Refetch provider list when AI Settings modal closes (user may have changed configs).
  const handleSettingsClose = () => { setSettingsOpen(false); refetch(); };

  // ── Default Primary Model (compact inline selector) ──
  const [primaryValue, setPrimaryValue] = useState('');
  const [primarySaving, setPrimarySaving] = useState(false);

  useEffect(() => {
    let mounted = true;
    (async () => {
      try {
        const r = await aiApi.getPrimary();
        if (mounted) setPrimaryValue(r.providerId ? `${r.providerId}|${r.model || ''}` : '');
      } catch { /* fetch failure → keep empty */ }
    })();
    return () => { mounted = false; };
  }, []);

  const modelOptions = useMemo(() => configs
    .filter(c => c.enabled && c.has_secret)
    .flatMap(c => {
      const models = c.models?.length ? c.models : c.default_model ? [c.default_model] : [];
      return models.map(m => ({
        value: `${c.provider_id}|${m}`,
        label: m,
      }));
    }), [configs]);

  // Auto-refresh model lists from providers on mount + when enabled providers change.
  const refreshKey = useMemo(() =>
    configs.filter(c => c.enabled && c.has_secret).map(c => c.provider_id).sort().join('|'),
  [configs]);
  const [refreshWarnings, setRefreshWarnings] = useState<string[]>([]);
  useEffect(() => {
    if (!refreshKey) return;
    const providers = configs.filter(c => c.enabled && c.has_secret);
    const failed: string[] = [];
    let cancelled = false;
    (async () => {
      for (const p of providers) {
        if (cancelled) break;
        try {
          await discoverSystemAIModels(p.provider_id);
        } catch {
          failed.push(p.name || p.provider_id);
        }
      }
      if (!cancelled) {
        setRefreshWarnings(failed);
        refetch();
      }
    })();
    return () => { cancelled = true; };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [refreshKey]);

  const savePrimary = async (next: string) => {
    setPrimarySaving(true);
    try {
      const dec = decodeModel(next);
      const saved = await aiApi.setPrimary({ providerId: dec.providerId, model: dec.model });
      setPrimaryValue(saved.providerId ? `${saved.providerId}|${saved.model || ''}` : '');
    } catch (e: unknown) {
      message.error(String((e as Error)?.message || 'Save failed'));
    } finally { setPrimarySaving(false); }
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      {/* Code editor */}
      <div>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 4 }}>
          <span style={{ fontSize: 13, fontWeight: 500, color: '#595959' }}>
            {t('strategy.workspace.code', 'Strategy Code')}
          </span>
          <Space size={4}>
            <Tooltip title={t('strategy.workspace.copy', 'Copy')}>
              <Button size="small" icon={<CopyOutlined />} onClick={onCopy} disabled={!code} style={btnStyle} />
            </Tooltip>
            <Tooltip title={t('strategy.workspace.validate', 'Validate')}>
              <Button size="small" icon={<CheckCircleOutlined />} loading={validating} onClick={onValidate} style={btnStyle} />
            </Tooltip>
            <Tooltip title={t('strategy.workspace.save', 'Save')}>
              <Button size="small" type="primary" icon={<SaveOutlined />} onClick={onSave} disabled={!canSave} style={btnStyle} />
            </Tooltip>
            <Tooltip title={t('strategy.workspace.runBacktest', 'Run Backtest')}>
              <Button size="small" type="primary" icon={<PlayCircleOutlined />}
                loading={backtestSubmitting} onClick={onRunBacktest} disabled={!code} style={btnStyle} />
            </Tooltip>
            <Tooltip title={t('strategy.ai.settings', 'AI Settings')}>
              <Button size="small" icon={<SettingOutlined />} onClick={() => setSettingsOpen(true)} style={btnStyle} />
            </Tooltip>
          </Space>
        </div>

        {/* Default Primary Model — inline below title, above editor */}
        <div style={{
          display: 'flex', alignItems: 'center', gap: 6,
          marginBottom: 6, padding: '4px 8px',
          background: '#fafafa', borderRadius: 4, border: '1px solid #f0f0f0',
        }}>
          <RobotOutlined style={{ fontSize: 11, color: '#8c8c8c' }} />
          <Select
            size="small"
            style={{ flex: 1, minWidth: 0 }}
            variant="borderless"
            value={primaryValue || undefined}
            placeholder={t('ai.settings.primary.placeholder', { defaultValue: 'Default AI model...' })}
            options={modelOptions}
            onChange={(v) => setPrimaryValue(v || '')}
            showSearch
            optionFilterProp="label"
            notFoundContent={t('ai.settings.agent.fields.modelProfileEmpty', { defaultValue: 'No model — configure in AI Settings' })}
          />
          {primaryValue && (
            <Button size="small" type="link" loading={primarySaving}
              onClick={() => savePrimary(primaryValue)}
              style={{ padding: 0, fontSize: 12 }}>
              {t('common.save', { defaultValue: 'Save' })}
            </Button>
          )}
        </div>

        {/* Model refresh warnings */}
        {refreshWarnings.length > 0 && (
          <div style={{
            marginBottom: 6, padding: '3px 8px', fontSize: 11,
            color: '#ad6800', background: '#fffbe6', borderRadius: 4,
            border: '1px solid #ffe58f',
          }}>
            ⚠ {t('strategy.ai.refreshFailed', { defaultValue: 'Model list refresh failed for' })}:{' '}
            {refreshWarnings.join(', ')}.{' '}
            <span style={{ cursor: 'pointer', textDecoration: 'underline' }}
              onClick={() => setSettingsOpen(true)}>
              {t('strategy.ai.checkSettings', { defaultValue: 'Check AI Settings →' })}
            </span>
          </div>
        )}

        <StrategyCodeEditor
          value={code}
          onChange={onCodeChange}
          style={{ height: 420 }}
        />
      </div>

      <ValidationResultAlert
        validationResult={validationResult}
        autoFixing={autoFixing} onAutoFix={onAutoFix} onAskAI={onAskAI}
      />

      <AISettingsModal open={settingsOpen} onClose={handleSettingsClose} />
    </div>
  );
}
