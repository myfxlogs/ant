import { Alert, Space, Button, Tag } from 'antd';
import { RobotOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import type { ValidateExtendedResult } from '@/client/codeAssist';

const SEVERITY_COLOR: Record<string, string> = {
  error: '#cf1322',
  warn: '#ad6800',
  info: '#1677ff',
};

const CATEGORY_LABEL: Record<string, string> = {
  FUTURE_DATA_LEAK: 'Future Data Leak',
  MISSING_PARAM: 'Missing Param',
  UNREAD_PARAM: 'Unread Param',
};

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

  const { errors, warnings, qualityHints, valid } = validationResult;
  const hasIssues = errors.length > 0 || warnings.length > 0 || qualityHints.length > 0;

  return (
    <Alert
      type={valid ? 'success' : 'warning'} showIcon
      message={valid
        ? t('strategy.workspace.validatePass', 'Validation passed')
        : t('strategy.workspace.validateFailed', 'Validation failed')}
      description={
        hasIssues ? (
          <div style={{
            maxHeight: 260, overflowY: 'auto',
            marginTop: 4, padding: '6px 8px',
            background: '#fff', borderRadius: 4,
            border: '1px solid #f0f0f0',
          }}>
            {errors.map((e, i) => (
              <div key={`err-${i}`} style={{
                fontSize: 11, lineHeight: '1.6', color: '#cf1322',
                padding: '2px 0', borderBottom: '1px solid #fff1f0',
                wordBreak: 'break-word',
              }}>
                <span style={{ fontWeight: 600, marginRight: 4 }}>✕</span>
                {e}
              </div>
            ))}
            {warnings.map((w, i) => (
              <div key={`warn-${i}`} style={{
                fontSize: 11, lineHeight: '1.6', color: '#ad6800',
                padding: '2px 0', borderBottom: '1px solid #fffbe6',
                wordBreak: 'break-word',
              }}>
                <span style={{ fontWeight: 600, marginRight: 4 }}>⚠</span>
                {w}
              </div>
            ))}
            {qualityHints.map((h, i) => (
              <div key={`hint-${i}`} style={{
                fontSize: 11, lineHeight: '1.6',
                color: SEVERITY_COLOR[h.severity] || '#666',
                padding: '3px 0', borderBottom: '1px solid #f5f5f5',
              }}>
                <Tag color={h.severity === 'error' ? 'red' : h.severity === 'warn' ? 'orange' : 'blue'}
                  style={{ fontSize: 9, lineHeight: '16px', marginRight: 4 }}>
                  {CATEGORY_LABEL[h.category] || h.category}
                </Tag>
                <span style={{ fontWeight: 500 }}>{h.message}</span>
                {h.line > 0 && (
                  <span style={{ color: '#8c8c8c', marginLeft: 4 }}>
                    (line {h.line})
                  </span>
                )}
              </div>
            ))}
          </div>
        ) : undefined
      }
      action={
        !valid ? (
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
