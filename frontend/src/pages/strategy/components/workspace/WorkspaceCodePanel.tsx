import { Button, Space, Input, Alert, Tag, Tooltip } from 'antd';
import { CheckCircleOutlined, PlayCircleOutlined, CopyOutlined, SaveOutlined, ThunderboltOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';

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
  // AI Generate (optional)
  aiPrompt?: string; onAiPromptChange?: (v: string) => void;
  aiGenerating?: boolean; onGenerateCode?: () => void;
}

export default function WorkspaceCodePanel({
  code, onCodeChange,
  validating, onValidate, validationResult,
  onRunBacktest, backtestSubmitting, canSave, onSave, onCopy,
  aiPrompt, onAiPromptChange, aiGenerating, onGenerateCode,
}: Props) {
  const { t } = useTranslation();

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      {/* Code editor */}
      <div>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 4 }}>
          <span style={{ fontSize: 13, fontWeight: 500, color: '#595959' }}>
            {t('strategy.workspace.code', 'Strategy Code')}
          </span>
          <Space size="small">
            <Tooltip title={t('strategy.workspace.copy', 'Copy')}>
              <Button size="small" icon={<CopyOutlined />} onClick={onCopy} disabled={!code} />
            </Tooltip>
            <Button size="small" icon={<CheckCircleOutlined />} loading={validating} onClick={onValidate}>
              {t('strategy.workspace.validate', 'Validate')}
            </Button>
            <Button size="small" type="primary" icon={<SaveOutlined />} onClick={onSave} disabled={!canSave}>
              {t('strategy.workspace.save', 'Save')}
            </Button>
            <Button size="small" type="primary" icon={<PlayCircleOutlined />}
              loading={backtestSubmitting} onClick={onRunBacktest} disabled={!code}>
              {t('strategy.workspace.runBacktest', 'Run Backtest')}
            </Button>
          </Space>
        </div>
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

      {/* AI Generate section */}
      {onGenerateCode && (
        <div style={{
          borderTop: '1px solid #e8e8e8', paddingTop: 8,
          background: '#fafbfc', borderRadius: 6,
          border: '1px solid #e8e8e8',
        }}>
          <div style={{ padding: '6px 12px', borderBottom: '1px solid #e8e8e8', background: 'linear-gradient(180deg, #f0f7ff 0%, #e6f4ff 100%)' }}>
            <span style={{ fontSize: 11, fontWeight: 700, color: '#262626' }}>
              <ThunderboltOutlined style={{ marginRight: 4, color: '#1890ff' }} />
              AI Generate
            </span>
          </div>
          <div style={{ padding: '8px 12px', display: 'flex', flexDirection: 'column', gap: 8 }}>
            <Input.TextArea
              value={aiPrompt} onChange={(e) => onAiPromptChange?.(e.target.value)}
              rows={3} size="small" placeholder="Describe the strategy you want to generate..."
              spellCheck={false} />
            <Button type="primary" block size="small" loading={aiGenerating}
              disabled={!aiPrompt?.trim()} onClick={onGenerateCode}
              style={{ borderRadius: 6, fontWeight: 600, boxShadow: '0 2px 8px rgba(24,144,255,0.25)' }}>
              ⚡ Generate Code
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}
