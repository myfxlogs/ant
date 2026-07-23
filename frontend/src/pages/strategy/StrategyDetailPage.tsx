import { useMemo } from 'react';
import { Typography, Tabs, Tag, Space, Button, Descriptions, Spin, Empty, message } from 'antd';
import { EditOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { useNavigate, useParams } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { LineChart, Line, ResponsiveContainer, YAxis, Tooltip as RTooltip } from 'recharts';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { vscDarkPlus } from 'react-syntax-highlighter/dist/esm/styles/prism';
import { strategyApi } from '@/client/strategy';
import { queryKeys } from '@/queries/queryKeys';
import Seo from '@/components/common/Seo';

const { Title, Paragraph } = Typography;

export default function StrategyDetailPage() {
  const { id } = useParams<{ id: string }>();
  const { t } = useTranslation();
  const navigate = useNavigate();

  const { data: template, isLoading } = useQuery({
    queryKey: queryKeys.templates.detail(id!),
    queryFn: () => strategyApi.getTemplate(id!),
    enabled: !!id,
  });

  const handleEdit = () => navigate(`/strategy/${id}/edit`);

  const handleForkAndEdit = async () => {
    if (!id) return;
    try {
      const draftId = await strategyApi.forkTemplate(id, `${template?.name || 'Strategy'} (Fork)`);
      navigate(`/strategy/${draftId}/edit`);
    } catch (e: unknown) {
      message.error(e instanceof Error ? e.message : t('strategy.gallery.forkFailed', { defaultValue: 'Fork failed' }));
    }
  };

  const tags = useMemo(() => template?.tags || [], [template]);

  const sparklineData = useMemo(() => {
    if (!template?.sparkline || template.sparkline.length < 2) return [];
    return template.sparkline.map((v, i) => ({ idx: i, value: parseFloat(v) || 0 }));
  }, [template]);

  const statsRows = useMemo(() => {
    if (!template) return [];
    const rows: { key: string; label: string; value: string }[] = [];
    if (template.winRate) rows.push({ key: 'winRate', label: t('strategy.detail.winRate', { defaultValue: 'Win Rate' }), value: `${(parseFloat(template.winRate) * 100).toFixed(1)}%` });
    if (template.profitFactor) rows.push({ key: 'pf', label: t('strategy.detail.profitFactor', { defaultValue: 'Profit Factor' }), value: parseFloat(template.profitFactor).toFixed(2) });
    if (template.maxDrawdown) rows.push({ key: 'dd', label: t('strategy.detail.maxDrawdown', { defaultValue: 'Max Drawdown' }), value: `${(parseFloat(template.maxDrawdown) * 100).toFixed(1)}%` });
    if (template.sharpeRatio) rows.push({ key: 'sharpe', label: t('strategy.detail.sharpe', { defaultValue: 'Sharpe Ratio' }), value: parseFloat(template.sharpeRatio).toFixed(2) });
    return rows;
  }, [template, t]);

  if (isLoading) {
    return <div style={{ textAlign: 'center', padding: 80 }}><Spin size="large" /></div>;
  }

  if (!template) {
    return <Empty description={t('strategy.detail.notFound', { defaultValue: 'Strategy not found' })} />;
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
                {template.isSystem && <Tag color="blue">{t('strategy.gallery.system', { defaultValue: 'System' })}</Tag>}
                {template.isPublic && !template.isSystem && <Tag color="green">{t('strategy.gallery.shared', { defaultValue: 'Shared' })}</Tag>}
                {tags.map(tag => <Tag key={tag}>{tag}</Tag>)}
              </Space>
            </div>
            <Space>
              <Button type="primary" icon={<EditOutlined />} onClick={template.isSystem ? handleForkAndEdit : handleEdit}>
                {template.isSystem
                  ? t('strategy.detail.forkAndEdit', { defaultValue: 'Fork & Edit' })
                  : t('strategy.gallery.edit', { defaultValue: 'Edit' })}
              </Button>
            </Space>
          </div>

          <Tabs
            defaultActiveKey="overview"
            items={[
              {
                key: 'overview',
                label: t('strategy.detail.overview', { defaultValue: 'Overview' }),
                children: (
                  <div style={{ padding: '8px 0' }}>
                    <Paragraph type="secondary">
                      {template.description || t('strategy.detail.noDescription', { defaultValue: 'No description' })}
                    </Paragraph>

                    {sparklineData.length > 0 && (
                      <div style={{ marginTop: 16, marginBottom: 24 }}>
                        <Title level={5}>{t('strategy.detail.equityCurve', { defaultValue: 'Equity Curve' })}</Title>
                        <ResponsiveContainer width="100%" height={240}>
                          <LineChart data={sparklineData}>
                            <YAxis domain={['auto', 'auto']} style={{ fontSize: 11 }} />
                            <RTooltip formatter={(v: number) => v.toFixed(2)} />
                            <Line type="monotone" dataKey="value" stroke="#58a6ff" strokeWidth={2} dot={false} />
                          </LineChart>
                        </ResponsiveContainer>
                      </div>
                    )}

                    {statsRows.length > 0 && (
                      <div style={{ marginBottom: 24 }}>
                        <Title level={5}>{t('strategy.detail.tradeStats', { defaultValue: 'Trade Statistics' })}</Title>
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
                        { key: 'useCount', label: t('strategy.detail.useCount', { defaultValue: 'Use Count' }), children: template.useCount },
                        { key: 'created', label: t('strategy.detail.created', { defaultValue: 'Created' }), children: template.createdAt ? new Date(template.createdAt.toDate()).toLocaleDateString() : '-' },
                        { key: 'visibility', label: t('strategy.detail.visibility', { defaultValue: 'Visibility' }), children: template.isSystem ? 'System' : template.isPublic ? 'Public' : 'Private' },
                        { key: 'status', label: t('strategy.detail.status', { defaultValue: 'Status' }), children: template.status || '-' },
                      ]}
                    />
                    {template.parameters && template.parameters.length > 0 && (
                      <div style={{ marginTop: 24 }}>
                        <Title level={5}>{t('strategy.detail.parameters', { defaultValue: 'Parameters' })}</Title>
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
              {
                key: 'code',
                label: t('strategy.detail.code', { defaultValue: 'Code' }),
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
              },
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
            <Button type="primary" icon={<EditOutlined />} onClick={template.isSystem ? handleForkAndEdit : handleEdit}>
              {template.isSystem
                ? t('strategy.detail.forkAndEdit', { defaultValue: 'Fork & Edit' })
                : t('strategy.gallery.edit', { defaultValue: 'Edit' })}
            </Button>
          </Space>
        </div>
      </div>
    </>
  );
}
