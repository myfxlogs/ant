import { Tag, Typography, Empty, Spin, List, Alert } from 'antd';
import { HistoryOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import type { StrategyVersionInfo } from '@/gen/ant/v1/strategy_runtime_pb';
import dayjs from 'dayjs';

const { Text } = Typography;

interface VersionHistoryTabProps {
  versions: StrategyVersionInfo[];
  versionsLoading: boolean;
  isPurchased: boolean;
}

export function VersionHistoryTab({ versions, versionsLoading, isPurchased }: VersionHistoryTabProps) {
  const { t } = useTranslation();

  return (
    <div>
      {versionsLoading ? (
        <div style={{ textAlign: 'center', padding: 20 }}><Spin /></div>
      ) : versions.length === 0 ? (
        <Empty description={t('marketplace.detail.noVersions')} />
      ) : (
        <List
          size="small"
          dataSource={versions}
          renderItem={(v: StrategyVersionInfo) => (
            <List.Item>
              <div style={{ width: '100%', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <div>
                  <Tag color="blue">v{v.versionNumber}</Tag>
                  <Text type="secondary" style={{ fontSize: 12, marginLeft: 8 }}>{v.changeSummary || '-'}</Text>
                  <Text type="secondary" style={{ fontSize: 11, marginLeft: 8 }}>
                    {v.createdAt ? dayjs(typeof v.createdAt === 'object' && 'seconds' in v.createdAt ? new Date(Number((v.createdAt as any).seconds) * 1000) : v.createdAt as any).format('YYYY-MM-DD HH:mm') : ''}
                  </Text>
                </div>
                {isPurchased && v.versionNumber > 1 && (
                  <Tag color="cyan">{t('marketplace.detail.newVersion')}</Tag>
                )}
              </div>
            </List.Item>
          )}
        />
      )}
      {isPurchased && versions.length > 1 && (
        <Alert
          type="info"
          showIcon
          style={{ marginTop: 12 }}
          message={t('marketplace.detail.upgradeAvailable')}
          description={t('marketplace.detail.upgradeHint')}
        />
      )}
    </div>
  );
}

export function versionHistoryTabLabel(t: (key: string, opts?: any) => string) {
  return <span><HistoryOutlined /> {t('marketplace.detail.versionHistory')}</span>;
}
