import { Alert, Space, Button, Tag } from 'antd';
import { CheckCircleOutlined, ExclamationCircleOutlined, RobotOutlined, WarningOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import type { ValidateExtendedResult } from '@/client/codeAssist';
import type { AutoFixDebug } from '@/pages/strategy/hooks/useAIWorkflow';

const SEVERITY_COLOR: Record<string, string> = {
  error: '#cf1322',
  warn: '#ad6800',
  info: '#1677ff',
};

const CATEGORY_LABEL: Record<string, string> = {
  FUTURE_DATA_LEAK: 'Future Data Leak',
  MISSING_PARAM: 'Missing Param',
  UNREAD_PARAM: 'Unread Param',
  NDARRAY_PANDAS_MISUSE: 'ndarray/pandas Misuse',
  NO_STOP_AND_TAKE_PROFIT: 'Missing Stop/Take Profit',
  NO_ENTRY_PCT: 'Missing Entry %',
};

interface Props {
  validationResult: ValidateExtendedResult;
  autoFixing?: boolean;
  autoFixDebug?: AutoFixDebug | null;
  onAutoFix?: () => void;
  onAskAI?: () => void;
  onDismissDebug?: () => void;
}

function issueCount(s: { errors: string[]; warnings: string[]; hints: { category: string; message: string; line: number }[] }): number {
  return s.errors.length + s.warnings.length + s.hints.length;
}

function renderIssueList(
  errors: string[], warnings: string[], hints: { category: string; message: string }[],
  prefix: string, color: string, borderColor: string,
) {
  const items: { type: 'e' | 'w' | 'h'; text: string; cat?: string }[] = [
    ...errors.map(e => ({ type: 'e' as const, text: e })),
    ...warnings.map(w => ({ type: 'w' as const, text: w })),
    ...hints.map(h => ({ type: 'h' as const, text: h.message, cat: h.category })),
  ];
  if (items.length === 0) return null;
  return (
    <>
      {items.map((item, i) => (
        <div key={`${prefix}-${i}`} style={{
          fontSize: 11, lineHeight: '1.6', color,
          padding: '2px 0', borderBottom: `1px solid ${borderColor}`,
          wordBreak: 'break-word',
        }}>
          {item.type === 'e' ? <span style={{ fontWeight: 600, marginRight: 4 }}>✕</span>
            : item.type === 'w' ? <span style={{ fontWeight: 600, marginRight: 4 }}>⚠</span>
              : item.cat ? <Tag color="blue" style={{ fontSize: 9, lineHeight: '16px', marginRight: 4 }}>{CATEGORY_LABEL[item.cat] || item.cat}</Tag>
                : null}
          {item.text}
        </div>
      ))}
    </>
  );
}

export default function ValidationResultAlert({
  validationResult, autoFixing, autoFixDebug, onAutoFix, onAskAI, onDismissDebug,
}: Props) {
  const { t } = useTranslation();
  if (!validationResult) return null;

  const { errors, warnings, qualityHints, valid } = validationResult;
  const hasIssues = errors.length > 0 || warnings.length > 0 || qualityHints.length > 0;

  return (
    <>
      {/* Auto-fix debug card — shown after auto-fix completes */}
      {autoFixDebug && (
        <Alert
          type={autoFixDebug.passed ? 'success' : 'warning'}
          showIcon
          icon={autoFixDebug.passed ? <CheckCircleOutlined /> : <ExclamationCircleOutlined />}
          message={
            autoFixDebug.passed
              ? `Auto-fix passed in ${autoFixDebug.iterations} iteration${autoFixDebug.iterations > 1 ? 's' : ''}`
              : `Auto-fix: ${issueCount(autoFixDebug.remaining)} issue(s) remain after ${autoFixDebug.iterations} iterations`
          }
          description={
            <div style={{ fontSize: 12, marginTop: 4 }}>
              {/* Fixed issues */}
              {issueCount(autoFixDebug.fixed) > 0 && (
                <div style={{ marginBottom: 8 }}>
                  <div style={{ fontWeight: 600, color: '#389e0d', marginBottom: 4 }}>
                    ✅ Fixed ({issueCount(autoFixDebug.fixed)})
                  </div>
                  <div style={{
                    maxHeight: 120, overflowY: 'auto', padding: '4px 6px',
                    background: '#f6ffed', borderRadius: 4, border: '1px solid #d9f7be',
                  }}>
                    {renderIssueList(
                      autoFixDebug.fixed.errors, autoFixDebug.fixed.warnings, autoFixDebug.fixed.hints,
                      'fixed', '#389e0d', '#d9f7be',
                    )}
                  </div>
                </div>
              )}

              {/* Remaining issues */}
              {issueCount(autoFixDebug.remaining) > 0 && (
                <div style={{ marginBottom: 8 }}>
                  <div style={{ fontWeight: 600, color: '#ad6800', marginBottom: 4 }}>
                    <WarningOutlined /> Remaining ({issueCount(autoFixDebug.remaining)})
                  </div>
                  <div style={{
                    maxHeight: 120, overflowY: 'auto', padding: '4px 6px',
                    background: '#fffbe6', borderRadius: 4, border: '1px solid #ffe58f',
                  }}>
                    {renderIssueList(
                      autoFixDebug.remaining.errors, autoFixDebug.remaining.warnings, autoFixDebug.remaining.hints,
                      'rem', '#ad6800', '#ffe58f',
                    )}
                  </div>
                </div>
              )}

              {/* Introduced (regression) issues */}
              {issueCount(autoFixDebug.introduced) > 0 && (
                <div style={{ marginBottom: 4 }}>
                  <div style={{ fontWeight: 600, color: '#cf1322', marginBottom: 4 }}>
                    ❌ New (regression) ({issueCount(autoFixDebug.introduced)})
                  </div>
                  <div style={{
                    maxHeight: 120, overflowY: 'auto', padding: '4px 6px',
                    background: '#fff1f0', borderRadius: 4, border: '1px solid #ffa39e',
                  }}>
                    {renderIssueList(
                      autoFixDebug.introduced.errors, autoFixDebug.introduced.warnings, autoFixDebug.introduced.hints,
                      'intro', '#cf1322', '#ffa39e',
                    )}
                  </div>
                </div>
              )}

              {onDismissDebug && (
                <Button size="small" type="link" onClick={onDismissDebug} style={{ padding: 0, marginTop: 4 }}>
                  Dismiss
                </Button>
              )}
            </div>
          }
          style={{ marginBottom: 8 }}
          closable
          onClose={onDismissDebug}
        />
      )}

      {/* Main validation result alert */}
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
    </>
  );
}
