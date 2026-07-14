import { useState, useEffect, useMemo } from 'react';
import { Button, Space, Tooltip, Select, message, Segmented, Tag } from 'antd';
import {
  CheckCircleOutlined, CopyOutlined,
  SaveOutlined, SettingOutlined, RobotOutlined,
  ThunderboltOutlined, KeyOutlined, HistoryOutlined,
} from '@ant-design/icons';
import { useTranslation } from 'react-i18next'
import { CODE_KEY, COPY_KEY, RUNTIME_MODE_KEY, SAVE_FAILED_KEY, SAVE_KEY, VALIDATE_KEY } from '@/gen/ant/v1/i18n/strategy_workspace_keys';
import { AGENT_FIELDS_MODEL_PROFILE_EMPTY_KEY, PRIMARY_PLACEHOLDER_KEY } from '@/gen/ant/v1/i18n/ai_settings_keys';
import { CHECK_SETTINGS_KEY, REFRESH_FAILED_KEY, SETTINGS_KEY } from '@/gen/ant/v1/i18n/strategy_ai_keys';
import { GATEWAY_GROUP_MY_KEYS_KEY, GATEWAY_GROUP_GATEWAY_KEY, GATEWAY_GROUP_CURRENT_KEY } from '@/gen/ant/v1/i18n/ai_core_keys';
import { EVENT_DRIVEN_MODE_KEY, VECTORIZED_MODE_KEY } from '@/gen/ant/v1/i18n/strategy_backtest_params_keys';

;
import { useSystemAIConfigsQuery } from '@/queries/useSystemAIConfigsQuery';
import { aiApi } from '@/client/ai';
import { aiGatewayApi } from '@/client/aiGateway';
import { discoverSystemAIModels } from '@/pages/ai/systemai/api';
import type { ValidateExtendedResult } from '@/client/codeAssist';
import type { AutoFixDebug } from '@/pages/strategy/hooks/useAIWorkflow';
import StrategyCodeEditor from '@/components/strategy/StrategyCodeEditor';
import ValidationResultAlert from './ValidationResultAlert';
import AISettingsModal from './AISettingsModal';

interface Props {
  code: string;
  onCodeChange: (code: string) => void;
  validating: boolean; onValidate: () => void;
  validationResult: ValidateExtendedResult | null;
  onRunBacktest?: () => void; backtestSubmitting?: boolean;
  canSave: boolean; onSave: () => void; onCopy: () => void;
  onAskAI?: () => void;
  onAutoFix?: () => void;
  autoFixing?: boolean;
  autoFixDebug?: AutoFixDebug | null;
  onDismissDebug?: () => void;
  onShowHistory?: () => void;
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
  onAskAI, onAutoFix, autoFixing, autoFixDebug, onDismissDebug,
  onShowHistory,
}: Props) {
  const { t } = useTranslation();
  const [settingsOpen, setSettingsOpen] = useState(false);
  const { data, refetch } = useSystemAIConfigsQuery();
  const configs = data?.items ?? [];

  // Refetch provider list and primary when AI Settings modal closes.
  const handleSettingsClose = () => { setSettingsOpen(false); refetch(); refreshPrimary(); };

  // ── Default Primary Model (compact inline selector) ──
  const [primaryValue, setPrimaryValue] = useState('');
  const [primarySaving, setPrimarySaving] = useState(false);
  const [gatewayModels, setGatewayModels] = useState<Array<{ providerId: string; model: string; label: string }>>([]);

  const refreshPrimary = async () => {
    try {
      const r = await aiApi.getPrimary();
      setPrimaryValue(r.providerId ? `${r.providerId}|${r.model || ''}` : '');
    } catch { /* fetch failure → keep current */ }
  };

  useEffect(() => { refreshPrimary(); }, []);

  // Fetch gateway models for workspace selector (available even without own API key).
  useEffect(() => {
    let mounted = true;
    (async () => {
      try {
        const list = await aiGatewayApi.listSystemModels();
        if (mounted) {
          setGatewayModels(list.map(m => ({
            providerId: m.providerId, model: m.modelName,
            label: m.displayName || m.modelName,
          })));
        }
      } catch { /* best effort */ }
    })();
    return () => { mounted = false; };
  }, []);

  const modelOptions = useMemo(() => {
    const seen = new Set<string>();
    const groups: Array<{ label: React.ReactNode; title: string; options: Array<{ value: string; label: React.ReactNode; searchLabel: string }> }> = [];
    // User's own API key models.
    const ownItems: Array<{ value: string; label: React.ReactNode; searchLabel: string }> = [];
    configs
      .filter(c => c.enabled && c.has_secret)
      .flatMap(c => {
        const models = c.models?.length ? c.models : c.default_model ? [c.default_model] : [];
        return models.map(m => ({ value: `${c.provider_id}|${m}`, label: m }));
      }).forEach(o => {
        if (!seen.has(o.value)) {
          seen.add(o.value);
          ownItems.push({
            value: o.value,
            label: <span style={{ display: 'flex', alignItems: 'center', gap: 6 }}><KeyOutlined style={{ color: '#1677ff', fontSize: 12 }} />{o.label}</span>,
            searchLabel: o.label,
          });
        }
      });
    if (ownItems.length > 0) {
      groups.push({ label: <span style={{ fontSize: 11, color: '#1677ff', fontWeight: 600 }}><KeyOutlined style={{ fontSize: 11 }} /> {t(GATEWAY_GROUP_MY_KEYS_KEY)}</span>, title: 'My Keys', options: ownItems });
    }
    // Gateway models (platform-provided).
    const gwItems: Array<{ value: string; label: React.ReactNode; searchLabel: string }> = [];
    gatewayModels.forEach(g => {
      const v = `${g.providerId}|${g.model}`;
      if (!seen.has(v)) {
        seen.add(v);
        gwItems.push({
          value: v,
          label: <span style={{ display: 'flex', alignItems: 'center', gap: 6 }}><ThunderboltOutlined style={{ color: '#722ed1', fontSize: 12 }} />{g.label}</span>,
          searchLabel: g.label,
        });
      }
    });
    if (gwItems.length > 0) {
      groups.push({ label: <span style={{ fontSize: 11, color: '#722ed1', fontWeight: 600 }}><ThunderboltOutlined style={{ fontSize: 11 }} /> {t(GATEWAY_GROUP_GATEWAY_KEY)}</span>, title: 'Gateway', options: gwItems });
    }
    // Always include the currently saved primary if not in any group.
    if (primaryValue && !seen.has(primaryValue)) {
      const [pid, mdl] = primaryValue.split('|');
      if (pid && mdl) {
        groups.unshift({
          label: <span style={{ fontSize: 11, color: '#faad14', fontWeight: 600 }}>{t(GATEWAY_GROUP_CURRENT_KEY)}</span>, title: 'Current', options: [{
            value: primaryValue,
            label: <span style={{ display: 'flex', alignItems: 'center', gap: 6 }}>{mdl}</span>,
            searchLabel: mdl,
          }],
        });
      }
    }
    return groups;
  }, [configs, primaryValue, gatewayModels]);

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
      message.error(String((e as Error)?.message || t(SAVE_FAILED_KEY)));
    } finally { setPrimarySaving(false); }
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      {/* Code editor */}
      <div>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 4 }}>
          <span style={{ fontSize: 13, fontWeight: 500, color: '#595959' }}>
            {t(CODE_KEY, 'Strategy Code')}
          </span>
          <Space size={4}>
            <Tooltip title={t(COPY_KEY, 'Copy')}>
              <Button size="small" icon={<CopyOutlined />} onClick={onCopy} disabled={!code} style={btnStyle} />
            </Tooltip>
            <Tooltip title={t(VALIDATE_KEY, 'Validate')}>
              <Button size="small" icon={<CheckCircleOutlined />} loading={validating} onClick={onValidate} style={btnStyle} />
            </Tooltip>
            <Tooltip title={t(SAVE_KEY, 'Save')}>
              <Button size="small" type="primary" icon={<SaveOutlined />} onClick={onSave} disabled={!canSave} style={btnStyle} />
            </Tooltip>
            {onShowHistory && (
              <Tooltip title={t('strategy.version.history', { defaultValue: 'Version History' })}>
                <Button size="small" icon={<HistoryOutlined />} onClick={onShowHistory} style={btnStyle} />
              </Tooltip>
            )}
            <Tooltip title={t(SETTINGS_KEY, 'AI Settings')}>
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
          {modelOptions.length === 0 ? (
            <span style={{ flex: 1, fontSize: 12, color: '#8c8c8c' }}>
              {t(PRIMARY_PLACEHOLDER_KEY, { defaultValue: 'Loading models...' })}
            </span>
          ) : (
            <Select
              size="small"
              style={{ flex: 1, minWidth: 0 }}
              variant="borderless"
              value={primaryValue || undefined}
              placeholder={t(PRIMARY_PLACEHOLDER_KEY, { defaultValue: 'Default AI model...' })}
              options={modelOptions}
              onChange={(v) => { if (v) savePrimary(v); }}
              showSearch
              filterOption={(input, option) => {
                if (!option) return false;
                const label = option.label as any;
                const text = typeof label === 'string' ? label : (label?.props?.children?.[1] || label?.props?.children || '');
                return String(text).toLowerCase().includes(input.toLowerCase());
              }}
              notFoundContent={t(AGENT_FIELDS_MODEL_PROFILE_EMPTY_KEY, { defaultValue: 'No model — configure in AI Settings' })}
            />
          )}
          {primarySaving && <span style={{ fontSize: 11, color: '#1677ff' }}>{t('common.saving', { defaultValue: 'saving...' })}</span>}
        </div>

        {/* Model refresh warnings */}
        {refreshWarnings.length > 0 && (
          <div style={{
            marginBottom: 6, padding: '3px 8px', fontSize: 11,
            color: '#ad6800', background: '#fffbe6', borderRadius: 4,
            border: '1px solid #ffe58f',
          }}>
            ⚠ {t(REFRESH_FAILED_KEY, { defaultValue: 'Model list refresh failed for' })}:{' '}
            {refreshWarnings.join(', ')}.{' '}
            <span style={{ cursor: 'pointer', textDecoration: 'underline' }}
              onClick={() => setSettingsOpen(true)}>
              {t(CHECK_SETTINGS_KEY, { defaultValue: 'Check AI Settings →' })}
            </span>
          </div>
        )}

        {/* Strategy runtime mode indicator */}
        {validationResult?.strategyType && (
          <div style={{
            display: 'flex', alignItems: 'center', gap: 8,
            padding: '2px 0', marginBottom: 4,
          }}>
            <span style={{ fontSize: 10, color: '#8c8c8c' }}>
              {t(RUNTIME_MODE_KEY, 'Runtime')}:
            </span>
            <Tag color={validationResult.strategyType === 'run_dataframe' ? 'green' : 'blue'}
              style={{ fontSize: 10, margin: 0, lineHeight: '18px' }}>
              {validationResult.strategyType === 'run_dataframe'
                ? t(VECTORIZED_MODE_KEY, 'Vectorized')
                : t(EVENT_DRIVEN_MODE_KEY, 'Run(context)')}
            </Tag>
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
        autoFixing={autoFixing} autoFixDebug={autoFixDebug}
        onAutoFix={onAutoFix} onAskAI={onAskAI} onDismissDebug={onDismissDebug}
      />

      <AISettingsModal open={settingsOpen} onClose={handleSettingsClose} />
    </div>
  );
}
