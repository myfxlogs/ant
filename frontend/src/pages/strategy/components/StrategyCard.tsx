import { memo, useState } from 'react';
import { Card, Tag, Typography, Space, Button, Popconfirm, message } from 'antd';
import { RocketOutlined, ForkOutlined, ShareAltOutlined, DeleteOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import { useQueryClient } from '@tanstack/react-query';
import type { StrategyCard as StrategyCardType, StrategyTemplate } from '@/client/strategy';
import { strategyApi } from '@/client/strategy';
import { queryKeys } from '@/queries/queryKeys';
import DeployScheduleModal from './DeployScheduleModal';
import PublishToMarketModal from './PublishToMarketModal';

const { Text } = Typography;

function Sparkline({ data }: { data: number[] }) {
  if (!data || data.length < 2) return null;
  const nums = data.filter(n => !isNaN(n));
  if (nums.length < 2) return null;
  const min = nums.reduce((a, b) => (b < a ? b : a), nums[0]);
  const max = nums.reduce((a, b) => (b > a ? b : a), nums[0]);
  const range = max - min || 1;
  const w = 200;
  const h = 48;
  const step = w / (nums.length - 1);
  const points = nums.map((v, i) => `${(i * step).toFixed(1)},${(h - ((v - min) / range) * h).toFixed(1)}`).join(' ');
  const isUp = nums[nums.length - 1] >= nums[0];
  const color = isUp ? '#52c41a' : '#ff4d4f';
  return (
    <svg width={w} height={h} style={{ display: 'block' }}>
      <polyline points={points} fill="none" stroke={color} strokeWidth={1.5} />
    </svg>
  );
}

interface Props {
  card: StrategyCardType;
}

function StrategyCardImpl({ card }: Props) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [deployOpen, setDeployOpen] = useState(false);
  const [publishOpen, setPublishOpen] = useState(false);
  const [publishTemplate, setPublishTemplate] = useState<StrategyTemplate | null>(null);
  const [publishLoading, setPublishLoading] = useState(false);

  const handleDetail = () => navigate(`/strategy/view/${card.id}`);

  const handleFork = async () => {
    try {
      const draftId = await strategyApi.forkTemplate(card.id, `${card.name} (Fork)`);
      navigate(`/strategy/${draftId}/edit`);
    } catch (e: unknown) {
      message.error(e instanceof Error ? e.message : t('strategy.gallery.forkFailed', { defaultValue: 'Fork failed' }));
    }
  };

  const handlePublish = async () => {
    try {
      setPublishLoading(true);
      const tpl = await strategyApi.getTemplate(card.id);
      setPublishTemplate(tpl);
      setPublishOpen(true);
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('strategy.gallery.publishFailed', { defaultValue: 'Publish failed' }));
    } finally {
      setPublishLoading(false);
    }
  };

  const handleUnpublish = async () => {
    try {
      await strategyApi.cancelTemplateDraft(card.id);
      message.success(t('strategy.gallery.unpublishSuccess', { defaultValue: 'Unpublished' }));
      queryClient.invalidateQueries({ queryKey: queryKeys.strategyCards.all });
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('strategy.gallery.unpublishFailed', { defaultValue: 'Unpublish failed' }));
    }
  };

  const handleDelete = async () => {
    try {
      await strategyApi.deleteTemplate(card.id);
      message.success(t('strategy.gallery.deleteSuccess', { defaultValue: 'Deleted' }));
      queryClient.invalidateQueries({ queryKey: queryKeys.strategyCards.all });
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('strategy.gallery.deleteFailed', { defaultValue: 'Delete failed' }));
    }
  };

  const isSystem = card.isSystem;
  const isPublished = card.isPublic;

  return (
    <Card
      hoverable
      size="small"
      style={{ display: 'flex', flexDirection: 'column', height: '100%' }}
      bodyStyle={{ display: 'flex', flexDirection: 'column', height: '100%', padding: 16 }}
      onClick={handleDetail}
    >
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 8 }}>
        <Text strong ellipsis style={{ fontSize: 15, maxWidth: 200 }}>{card.name || 'Untitled'}</Text>
        <Space size={4}>
          {isSystem && <Tag color="blue">{t('strategy.gallery.system')}</Tag>}
          {isPublished && !isSystem && <Tag color="green">{t('strategy.gallery.shared')}</Tag>}
          {card.runningSchedules > 0 && (
            <Tag color="orange"><RocketOutlined /> {card.runningSchedules}</Tag>
          )}
        </Space>
      </div>

      <Text type="secondary" ellipsis style={{ fontSize: 12, minHeight: 18, marginBottom: 8 }}>
        {card.description?.slice(0, 100) || ''}
      </Text>

      {card.sparkline.length > 0 && (
        <div style={{ marginBottom: 8 }}>
          <Sparkline data={card.sparkline} />
        </div>
      )}

      <div style={{ display: 'flex', gap: 16, marginBottom: 12, marginTop: 'auto' }}>
        {card.winRate && (
          <div>
            <Text type="secondary" style={{ fontSize: 11 }}>{t('strategy.gallery.winRate')}</Text>
            <div><Text strong>{(parseFloat(card.winRate) * 100).toFixed(1)}%</Text></div>
          </div>
        )}
        {card.maxDrawdown && (
          <div>
            <Text type="secondary" style={{ fontSize: 11 }}>{t('strategy.gallery.maxDrawdown')}</Text>
            <div><Text strong style={{ color: '#ff4d4f' }}>{(parseFloat(card.maxDrawdown) * 100).toFixed(1)}%</Text></div>
          </div>
        )}
        {card.sharpeRatio && (
          <div>
            <Text type="secondary" style={{ fontSize: 11 }}>{t('strategy.gallery.sharpe')}</Text>
            <div><Text strong>{parseFloat(card.sharpeRatio).toFixed(2)}</Text></div>
          </div>
        )}
      </div>

      <Space size={4} onClick={e => e.stopPropagation()}>
        {!isSystem && (
          <Button size="small" icon={<RocketOutlined />} onClick={() => setDeployOpen(true)}>
            {t('strategy.gallery.deploy', { defaultValue: 'Deploy' })}
          </Button>
        )}
        {!isSystem && (
          <Button size="small" icon={<ForkOutlined />} onClick={handleFork}>
            {t('strategy.gallery.fork', { defaultValue: 'Fork' })}
          </Button>
        )}
        {!isSystem && !isPublished && (
          <Button size="small" icon={<ShareAltOutlined />} loading={publishLoading} onClick={handlePublish}>
            {t('strategy.gallery.publish', { defaultValue: 'Publish' })}
          </Button>
        )}
        {!isSystem && isPublished && (
          <Button size="small" icon={<ShareAltOutlined />} onClick={handleUnpublish}>
            {t('strategy.gallery.unpublish', { defaultValue: 'Unpublish' })}
          </Button>
        )}
        {!isSystem && (
          <Popconfirm title={t('strategy.gallery.deleteConfirm', { defaultValue: 'Delete this strategy?' })} onConfirm={handleDelete}>
            <Button size="small" danger icon={<DeleteOutlined />}>
              {t('strategy.gallery.delete', { defaultValue: 'Delete' })}
            </Button>
          </Popconfirm>
        )}
      </Space>
      <DeployScheduleModal
        open={deployOpen}
        templateId={card.id}
        templateName={card.name}
        onClose={() => setDeployOpen(false)}
      />
      <PublishToMarketModal
        open={publishOpen}
        template={publishTemplate}
        onClose={() => { setPublishOpen(false); setPublishTemplate(null); }}
        onPublished={() => {
          queryClient.invalidateQueries({ queryKey: queryKeys.strategyCards.all });
          setPublishOpen(false);
          setPublishTemplate(null);
        }}
      />
    </Card>
  );
}

export const StrategyCard = memo(StrategyCardImpl);
