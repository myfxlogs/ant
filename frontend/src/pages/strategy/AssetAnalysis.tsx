import { useState, useCallback } from 'react';
import {
  Card, Input, Button, Typography, Tag, Space, Spin, Progress,
  Descriptions, Row, Col, Alert, Divider,
} from 'antd';
import {
  SearchOutlined, ThunderboltOutlined, DashboardOutlined,
  BulbOutlined, RiseOutlined, FallOutlined, MinusOutlined,
} from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import ReactMarkdown from 'react-markdown';
import { assetAnalysisClient } from '@/client/connect';
import type { AnalyzeAssetResponse, TfOutlook, SRLevel as SRLevelType } from '@/gen/ant/v1/asset_analysis_pb';
import type { PartialMessage } from '@bufbuild/protobuf';

const { Title, Text, Paragraph } = Typography;

type Phase = 'idle' | 'mtf_outlook' | 'sr_levels' | 'volatility' | 'ai_recommendation' | 'complete';

const TREND_ICONS: Record<string, React.ReactNode> = {
  BULLISH: <RiseOutlined style={{ color: '#00A651' }} />,
  BEARISH: <FallOutlined style={{ color: '#E53935' }} />,
  NEUTRAL: <MinusOutlined style={{ color: '#8A9AA5' }} />,
};

const TREND_COLORS: Record<string, string> = {
  BULLISH: '#00A651',
  BEARISH: '#E53935',
  NEUTRAL: '#8A9AA5',
};

const VOL_COLORS: Record<string, string> = {
  LOW: '#00A651',
  NORMAL: '#1890FF',
  HIGH: '#FF9800',
  EXTREME: '#E53935',
};

const PHASES: Phase[] = ['mtf_outlook', 'sr_levels', 'volatility', 'ai_recommendation', 'complete'];

export default function AssetAnalysisPage() {
  const { t } = useTranslation();
  const [symbol, setSymbol] = useState('');
  const [phase, setPhase] = useState<Phase>('idle');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [result, setResult] = useState<PartialMessage<AnalyzeAssetResponse>>({});
  const [progress, setProgress] = useState(0);

  const handleAnalyze = useCallback(async () => {
    if (!symbol.trim()) return;
    setLoading(true);
    setPhase('idle');
    setError('');
    setResult({});
    setProgress(0);

    const ctrl = new AbortController();
    try {
      const stream = await assetAnalysisClient.analyzeAsset(
        { symbol: symbol.trim().toUpperCase(), klineCount: 200 },
        { signal: ctrl.signal },
      );

      let idx = 0;
      for await (const frame of stream) {
        if (frame.phase === 'complete') {
          setPhase('complete');
          setProgress(100);
          break;
        }
        if (frame.error) {
          setError(frame.error);
          setPhase('complete');
          setProgress(100);
          break;
        }
        setResult(frame);
        setPhase(frame.phase as Phase);
        idx = PHASES.indexOf(frame.phase as Phase);
        setProgress(Math.round(((idx + 1) / PHASES.length) * 100));
      }
    } catch (err: any) {
      if (err?.name !== 'AbortError') {
        setError(err?.message || 'Analysis failed');
        setPhase('complete');
      }
    } finally {
      setLoading(false);
    }
  }, [symbol]);

  const hasData = phase !== 'idle' && Object.keys(result).length > 0;

  const renderTfCard = (label: string, tf: TfOutlook | undefined) => {
    if (!tf) return null;
    return (
      <Card size="small" style={{ textAlign: 'center', minWidth: 130 }}>
        <div style={{ fontSize: 12, color: '#8A9AA5', marginBottom: 4 }}>{label}</div>
        <div style={{ fontSize: 20, marginBottom: 4 }}>{TREND_ICONS[tf.trend] || null}</div>
        <Tag color={TREND_COLORS[tf.trend] || 'default'} style={{ marginBottom: 4 }}>
          {tf.trend}
        </Tag>
        <Progress
          percent={Math.round((tf.strength || 0) * 100)}
          size="small"
          strokeColor={TREND_COLORS[tf.trend]}
          format={() => `${Math.round((tf.strength || 0) * 100)}%`}
        />
        <div style={{ fontSize: 11, color: '#B0BEC5', marginTop: 4 }}>
          EMA gap: {tf.emaGapPct?.toFixed(2)}%
          <br />
          Change: {tf.priceChangePct?.toFixed(2)}%
        </div>
      </Card>
    );
  };

  const renderSRLevels = (levels: SRLevelType[]) => {
    if (!levels?.length) return <Text type="secondary">No significant levels detected</Text>;
    return (
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
        {levels.map((lvl, i) => (
          <Tag
            key={i}
            color={lvl.type === 'RESISTANCE' ? 'red' : 'green'}
            style={{ padding: '4px 12px', fontSize: 13 }}
          >
            {lvl.price?.toFixed(5)}
            <span style={{ fontSize: 10, marginLeft: 6 }}>
              {lvl.strength} | {lvl.touches} touches
            </span>
          </Tag>
        ))}
      </div>
    );
  };

  return (
    <div style={{ maxWidth: 960, margin: '0 auto', padding: '24px 16px' }}>
      <Title level={3}>
        <BulbOutlined style={{ marginRight: 8 }} />
        AI Asset Analysis
      </Title>
      <Text type="secondary" style={{ display: 'block', marginBottom: 24 }}>
        Multi-timeframe trend outlook, S/R level detection, volatility classification, and AI strategy recommendation
      </Text>

      <Space style={{ marginBottom: 24, width: '100%' }} size={12}>
        <Input
          prefix={<SearchOutlined />}
          placeholder="Enter symbol (e.g. EURUSD, XAUUSD, BTCUSD)"
          value={symbol}
          onChange={e => setSymbol(e.target.value)}
          onPressEnter={handleAnalyze}
          style={{ width: 320, borderRadius: 8 }}
          disabled={loading}
        />
        <Button
          type="primary"
          icon={<ThunderboltOutlined />}
          onClick={handleAnalyze}
          loading={loading}
          disabled={!symbol.trim()}
          style={{ borderRadius: 8 }}
        >
          Analyze
        </Button>
      </Space>

      {loading && (
        <div style={{ marginBottom: 16 }}>
          <Progress percent={progress} status="active" size="small" />
          <Text type="secondary" style={{ fontSize: 12 }}>
            {phase === 'idle' ? 'Fetching market data...' : `Phase: ${phase}`}
          </Text>
        </div>
      )}

      {error && (
        <Alert type="error" message={error} style={{ marginBottom: 16 }} closable onClose={() => setError('')} />
      )}

      {hasData && (
        <>
          {/* MTF Outlook */}
          {(phase === 'mtf_outlook' || phase === 'sr_levels' || phase === 'volatility' || phase === 'ai_recommendation' || phase === 'complete') && result.multiTf && (
            <Card
              title={<><DashboardOutlined /> Multi-Timeframe Outlook</>}
              size="small"
              style={{ marginBottom: 16, borderRadius: 12 }}
            >
              <Row gutter={[12, 12]} justify="space-around">
                <Col>{renderTfCard('1H', result.multiTf.h1)}</Col>
                <Col>{renderTfCard('4H', result.multiTf.h4)}</Col>
                <Col>{renderTfCard('D1', result.multiTf.d1)}</Col>
                <Col>{renderTfCard('W1', result.multiTf.w1)}</Col>
              </Row>
            </Card>
          )}

          {/* S/R Levels */}
          {(phase === 'sr_levels' || phase === 'volatility' || phase === 'ai_recommendation' || phase === 'complete') && (
            <Card
              title="Support / Resistance Levels"
              size="small"
              style={{ marginBottom: 16, borderRadius: 12 }}
              extra={result.keyLevels?.length ? <Tag>{result.keyLevels.length} levels</Tag> : null}
            >
              {renderSRLevels(result.keyLevels || [])}
            </Card>
          )}

          {/* Volatility */}
          {(phase === 'volatility' || phase === 'ai_recommendation' || phase === 'complete') && result.volatilityState && (
            <Card
              title="Volatility"
              size="small"
              style={{ marginBottom: 16, borderRadius: 12 }}
            >
              <Descriptions column={2} size="small">
                <Descriptions.Item label="State">
                  <Tag color={VOL_COLORS[result.volatilityState] || 'default'}>
                    {result.volatilityState}
                  </Tag>
                </Descriptions.Item>
                <Descriptions.Item label="ATR %">
                  <Text strong>{result.volatilityValue?.toFixed(2)}%</Text>
                </Descriptions.Item>
              </Descriptions>
              <Paragraph style={{ fontSize: 12, color: '#8A9AA5', marginTop: 8, marginBottom: 0 }}>
                {result.volatilityState === 'LOW' && 'Low volatility — consider breakout or mean-reversion strategies with tight stops.'}
                {result.volatilityState === 'NORMAL' && 'Normal volatility — suitable for most strategy types.'}
                {result.volatilityState === 'HIGH' && 'High volatility — wider stops recommended; trend-following and breakout strategies favored.'}
                {result.volatilityState === 'EXTREME' && 'Extreme volatility — reduce position sizes significantly; wide stops required.'}
              </Paragraph>
            </Card>
          )}

          {/* AI Recommendation */}
          {(phase === 'ai_recommendation' || phase === 'complete') && (
            <Card
              title={<><BulbOutlined /> AI Strategy Recommendation</>}
              size="small"
              style={{ marginBottom: 16, borderRadius: 12, borderColor: '#D4AF37' }}
            >
              {result.aiRecommendation ? (
                <div style={{ fontSize: 13, lineHeight: 1.8 }}>
                  <ReactMarkdown>{result.aiRecommendation}</ReactMarkdown>
                </div>
              ) : (
                <Text type="secondary">
                  AI recommendation unavailable. Please configure an AI provider in Settings.
                </Text>
              )}
            </Card>
          )}
        </>
      )}

      {phase === 'complete' && !hasData && !error && (
        <Alert type="info" message="No analysis results returned. Try a different symbol." />
      )}
    </div>
  );
}
