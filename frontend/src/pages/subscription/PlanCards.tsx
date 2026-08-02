import React from 'react';
import { Row, Col, Card, Button, Empty } from 'antd';
import { CrownOutlined, CheckCircleOutlined, ThunderboltOutlined } from '@ant-design/icons';
import type { TFunction } from 'i18next';

const PRIMARY_GRADIENT = 'linear-gradient(135deg, #D4AF37 0%, #F4D03F 100%)';

interface PlanItem {
  id: string;
  name: string;
  displayName: string;
  priceMonthly: string | number;
  priceYearly: string | number;
  maxAiTokensMonthly: number;
  maxStrategies: number;
  maxBacktestsDaily: number;
  maxLiveStrategies: number;
  maxSymbolsPerStrategy: number;
}

interface Props {
  plans: PlanItem[];
  currentPlanName?: string;
  t: TFunction;
  onSubscribe: (planName: string) => void;
}

export default function PlanCards({ plans, currentPlanName, t, onSubscribe }: Props) {
  const isCurrentPlan = (planName: string) => currentPlanName === planName;

  const getPlanIcon = (name: string) => {
    if (name === 'free') return <ThunderboltOutlined style={{ fontSize: 28, color: '#8c8c8c' }} />;
    if (name === 'pro') return <CrownOutlined style={{ fontSize: 28, color: '#D4AF37' }} />;
    return <CrownOutlined style={{ fontSize: 28, color: '#722ed1' }} />;
  };

  return (
    <>
      <Row gutter={[16, 16]}>
        {plans.map((plan) => (
          <Col xs={24} sm={12} lg={8} key={plan.id}>
            <Card
              hoverable
              style={{
                borderColor: isCurrentPlan(plan.name) ? '#D4AF37' : undefined,
                borderWidth: isCurrentPlan(plan.name) ? 2 : 1,
              }}
            >
              <div className="text-center mb-4">
                {getPlanIcon(plan.name)}
                <h2 className="mt-3 text-xl font-bold" style={{ color: 'var(--color-text)' }}>
                  {t(`subscription.planName.${plan.name}`, { defaultValue: plan.displayName })}
                </h2>
              </div>

              <div className="text-center mb-4">
                <span className="text-3xl font-bold" style={{ color: '#D4AF37' }}>
                  ${plan.priceMonthly}
                </span>
                <span style={{ color: 'var(--color-text-muted)' }}> / {t('subscription.month', { defaultValue: 'mo' })}</span>
                {Number(plan.priceYearly) > 0 && (
                  <div className="text-sm mt-1" style={{ color: 'var(--color-text-muted)' }}>
                    ${plan.priceYearly} / {t('subscription.year', { defaultValue: 'yr' })}
                  </div>
                )}
              </div>

              <div className="space-y-2 mb-4">
                {plan.maxAiTokensMonthly > 0 && (
                  <div className="flex items-center gap-2 text-sm">
                    <CheckCircleOutlined style={{ color: '#52c41a' }} />
                    {t('subscription.feature.aiTokens', { count: plan.maxAiTokensMonthly.toLocaleString(), defaultValue: '{{count}} AI tokens/mo' })}
                  </div>
                )}
                {plan.maxStrategies > 0 && (
                  <div className="flex items-center gap-2 text-sm">
                    <CheckCircleOutlined style={{ color: '#52c41a' }} />
                    {t('subscription.feature.strategies', { count: plan.maxStrategies, defaultValue: '{{count}} strategies' })}
                  </div>
                )}
                {plan.maxBacktestsDaily > 0 && (
                  <div className="flex items-center gap-2 text-sm">
                    <CheckCircleOutlined style={{ color: '#52c41a' }} />
                    {t('subscription.feature.backtests', { count: plan.maxBacktestsDaily, defaultValue: '{{count}} backtests/day' })}
                  </div>
                )}
                {plan.maxLiveStrategies > 0 && (
                  <div className="flex items-center gap-2 text-sm">
                    <CheckCircleOutlined style={{ color: '#52c41a' }} />
                    {t('subscription.feature.liveStrategies', { count: plan.maxLiveStrategies, defaultValue: '{{count}} live strategies' })}
                  </div>
                )}
                {plan.maxSymbolsPerStrategy > 0 && (
                  <div className="flex items-center gap-2 text-sm">
                    <CheckCircleOutlined style={{ color: '#52c41a' }} />
                    {t('subscription.feature.symbols', { count: plan.maxSymbolsPerStrategy, defaultValue: '{{count}} symbols/strategy' })}
                  </div>
                )}
                {plan.name === 'free' && (
                  <div className="flex items-center gap-2 text-sm">
                    <CheckCircleOutlined style={{ color: '#52c41a' }} />
                    {t('subscription.freeForever', { defaultValue: 'Free forever' })}
                  </div>
                )}
              </div>

              <Button
                block
                disabled={isCurrentPlan(plan.name)}
                style={
                  isCurrentPlan(plan.name)
                    ? undefined
                    : { background: PRIMARY_GRADIENT, border: 'none', borderRadius: '10px', height: '44px', color: '#fff', fontWeight: 600 }
                }
                onClick={() => onSubscribe(plan.name)}
              >
                {isCurrentPlan(plan.name)
                  ? t('subscription.currentPlan', { defaultValue: 'Current Plan' })
                  : t('subscription.choosePlan', { defaultValue: 'Choose Plan' })}
              </Button>
            </Card>
          </Col>
        ))}
      </Row>

      {plans.length === 0 && (
        <Empty description={t('subscription.noPlans', { defaultValue: 'No plans available' })} />
      )}
    </>
  );
}

export { PRIMARY_GRADIENT };
