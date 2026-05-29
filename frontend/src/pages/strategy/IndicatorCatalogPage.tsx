import { useState, useEffect } from 'react';
import { Card, Table, Tag, Typography, Spin, Alert, Collapse } from 'antd';
import { FunctionOutlined, SafetyOutlined } from '@ant-design/icons';
import { indicatorCatalogClient } from '@/client/connect';
import type { IndicatorCatalogResponse, IndicatorCatalogItem, IndicatorCatalogParam } from '@/gen/ant/v1/indicator_catalog_pb';
import { useTranslation } from 'react-i18next';

const { Title, Paragraph, Text } = Typography;

export default function IndicatorCatalogPage() {
  const { t } = useTranslation();
  const [data, setData] = useState<IndicatorCatalogResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    (async () => {
      try {
        const res = await indicatorCatalogClient.getIndicatorCatalog({});
        setData(res);
      } catch (e) {
        setError(String(e));
      } finally {
        setLoading(false);
      }
    })();
  }, []);

  if (loading) return <div className="flex justify-center py-20"><Spin size="large" /></div>;
  if (error) return <Alert type="error" message={t('common.loadingFailed')} description={error} showIcon />;
  if (!data) return null;

  const paramColumns = [
    { title: t('indicatorCatalog.paramKey'), dataIndex: 'key', key: 'key', width: 180 },
    { title: t('indicatorCatalog.paramLabel'), dataIndex: 'label', key: 'label', width: 180 },
    {
      title: t('indicatorCatalog.paramType'),
      dataIndex: 'type',
      key: 'type',
      width: 100,
      render: (v: string) => <Tag>{v}</Tag>,
    },
    { title: t('indicatorCatalog.paramDefault'), dataIndex: 'default', key: 'default', width: 100 },
    { title: t('indicatorCatalog.paramRange'), key: 'range', width: 140, render: (_: unknown, r: IndicatorCatalogParam) =>
      r.min !== r.max ? `${r.min} ~ ${r.max}` : '-'
    },
    { title: t('indicatorCatalog.paramDescription'), dataIndex: 'description', key: 'description' },
  ];

  const indicatorItems = data.indicators.map((ind: IndicatorCatalogItem) => ({
    key: ind.name,
    label: (
      <span className="font-mono">
        <FunctionOutlined className="mr-2" style={{ color: '#D4AF37' }} />
        <Text code className="text-base">{ind.callSignature}</Text>
      </span>
    ),
    children: (
      <div>
        <Paragraph type="secondary" className="mb-3">{ind.description}</Paragraph>
        <Table
          dataSource={ind.paramKeys.map((p: IndicatorCatalogParam, i: number) => ({ ...p, _key: i }))}
          columns={paramColumns}
          rowKey="_key"
          pagination={false}
          size="small"
          bordered
        />
      </div>
    ),
  }));

  const riskItems = [{
    key: 'risk',
    label: (
      <span>
        <SafetyOutlined className="mr-2" style={{ color: '#E53935' }} />
        <Text strong>{t('indicatorCatalog.riskParamsTitle')}</Text>
      </span>
    ),
    children: (
      <div>
        <Paragraph type="secondary" className="mb-3">{t('indicatorCatalog.riskParamsDesc')}</Paragraph>
        <Table
          dataSource={data.riskParams.map((p: IndicatorCatalogParam, i: number) => ({ ...p, _key: i }))}
          columns={paramColumns}
          rowKey="_key"
          pagination={false}
          size="small"
          bordered
        />
      </div>
    ),
  }];

  return (
    <div className="space-y-6">
      <Title level={4}>{t('indicatorCatalog.title')}</Title>
      <Paragraph type="secondary">{t('indicatorCatalog.description')}</Paragraph>

      <Card title={t('indicatorCatalog.indicatorsTitle')}>
        <Collapse items={indicatorItems} bordered={false} />
      </Card>

      <Card title={t('indicatorCatalog.riskSectionTitle')}>
        <Collapse items={riskItems} bordered={false} defaultActiveKey={['risk']} />
      </Card>
    </div>
  );
}
