import React from 'react';
import { Input, Select, Space, Button, Typography } from 'antd';
import { RocketOutlined } from '@ant-design/icons';
import type { TFunction } from 'i18next';

const { Text } = Typography;
const { TextArea } = Input;

interface Props {
  t: TFunction;
  description: string;
  setDescription: (v: string) => void;
  assetClass: string;
  setAssetClass: (v: string) => void;
  symbol: string;
  setSymbol: (v: string) => void;
  timeframe: string;
  setTimeframe: (v: string) => void;
  riskLevel: string;
  setRiskLevel: (v: string) => void;
  strategyType: string;
  setStrategyType: (v: string) => void;
  autoPublish: boolean;
  setAutoPublish: (v: boolean) => void;
  onGenerate: () => void;
}

export default function FreeFormConfig({
  t, description, setDescription,
  assetClass, setAssetClass,
  symbol, setSymbol,
  timeframe, setTimeframe,
  riskLevel, setRiskLevel,
  strategyType, setStrategyType,
  autoPublish, setAutoPublish,
  onGenerate,
}: Props) {
  return (
    <>
      <div style={{ marginBottom: 12 }}>
        <Text strong>{t('marketplace.autogen.description')}</Text>
        <TextArea
          value={description}
          onChange={e => setDescription(e.target.value)}
          placeholder={t('marketplace.autogen.placeholder')}
          rows={4}
          style={{ marginTop: 4 }}
        />
      </div>

      <Space size="large" wrap style={{ marginBottom: 16 }}>
        <div>
          <Text type="secondary" style={{ display: 'block', marginBottom: 4 }}>{t('marketplace.autogen.assetClass')}</Text>
          <Select value={assetClass} onChange={setAssetClass} style={{ width: 120 }}
            options={[
              { value: 'forex', label: 'Forex' },
              { value: 'crypto', label: 'Crypto' },
              { value: 'commodity', label: 'Commodity' },
              { value: 'index', label: 'Index' },
            ]}
          />
        </div>
        <div>
          <Text type="secondary" style={{ display: 'block', marginBottom: 4 }}>{t('marketplace.autogen.symbol')}</Text>
          <Input value={symbol} onChange={e => setSymbol(e.target.value)} style={{ width: 120 }} />
        </div>
        <div>
          <Text type="secondary" style={{ display: 'block', marginBottom: 4 }}>{t('marketplace.autogen.timeframe')}</Text>
          <Select value={timeframe} onChange={setTimeframe} style={{ width: 100 }}
            options={['M5', 'M15', 'M30', 'H1', 'H4', 'D1'].map(tf => ({ value: tf, label: tf }))}
          />
        </div>
        <div>
          <Text type="secondary" style={{ display: 'block', marginBottom: 4 }}>{t('marketplace.autogen.risk')}</Text>
          <Select value={riskLevel} onChange={setRiskLevel} style={{ width: 140 }}
            options={[
              { value: 'conservative', label: 'Conservative' },
              { value: 'moderate', label: 'Moderate' },
              { value: 'aggressive', label: 'Aggressive' },
            ]}
          />
        </div>
        <div>
          <Text type="secondary" style={{ display: 'block', marginBottom: 4 }}>{t('marketplace.autogen.type')}</Text>
          <Select value={strategyType} onChange={setStrategyType} style={{ width: 160 }}
            options={[
              { value: 'auto', label: 'Auto-detect' },
              { value: 'trend_following', label: 'Trend Following' },
              { value: 'mean_reversion', label: 'Mean Reversion' },
              { value: 'breakout', label: 'Breakout' },
            ]}
          />
        </div>
      </Space>

      <Space>
        <Button type="primary" icon={<RocketOutlined />} onClick={onGenerate} size="large">
          {t('marketplace.autogen.start')}
        </Button>
        <Button onClick={() => setAutoPublish(!autoPublish)} type={autoPublish ? 'default' : 'dashed'}>
          {autoPublish ? t('marketplace.autogen.autoPublishOn') : t('marketplace.autogen.autoPublishOff')}
        </Button>
      </Space>
    </>
  );
}
