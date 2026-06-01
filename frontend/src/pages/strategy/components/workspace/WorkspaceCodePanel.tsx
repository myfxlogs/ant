import { Button, Space, Input, Alert, Tag, Tooltip } from 'antd';
import { CheckCircleOutlined, PlayCircleOutlined, CopyOutlined, SaveOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import SymbolPicker from '@/components/chart/SymbolPicker';

const TIMEFRAMES = [
  { label: 'M1', value: '1m' },
  { label: 'M5', value: '5m' },
  { label: 'M15', value: '15m' },
  { label: 'M30', value: '30m' },
  { label: 'H1', value: '1h' },
  { label: 'H4', value: '4h' },
  { label: 'D1', value: '1d' },
  { label: 'W1', value: '1w' },
];

interface ValidationResult {
  valid: boolean;
  errors?: string[];
  warnings?: string[];
}

interface Props {
  code: string;
  onCodeChange: (code: string) => void;
  accountId: string;
  onAccountIdChange: (id: string) => void;
  symbol: string;
  onSymbolChange: (symbol: string) => void;
  timeframe: string;
  onTimeframeChange: (tf: string) => void;
  validating: boolean;
  onValidate: () => void;
  validationResult: ValidationResult | null;
  onRunBacktest: () => void;
  backtestSubmitting: boolean;
  canSave: boolean;
  onSave: () => void;
  onCopy: () => void;
}

export default function WorkspaceCodePanel({
  code, onCodeChange, accountId, onAccountIdChange,
  symbol, onSymbolChange, timeframe, onTimeframeChange,
  validating, onValidate, validationResult,
  onRunBacktest, backtestSubmitting, canSave, onSave, onCopy,
}: Props) {
  const { t } = useTranslation();

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 12, padding: '0 4px' }}>
      {/* Symbol selection row */}
      <Space wrap size="small">
        <SymbolPicker
          accountId={accountId}
          value={symbol}
          onChange={onSymbolChange}
          placeholder={t('strategy.workspace.selectSymbol', 'Symbol')}
          style={{ width: 140 }}
        />
        <select
          value={timeframe}
          onChange={(e) => onTimeframeChange(e.target.value)}
          style={{
            padding: '4px 8px', borderRadius: 6, border: '1px solid #d9d9d9',
            fontSize: 13, background: '#fff', cursor: 'pointer',
          }}
        >
          {TIMEFRAMES.map((tf) => (
            <option key={tf.value} value={tf.value}>{tf.label}</option>
          ))}
        </select>
        <span style={{ fontSize: 12, color: '#8c8c8c' }}>
          {t('strategy.workspace.account', 'Account')}:
        </span>
        <input
          value={accountId}
          onChange={(e) => onAccountIdChange(e.target.value)}
          placeholder={t('strategy.workspace.accountPlaceholder', 'Account ID')}
          style={{
            padding: '4px 8px', borderRadius: 6, border: '1px solid #d9d9d9',
            fontSize: 13, width: 120,
          }}
        />
      </Space>

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
              loading={backtestSubmitting} onClick={onRunBacktest}
              disabled={!code || !symbol}
            >
              {t('strategy.workspace.runBacktest', 'Run Backtest')}
            </Button>
          </Space>
        </div>
        <Input.TextArea
          value={code}
          onChange={(e) => onCodeChange(e.target.value)}
          rows={18}
          style={{ fontFamily: "'Fira Code', 'Cascadia Code', 'Consolas', monospace", fontSize: 13 }}
          placeholder={t('strategy.workspace.codePlaceholder', '# Python strategy code...\ndef run(context):\n    return {"signal": "hold"}')}
          spellCheck={false}
        />
      </div>

      {/* Validation result */}
      {validationResult && (
        <Alert
          type={validationResult.valid ? 'success' : 'warning'}
          showIcon
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
          )}
        />
      )}
    </div>
  );
}
