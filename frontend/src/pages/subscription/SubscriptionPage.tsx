import { useState } from 'react';
import { Card, Button, Tag, Spin, message, Modal, Radio, Progress, Statistic, Row, Col } from 'antd';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { subscriptionApi } from '@/client/subscription';
import { formatDateTime } from '@/utils/date';
import Seo from '@/components/common/Seo';
import PlanCards from './PlanCards';
import BoundAccountsCard from './BoundAccountsCard';

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
                  title={t('subscription.aiTokensRemaining', { defaultValue: 'AI Tokens Remaining' })}
                  value={usage.aiTokensLimit > 0 ? Math.max(0, usage.aiTokensLimit - usage.aiTokensUsed) : '∞'}
                  suffix={usage.aiTokensLimit > 0 ? ` / ${usage.aiTokensLimit}` : ''}
                />
                {usage.aiTokensLimit > 0 && (
                  <Progress
                    percent={Math.min(100, (usage.aiTokensUsed / usage.aiTokensLimit) * 100)}
                    size="small"
                    strokeColor={usage.aiTokensUsed / usage.aiTokensLimit > 0.8 ? '#ff4d4f' : '#D4AF37'}
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

        {/* Bound accounts management (LEAKAGE-1) */}
        <Card title={t('subscription.boundAccountsTitle', { defaultValue: 'Bound MT Accounts' })} size="small">
          <BoundAccountsCard />
        </Card>

        {/* Plan cards */}
        <PlanCards plans={plans} currentPlanName={currentPlan?.name} t={t} onSubscribe={handleSubscribe} />

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
