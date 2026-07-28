import { useState } from 'react';
import { Card, Row, Col, Button, Tag, Spin, message, Modal, Radio, Progress, Statistic, Empty } from 'antd';
import { CrownOutlined, CheckCircleOutlined, ThunderboltOutlined } from '@ant-design/icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { subscriptionApi } from '@/client/subscription';
import { formatDateTime } from '@/utils/date';
import Seo from '@/components/common/Seo';

const PRIMARY_GRADIENT = 'linear-gradient(135deg, #D4AF37 0%, #F4D03F 100%)';

export default function SubscriptionPage() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [subscribeOpen, setSubscribeOpen] = useState(false);
  const [selectedPlan, setSelectedPlan] = useState<string>('');
  const [billingCycle, setBillingCycle] = useState<string>('monthly');

  const { data: plans = [], isLoading: plansLoading } = useQuery({
    queryKey: ['subscription', 'plans'],
    queryFn: () => subscriptionApi.listPlans(),
  });

  const { data: subData, isLoading: subLoading } = useQuery({
    queryKey: ['subscription', 'mine'],
    queryFn: () => subscriptionApi.getMySubscription(),
  });

  const { data: usageData } = useQuery({
    queryKey: ['subscription', 'usage'],
    queryFn: () => subscriptionApi.getUsageSummary(),
  });

  const subscribeMutation = useMutation({
    mutationFn: () => subscriptionApi.subscribe(selectedPlan, billingCycle, true),
    onSuccess: (res) => {
      message.success(t('subscription.subscribeSuccess', { defaultValue: 'Subscription activated successfully!' }));
      setSubscribeOpen(false);
      queryClient.invalidateQueries({ queryKey: ['subscription'] });
      if (res.amountCharged && res.amountCharged !== '0') {
        message.info(t('subscription.charged', { defaultValue: `Charged: ${res.amountCharged}, Balance: ${res.balanceAfter}` }));
      }
    },
    onError: (err: Error) => {
      const msg = String((err as unknown)?.message || '');
      if (msg.includes('insufficient balance')) {
        message.error(t('subscription.insufficientBalance', { defaultValue: 'Insufficient wallet balance. Please top up your wallet first.' }));
      } else {
        message.error(t('subscription.subscribeFailed', { defaultValue: 'Subscription failed. Please try again.' }));
      }
    },
  });

  const cancelMutation = useMutation({
    mutationFn: () => subscriptionApi.cancelSubscription(),
    onSuccess: () => {
      message.success(t('subscription.cancelSuccess', { defaultValue: 'Auto-renewal cancelled. Your subscription remains active until the period ends.' }));
      queryClient.invalidateQueries({ queryKey: ['subscription'] });
    },
    onError: () => {
      message.error(t('subscription.cancelFailed', { defaultValue: 'Failed to cancel. Please try again.' }));
    },
  });

  const changePlanMutation = useMutation({
    mutationFn: (newPlan: string) => subscriptionApi.changePlan(newPlan, billingCycle),
    onSuccess: (res) => {
      message.success(t('subscription.changeSuccess', { defaultValue: 'Plan changed successfully!' }));
      setSubscribeOpen(false);
      queryClient.invalidateQueries({ queryKey: ['subscription'] });
      if (res.amountCharged && res.amountCharged !== '0') {
        message.info(t('subscription.charged', { defaultValue: `Charged: ${res.amountCharged}, Balance: ${res.balanceAfter}` }));
      }
    },
    onError: (err: Error) => {
      const msg = String((err as unknown)?.message || '');
      if (msg.includes('insufficient balance')) {
        message.error(t('subscription.insufficientBalance', { defaultValue: 'Insufficient wallet balance. Please top up your wallet first.' }));
      } else {
        message.error(t('subscription.changeFailed', { defaultValue: 'Plan change failed. Please try again.' }));
      }
    },
  });

  const currentPlan = subData?.plan;
  const currentSub = subData?.subscription;
  const usage = usageData?.summary;

  const handleSubscribe = (planName: string) => {
    setSelectedPlan(planName);
    setBillingCycle('monthly');
    setSubscribeOpen(true);
  };

  const handleConfirmSubscribe = () => {
    if (!selectedPlan) return;
    if (currentSub && currentPlan && currentPlan.name !== 'free') {
      changePlanMutation.mutate(selectedPlan);
    } else {
      subscribeMutation.mutate();
    }
  };

  const isCurrentPlan = (planName: string) => currentPlan?.name === planName;

  const getPlanIcon = (name: string) => {
    if (name === 'free') return <ThunderboltOutlined style={{ fontSize: 28, color: '#8c8c8c' }} />;
    if (name === 'pro') return <CrownOutlined style={{ fontSize: 28, color: '#D4AF37' }} />;
    return <CrownOutlined style={{ fontSize: 28, color: '#722ed1' }} />;
  };

  if (plansLoading || subLoading) {
    return (
      <div className="min-h-[400px] flex items-center justify-center">
        <Spin size="large" />
      </div>
    );
  }

  return (
    <>
      <Seo title={t('subscription.seoTitle')} description={t('subscription.seoDesc')} path="/subscription" />
      <div className="space-y-6">
        <h1 className="text-2xl font-bold" style={{ color: 'var(--color-text)' }}>
          {t('subscription.title', { defaultValue: 'Subscription Plans' })}
        </h1>

        {/* Current subscription status */}
        {currentSub && currentPlan && currentPlan.name !== 'free' && (
          <Card>
            <Row gutter={[24, 16]} align="middle">
              <Col flex="auto">
                <div className="flex items-center gap-3">
                  <Tag color="gold" style={{ fontSize: 16, padding: '4px 12px' }}>
                    {currentPlan.displayName}
                  </Tag>
                  <Tag color={currentSub.status === 'active' ? 'green' : 'default'}>
                    {currentSub.status}
                  </Tag>
                  <span style={{ color: 'var(--color-text-muted)', fontSize: 14 }}>
                    {t('subscription.billingCycle', { defaultValue: 'Billing' })}: {currentSub.billingCycle}
                  </span>
                  {currentSub.autoRenew && <Tag color="blue">{t('subscription.autoRenew', { defaultValue: 'Auto-renew' })}</Tag>}
                </div>
                <div className="mt-3 text-sm" style={{ color: 'var(--color-text-muted)' }}>
                  {t('subscription.period', { defaultValue: 'Current period' })}: {formatDateTime(currentSub.currentPeriodStart)} → {formatDateTime(currentSub.currentPeriodEnd)}
                </div>
              </Col>
              {currentSub.autoRenew && (
                <Col>
                  <Button
                    danger
                    onClick={() => cancelMutation.mutate()}
                    loading={cancelMutation.isPending}
                  >
                    {t('subscription.cancelAutoRenew', { defaultValue: 'Cancel Auto-renew' })}
                  </Button>
                </Col>
              )}
            </Row>
          </Card>
        )}

        {/* Usage summary */}
        {usage && (
          <Card title={t('subscription.usageTitle', { defaultValue: 'Current Month Usage' })} size="small">
            <Row gutter={[24, 16]}>
              <Col xs={12} sm={6}>
                <Statistic
                  title={t('subscription.aiTokens', { defaultValue: 'AI Tokens' })}
                  value={usage.aiTokensUsed}
                  suffix={usage.aiTokensLimit > 0 ? `/ ${usage.aiTokensLimit}` : ''}
                />
                {usage.aiTokensLimit > 0 && (
                  <Progress
                    percent={Math.min(100, (usage.aiTokensUsed / usage.aiTokensLimit) * 100)}
                    size="small"
                    strokeColor="#D4AF37"
                  />
                )}
              </Col>
              <Col xs={12} sm={6}>
                <Statistic
                  title={t('subscription.activeStrategies', { defaultValue: 'Active Strategies' })}
                  value={usage.activeStrategies}
                  suffix={usage.maxStrategies > 0 ? `/ ${usage.maxStrategies}` : ''}
                />
              </Col>
              <Col xs={12} sm={6}>
                <Statistic
                  title={t('subscription.runtimeMinutes', { defaultValue: 'Runtime (min)' })}
                  value={usage.strategyRuntimeMinutes}
                />
              </Col>
              <Col xs={12} sm={6}>
                <Statistic
                  title={t('subscription.walletBalance', { defaultValue: 'Wallet Balance' })}
                  value={usage.walletBalance}
                />
              </Col>
            </Row>
          </Card>
        )}

        {/* Plan cards */}
        <Row gutter={[16, 16]}>
          {plans.map((plan: unknown) => (
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
                    {plan.priceMonthly}
                  </span>
                  <span style={{ color: 'var(--color-text-muted)' }}> / {t('subscription.month', { defaultValue: 'mo' })}</span>
                  {Number(plan.priceYearly) > 0 && (
                    <div className="text-sm mt-1" style={{ color: 'var(--color-text-muted)' }}>
                      {plan.priceYearly} / {t('subscription.year', { defaultValue: 'yr' })}
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
                  onClick={() => handleSubscribe(plan.name)}
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

        {/* Subscribe / Change Plan Modal */}
        <Modal
          open={subscribeOpen}
          title={currentSub && currentPlan && currentPlan.name !== 'free'
            ? t('subscription.changePlanTitle', { defaultValue: 'Change Plan' })
            : t('subscription.subscribeTitle', { defaultValue: 'Subscribe to Plan' })}
          onCancel={() => setSubscribeOpen(false)}
          onOk={handleConfirmSubscribe}
          confirmLoading={subscribeMutation.isPending || changePlanMutation.isPending}
          okText={t('common.confirm', { defaultValue: 'Confirm' })}
          cancelText={t('common.cancel', { defaultValue: 'Cancel' })}
        >
          <div className="space-y-4 py-2">
            <div>
              <p className="mb-2 text-sm" style={{ color: 'var(--color-text-muted)' }}>
                {t('subscription.selectBillingCycle', { defaultValue: 'Billing Cycle' })}
              </p>
              <Radio.Group
                value={billingCycle}
                onChange={(e) => setBillingCycle(e.target.value)}
              >
                <Radio.Button value="monthly">{t('subscription.monthly', { defaultValue: 'Monthly' })}</Radio.Button>
                <Radio.Button value="yearly">{t('subscription.yearly', { defaultValue: 'Yearly' })}</Radio.Button>
              </Radio.Group>
            </div>
            <p className="text-sm" style={{ color: 'var(--color-text-muted)' }}>
              {t('subscription.chargeNotice', { defaultValue: 'Your wallet will be charged for paid plans. Free plans have no charge.' })}
            </p>
          </div>
        </Modal>
      </div>
    </>
  );
}
