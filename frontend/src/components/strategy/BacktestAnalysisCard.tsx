import { Card, Tag, Typography, Space, Descriptions, Empty, Statistic } from 'antd';
import { BulbOutlined, ExperimentOutlined, TrendingUpOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import type { BacktestAnalysis } from '@/gen/ant/v1/agent_analysis_pb';

interface Props {
  analysis: BacktestAnalysis | null;
  loading?: boolean;
}

const gradeColor: Record<string, string> = {
  A: '#52c41a', B: '#1677ff', C: '#faad14', D: '#fa8c16', F: '#cf1322',
};

const actionColor: Record<string, string> = {
  deploy: 'success', optimize: 'processing', refine: 'warning', discard: 'error',
};

export default function BacktestAnalysisCard({ analysis, loading }: Props) {
  const { t } = useTranslation();

  if (loading) {
    return <Card size="small" loading style={{ marginBottom: 8 }} />;
  }

  if (!analysis) {
    return null;
  }

  return (
    <Card
      size="small"
      style={{ marginBottom: 8, borderColor: '#91caff' }}
      title={
        <Space size={4}>
          <BulbOutlined style={{ color: '#1677ff' }} />
          <Typography.Text strong style={{ fontSize: 13 }}>
            {t('agent.analysis.title', 'Backtest Analysis')}
          </Typography.Text>
          {analysis.performanceGrade && (
            <Tag style={{ fontWeight: 700, color: gradeColor[analysis.performanceGrade] || '#595959' }}>
              {analysis.performanceGrade}
            </Tag>
          )}
          {analysis.recommendedAction && (
            <Tag color={actionColor[analysis.recommendedAction] || 'default'}>
              {analysis.recommendedAction}
            </Tag>
          )}
        </Space>
      }
    >
      {/* Summary */}
      {analysis.summary && (
        <Typography.Paragraph style={{ fontSize: 12, marginBottom: 8, color: '#595959' }}>
          {analysis.summary}
        </Typography.Paragraph>
      )}

      {/* Score gauges */}
      <Space size="small" style={{ width: '100%', marginBottom: 8 }} styles={{ item: { flex: 1 } }}>
        {analysis.sharpeAssessment !== 0 && (
          <Card size="small" style={{ textAlign: 'center' }} styles={{ body: { padding: '8px 4px' } }}>
            <Statistic
              title={<span style={{ fontSize: 10 }}>{t('agent.analysis.sharpe', 'Sharpe')}</span>}
              value={analysis.sharpeAssessment}
              precision={2}
              valueStyle={{ fontSize: 16, color: analysis.sharpeAssessment >= 0.7 ? '#52c41a' : '#faad14' }}
            />
          </Card>
        )}
        {analysis.drawdownAssessment !== 0 && (
          <Card size="small" style={{ textAlign: 'center' }} styles={{ body: { padding: '8px 4px' } }}>
            <Statistic
              title={<span style={{ fontSize: 10 }}>{t('agent.analysis.drawdown', 'DD')}/span>}
              value={analysis.drawdownAssessment}
              precision={2}
              valueStyle={{ fontSize: 16, color: analysis.drawdownAssessment >= 0.7 ? '#52c41a' : '#faad14' }}
            />
          </Card>
        )}
        {analysis.winRateAssessment !== 0 && (
          <Card size="small" style={{ textAlign: 'center' }} styles={{ body: { padding: '8px 4px' } }}>
            <Statistic
              title={<span style={{ fontSize: 10 }}>{t('agent.analysis.winrate', 'Win Rate')}/span>}
              value={analysis.winRateAssessment}
              precision={2}
              valueStyle={{ fontSize: 16, color: analysis.winRateAssessment >= 0.6 ? '#52c41a' : '#faad14' }}
            />
          </Card>
        )}
      </Space>

      {/* Assessments */}
      <Descriptions size="small" column={2} style={{ marginBottom: 6 }}>
        {analysis.profitConsistency && (
          <Descriptions.Item label={t('agent.analysis.consistency', 'Consistency')}>
            {analysis.profitConsistency}
          </Descriptions.Item>
        )}
        {analysis.riskAdjustedReturn && (
          <Descriptions.Item label={t('agent.analysis.risk_adj', 'Risk-Adj Return')}>
            {analysis.riskAdjustedReturn}
          </Descriptions.Item>
        )}
        {analysis.overfittingRisk && (
          <Descriptions.Item label={t('agent.analysis.overfitting', 'Overfitting Risk')}>
            <Tag color={analysis.overfittingRisk === 'low' ? 'success' : analysis.overfittingRisk === 'high' ? 'error' : 'warning'}>
              {analysis.overfittingRisk}
            </Tag>
          </Descriptions.Item>
        )}
      </Descriptions>

      {/* Key observations */}
      {analysis.keyObservations.length > 0 && (
        <div style={{ marginBottom: 6 }}>
          <Typography.Text type="secondary" style={{ fontSize: 11 }}>
            <ExperimentOutlined /> {t('agent.analysis.observations', 'Key Observations')}
          </Typography.Text>
          {analysis.keyObservations.map((obs, i) => (
            <div key={i} style={{ fontSize: 11, color: '#595959', paddingLeft: 14 }}>• {obs}</div>
          ))}
        </div>
      )}

      {/* Improvement suggestions */}
      {analysis.improvementSuggestions.length > 0 && (
        <div style={{ marginBottom: 6 }}>
          <Typography.Text style={{ fontSize: 11, color: '#1677ff' }}>
            <TrendingUpOutlined /> {t('agent.analysis.suggestions', 'Improvement Suggestions')}
          </Typography.Text>
          {analysis.improvementSuggestions.map((sug, i) => (
            <div key={i} style={{ fontSize: 11, color: '#595959', paddingLeft: 14 }}>• {sug}</div>
          ))}
        </div>
      )}

      {/* Detailed analysis */}
      {analysis.detailedAnalysis && (
        <div style={{ paddingTop: 6, borderTop: '1px solid #f0f0f0' }}>
          <Typography.Text type="secondary" style={{ fontSize: 11 }}>
            {t('agent.analysis.detailed', 'Detailed Analysis')}
          </Typography.Text>
          <Typography.Paragraph style={{ fontSize: 11, marginTop: 2, color: '#595959', whiteSpace: 'pre-wrap' }}>
            {analysis.detailedAnalysis}
          </Typography.Paragraph>
        </div>
      )}
    </Card>
  );
}
