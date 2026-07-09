import { useState, useEffect } from 'react';
import { Card, Table, Tag, Typography, Spin, Alert, Collapse } from 'antd';
import { FunctionOutlined, SafetyOutlined } from '@ant-design/icons';
import { indicatorCatalogClient } from '@/client/connect';
import type { IndicatorCatalogResponse, IndicatorCatalogItem, IndicatorCatalogParam } from '@/gen/ant/v1/indicator_catalog_pb';
import { useTranslation } from 'react-i18next';
import {
  PARAM_KEY_KEY, PARAM_LABEL_KEY, PARAM_TYPE_KEY, PARAM_DEFAULT_KEY,
  PARAM_RANGE_KEY, PARAM_DESCRIPTION_KEY, RISK_PARAMS_TITLE_KEY,
  RISK_PARAMS_DESC_KEY, INDICATORS_TITLE_KEY, RISK_SECTION_TITLE_KEY,
} from '@/gen/ant/v1/i18n/indicator_catalog_keys';
import { COMMON_LOADING_FAILED_KEY } from '@/gen/ant/v1/i18n/base_keys';

const { Paragraph, Text } = Typography;

export default function IndicatorCatalogContent() {
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

  if (loading) return <div style={{ display: 'flex', justifyContent: 'center', padding: 80 }}><Spin size="large" /></div>;
  if (error) return <Alert type="error" message={t(COMMON_LOADING_FAILED_KEY)} description={error} showIcon />;
  if (!data) return null;

  const paramColumns = [
    { title: t(PARAM_KEY_KEY), dataIndex: 'key', key: 'key', width: 180 },
    { title: t(PARAM_LABEL_KEY), dataIndex: 'label', key: 'label', width: 180 },
    { title: t(PARAM_TYPE_KEY), dataIndex: 'type', key: 'type', width: 100, render: (v: string) => <Tag>{v}</Tag> },
    { title: t(PARAM_DEFAULT_KEY), dataIndex: 'default', key: 'default', width: 100 },
    { title: t(PARAM_RANGE_KEY), key: 'range', width: 140, render: (_: unknown, r: IndicatorCatalogParam) => r.min !== r.max ? `${r.min} ~ ${r.max}` : '-' },
    { title: t(PARAM_DESCRIPTION_KEY), dataIndex: 'description', key: 'description' },
  ];

  const indicatorItems = data.indicators.map((ind: IndicatorCatalogItem) => ({
    key: ind.name,
    label: (
      <span style={{ fontFamily: 'monospace' }}>
        <FunctionOutlined style={{ marginRight: 8, color: '#D4AF37' }} />
        <Text code style={{ fontSize: 13 }}>{ind.callSignature}</Text>
      </span>
    ),
    children: (
      <div>
        <Paragraph type="secondary" style={{ marginBottom: 12 }}>{ind.description}</Paragraph>
        <Table dataSource={ind.paramKeys.map((p: IndicatorCatalogParam, i: number) => ({ ...p, _key: i }))} columns={paramColumns} rowKey="_key" pagination={false} size="small" bordered />
      </div>
    ),
  }));

  const riskItems = [{
    key: 'risk',
    label: (
      <span>
        <SafetyOutlined style={{ marginRight: 8, color: '#E53935' }} />
        <Text strong>{t(RISK_PARAMS_TITLE_KEY)}</Text>
      </span>
    ),
    children: (
      <div>
        <Paragraph type="secondary" style={{ marginBottom: 12 }}>{t(RISK_PARAMS_DESC_KEY)}</Paragraph>
        <Table dataSource={data.riskParams.map((p: IndicatorCatalogParam, i: number) => ({ ...p, _key: i }))} columns={paramColumns} rowKey="_key" pagination={false} size="small" bordered />
      </div>
    ),
  }];

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <Card title={t(INDICATORS_TITLE_KEY)}>
        <Collapse items={indicatorItems} bordered={false} />
      </Card>
      <Card title={t(RISK_SECTION_TITLE_KEY)}>
        <Collapse items={riskItems} bordered={false} defaultActiveKey={['risk']} />
      </Card>
    </div>
  );
}
