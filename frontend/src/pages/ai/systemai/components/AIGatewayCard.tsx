import { useEffect, useState } from 'react';
import { Card, Select, Tag, Typography, Row, Col, Progress, Button, Spin, message } from 'antd';
import { ThunderboltOutlined, ApiOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next'
import { GATEWAY_MODEL_PLACEHOLDER_KEY, GATEWAY_MONTHLY_COST_KEY, GATEWAY_MONTHLY_TOKENS_KEY, GATEWAY_NO_MODELS_KEY, GATEWAY_SELECT_MODEL_KEY, GATEWAY_TITLE_KEY, GATEWAY_USAGE_BY_FEATURE_KEY, GATEWAY_USE_GATEWAY_DESC_KEY, GATEWAY_USE_GATEWAY_KEY, GATEWAY_USE_OWN_KEY_DESC_KEY, GATEWAY_USE_OWN_KEY_HINT_KEY, GATEWAY_USE_OWN_KEY_KEY } from '@/gen/ant/v1/i18n/ai_core_keys';

;
import { aiGatewayApi, type SystemModelInfo, type TokenUsageInfo } from '@/client/aiGateway';
import { aiApi } from '@/client/ai';

const { Text, Title } = Typography;

interface AIGatewayCardProps {
  useGateway: boolean;
  onToggle: (v: boolean) => void;
  selectedModel: string | undefined;
  onModelChange: (v: string) => void;
}

export default function AIGatewayCard({ useGateway, onToggle, selectedModel, onModelChange }: AIGatewayCardProps) {
  const { t } = useTranslation();
  const [models, setModels] = useState<SystemModelInfo[]>([]);
  const [modelsLoading, setModelsLoading] = useState(false);
  const [usage, setUsage] = useState<TokenUsageInfo | null>(null);
  const [usageLoading, setUsageLoading] = useState(false);

  useEffect(() => {
    if (!useGateway) return;
    setModelsLoading(true);
    aiGatewayApi.listSystemModels()
      .then(setModels)
      .catch((e) => { console.error('listSystemModels failed', e); })
      .finally(() => setModelsLoading(false));
  }, [useGateway]);

  useEffect(() => {
    if (!useGateway) return;
    setUsageLoading(true);
    aiGatewayApi.getTokenUsage()
      .then(setUsage)
      .catch((e) => { console.error('getTokenUsage failed', e); })
      .finally(() => setUsageLoading(false));
  }, [useGateway]);

  const handleModelSelect = async (modelId: string) => {
    onModelChange(modelId);
    const m = models.find(x => x.id === modelId);
    if (m) {
      // Save as global default — all 15 AI callers use this model.
      try {
        await aiApi.setPrimary({ providerId: m.providerId, model: m.modelName });
      } catch { /* non-blocking */ }
      const priceIn = parseFloat(m.pricePer1mInput || '0');
      message.success({
        content: t('ai.gateway.modelSelected', '已选择 {{model}} · ${{price}}/1M tokens', { model: m.displayName || m.modelName, price: priceIn.toFixed(2) }),
        duration: 2,
      });
    }
  };

  const totalTokens = Object.values(usage?.featureTokens || {}).reduce((a, b) => a + b, 0);
  const featureBars = Object.entries(usage?.featureTokens || {}).map(([name, tokens]) => ({
    name,
    tokens,
    pct: totalTokens > 0 ? Math.round((tokens / totalTokens) * 100) : 0,
  }));

  return (
    <Card
      title={
        <span style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <ApiOutlined style={{ color: '#722ed1' }} />
          <span>{t(GATEWAY_TITLE_KEY, 'AI 网关')}</span>
          <Tag color="purple" style={{ fontSize: 11 }}>Beta</Tag>
        </span>
      }
      style={{ borderRadius: 12, borderColor: '#d3adf7', marginBottom: 16 }}
    >
      {/* Mode selector */}
      <Row gutter={[16, 12]} align="middle" style={{ marginBottom: 16 }}>
        <Col span={24}>
          <div style={{ display: 'flex', gap: 12, background: '#f5f5f5', borderRadius: 8, padding: 4 }}>
            <Button
              type={useGateway ? 'primary' : 'default'}
              size="small"
              style={{ flex: 1, borderRadius: 6 }}
              onClick={() => onToggle(true)}
            >
              <ThunderboltOutlined /> {t(GATEWAY_USE_GATEWAY_KEY, 'AI 网关')}
              <div style={{ fontSize: 10, color: useGateway ? '#fff' : '#8c8c8c' }}>
                {t(GATEWAY_USE_GATEWAY_DESC_KEY, '扣钱包余额 · 按 Token 计费')}
              </div>
            </Button>
            <Button
              type={!useGateway ? 'primary' : 'default'}
              size="small"
              style={{ flex: 1, borderRadius: 6 }}
              onClick={() => onToggle(false)}
            >
              <ApiOutlined /> {t(GATEWAY_USE_OWN_KEY_KEY, '我的 API Key')}
              <div style={{ fontSize: 10, color: !useGateway ? '#fff' : '#8c8c8c' }}>
                {t(GATEWAY_USE_OWN_KEY_DESC_KEY, '直付厂商 · 自行管理')}
              </div>
            </Button>
          </div>
        </Col>
      </Row>

      {/* Gateway mode content */}
      {useGateway && (
        <>
          <Row gutter={[16, 12]} align="middle" style={{ marginBottom: 16 }}>
            <Col xs={24} sm={8}>
              <Text type="secondary" style={{ fontSize: 12 }}>{t(GATEWAY_SELECT_MODEL_KEY, '选择模型')}</Text>
            </Col>
            <Col xs={24} sm={16}>
              {modelsLoading ? <Spin size="small" /> : (
                <Select
                  showSearch
                  value={selectedModel}
                  onChange={handleModelSelect}
                  style={{ width: '100%' }}
                  placeholder={t(GATEWAY_MODEL_PLACEHOLDER_KEY, '选择 AI 模型')}
                  notFoundContent={t(GATEWAY_NO_MODELS_KEY, '暂无可用模型')}
                  options={models.map(m => ({
                    value: m.id,
                    label: (
                      <span style={{ display: 'flex', justifyContent: 'space-between' }}>
                        <span>{m.displayName || m.modelName}</span>
                        <Tag color="blue" style={{ fontSize: 10, marginLeft: 8 }}>{m.providerId}</Tag>
                      </span>
                    ),
                  }))}
                />
              )}
            </Col>
          </Row>

          {/* Token stats — wallet balance is shown in the nav bar, unified wallet system */}
          <Row gutter={16} style={{ marginBottom: 12 }}>
            <Col span={12}>
              <Card size="small" style={{ textAlign: 'center', borderRadius: 8 }}>
                <ThunderboltOutlined style={{ fontSize: 18, color: '#722ed1' }} />
                <div style={{ fontSize: 12, color: '#8c8c8c', marginTop: 4 }}>{t(GATEWAY_MONTHLY_TOKENS_KEY, '本月 Token')}</div>
                <Title level={5} style={{ margin: 0, color: '#722ed1' }}>
                  {usageLoading ? <Spin size="small" /> : `${(totalTokens / 1000).toFixed(1)}K`}
                </Title>
              </Card>
            </Col>
            <Col span={12}>
              <Card size="small" style={{ textAlign: 'center', borderRadius: 8 }}>
                <span style={{ fontSize: 18 }}>💸</span>
                <div style={{ fontSize: 12, color: '#8c8c8c', marginTop: 4 }}>{t(GATEWAY_MONTHLY_COST_KEY, '本月费用')}</div>
                <Title level={5} style={{ margin: 0, color: '#52c41a' }}>
                  {parseFloat(usage?.monthlyCost || '0').toFixed(4)}$</Title>
              </Card>
            </Col>
          </Row>

          {/* Per-feature token usage */}
          {featureBars.length > 0 && (
            <>
              <Text type="secondary" style={{ fontSize: 12, display: 'block', marginBottom: 8 }}>
                {t(GATEWAY_USAGE_BY_FEATURE_KEY, '按功能用量')}
              </Text>
              {featureBars.map(f => (
                <div key={f.name} style={{ marginBottom: 6 }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 12, marginBottom: 2 }}>
                    <span>{f.name}</span>
                    <span style={{ color: '#8c8c8c' }}>{(f.tokens / 1000).toFixed(0)}K tokens</span>
                  </div>
                  <Progress percent={f.pct} size="small" showInfo={false} strokeColor="#722ed1" trailColor="#f0f0f0" />
                </div>
              ))}
            </>
          )}
        </>
      )}

      {!useGateway && (
        <Text type="secondary" style={{ fontSize: 13 }}>
          {t(GATEWAY_USE_OWN_KEY_HINT_KEY, '使用你自己配置的 API Key，直接向所选厂商付费。在下方选择厂商卡片进行配置。')}
        </Text>
      )}
    </Card>
  );
}
