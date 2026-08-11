import { useMemo } from 'react';
import { Typography, Tabs, Tag, Space, Button, Descriptions, Spin, Empty } from 'antd';
import { EditOutlined, ForkOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { useNavigate, useParams } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { LineChart, Line, ResponsiveContainer, YAxis, Tooltip as RTooltip } from 'recharts';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { vscDarkPlus } from 'react-syntax-highlighter/dist/esm/styles/prism';
import { strategyApi } from '@/client/strategy';
import type { StrategyTemplate } from '@/gen/ant/v1/strategy_template_entity_pb';
import { queryKeys } from '@/queries/queryKeys';
import { useAuthStore } from '@/stores/authStore';
import Seo from '@/components/common/Seo';

type TemplateWithMetrics = StrategyTemplate & {
  sparkline?: string[];
  winRate?: string;
  maxDrawdown?: string;
  profitFactor?: string;
  sharpeRatio?: string;
};

const { Title, Paragraph } = Typography;

export default function StrategyDetailPage() {
  const { id } = useParams<{ id: string }>();
  const { t } = useTranslation();
  const navigate = useNavigate();

  const { data: template, isLoading } = useQuery({
    queryKey: queryKeys.templates.detail(id!),
    queryFn: async () => strategyApi.getTemplate(id!) as unknown as TemplateWithMetrics,
    enabled: !!id,
  });

  const currentUserId = useAuthStore(s => s.user?.id);
  const isOwner = !!template && !!currentUserId && template.userId === currentUserId;
  const isSystem = !!template?.isSystem;
  const canEdit = isOwner || isSystem;

  const handleEdit = () => navigate(`/strategy/${id}/edit`);
  const handleFork = async () => {
    try {
      const draft = await strategyApi.createTemplateDraft({ name: `${template?.name || 'Strategy'} (Fork)` });
      if (!draft.id) throw new Error('Draft creation returned empty id');
      await strategyApi.updateTemplateDraft({ id: draft.id, name: `${template?.name || 'Strategy'} (Fork)`, description: template?.description, code: template?.code, tags: template?.tags });
      navigate(`/strategy/${draft.id}/edit`);
    } catch (_e) {
      // navigate to edit as fallback
      navigate(`/strategy/${id}/edit`);
    }
  };

  const tags = useMemo(() => template?.tags || [], [template]);

  const sparklineData = useMemo(() => {
    if (!template?.sparkline || template.sparkline.length < 2) return [];
    return template.sparkline.map((v: string, i: number) => ({ idx: i, value: parseFloat(v) || 0 }));
  }, [template]);

  const statsRows = useMemo(() => {
    if (!template) return [];
    const rows: { key: string; label: string; value: string }[] = [];
    if (template.winRate) rows.push({ key: 'winRate', label: t('strategy.templates.scheduleLaunch.metrics.winRate', { defaultValue: 'Win Rate' }), value: `${(parseFloat(template.winRate) * 100).toFixed(1)}%` });
    if (template.profitFactor) rows.push({ key: 'pf', label: t('strategy.templates.detail.profitFactor', { defaultValue: 'Profit Factor' }), value: parseFloat(template.profitFactor).toFixed(2) });
    if (template.maxDrawdown) rows.push({ key: 'dd', label: t('strategy.templates.scheduleLaunch.metrics.maxDrawdown', { defaultValue: 'Max Drawdown' }), value: `${(parseFloat(template.maxDrawdown) * 100).toFixed(1)}%` });
    if (template.sharpeRatio) rows.push({ key: 'sharpe', label: t('strategy.templates.scheduleLaunch.metrics.sharpe', { defaultValue: 'Sharpe Ratio' }), value: parseFloat(template.sharpeRatio).toFixed(2) });
    return rows;
  }, [template, t]);

  if (isLoading) {
    return <div style={{ textAlign: 'center', padding: 80 }}><Spin size="large" /></div>;
  }

  if (!template) {
    return <Empty description={t('strategy.templates.detail.notFound', { defaultValue: 'Strategy not found' })} />;
  }

  return (
    <>
      <Seo title={template.name} path={`/strategy/view/${id}`} />
      <div style={{ padding: '24px 24px 96px', background: 'var(--color-bg-secondary)', minHeight: '100vh' }}>
        <div className="max-w-5xl mx-auto">
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 16 }}>
            <div>
              <Title level={3} style={{ margin: 0 }}>{template.name}</Title>
              <Space size={8} style={{ marginTop: 8 }}>
                {template.isSystem && <Tag color="blue">{t('strategy.templates.gallery.system', { defaultValue: 'System' })}</Tag>}
                {template.isPublic && !template.isSystem && <Tag color="green">{t('strategy.templates.gallery.shared', { defaultValue: 'Shared' })}</Tag>}
                {tags.map(tag => <Tag key={tag}>{tag}</Tag>)}
              </Space>
            </div>
            <Space>
              {isSystem && (
                <Button type="primary" icon={<ForkOutlined />} onClick={handleFork}>
                  {t('strategy.templates.gallery.forkEdit', { defaultValue: 'Fork & Edit' })}
                </Button>
              )}
              {isOwner && !isSystem && (
                <Button type="primary" icon={<EditOutlined />} onClick={handleEdit}>
                  {t('strategy.templates.detail.openInWorkspace', { defaultValue: 'Open in Workspace' })}
                </Button>
              )}
            </Space>
          </div>

          <Tabs
            defaultActiveKey="overview"
            items={[
              {
                key: 'overview',
                label: t('strategy.templates.detail.overview', { defaultValue: 'Overview' }),
                children: (
                  <div style={{ padding: '8px 0' }}>
                    <Paragraph type="secondary">
                      {template.description || t('strategy.templates.detail.noDescription', { defaultValue: 'No description' })}
                    </Paragraph>

                    {sparklineData.length > 0 && (
                      <div style={{ marginTop: 16, marginBottom: 24 }}>
                        <Title level={5}>{t('strategy.templates.detail.equityCurve', { defaultValue: 'Equity Curve' })}</Title>
                        <ResponsiveContainer width="100%" height={240}>
                          <LineChart data={sparklineData}>
                            <YAxis domain={['auto', 'auto']} style={{ fontSize: 11 }} />
                            <RTooltip formatter={(v: number | string | undefined) => Number(v ?? 0).toFixed(2)} />
                            <Line type="monotone" dataKey="value" stroke="#58a6ff" strokeWidth={2} dot={false} />
                          </LineChart>
                        </ResponsiveContainer>
                      </div>
                    )}

                    {statsRows.length > 0 && (
                      <div style={{ marginBottom: 24 }}>
                        <Title level={5}>{t('strategy.templates.detail.tradeStats', { defaultValue: 'Trade Statistics' })}</Title>
                        <Descriptions column={2} bordered size="small"
                          items={statsRows.map(r => ({ key: r.key, label: r.label, children: r.value }))}
                        />
                      </div>
                    )}

                    <Descriptions
                      column={2}
                      bordered
                      size="small"
                      items={[
                        { key: 'useCount', label: t('strategy.templates.table.useCount', { defaultValue: 'Use Count' }), children: template.useCount },
                        { key: 'created', label: t('strategy.templates.table.createdAt', { defaultValue: 'Created' }), children: template.createdAt ? new Date(template.createdAt as unknown as string).toLocaleDateString() : '-' },
                        { key: 'visibility', label: t('strategy.templates.table.visibility', { defaultValue: 'Visibility' }), children: template.isSystem ? t('strategy.templates.gallery.system', { defaultValue: 'System' }) : template.isPublic ? t('strategy.templates.visibility.public', { defaultValue: 'Public' }) : t('strategy.templates.visibility.private', { defaultValue: 'Private' }) },
                        { key: 'status', label: t('strategy.templates.table.status', { defaultValue: 'Status' }), children: template.status || '-' },
                      ]}
                    />
                    {template.parameters && template.parameters.length > 0 && (
                      <div style={{ marginTop: 24 }}>
                        <Title level={5}>{t('strategy.templates.detail.parameters', { defaultValue: 'Parameters' })}</Title>
                        <Descriptions
                          column={1}
                          bordered
                          size="small"
                          items={template.parameters.map(p => ({
                            key: p.name,
                            label: p.label || p.name,
                            children: `${p.type} (default: ${p.default || '-'})`,
                          }))}
                        />
                      </div>
                    )}
                  </div>
                ),
              },
              ...(canEdit ? [{
                key: 'code',
                label: t('strategy.templates.codeModal.title', { defaultValue: 'Code' }),
                children: (
                  <div style={{ padding: '8px 0' }}>
                    <SyntaxHighlighter
                      language="mql"
                      style={vscDarkPlus}
                      customStyle={{ borderRadius: 8, maxHeight: '70vh', fontSize: 13 }}
                      showLineNumbers
                    >
                      {template.code || ''}
                    </SyntaxHighlighter>
                  </div>
                ),
              }] : []),
            ]}
          />
        </div>

        <div style={{
          position: 'fixed', bottom: 0, left: 0, right: 0,
          background: 'var(--ant-color-bg-container)',
          borderTop: '1px solid var(--ant-color-border)',
          padding: '12px 24px',
          display: 'flex', justifyContent: 'center',
          zIndex: 10,
        }}>
          <Space>
            {isSystem && (
              <Button type="primary" icon={<ForkOutlined />} onClick={handleFork}>
                {t('strategy.templates.gallery.forkEdit', { defaultValue: 'Fork & Edit' })}
              </Button>
            )}
            {isOwner && !isSystem && (
              <Button type="primary" icon={<EditOutlined />} onClick={handleEdit}>
                {t('strategy.templates.detail.openInWorkspace', { defaultValue: 'Open in Workspace' })}
              </Button>
            )}
          </Space>
        </div>
      </div>
    </>
  );
}
