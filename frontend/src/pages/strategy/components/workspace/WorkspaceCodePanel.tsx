import { Button, Space, Input, Alert, Tag, Tooltip } from 'antd';
import { CheckCircleOutlined, PlayCircleOutlined, CopyOutlined, SaveOutlined, SettingOutlined, RobotOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { useState, useMemo } from 'react';
import { useSystemAIConfigsQuery } from '@/queries/useSystemAIConfigsQuery';
import AISettingsModal from './AISettingsModal';

interface ValidationResult {
  valid: boolean;
  errors?: string[];
  warnings?: string[];
}

interface Props {
  code: string;
  onCodeChange: (code: string) => void;
  validating: boolean; onValidate: () => void;
  validationResult: ValidationResult | null;
  onRunBacktest: () => void; backtestSubmitting: boolean;
  canSave: boolean; onSave: () => void; onCopy: () => void;
}

const btnStyle: React.CSSProperties = { width: 30, height: 30, borderRadius: 6, padding: 0,
  display: 'flex', alignItems: 'center', justifyContent: 'center' };

export default function WorkspaceCodePanel({
  code, onCodeChange,
  validating, onValidate, validationResult,
  onRunBacktest, backtestSubmitting, canSave, onSave, onCopy,
}: Props) {
  const { t } = useTranslation();
  const [settingsOpen, setSettingsOpen] = useState(false);
  const { data } = useSystemAIConfigsQuery();
  const configs = data?.items ?? [];

  // Resolve saved workspace model to a display label
  const modelLabel = useMemo(() => {
    try {
      const key = localStorage.getItem('workspace_ai_model');
      if (!key) return null;
      const [providerId, model] = key.split('|');
      const cfg = configs.find(c => c.provider_id === providerId);
      const name = cfg?.name || providerId;
      return `${name} · ${model}`;
    } catch { return null; }
  }, [configs]);

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
        {/* Selected model indicator */}
        {modelLabel && (
          <div style={{ display: 'flex', alignItems: 'center', gap: 4, marginTop: 2, marginBottom: 6 }}>
            <RobotOutlined style={{ fontSize: 10, color: '#8c8c8c' }} />
            <span style={{ fontSize: 10, color: '#8c8c8c' }}>{modelLabel}</span>
          </div>
        )}
        <Input.TextArea value={code} onChange={(e) => onCodeChange(e.target.value)}
          rows={18} style={{ fontFamily: "'Fira Code', 'Cascadia Code', 'Consolas', monospace", fontSize: 13 }}
          placeholder={t('strategy.workspace.codePlaceholder', '# Python strategy code...\ndef run(context):\n    return {"signal": "hold"}')}
          spellCheck={false} />
      </div>

      {/* Validation result */}
      {validationResult && (
        <Alert
          type={validationResult.valid ? 'success' : 'warning'} showIcon
          message={validationResult.valid
            ? t('strategy.workspace.validatePass', 'Validation passed')
            : t('strategy.workspace.validateFailed', 'Validation failed')}
          description={(
            <div style={{ marginTop: 4 }}>
              {validationResult.errors?.map((e, i) => (
                <Tag key={`err-${i}`} color="error" style={{ marginBottom: 4 }}>{e}</Tag>
              ))}
              {validationResult.warnings?.map((w, i) => (
                <Tag key={`warn-${i}`} color="warning" style={{ marginBottom: 4 }}>{w}</Tag>
              ))}
            </div>
          )} />
      )}

      <AISettingsModal open={settingsOpen} onClose={() => setSettingsOpen(false)} />
    </div>
  );
}
