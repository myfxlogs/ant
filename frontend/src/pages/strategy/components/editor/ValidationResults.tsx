import { Button, Tag, Typography } from 'antd';
import { BulbOutlined, CheckCircleOutlined, CloseCircleOutlined, WarningOutlined, InfoCircleOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';

const { Text } = Typography;

interface Props {
  result: { valid?: boolean; errors?: string[]; warnings?: string[]; parameters?: { key: string; defaultValue?: string }[]; qualityHints?: { message?: string; description?: string }[] } | null;
  onFixWithAI: () => void;
}

export default function ValidationResults({ result, onFixWithAI }: Props) {
  const { t } = useTranslation();
  if (!result) return null;

  return (
    <div>
      <div style={{
        padding: '10px 14px', borderRadius: 8, marginBottom: 12,
        background: result.valid ? '#f6ffed' : '#fff2f0',
        border: `1px solid ${result.valid ? '#b7eb8f' : '#ffccc7'}`,
        display: 'flex', alignItems: 'center', gap: 8,
      }}>
        {result.valid
          ? <CheckCircleOutlined style={{ color: '#52c41a', fontSize: 18 }} />
          : <CloseCircleOutlined style={{ color: '#ff4d4f', fontSize: 18 }} />}
        <Text strong style={{ color: result.valid ? '#52c41a' : '#ff4d4f' }}>
          {result.valid ? t('strategy.validate.passed') : t('strategy.validate.failed')}
        </Text>
      </div>

      {result.errors?.length! > 0 && (
        <div style={{ marginBottom: 10 }}>
          <Text strong style={{ fontSize: 12, color: '#ff4d4f' }}>❌ {t('strategy.validate.errors', { defaultValue: 'Errors' })}</Text>
          {result.errors!.map((e, i) => (
            <div key={i} style={{ fontSize: 12, padding: '4px 8px', marginTop: 2, background: '#fff2f0', borderRadius: 4 }}>
              <CloseCircleOutlined style={{ color: '#ff4d4f', marginRight: 4 }} />{e}
            </div>
          ))}
        </div>
      )}

      {result.warnings?.length! > 0 && (
        <div style={{ marginBottom: 10 }}>
          <Text strong style={{ fontSize: 12, color: '#fa8c16' }}>⚠️ {t('strategy.validate.warnings', { defaultValue: 'Warnings' })}</Text>
          {result.warnings!.map((w, i) => (
            <div key={i} style={{ fontSize: 12, padding: '4px 8px', marginTop: 2, background: '#fff7e6', borderRadius: 4 }}>
              <WarningOutlined style={{ color: '#fa8c16', marginRight: 4 }} />{w}
            </div>
          ))}
        </div>
      )}

      {!result.valid && (result.errors?.length! > 0 || result.warnings?.length! > 0) && (
        <div style={{ textAlign: 'center', marginBottom: 12 }}>
          <Button type="primary" size="small" icon={<BulbOutlined />} onClick={onFixWithAI}>
            {t('strategy.validate.fixWithAI', { defaultValue: 'Send errors to AI Revise' })}
          </Button>
        </div>
      )}

      {result.parameters?.length! > 0 && (
        <div style={{ marginBottom: 10 }}>
          <Text strong style={{ fontSize: 12, color: '#1677ff' }}>📋 {result.parameters!.length} {t('strategy.validate.parameters', { defaultValue: 'parameters' })}</Text>
          {result.parameters!.map((p, i) => <Tag key={i} style={{ marginTop: 2 }}>{p.key}{p.defaultValue ? ` = ${p.defaultValue}` : ''}</Tag>)}
        </div>
      )}

      {result.qualityHints?.length! > 0 && (
        <div style={{ marginBottom: 10 }}>
          <Text strong style={{ fontSize: 12 }}>💡 {t('strategy.validate.hints', { defaultValue: 'Suggestions' })}</Text>
          {result.qualityHints!.map((h, i) => (
            <div key={i} style={{ fontSize: 11, padding: '3px 6px', marginTop: 2, background: '#f0f5ff', borderRadius: 4 }}>
              <InfoCircleOutlined style={{ color: '#1677ff', marginRight: 4 }} />{h.message || h.description || JSON.stringify(h)}
            </div>
          ))}
        </div>
      )}

      {!result.errors?.length && !result.warnings?.length && result.valid && (
        <Text type="secondary" style={{ textAlign: 'center', display: 'block', padding: 12 }}>
          {t('strategy.validate.allClear', { defaultValue: 'All checks passed — no issues found.' })}
        </Text>
      )}
    </div>
  );
}
