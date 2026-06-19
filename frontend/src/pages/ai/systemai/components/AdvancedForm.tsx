import { Button, Switch, Slider, InputNumber, Checkbox } from 'antd';
import { CheckOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next'
import { ACTIONS_SAVE_CONFIG_KEY, FIELDS_ENABLED_OFF_KEY, FIELDS_ENABLED_ON_KEY, FIELDS_ENABLED_STATUS_KEY, FIELDS_MAX_TOKENS_KEY, FIELDS_TEMPERATURE_KEY, FIELDS_TIMEOUT_SECONDS_KEY, SECTIONS_ADVANCED_HINT_KEY, SECTIONS_ADVANCED_KEY } from '@/gen/ant/v1/i18n/ai_settings_keys';

;
import { ALL_PURPOSES } from '../constants';
import { Section, Label, SoftTag } from './SharedComponents';
import type { AIConfig } from '../model';

export function AdvancedSection(props: {
  draft: AIConfig;
  onDraftChange: (patch: Partial<AIConfig>) => void;
}) {
  const { t } = useTranslation();
  const { draft, onDraftChange } = props;

  return (
    <Section
      step={3}
      title={t(SECTIONS_ADVANCED_KEY, { defaultValue: '高级参数' })}
      subtitle={t(SECTIONS_ADVANCED_HINT_KEY, { defaultValue: '仅在了解含义时调整；默认值已适配大多数场景' })}
    >
      <div className="space-y-6">
        <div>
          <Label text={t(FIELDS_ENABLED_STATUS_KEY, { defaultValue: '启用状态' })} hint={t(SYSTEM_A_I_FIELDS_ENABLED_HINT_KEY, { defaultValue: '关闭后该厂商不参与系统路由' })} />
          <label className="flex items-center gap-3 cursor-pointer select-none">
            <Switch checked={draft.enabled} onChange={(v) => onDraftChange({ enabled: v })} />
            <span className="text-sm text-gray-700">
              {draft.enabled ? (
                <span className="text-slate-700 font-medium">{t(FIELDS_ENABLED_ON_KEY, { defaultValue: '已启用 → 点击关闭' })}</span>
              ) : (
                <span className="text-gray-500">{t(FIELDS_ENABLED_OFF_KEY, { defaultValue: '未启用 → 点击开启' })}</span>
              )}
            </span>
          </label>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div>
            <Label text={`${t(FIELDS_TEMPERATURE_KEY, { defaultValue: 'Temperature' })}（${draft.temperature}）`} hint={t(SYSTEM_A_I_FIELDS_TEMPERATURE_HINT_KEY, { defaultValue: '越高越发散，越低越稳定' })} />
            <Slider
              min={0}
              max={2}
              step={0.1}
              value={draft.temperature}
              onChange={(v) => onDraftChange({ temperature: Number(v) })}
            />
          </div>
          <div>
            <Label text={t(FIELDS_TIMEOUT_SECONDS_KEY, { defaultValue: 'Timeout（秒）' })} hint={t(SYSTEM_A_I_FIELDS_TIMEOUT_HINT_KEY, { defaultValue: '单次请求最长等待时间' })} />
            <InputNumber
              value={draft.timeout_seconds}
              min={1}
              onChange={(v) => onDraftChange({ timeout_seconds: Number(v || 0) })}
              style={{ width: '100%' }}
            />
          </div>
          <div>
            <Label text={t(FIELDS_MAX_TOKENS_KEY, { defaultValue: 'Max Tokens' })} hint={t(SYSTEM_A_I_FIELDS_MAX_TOKENS_HINT_KEY, { defaultValue: '单次响应最大 token 数' })} />
            <InputNumber
              value={draft.max_tokens}
              min={1}
              onChange={(v) => onDraftChange({ max_tokens: Number(v || 0) })}
              style={{ width: '100%' }}
            />
          </div>
        </div>

        <div>
          <Label
            text={t(SYSTEM_A_I_FIELDS_PRIMARY_FOR_KEY, { defaultValue: '主要用途（Primary For）' })}
            hint={t(SYSTEM_A_I_FIELDS_PRIMARY_FOR_HINT_KEY, { defaultValue: '仅用于服务内部路由：chat / embedding / summarizer / reasoning' })}
          />
          <Checkbox.Group
            value={draft.primary_for}
            onChange={(vals) => onDraftChange({ primary_for: vals as string[] })}
            options={ALL_PURPOSES.map((p) => ({ label: p, value: p }))}
          />
        </div>
      </div>
    </Section>
  );
}

export function SaveSection(props: {
  draft: AIConfig;
  savingConfig: boolean;
  validating: boolean;
  validated: boolean;
  hasError: boolean;
  onSave: () => void;
}) {
  const { t } = useTranslation();
  const { draft, savingConfig, validating, validated, hasError, onSave } = props;

  return (
    <Section
      step={4}
      title={t(ACTIONS_SAVE_CONFIG_KEY, { defaultValue: '保存配置' })}
    >
      <div className="flex items-center justify-between">
        <div className="text-sm text-gray-600 flex items-center gap-2 flex-wrap">
          {draft.enabled
            ? <SoftTag>{t(SYSTEM_A_I_STATUS_BAR_ENABLED_KEY, { defaultValue: '已启用' })}</SoftTag>
            : <SoftTag>{t(SYSTEM_A_I_STATUS_BAR_DISABLED_KEY, { defaultValue: '未启用' })}</SoftTag>}
          {draft.has_secret && <SoftTag>{t(SYSTEM_A_I_STATUS_BAR_KEY_READY_KEY, { defaultValue: '密钥就绪' })}</SoftTag>}
          {validating && <SoftTag>{t(SYSTEM_A_I_STATUS_BAR_CHECKING_KEY, { defaultValue: '连通性检测中…' })}</SoftTag>}
          {!validating && validated && <SoftTag>{t(SYSTEM_A_I_STATUS_BAR_CONNECTED_KEY, { defaultValue: '连接正常' })}</SoftTag>}
          {!validating && !validated && (draft.models || []).length > 0 && hasError && (
            <span className="text-xs text-slate-600">{t(SYSTEM_A_I_STATUS_CONNECTION_FAILED_KEY, { defaultValue: '连接异常，请检查上方提示' })}</span>
          )}
        </div>
        <Button
          size="large"
          onClick={onSave}
          loading={savingConfig}
          icon={<CheckOutlined style={{ fontSize: 16 }} />}
          type="primary"
        >
          {t(ACTIONS_SAVE_CONFIG_KEY, { defaultValue: '保存配置' })}
        </Button>
      </div>
    </Section>
  );
}
