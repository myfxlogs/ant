import { useState, useRef, useCallback } from 'react';
import { Card, Typography, Segmented, message } from 'antd';
import { RobotOutlined, AppstoreOutlined, EditOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { marketplaceClient } from '@/client/connect';
import { create } from '@bufbuild/protobuf';
import { GenerateAndPublishRequestSchema, GenerateFromTemplateRequestSchema, SetStrategyPricingRequestSchema } from '@/gen/ant/v1/marketplace_service_pb';
import TemplateSelector from './TemplateSelector';
import AutoGenerateProgress from './AutoGenerateProgress';
import AutoGenerateResult from './AutoGenerateResult';
import FreeFormConfig from './FreeFormConfig';
import PricingModal from './PricingModal';
import { resetGenerationState, processStreamEvent, handleStreamError, type Stage } from './AutoGenerateStreamHelpers';

const { Text, Title } = Typography;

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
  const [violations, setViolations] = useState<Array<{ metric: string; actual: string | number; threshold: string | number }>>([]);
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
    resetGenerationState({ setStage, setProgress, setDelta, setErrorStage, setErrorDetail, setRetryable, setResult, setViolations });

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
        processStreamEvent(ev, { setStage, setProgress, setDelta, setErrorStage, setErrorDetail, setRetryable, setResult, setViolations });
      }
    } catch (e: unknown) {
      handleStreamError(e, { setStage, setProgress, setDelta, setErrorStage, setErrorDetail, setRetryable, setResult, setViolations });
    }
  }, [symbol, timeframe, autoPublish]);

  const handleGenerate = useCallback(async () => {
    if (!description.trim()) {
      message.warning(t('marketplace.autogen.needDescription'));
      return;
    }

    const ac = new AbortController();
    abortRef.current = ac;
    resetGenerationState({ setStage, setProgress, setDelta, setErrorStage, setErrorDetail, setRetryable, setResult, setViolations });

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
        processStreamEvent(ev, { setStage, setProgress, setDelta, setErrorStage, setErrorDetail, setRetryable, setResult, setViolations });
      }
    } catch (e: unknown) {
      handleStreamError(e, { setStage, setProgress, setDelta, setErrorStage, setErrorDetail, setRetryable, setResult, setViolations });
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
            <FreeFormConfig
              t={t}
              description={description}
              setDescription={setDescription}
              assetClass={assetClass}
              setAssetClass={setAssetClass}
              symbol={symbol}
              setSymbol={setSymbol}
              timeframe={timeframe}
              setTimeframe={setTimeframe}
              riskLevel={riskLevel}
              setRiskLevel={setRiskLevel}
              strategyType={strategyType}
              setStrategyType={setStrategyType}
              autoPublish={autoPublish}
              setAutoPublish={setAutoPublish}
              onGenerate={handleGenerate}
            />
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
      <PricingModal
        t={t}
        open={pricingModalOpen}
        priceModel={priceModel}
        setPriceModel={setPriceModel}
        priceAmount={priceAmount}
        setPriceAmount={setPriceAmount}
        saving={pricingSaving}
        onSave={handleSavePricing}
        onCancel={() => setPricingModalOpen(false)}
      />
    </Card>
  );
}
