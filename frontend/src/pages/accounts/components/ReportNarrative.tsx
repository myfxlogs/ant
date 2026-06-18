import { Button, Typography } from 'antd';
import { TrophyOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';

const { Title, Text, Paragraph } = Typography;

type Props = {
  narrative: string;
  sections: { summary?: string; findings?: string; recommendations?: string };
  reportError: string | null;
  generating: boolean;
  onNavigateAISettings: () => void;
};

export default function ReportNarrative({
  narrative, sections, reportError, generating, onNavigateAISettings,
}: Props) {
  const { t } = useTranslation();

  if (!narrative && !generating && !reportError) return null;

  return (
    <div className="rounded-2xl p-6 mb-6" style={{ background: 'var(--color-bg-card)', boxShadow: '0 2px 8px var(--color-shadow)' }}>
      <Title level={5} style={{ marginBottom: 16 }}>
        <TrophyOutlined /> {t('accounts.report.aiAnalysis')}
      </Title>
      {reportError && (
        <div className="rounded-lg p-3 mb-4" style={{ background: 'var(--color-danger-bg)', border: '1px solid var(--color-danger-bg-subtle)', color: 'var(--color-danger)' }}>
          <div>{reportError}</div>
          {/AI|api[_\s]?key|api key|未配置|not configured|API Key/i.test(reportError) && (
            <Button type="link" size="small" style={{ padding: 0, marginTop: 8 }}
              onClick={onNavigateAISettings}>
              {t('accounts.report.goToAISettings', 'AI Settings →')}
            </Button>
          )}
        </div>
      )}
      {sections.summary && (
        <div className="mb-4">
          <Text strong style={{ color: 'var(--color-text)' }}>{t('accounts.report.sections.summary')}</Text>
          <Paragraph style={{ color: 'var(--color-text-secondary)', marginTop: 4 }}>{sections.summary}</Paragraph>
        </div>
      )}
      {sections.findings && (
        <div className="mb-4">
          <Text strong style={{ color: 'var(--color-text)' }}>{t('accounts.report.sections.findings')}</Text>
          <Paragraph style={{ color: 'var(--color-text-secondary)', marginTop: 4, whiteSpace: 'pre-wrap' }}>{sections.findings}</Paragraph>
        </div>
      )}
      {sections.recommendations && (
        <div className="mb-4">
          <Text strong style={{ color: 'var(--color-text)' }}>{t('accounts.report.sections.recommendations')}</Text>
          <Paragraph style={{ color: 'var(--color-text-secondary)', marginTop: 4, whiteSpace: 'pre-wrap' }}>{sections.recommendations}</Paragraph>
        </div>
      )}
      {!sections.summary && narrative && (
        <div className="rounded-lg p-4" style={{ background: 'var(--color-bg-secondary)', whiteSpace: 'pre-wrap', color: 'var(--color-text-secondary)', fontSize: 14, lineHeight: 1.8 }}>
          {narrative}
          {generating && <span className="inline-block w-2 h-4 ml-1 animate-pulse" style={{ background: 'var(--color-primary)' }} />}
        </div>
      )}
    </div>
  );
}
