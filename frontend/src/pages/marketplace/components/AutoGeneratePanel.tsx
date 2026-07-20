import { useState, useRef, useCallback } from 'react';
import { Card, Input, Select, Button, Steps, Alert, Typography, Space, Tag, Statistic, Row, Col, Progress, message } from 'antd';
import { RobotOutlined, RocketOutlined, CheckCircleOutlined, CloseCircleOutlined, LoadingOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { marketplaceClient } from '@/client/connect';
import { create } from '@bufbuild/protobuf';
import { GenerateAndPublishRequestSchema } from '@/gen/ant/v1/marketplace_service_pb';

const { TextArea } = Input;
const { Text, Title } = Typography;

type Stage = 'idle' | 'generating' | 'compiling' | 'backtesting' | 'evaluating' | 'publishing' | 'completed' | 'failed';

interface StageInfo {
  stage: Stage;
  label: string;
  status: 'wait' | 'process' | 'finish' | 'error';
}

const STAGE_ORDER: Stage[] = ['generating', 'compiling', 'backtesting', 'evaluating', 'publishing', 'completed'];

function stageToStepIndex(stage: Stage): number {
  const idx = STAGE_ORDER.indexOf(stage);
  return idx < 0 ? 0 : idx;
}

export default function AutoGeneratePanel() {
  const { t } = useTranslation();
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
  const [result, setResult] = useState<{ strategyId: string; publishId: string; backtest: any } | null>(null);
  const [violations, setViolations] = useState<any[]>([]);
  const abortRef = useRef<AbortController | null>(null);

  const isRunning = stage !== 'idle' && stage !== 'completed' && stage !== 'failed';

  const handleGenerate = useCallback(async () => {
    if (!description.trim()) {
      message.warning(t('marketplace.autogen.needDescription', { defaultValue: 'Please describe your strategy' }));
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
    } catch (e: any) {
      if (e?.name === 'AbortError') return;
      setStage('failed');
      setErrorStage('generating');
      setErrorDetail(e?.message || 'Generation failed');
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

  const steps: StageInfo[] = STAGE_ORDER.map(s => {
    const currentIdx = stageToStepIndex(stage);
    const idx = STAGE_ORDER.indexOf(s);
    let status: 'wait' | 'process' | 'finish' | 'error' = 'wait';
    if (stage === 'failed') {
      if (idx < currentIdx) status = 'finish';
      else if (idx === currentIdx) status = 'error';
    } else if (idx < currentIdx) {
      status = 'finish';
    } else if (idx === currentIdx) {
      status = 'process';
    }
    return { stage: s, label: t(`marketplace.autogen.stages.${s}`, { defaultValue: s }), status };
  });

  return (
    <Card>
      <div style={{ marginBottom: 16 }}>
        <Title level={4}><RobotOutlined style={{ marginRight: 8 }} />{t('marketplace.autogen.title', { defaultValue: 'AI Strategy Generation' })}</Title>
        <Text type="secondary">{t('marketplace.autogen.subtitle', { defaultValue: 'Describe your strategy in natural language — AI will generate, compile, backtest, and publish it.' })}</Text>
      </div>

      {stage === 'idle' && (
        <>
          <div style={{ marginBottom: 12 }}>
            <Text strong>{t('marketplace.autogen.description', { defaultValue: 'Describe your strategy' })}</Text>
            <TextArea
              value={description}
              onChange={e => setDescription(e.target.value)}
              placeholder={t('marketplace.autogen.placeholder', { defaultValue: 'e.g. Trend following on EURUSD H1 using EMA crossover, 50 pip stop loss, 100 pip take profit...' })}
              rows={4}
              style={{ marginTop: 4 }}
            />
          </div>

          <Space size="large" wrap style={{ marginBottom: 16 }}>
            <div>
              <Text type="secondary" style={{ display: 'block', marginBottom: 4 }}>{t('marketplace.autogen.assetClass', { defaultValue: 'Asset Class' })}</Text>
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
              <Text type="secondary" style={{ display: 'block', marginBottom: 4 }}>{t('marketplace.autogen.symbol', { defaultValue: 'Symbol' })}</Text>
              <Input value={symbol} onChange={e => setSymbol(e.target.value)} style={{ width: 120 }} />
            </div>
            <div>
              <Text type="secondary" style={{ display: 'block', marginBottom: 4 }}>{t('marketplace.autogen.timeframe', { defaultValue: 'Timeframe' })}</Text>
              <Select value={timeframe} onChange={setTimeframe} style={{ width: 100 }}
                options={['M5', 'M15', 'M30', 'H1', 'H4', 'D1'].map(tf => ({ value: tf, label: tf }))}
              />
            </div>
            <div>
              <Text type="secondary" style={{ display: 'block', marginBottom: 4 }}>{t('marketplace.autogen.risk', { defaultValue: 'Risk Level' })}</Text>
              <Select value={riskLevel} onChange={setRiskLevel} style={{ width: 140 }}
                options={[
                  { value: 'conservative', label: 'Conservative' },
                  { value: 'moderate', label: 'Moderate' },
                  { value: 'aggressive', label: 'Aggressive' },
                ]}
              />
            </div>
            <div>
              <Text type="secondary" style={{ display: 'block', marginBottom: 4 }}>{t('marketplace.autogen.type', { defaultValue: 'Strategy Type' })}</Text>
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
              {t('marketplace.autogen.start', { defaultValue: 'Start Generation' })}
            </Button>
            <Button onClick={() => setAutoPublish(!autoPublish)} type={autoPublish ? 'default' : 'dashed'}>
              {autoPublish ? t('marketplace.autogen.autoPublishOn', { defaultValue: 'Auto-publish: ON' }) : t('marketplace.autogen.autoPublishOff', { defaultValue: 'Auto-publish: OFF' })}
            </Button>
          </Space>
        </>
      )}

      {isRunning && (
        <div>
          <Steps
            current={stageToStepIndex(stage)}
            items={steps.map(s => ({
              title: s.label,
              status: s.status,
              icon: s.status === 'process' ? <LoadingOutlined /> : s.status === 'error' ? <CloseCircleOutlined /> : s.status === 'finish' ? <CheckCircleOutlined /> : undefined,
            }))}
            size="small"
            style={{ marginBottom: 16 }}
          />
          <Progress percent={Math.round(progress * 100)} status="active" style={{ marginBottom: 16 }} />
          {delta && (
            <Card size="small" style={{ maxHeight: 200, overflow: 'auto', marginBottom: 16, background: 'var(--color-bg-secondary)' }}>
              <pre style={{ whiteSpace: 'pre-wrap', fontSize: 12, margin: 0 }}>{delta}</pre>
            </Card>
          )}
          <Button onClick={handleCancel} danger>{t('marketplace.autogen.cancel', { defaultValue: 'Cancel' })}</Button>
        </div>
      )}

      {stage === 'failed' && (
        <div>
          <Alert
            type="error"
            showIcon
            message={`${t('marketplace.autogen.failedAt', { defaultValue: 'Failed at' })}: ${errorStage}`}
            description={errorDetail}
            style={{ marginBottom: 16 }}
          />
          <Space>
            {retryable && <Button type="primary" onClick={handleGenerate}>{t('marketplace.autogen.retry', { defaultValue: 'Retry' })}</Button>}
            <Button onClick={handleReset}>{t('marketplace.autogen.modify', { defaultValue: 'Modify Request' })}</Button>
          </Space>
        </div>
      )}

      {stage === 'completed' && (
        <div>
          {violations.length > 0 ? (
            <Alert
              type="warning"
              showIcon
              message={t('marketplace.autogen.qualityFailed', { defaultValue: 'Strategy generated but did not pass quality gates' })}
              description={
                <div>
                  {violations.map((v, i) => (
                    <div key={i}>
                      <Tag color="orange">{v.metric}</Tag>
                      <Text>{t('marketplace.autogen.actual', { defaultValue: 'Actual' })}: {v.actual} / {t('marketplace.autogen.threshold', { defaultValue: 'Threshold' })}: {v.threshold}</Text>
                    </div>
                  ))}
                </div>
              }
              style={{ marginBottom: 16 }}
            />
          ) : (
            <Alert
              type="success"
              showIcon
              message={t('marketplace.autogen.success', { defaultValue: 'Strategy generated and published successfully!' })}
              style={{ marginBottom: 16 }}
            />
          )}

          {result?.backtest && (
            <Row gutter={16} style={{ marginBottom: 16 }}>
              <Col span={4}><Statistic title="Total Return" value={result.backtest.totalReturn} /></Col>
              <Col span={4}><Statistic title="Max DD" value={result.backtest.maxDrawdown} /></Col>
              <Col span={4}><Statistic title="Sharpe" value={result.backtest.sharpeRatio} /></Col>
              <Col span={4}><Statistic title="Win Rate" value={result.backtest.winRate} /></Col>
              <Col span={4}><Statistic title="Trades" value={result.backtest.totalTrades} /></Col>
            </Row>
          )}

          {result?.strategyId && (
            <Space>
              <Button type="primary" href={`#/marketplace?strategy=${result.strategyId}`}>
                {t('marketplace.autogen.viewDetail', { defaultValue: 'View Strategy' })}
              </Button>
              <Button onClick={handleReset}>{t('marketplace.autogen.generateAnother', { defaultValue: 'Generate Another' })}</Button>
            </Space>
          )}
          {!result?.strategyId && violations.length > 0 && (
            <Space>
              <Button onClick={handleGenerate}>{t('marketplace.autogen.retry', { defaultValue: 'Retry' })}</Button>
              <Button onClick={handleReset}>{t('marketplace.autogen.modify', { defaultValue: 'Modify Request' })}</Button>
            </Space>
          )}
        </div>
      )}
    </Card>
  );
}
