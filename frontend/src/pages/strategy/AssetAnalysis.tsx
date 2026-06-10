import {
  Card, Input, Button, Typography, Tag, Progress,
  Descriptions, Row, Col, Alert, Grid,
} from 'antd';
import {
  SearchOutlined, ThunderboltOutlined, DashboardOutlined,
  BulbOutlined, RiseOutlined, FallOutlined, MinusOutlined,
} from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import ReactMarkdown from 'react-markdown';
import type { TfOutlook, SRLevel as SRLevelType } from '@/gen/ant/v1/asset_analysis_pb';
import { useAssetAnalysis, type AnalysisPhase } from './hooks/useAssetAnalysis';

const { Title, Text, Paragraph } = Typography;

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

function renderTfCard(label: string, tf: TfOutlook | undefined) {
  if (!tf) return null;
  return (
    <Card size="small" style={{ textAlign: 'center' }}>
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
}

function renderSRLevels(levels: SRLevelType[], t: (key: string) => string) {
  if (!levels?.length) return <Text type="secondary">{t('strategy.assetAnalysis.noLevels')}</Text>;
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
}

export default function AssetAnalysisPage() {
  const { t } = useTranslation();
  const screens = Grid.useBreakpoint();
  const isMobile = !screens.sm;
  const { symbol, setSymbol, phase, loading, error, setError, result, progress, analyze } = useAssetAnalysis();

  const hasData = phase !== 'idle' && Object.keys(result).length > 0;

  return (
    <div style={{ maxWidth: 960, margin: '0 auto', padding: isMobile ? '16px 12px' : '24px 16px' }}>
      <Title level={3}>
        <BulbOutlined style={{ marginRight: 8 }} />
        {t('strategy.assetAnalysis.title')}
      </Title>
      <Text type="secondary" style={{ display: 'block', marginBottom: 24 }}>
        {t('strategy.assetAnalysis.subtitle')}
      </Text>

      <Row gutter={[12, 12]} style={{ marginBottom: 24 }}>
        <Col xs={24} sm={16} md={12}>
          <Input
            prefix={<SearchOutlined />}
            placeholder={t('strategy.assetAnalysis.symbolPlaceholder')}
            value={symbol}
            onChange={e => setSymbol(e.target.value)}
            onPressEnter={analyze}
            style={{ width: '100%', borderRadius: 8 }}
            disabled={loading}
          />
        </Col>
        <Col xs={12} sm={8} md={4}>
          <Button
            type="primary"
            icon={<ThunderboltOutlined />}
            onClick={analyze}
            loading={loading}
            disabled={!symbol.trim()}
            style={{ borderRadius: 8, width: '100%' }}
            block
          >
            {t('strategy.assetAnalysis.analyze')}
          </Button>
        </Col>
      </Row>

      {loading && (
        <div style={{ marginBottom: 16 }}>
          <Progress percent={progress} status="active" size="small" />
          <Text type="secondary" style={{ fontSize: 12 }}>
            {phase === 'idle' ? t('strategy.assetAnalysis.fetchingData') : t('strategy.assetAnalysis.phase', { phase })}
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
              title={<><DashboardOutlined /> {t('strategy.assetAnalysis.mtfOutlook')}</>}
              size="small"
              style={{ marginBottom: 16, borderRadius: 12 }}
            >
              <Row gutter={[12, 12]} justify="start">
                <Col xs={12} sm={12} md={6}>{renderTfCard('1H', result.multiTf.h1)}</Col>
                <Col xs={12} sm={12} md={6}>{renderTfCard('4H', result.multiTf.h4)}</Col>
                <Col xs={12} sm={12} md={6}>{renderTfCard('D1', result.multiTf.d1)}</Col>
                <Col xs={12} sm={12} md={6}>{renderTfCard('W1', result.multiTf.w1)}</Col>
              </Row>
            </Card>
          )}

          {/* S/R Levels */}
          {(phase === 'sr_levels' || phase === 'volatility' || phase === 'ai_recommendation' || phase === 'complete') && (
            <Card
              title={t('strategy.assetAnalysis.srLevels')}
              size="small"
              style={{ marginBottom: 16, borderRadius: 12 }}
              extra={result.keyLevels?.length ? <Tag>{result.keyLevels.length} levels</Tag> : null}
            >
              {renderSRLevels(result.keyLevels || [], t)}
            </Card>
          )}

          {/* Volatility */}
          {(phase === 'volatility' || phase === 'ai_recommendation' || phase === 'complete') && result.volatilityState && (
            <Card
              title={t('strategy.assetAnalysis.volatility')}
              size="small"
              style={{ marginBottom: 16, borderRadius: 12 }}
            >
              <Descriptions column={{ xs: 1, sm: 2 }} size="small">
                <Descriptions.Item label={t('strategy.assetAnalysis.state')}>
                  <Tag color={VOL_COLORS[result.volatilityState] || 'default'}>
                    {result.volatilityState}
                  </Tag>
                </Descriptions.Item>
                <Descriptions.Item label={t('strategy.assetAnalysis.atrPct')}>
                  <Text strong>{result.volatilityValue?.toFixed(2)}%</Text>
                </Descriptions.Item>
              </Descriptions>
              <Paragraph style={{ fontSize: 12, color: '#8A9AA5', marginTop: 8, marginBottom: 0 }}>
                {result.volatilityState === 'LOW' && t('strategy.assetAnalysis.volLow')}
                {result.volatilityState === 'NORMAL' && t('strategy.assetAnalysis.volNormal')}
                {result.volatilityState === 'HIGH' && t('strategy.assetAnalysis.volHigh')}
                {result.volatilityState === 'EXTREME' && t('strategy.assetAnalysis.volExtreme')}
              </Paragraph>
            </Card>
          )}

          {/* AI Recommendation */}
          {(phase === 'ai_recommendation' || phase === 'complete') && (
            <Card
              title={<><BulbOutlined /> {t('strategy.assetAnalysis.aiRecommendation')}</>}
              size="small"
              style={{ marginBottom: 16, borderRadius: 12, borderColor: '#D4AF37' }}
            >
              {result.aiRecommendation ? (
                <div style={{ fontSize: 13, lineHeight: 1.8 }}>
                  <ReactMarkdown>{result.aiRecommendation}</ReactMarkdown>
                </div>
              ) : (
                <Text type="secondary">
                  {t('strategy.assetAnalysis.aiUnavailable')}
                </Text>
              )}
            </Card>
          )}
        </>
      )}

      {phase === 'complete' && !hasData && !error && (
        <Alert type="info" message={t('strategy.assetAnalysis.noResults')} />
      )}
    </div>
  );
}
