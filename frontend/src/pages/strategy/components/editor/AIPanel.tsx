import { Tabs, Typography, Spin } from 'antd';
import { CodeOutlined, BulbOutlined, CheckCircleOutlined, ThunderboltOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { TAB_A_I_KEY, TAB_EXPLAIN_KEY } from '@/gen/ant/v1/i18n/strategy_code_assist_keys';
import { AICodeReviseChat, CodeExplainPanel } from '@/components/strategy/CodeAssist';
import ValidationResults from './ValidationResults';

const { Text } = Typography;

interface Props {
  activeTab: string;
  onTabChange: (key: string) => void;
  code: string;
  codeValidating: boolean;
  validationResult: { valid?: boolean; errors?: string[]; warnings?: string[]; parameters?: { key: string; defaultValue?: string }[]; qualityHints?: { message?: string; description?: string }[] } | null;
  validationStale?: boolean;
  fixInstruction: string;
  onApplyCode: (newCode: string) => void;
  onFixWithAI: () => void;
}

export default function AIPanel({ activeTab, onTabChange, code, codeValidating, validationResult, validationStale, fixInstruction, onApplyCode, onFixWithAI }: Props) {
  const { t } = useTranslation();
  const hasCode = !!code.trim();

  return (
    <div style={{ border: '1px solid var(--color-border)', borderRadius: 10, overflow: 'hidden', height: 466, display: 'flex', flexDirection: 'column' }}>
      <Tabs activeKey={activeTab} onChange={onTabChange} size="small" style={{ flex: 'none', padding: '0 12px' }} tabBarStyle={{ marginBottom: 0 }}
        items={[
          {
            key: 'revise',
            label: <span><BulbOutlined /> {t(TAB_A_I_KEY, { defaultValue: 'AI Revise' })}</span>,
            children: (
              <div style={{ height: 416, overflow: 'auto', padding: '8px 4px' }}>
                {!hasCode ? (
                  <Text type="secondary" style={{ display: 'block', textAlign: 'center', padding: 40 }}>
                    {t('strategy.ai.reviseHint', { defaultValue: 'Write code first, then ask AI to improve it.' })}
                  </Text>
                ) : <AICodeReviseChat code={code} onApply={onApplyCode} initialInstruction={fixInstruction} />}
              </div>
            ),
          },
          {
            key: 'explain',
            label: <span><CodeOutlined /> {t(TAB_EXPLAIN_KEY, { defaultValue: 'Explain' })}</span>,
            children: (
              <div style={{ height: 416, overflow: 'auto', padding: '8px 4px' }}>
                {!hasCode ? (
                  <Text type="secondary" style={{ display: 'block', textAlign: 'center', padding: 40 }}>
                    {t('strategy.ai.explainHint', { defaultValue: 'Write code to see AI explanation.' })}
                  </Text>
                ) : <CodeExplainPanel code={code} />}
              </div>
            ),
          },
          {
            key: 'validate',
            label: <span><CheckCircleOutlined /> {t('strategy.validate.tab')}</span>,
            children: (
              <div style={{ height: 416, overflow: 'auto', padding: '12px' }}>
                {codeValidating ? (
                  <div style={{ textAlign: 'center', padding: 60 }}>
                    <Spin size="large" />
                    <Text type="secondary" style={{ display: 'block', marginTop: 12 }}>{t('strategy.validate.running', { defaultValue: 'Running validation...' })}</Text>
                  </div>
                ) : validationStale ? (
                  <div style={{ textAlign: 'center', padding: 30 }}>
                    <ThunderboltOutlined style={{ fontSize: 28, color: '#fa8c16' }} />
                    <Text type="warning" style={{ display: 'block', marginTop: 8, fontWeight: 500 }}>
                      {t('strategy.validate.stale', { defaultValue: 'Code was modified — re-validate to check.' })}
                    </Text>
                    <Text type="secondary" style={{ display: 'block', marginTop: 4, fontSize: 12 }}>
                      {t('strategy.validate.staleHint', { defaultValue: 'The validation results below are from the previous code version.' })}
                    </Text>
                    <div style={{ marginTop: 12 }}>
                      <ValidationResults result={validationResult} onFixWithAI={onFixWithAI} />
                    </div>
                  </div>
                ) : validationResult ? (
                  <ValidationResults result={validationResult} onFixWithAI={onFixWithAI} />
                ) : (
                  <div style={{ textAlign: 'center', padding: 40 }}>
                    <ThunderboltOutlined style={{ fontSize: 28, color: '#d9d9d9' }} />
                    <Text type="secondary" style={{ display: 'block', marginTop: 8 }}>
                      {t('strategy.validate.hint', { defaultValue: 'Click "Validate Code" to check syntax, imports, and strategy structure.' })}
                    </Text>
                  </div>
                )}
              </div>
            ),
          },
        ]}
      />
    </div>
  );
}
