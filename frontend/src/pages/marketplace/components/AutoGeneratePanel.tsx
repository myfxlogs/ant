import { useState, useRef, useCallback } from 'react';
import { Card, Input, Select, Button, Typography, Space, Segmented, Modal, message } from 'antd';
import { RobotOutlined, RocketOutlined, AppstoreOutlined, EditOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { marketplaceClient } from '@/client/connect';
import { create } from '@bufbuild/protobuf';
import { GenerateAndPublishRequestSchema, GenerateFromTemplateRequestSchema, SetStrategyPricingRequestSchema } from '@/gen/ant/v1/marketplace_service_pb';
import TemplateSelector from './TemplateSelector';
import AutoGenerateProgress from './AutoGenerateProgress';
import AutoGenerateResult from './AutoGenerateResult';

const { TextArea } = Input;
const { Text, Title } = Typography;

type Stage = 'idle' | 'generating' | 'compiling' | 'backtesting' | 'evaluating' | 'publishing' | 'completed' | 'failed';

export default function AutoGeneratePanel() {
  const { t } = useTranslation();
  const [mode, setMode] = useState<'freeform' | 'template'>('freeform');
  const [description, setDescription] = useState('');
  const [assetClass, setAssetClass] = useState('forex');
  const [symbol, setSymbol] = useState('EURUSD');
  const [timeframe, setTimeframe] = useState('H1');
  const [riskLevel, setRiskLevel] = useState('moderate');
  const [strategyType, setStrategyType] = useState('auto');
  const [autoPublish, setAutoPublish] = useState(true);
  const [stage, setStage] = useState<Stage>('idle');
  const [progress, setProgress] = useState(0);
  const [delta, setDelta] = useState('');
  const [errorStage, setErrorStage] = useState('');
  const [errorDetail, setErrorDetail] = useState('');
  const [retryable, setRetryable] = useState(false);
  const [result, setResult] = useState<{ strategyId: string; publishId: string; backtest: unknown } | null>(null);
  const [violations, setViolations] = useState<any[]>([]);
  const abortRef = useRef<AbortController | null>(null);
  const [pricingModalOpen, setPricingModalOpen] = useState(false);
  const [priceModel, setPriceModel] = useState('free');
  const [priceAmount, setPriceAmount] = useState('0');
  const [pricingSaving, setPricingSaving] = useState(false);

  const isRunning = stage !== 'idle' && stage !== 'completed' && stage !== 'failed';

  const handleSavePricing = useCallback(async () => {
    if (!result?.strategyId) return;
    setPricingSaving(true);
    try {
      await marketplaceClient.setStrategyPricing(create(SetStrategyPricingRequestSchema, {
        strategyId: result.strategyId,
        priceModel,
        priceAmount,
      }));
      message.success(t('marketplace.autogen.pricingSaved'));
      setPricingModalOpen(false);
    } catch (e: unknown) {
      message.error(e instanceof Error ? e.message : String(e) || t('marketplace.autogen.pricingFailed', { defaultValue: 'Failed to update pricing' }));
    } finally {
      setPricingSaving(false);
    }
  }, [result, priceModel, priceAmount, t]);

  const handleTemplateGenerate = useCallback(async (templateId: string, paramsJson: string) => {
    const ac = new AbortController();
    abortRef.current = ac;

    setStage('generating');
    setProgress(0);
    setDelta('');
    setErrorStage('');
    setErrorDetail('');
    setRetryable(false);
    setResult(null);
    setViolations([]);

    try {
      const msg = create(GenerateFromTemplateRequestSchema, {
        templateId,
        parametersJson: paramsJson,
        symbol,
        timeframe,
        autoPublish,
      });

      const stream = marketplaceClient.generateFromTemplate(msg, { signal: ac.signal });
      for await (const ev of stream) {
        const s = (ev.stage || 'generating') as Stage;
        setStage(s);
        if (ev.progress) setProgress(ev.progress);
        if (ev.delta) setDelta(prev => prev + ev.delta);
        if (ev.message) setDelta(prev => prev + ev.message + '\n');

        if (s === 'failed') {
          setErrorStage(ev.errorStage || '');
          setErrorDetail(ev.errorDetail || '');
          setRetryable(ev.retryable);
        } else if (s === 'completed') {
          if (ev.strategyId || ev.publishId) {
            setResult({ strategyId: ev.strategyId || '', publishId: ev.publishId || '', backtest: ev.backtest });
          }
          if (ev.violations && ev.violations.length > 0) {
            setViolations(ev.violations);
          }
        }
      }
    } catch (e: unknown) {
      if (e instanceof Error && e.name === 'AbortError') return;
      setStage('failed');
      setErrorStage('generating');
      setErrorDetail(e instanceof Error ? e.message : String(e));
      setRetryable(true);
    }
  }, [symbol, timeframe, autoPublish]);

  const handleGenerate = useCallback(async () => {
    if (!description.trim()) {
      message.warning(t('marketplace.autogen.needDescription'));
      return;
    }

    const ac = new AbortController();
    abortRef.current = ac;

    setStage('generating');
    setProgress(0);
    setDelta('');
    setErrorStage('');
    setErrorDetail('');
    setRetryable(false);
    setResult(null);
    setViolations([]);

    try {
      const msg = create(GenerateAndPublishRequestSchema, {
        description,
        assetClass,
        symbols: symbol ? [symbol] : [],
        timeframe,
        riskLevel,
        strategyType,
        autoPublish,
        language: 'en',
      });

      const stream = marketplaceClient.generateAndPublish(msg, { signal: ac.signal });
      for await (const ev of stream) {
        const s = (ev.stage || 'generating') as Stage;
        setStage(s);
        if (ev.progress) setProgress(ev.progress);
        if (ev.delta) setDelta(prev => prev + ev.delta);
        if (ev.message) setDelta(prev => prev + ev.message + '\n');

        if (s === 'failed') {
          setErrorStage(ev.errorStage || '');
          setErrorDetail(ev.errorDetail || '');
          setRetryable(ev.retryable);
        } else if (s === 'completed') {
          if (ev.strategyId || ev.publishId) {
            setResult({
              strategyId: ev.strategyId || '',
              publishId: ev.publishId || '',
              backtest: ev.backtest,
            });
          }
          if (ev.violations && ev.violations.length > 0) {
            setViolations(ev.violations);
          }
        }
      }
    } catch (e: unknown) {
      if (e instanceof Error && e.name === 'AbortError') return;
      setStage('failed');
      setErrorStage('generating');
      setErrorDetail(e instanceof Error ? e.message : String(e));
      setRetryable(true);
    }
  }, [description, assetClass, symbol, timeframe, riskLevel, strategyType, autoPublish, t]);

  const handleCancel = useCallback(() => {
    abortRef.current?.abort();
    setStage('idle');
  }, []);

  const handleReset = useCallback(() => {
    setStage('idle');
    setProgress(0);
    setDelta('');
    setResult(null);
    setViolations([]);
  }, []);

  return (
    <Card>
      <div style={{ marginBottom: 16 }}>
        <Title level={4}><RobotOutlined style={{ marginRight: 8 }} />{t('marketplace.autogen.title')}</Title>
        <Text type="secondary">{t('marketplace.autogen.subtitle')}</Text>
      </div>

      {stage === 'idle' && (
        <>
          <Segmented
            value={mode}
            onChange={v => setMode(v as 'freeform' | 'template')}
            options={[
              { value: 'freeform', label: <><EditOutlined /> {t('marketplace.autogen.modes.freeform')}</>, icon: <EditOutlined /> },
              { value: 'template', label: <><AppstoreOutlined /> {t('marketplace.autogen.modes.template')}</>, icon: <AppstoreOutlined /> },
            ]}
            style={{ marginBottom: 16 }}
          />

          {mode === 'template' ? (
            <TemplateSelector
              symbol={symbol}
              timeframe={timeframe}
              autoPublish={autoPublish}
              onGenerate={handleTemplateGenerate}
              onSymbolChange={setSymbol}
              onTimeframeChange={setTimeframe}
            />
          ) : (
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
            <Button type="primary" icon={<RocketOutlined />} onClick={handleGenerate} size="large">
              {t('marketplace.autogen.start')}
            </Button>
            <Button onClick={() => setAutoPublish(!autoPublish)} type={autoPublish ? 'default' : 'dashed'}>
              {autoPublish ? t('marketplace.autogen.autoPublishOn') : t('marketplace.autogen.autoPublishOff')}
            </Button>
          </Space>
            </>
          )}
        </>
      )}

      {isRunning && (
        <AutoGenerateProgress stage={stage} progress={progress} delta={delta} onCancel={handleCancel} t={t} />
      )}

      {stage === 'failed' && (
        <AutoGenerateResult
          stage="failed"
          result={null}
          violations={[]}
          errorStage={errorStage}
          errorDetail={errorDetail}
          retryable={retryable}
          onRetry={handleGenerate}
          onReset={handleReset}
          onEditPricing={() => {}}
          t={t}
        />
      )}

      {stage === 'completed' && (
        <AutoGenerateResult
          stage="completed"
          result={result}
          violations={violations}
          errorStage=""
          errorDetail=""
          retryable={false}
          onRetry={() => {}}
          onReset={handleReset}
          onEditPricing={() => setPricingModalOpen(true)}
          t={t}
        />
      )}
      <Modal
        title={t('marketplace.autogen.editPricing')}
        open={pricingModalOpen}
        onCancel={() => setPricingModalOpen(false)}
        onOk={handleSavePricing}
        confirmLoading={pricingSaving}
        okText={t('marketplace.autogen.save')}
      >
        <Space direction="vertical" style={{ width: '100%' }}>
          <div>
            <Text type="secondary" style={{ display: 'block', marginBottom: 4 }}>{t('marketplace.autogen.priceModel')}</Text>
            <Select value={priceModel} onChange={setPriceModel} style={{ width: '100%' }}
              options={[
                { value: 'free', label: t('marketplace.autogen.pricingFree') },
                { value: 'once', label: t('marketplace.autogen.pricingOnce') },
                { value: 'subscription', label: t('marketplace.autogen.pricingSubscription') },
              ]}
            />
          </div>
          {priceModel !== 'free' && (
            <div>
              <Text type="secondary" style={{ display: 'block', marginBottom: 4 }}>{t('marketplace.autogen.priceAmount')}</Text>
              <Input value={priceAmount} onChange={e => setPriceAmount(e.target.value)} type="number" prefix="$" />
            </div>
          )}
        </Space>
      </Modal>
    </Card>
  );
}
