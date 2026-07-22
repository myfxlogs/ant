import { useState, useEffect, useCallback } from 'react';
import { Card, Select, Button, Input, Typography, Space, Row, Col, Spin, message } from 'antd';
import { AppstoreOutlined, ThunderboltOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { marketplaceClient } from '@/client/connect';
import { create } from '@bufbuild/protobuf';
import {
  ListStrategyTemplatesRequestSchema,
  type StrategyParameterTemplate,
} from '@/gen/ant/v1/marketplace_service_pb';

const { Text, Title } = Typography;

interface ParamField {
  key: string;
  label: string;
  type: 'number' | 'string' | 'select';
  default?: string | number;
  options?: { value: string; label: string }[];
  min?: number;
  max?: number;
  step?: number;
}

function parseParamSchema(schemaJson: string): ParamField[] {
  try {
    const schema = JSON.parse(schemaJson);
    if (Array.isArray(schema.fields)) return schema.fields;
    if (Array.isArray(schema)) return schema;
    return [];
  } catch {
    return [];
  }
}

interface TemplateSelectorProps {
  symbol: string;
  timeframe: string;
  autoPublish: boolean;
  onGenerate: (templateId: string, paramsJson: string) => void;
  onSymbolChange: (v: string) => void;
  onTimeframeChange: (v: string) => void;
}

export default function TemplateSelector({
  symbol,
  timeframe,
  autoPublish,
  onGenerate,
  onSymbolChange,
  onTimeframeChange,
}: TemplateSelectorProps) {
  const { t } = useTranslation();
  const [templates, setTemplates] = useState<StrategyParameterTemplate[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedTpl, setSelectedTpl] = useState<StrategyParameterTemplate | null>(null);
  const [paramValues, setParamValues] = useState<Record<string, string | number>>({});

  useEffect(() => {
    let cancelled = false;
    (async () => {
      setLoading(true);
      try {
        const req = create(ListStrategyTemplatesRequestSchema, {});
        const resp = await marketplaceClient.listStrategyTemplates(req);
        if (!cancelled) {
          setTemplates(resp.templates);
        }
      } catch {
        if (!cancelled) {
          message.error(t('marketplace.autogen.templates.loadError'));
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => { cancelled = true; };
  }, [t]);

  const fields = selectedTpl ? parseParamSchema(selectedTpl.parametersSchema) : [];

  const handleSelect = useCallback((tpl: StrategyParameterTemplate) => {
    setSelectedTpl(tpl);
    const defaults: Record<string, string | number> = {};
    for (const f of parseParamSchema(tpl.parametersSchema)) {
      if (f.default !== undefined) defaults[f.key] = f.default;
    }
    setParamValues(defaults);
  }, []);

  const handleGenerate = useCallback(() => {
    if (!selectedTpl) return;
    onGenerate(selectedTpl.id, JSON.stringify(paramValues));
  }, [selectedTpl, paramValues, onGenerate]);

  if (loading) {
    return <div style={{ textAlign: 'center', padding: 40 }}><Spin /></div>;
  }

  if (selectedTpl) {
    return (
      <div>
        <div style={{ marginBottom: 16 }}>
          <Button type="link" onClick={() => setSelectedTpl(null)} style={{ paddingLeft: 0 }}>
            ← {t('marketplace.autogen.templates.backToList')}
          </Button>
        </div>
        <Card size="small" style={{ marginBottom: 16 }}>
          <Title level={5}>{selectedTpl.name}</Title>
          <Text type="secondary">{selectedTpl.description}</Text>
        </Card>

        {fields.length > 0 && (
          <div style={{ marginBottom: 16 }}>
            <Text strong style={{ display: 'block', marginBottom: 8 }}>
              {t('marketplace.autogen.templates.parameters')}
            </Text>
            <Row gutter={[12, 12]}>
              {fields.map(f => (
                <Col key={f.key} span={8}>
                  <div>
                    <Text type="secondary" style={{ display: 'block', marginBottom: 4 }}>{f.label}</Text>
                    {f.type === 'select' ? (
                      <Select
                        style={{ width: '100%' }}
                        value={String(paramValues[f.key] ?? '')}
                        options={f.options}
                        onChange={v => setParamValues(prev => ({ ...prev, [f.key]: v }))}
                      />
                    ) : (
                      <Input
                        type={f.type === 'number' ? 'number' : 'text'}
                        style={{ width: '100%' }}
                        value={paramValues[f.key] ?? ''}
                        min={f.min}
                        max={f.max}
                        step={f.step}
                        onChange={e => setParamValues(prev => ({
                          ...prev,
                          [f.key]: f.type === 'number' ? Number(e.target.value) : e.target.value,
                        }))}
                      />
                    )}
                  </div>
                </Col>
              ))}
            </Row>
          </div>
        )}

        <Space size="large" wrap style={{ marginBottom: 16 }}>
          <div>
            <Text type="secondary" style={{ display: 'block', marginBottom: 4 }}>
              {t('marketplace.autogen.symbol')}
            </Text>
            <Input value={symbol} onChange={e => onSymbolChange(e.target.value)} style={{ width: 120 }} />
          </div>
          <div>
            <Text type="secondary" style={{ display: 'block', marginBottom: 4 }}>
              {t('marketplace.autogen.timeframe')}
            </Text>
            <Select value={timeframe} onChange={onTimeframeChange} style={{ width: 100 }}
              options={['M5', 'M15', 'M30', 'H1', 'H4', 'D1'].map(tf => ({ value: tf, label: tf }))}
            />
          </div>
        </Space>

        <Button type="primary" icon={<ThunderboltOutlined />} onClick={handleGenerate} size="large">
          {t('marketplace.autogen.templates.generateFromTemplate')}
        </Button>
      </div>
    );
  }

  return (
    <div>
      <div style={{ marginBottom: 12 }}>
        <Text strong>
          <AppstoreOutlined style={{ marginRight: 8 }} />
          {t('marketplace.autogen.templates.title')}
        </Text>
        <Text type="secondary" style={{ marginLeft: 8 }}>
          {t('marketplace.autogen.templates.subtitle')}
        </Text>
      </div>
      {templates.length === 0 ? (
        <Text type="secondary">{t('marketplace.autogen.templates.empty')}</Text>
      ) : (
        <Row gutter={[12, 12]}>
          {templates.map(tpl => (
            <Col key={tpl.id} span={8}>
              <Card
                hoverable
                size="small"
                onClick={() => handleSelect(tpl)}
                style={{ height: '100%' }}
              >
                <div style={{ display: 'flex', alignItems: 'center', marginBottom: 8 }}>
                  {tpl.icon && <span style={{ fontSize: 20, marginRight: 8 }}>{tpl.icon}</span>}
                  <Text strong>{tpl.name}</Text>
                </div>
                <Text type="secondary" style={{ fontSize: 12 }}>{tpl.description}</Text>
              </Card>
            </Col>
          ))}
        </Row>
      )}
    </div>
  );
}
