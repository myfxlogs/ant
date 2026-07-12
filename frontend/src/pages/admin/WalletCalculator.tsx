import { useState, useMemo } from 'react';
import { InputNumber, Select, Space, Divider, Typography, Button } from 'antd';
import { CalculatorOutlined } from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { aiGatewayApi } from '@/client/aiGateway';

const { Text } = Typography;

interface CalculatorProps {
  onFillAdjust: (usd: string, tokens: string, modelLabel: string) => void;
}

export default function WalletCalculator({ onFillAdjust }: CalculatorProps) {
  const { t } = useTranslation();
  const [calcModel, setCalcModel] = useState<string | undefined>();
  const [calcUSD, setCalcUSD] = useState<number>(0.10);
  const [calcTokens, setCalcTokens] = useState<number>(714285);
  const [editingField, setEditingField] = useState<'usd' | 'tokens'>('usd');

  const { data: systemModels } = useQuery({
    queryKey: ['admin', 'ai-gateway', 'models'],
    queryFn: () => aiGatewayApi.listSystemModels().catch(() => []),
  });

  const modelOptions = useMemo(() =>
    (systemModels || []).map(m => ({
      value: m.id,
      label: `${m.displayName || m.modelName} ($${parseFloat(m.pricePer1mInput).toFixed(2)}/1M)`,
      pricePerToken: (parseFloat(m.pricePer1mInput) + parseFloat(m.pricePer1mOutput)) / 2 / 1000000,
    })),
    [systemModels]);

  const selectedModel = modelOptions.find(m => m.value === calcModel);

  const handleUSDChange = (v: number) => {
    setCalcUSD(v);
    setEditingField('usd');
    if (selectedModel && v > 0) {
      setCalcTokens(Math.round(v / selectedModel.pricePerToken));
    }
  };

  const handleTokensChange = (v: number) => {
    setCalcTokens(v);
    setEditingField('tokens');
    if (selectedModel && v > 0) {
      setCalcUSD(parseFloat((v * selectedModel.pricePerToken).toFixed(8)));
    }
  };

  const handleModelChange = (modelId: string) => {
    setCalcModel(modelId);
    const m = modelOptions.find(x => x.value === modelId);
    if (m) {
      if (editingField === 'usd') {
        if (calcUSD > 0) setCalcTokens(Math.round(calcUSD / m.pricePerToken));
      } else {
        if (calcTokens > 0) setCalcUSD(parseFloat((calcTokens * m.pricePerToken).toFixed(8)));
      }
    }
  };

  const fillAdjust = () => {
    onFillAdjust(String(calcUSD), String(calcTokens), selectedModel?.label?.split(' ')[0] || '');
  };

  return (
    <div style={{ background: '#f6ffed', borderRadius: 8, padding: 12, border: '1px solid #b7eb8f' }}>
      <div style={{ fontSize: 12, color: '#52c41a', marginBottom: 8, fontWeight: 500 }}>
        <CalculatorOutlined /> {t('admin.walletCalculator.title', { defaultValue: 'Token ↔ USD Calculator' })}
      </div>
      <Select
        showSearch
        value={calcModel}
        onChange={handleModelChange}
        style={{ width: '100%', marginBottom: 12 }}
        placeholder={t('admin.walletCalculator.selectModel', { defaultValue: 'Select model (pricing basis)' })}
        options={modelOptions}
        filterOption={(input, option) =>
          (option?.label as string || '').toLowerCase().includes(input.toLowerCase())
        }
      />
      {selectedModel && (
        <Space style={{ width: '100%' }} size={12}>
          <div style={{ flex: 1 }}>
            <Text type="secondary" style={{ fontSize: 11, display: 'block', marginBottom: 4 }}>{t('admin.walletCalculator.usdAmount', { defaultValue: 'USD Amount' })}</Text>
            <InputNumber
              value={calcUSD}
              onChange={(v) => handleUSDChange(v || 0)}
              min={0}
              step={0.01}
              style={{ width: '100%' }}
              addonBefore="$"
              precision={8}
            />
          </div>
          <div style={{ textAlign: 'center', paddingTop: 18 }}>
            <Text type="secondary">⇄</Text>
          </div>
          <div style={{ flex: 1 }}>
            <Text type="secondary" style={{ fontSize: 11, display: 'block', marginBottom: 4 }}>{t('admin.walletCalculator.tokenAmount', { defaultValue: 'Token Amount' })}</Text>
            <InputNumber
              value={calcTokens}
              onChange={(v) => handleTokensChange(v || 0)}
              min={0}
              step={10000}
              style={{ width: '100%' }}
              addonAfter="tokens"
            />
          </div>
        </Space>
      )}
      <Divider style={{ margin: '12px 0 0' }} />
      <Button size="small" icon={<CalculatorOutlined />} onClick={fillAdjust} style={{ marginTop: 12 }}>
        {t('admin.walletCalculator.fillResult', { defaultValue: 'Fill Result' })}
      </Button>
    </div>
  );
}
