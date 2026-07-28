import { Card, Tag, Typography, Space, Progress, Descriptions } from 'antd';
import { RobotOutlined, WarningOutlined, CheckCircleOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import type { StrategyProfile } from '@/gen/ant/v1/agent_profile_pb';

interface Props {
  profile: StrategyProfile | null;
  loading?: boolean;
}

export default function StrategyProfileCard({ profile, loading }: Props) {
  const { t } = useTranslation();

  if (loading) {
    return (
      <Card size="small" loading style={{ marginBottom: 8 }} />
    );
  }

  if (!profile) {
    return null;
  }

  const coveragePct = Math.round((profile.coverageScore || 0) * 100);
  const coverageColor = coveragePct >= 80 ? '#52c41a' : coveragePct >= 50 ? '#faad14' : '#cf1322';

  return (
    <Card
      size="small"
      style={{ marginBottom: 8, borderColor: '#d9d9d9' }}
      title={
        <Space size={4}>
          <RobotOutlined style={{ color: '#1677ff' }} />
          <Typography.Text strong style={{ fontSize: 13 }}>
            {t('agent.profile.title', 'Strategy Profile')}
          </Typography.Text>
          {profile.strategyType && (
            <Tag color="blue">{profile.strategyType}</Tag>
          )}
        </Space>
      }
    >
      {profile.description && (
        <Typography.Paragraph style={{ fontSize: 12, marginBottom: 8, color: '#595959' }}>
          {profile.description}
        </Typography.Paragraph>
      )}

      <Descriptions size="small" column={2} style={{ marginBottom: 4 }}>
        {profile.timeframePreference && (
          <Descriptions.Item label={t('agent.profile.timeframe', 'Timeframe')}>
            {profile.timeframePreference}
          </Descriptions.Item>
        )}
        {profile.marketRegime && (
          <Descriptions.Item label={t('agent.profile.regime', 'Market Regime')}>
            {profile.marketRegime}
          </Descriptions.Item>
        )}
      </Descriptions>

      {profile.indicatorsUsed.length > 0 && (
        <div style={{ marginBottom: 6 }}>
          <Typography.Text type="secondary" style={{ fontSize: 11 }}>
            {t('agent.profile.indicators', 'Indicators')}
          </Typography.Text>
          <div style={{ marginTop: 2 }}>
            {profile.indicatorsUsed.map((ind, i) => (
              <Tag key={i} style={{ fontSize: 11, marginBottom: 2 }}>{ind}</Tag>
            ))}
          </div>
        </div>
      )}

      {profile.entryLogic && (
        <Typography.Paragraph style={{ fontSize: 12, marginBottom: 4 }}>
          <Typography.Text strong style={{ fontSize: 11 }}>{t('agent.profile.entry', 'Entry')}: </Typography.Text>
          {profile.entryLogic}
        </Typography.Paragraph>
      )}

      {profile.exitLogic && (
        <Typography.Paragraph style={{ fontSize: 12, marginBottom: 4 }}>
          <Typography.Text strong style={{ fontSize: 11 }}>{t('agent.profile.exit', 'Exit')}: </Typography.Text>
          {profile.exitLogic}
        </Typography.Paragraph>
      )}

      {profile.riskManagement && (
        <Typography.Paragraph style={{ fontSize: 12, marginBottom: 6 }}>
          <Typography.Text strong style={{ fontSize: 11 }}>{t('agent.profile.risk', 'Risk Management')}: </Typography.Text>
          {profile.riskManagement}
        </Typography.Paragraph>
      )}

      {/* Coverage */}
      <div style={{ marginBottom: 6 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 2 }}>
          <Typography.Text type="secondary" style={{ fontSize: 11 }}>
            {t('agent.profile.coverage', 'Coverage')}
          </Typography.Text>
          <Typography.Text style={{ fontSize: 11, color: coverageColor }}>
            {coveragePct}%
          </Typography.Text>
        </div>
        <Progress
          percent={coveragePct}
          size="small"
          strokeColor={coverageColor}
          showInfo={false}
        />
      </div>

      {/* Strengths & Weaknesses */}
      <Space size="small" style={{ width: '100%' }} styles={{ item: { flex: 1 } }}>
        {profile.strengths.length > 0 && (
          <div style={{ flex: 1 }}>
            <Typography.Text style={{ fontSize: 11, color: '#52c41a' }}>
              <CheckCircleOutlined /> {t('agent.profile.strengths', 'Strengths')}
            </Typography.Text>
            {profile.strengths.map((s, i) => (
              <div key={i} style={{ fontSize: 11, color: '#595959', paddingLeft: 14 }}>• {s}</div>
            ))}
          </div>
        )}
        {profile.weaknesses.length > 0 && (
          <div style={{ flex: 1 }}>
            <Typography.Text style={{ fontSize: 11, color: '#faad14' }}>
              <WarningOutlined /> {t('agent.profile.weaknesses', 'Weaknesses')}
            </Typography.Text>
            {profile.weaknesses.map((w, i) => (
              <div key={i} style={{ fontSize: 11, color: '#595959', paddingLeft: 14 }}>• {w}</div>
            ))}
          </div>
        )}
      </Space>

      {/* Blind spots */}
      {profile.blindSpots.length > 0 && (
        <div style={{ marginTop: 6, paddingTop: 6, borderTop: '1px solid #f0f0f0' }}>
          <Typography.Text type="secondary" style={{ fontSize: 11 }}>
            <WarningOutlined /> {t('agent.profile.blind_spots', 'Blind Spots')}
          </Typography.Text>
          <div style={{ marginTop: 2 }}>
            {profile.blindSpots.map((bs, i) => (
              <Tag key={i} color="warning" style={{ fontSize: 10, marginBottom: 2 }}>{bs}</Tag>
            ))}
          </div>
        </div>
      )}
    </Card>
  );
}
