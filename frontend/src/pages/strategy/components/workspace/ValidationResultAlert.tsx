import { Alert, Space, Button } from 'antd';
import { RobotOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import type { ValidateExtendedResult } from '@/client/codeAssist';

interface Props {
  validationResult: ValidateExtendedResult;
  autoFixing?: boolean;
  onAutoFix?: () => void;
  onAskAI?: () => void;
}

export default function ValidationResultAlert({
  validationResult, autoFixing, onAutoFix, onAskAI,
}: Props) {
  const { t } = useTranslation();
  if (!validationResult) return null;

  return (
    <Alert
      type={validationResult.valid ? 'success' : 'warning'} showIcon
      message={validationResult.valid
        ? t('strategy.workspace.validatePass', 'Validation passed')
        : t('strategy.workspace.validateFailed', 'Validation failed')}
      description={
        !validationResult.valid ? (
          <div style={{
            maxHeight: 220, overflowY: 'auto',
            marginTop: 4, padding: '6px 8px',
            background: '#fff', borderRadius: 4,
            border: '1px solid #f0f0f0',
          }}>
            {validationResult.errors?.map((e, i) => (
              <div key={`err-${i}`} style={{
                fontSize: 11, lineHeight: '1.6', color: '#cf1322',
                padding: '2px 0', borderBottom: '1px solid #fff1f0',
                wordBreak: 'break-word',
              }}>
                <span style={{ fontWeight: 600, marginRight: 4 }}>✕</span>
                {e}
              </div>
            ))}
            {validationResult.warnings?.map((w, i) => (
              <div key={`warn-${i}`} style={{
                fontSize: 11, lineHeight: '1.6', color: '#ad6800',
                padding: '2px 0', borderBottom: '1px solid #fffbe6',
                wordBreak: 'break-word',
              }}>
                <span style={{ fontWeight: 600, marginRight: 4 }}>⚠</span>
                {w}
              </div>
            ))}
          </div>
        ) : undefined
      }
      action={
        !validationResult.valid ? (
          <Space direction="vertical" size={4}>
            {onAutoFix && (
              <Button size="small" type="primary" icon={<RobotOutlined />}
                loading={autoFixing} onClick={onAutoFix}>
                {autoFixing ? 'Fixing...' : 'Auto Fix'}
              </Button>
            )}
            {onAskAI && (
              <Button size="small" type="primary" ghost icon={<RobotOutlined />}
                onClick={onAskAI}>
                Ask AI
              </Button>
            )}
          </Space>
        ) : undefined
      }
    />
  );
}
