import { Alert, Button, Space, Row, Col, Statistic, Tag } from 'antd';
import { DollarOutlined } from '@ant-design/icons';
import type { TFunction } from 'i18next';

interface AutoGenerateResultProps {
  stage: 'completed' | 'failed';
  result: { strategyId: string; publishId: string; backtest: unknown } | null;
  violations: any[];
  errorStage: string;
  errorDetail: string;
  retryable: boolean;
  onRetry: () => void;
  onReset: () => void;
  onEditPricing: () => void;
  t: TFunction;
}

export default function AutoGenerateResult({
  stage,
  result,
  violations,
  errorStage,
  errorDetail,
  retryable,
  onRetry,
  onReset,
  onEditPricing,
  t,
}: AutoGenerateResultProps) {
  if (stage === 'failed') {
    return (
      <div>
        <Alert
          type="error"
          showIcon
          message={`${t('marketplace.autogen.failedAt')}: ${errorStage}`}
          description={errorDetail}
          style={{ marginBottom: 16 }}
        />
        <Space>
          {retryable && <Button type="primary" onClick={onRetry}>{t('marketplace.autogen.retry')}</Button>}
          <Button onClick={onReset}>{t('marketplace.autogen.modify')}</Button>
        </Space>
      </div>
    );
  }

  return (
    <div>
      {violations.length > 0 ? (
        <Alert
          type="warning"
          showIcon
          message={t('marketplace.autogen.qualityFailed')}
          description={
            <div>
              {violations.map((v, i) => (
                <div key={i}>
                  <Tag color="orange">{v.metric}</Tag>
                  <span>{t('marketplace.autogen.actual')}: {v.actual} / {t('marketplace.autogen.threshold')}: {v.threshold}</span>
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
          message={t('marketplace.autogen.success')}
          style={{ marginBottom: 16 }}
        />
      )}

      {result?.backtest && (
        <Row gutter={16} style={{ marginBottom: 16 }}>
          <Col span={4}><Statistic title={t('marketplace.totalReturn', { defaultValue: 'Total Return' })} value={result.backtest.totalReturn} /></Col>
          <Col span={4}><Statistic title={t('marketplace.maxDD', { defaultValue: 'Max DD' })} value={result.backtest.maxDrawdown} /></Col>
          <Col span={4}><Statistic title="Sharpe" value={result.backtest.sharpeRatio} /></Col>
          <Col span={4}><Statistic title={t('marketplace.winRate', { defaultValue: 'Win Rate' })} value={result.backtest.winRate} /></Col>
          <Col span={4}><Statistic title={t('marketplace.trades', { defaultValue: 'Trades' })} value={result.backtest.totalTrades} /></Col>
        </Row>
      )}

      {result?.strategyId && (
        <Space>
          <Button type="primary" href={`#/marketplace?strategy=${result.strategyId}`}>
            {t('marketplace.autogen.viewDetail')}
          </Button>
          <Button icon={<DollarOutlined />} onClick={onEditPricing}>
            {t('marketplace.autogen.editPricing')}
          </Button>
          <Button onClick={onReset}>{t('marketplace.autogen.generateAnother')}</Button>
        </Space>
      )}
    </div>
  );
}
