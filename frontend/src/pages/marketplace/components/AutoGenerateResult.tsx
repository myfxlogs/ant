import { Alert, Button, Space, Row, Col, Statistic, Tag } from 'antd';
import { DollarOutlined } from '@ant-design/icons';
import type { TFunction } from 'react-i18next';

interface AutoGenerateResultProps {
  stage: 'completed' | 'failed';
  result: { strategyId: string; publishId: string; backtest: any } | null;
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
          message={`${t('marketplace.autogen.failedAt', { defaultValue: 'Failed at' })}: ${errorStage}`}
          description={errorDetail}
          style={{ marginBottom: 16 }}
        />
        <Space>
          {retryable && <Button type="primary" onClick={onRetry}>{t('marketplace.autogen.retry', { defaultValue: 'Retry' })}</Button>}
          <Button onClick={onReset}>{t('marketplace.autogen.modify', { defaultValue: 'Modify Request' })}</Button>
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
          message={t('marketplace.autogen.qualityFailed', { defaultValue: 'Strategy generated but did not pass quality gates' })}
          description={
            <div>
              {violations.map((v, i) => (
                <div key={i}>
                  <Tag color="orange">{v.metric}</Tag>
                  <span>{t('marketplace.autogen.actual', { defaultValue: 'Actual' })}: {v.actual} / {t('marketplace.autogen.threshold', { defaultValue: 'Threshold' })}: {v.threshold}</span>
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
          <Button icon={<DollarOutlined />} onClick={onEditPricing}>
            {t('marketplace.autogen.editPricing', { defaultValue: 'Edit Pricing' })}
          </Button>
          <Button onClick={onReset}>{t('marketplace.autogen.generateAnother', { defaultValue: 'Generate Another' })}</Button>
        </Space>
      )}
    </div>
  );
}
