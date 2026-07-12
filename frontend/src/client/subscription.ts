import { subscriptionClient } from './connect';

export type { Plan, UserSubscription, UsageSummary } from '../gen/ant/v1/subscription_pb';

export const subscriptionApi = {
  listPlans: async () => {
    const response: any = await subscriptionClient.listPlans({});
    return (response.plans || []) as any[];
  },

  getMySubscription: async () => {
    const response: any = await subscriptionClient.getMySubscription({});
    return {
      subscription: response.subscription || null,
      plan: response.plan || null,
    };
  },

  subscribe: async (planName: string, billingCycle: string, autoRenew: boolean) => {
    const response: any = await subscriptionClient.subscribe({ planName, billingCycle, autoRenew });
    return {
      subscription: response.subscription || null,
      transactionId: response.transactionId || '',
      amountCharged: response.amountCharged || '0',
      balanceAfter: response.balanceAfter || '0',
    };
  },

  cancelSubscription: async () => {
    const response: any = await subscriptionClient.cancelSubscription({});
    return { subscription: response.subscription || null };
  },

  changePlan: async (newPlanName: string, billingCycle: string) => {
    const response: any = await subscriptionClient.changePlan({ newPlanName, billingCycle });
    return {
      subscription: response.subscription || null,
      transactionId: response.transactionId || '',
      amountCharged: response.amountCharged || '0',
      balanceAfter: response.balanceAfter || '0',
    };
  },

  getUsageSummary: async () => {
    const response: any = await subscriptionClient.getUsageSummary({});
    return {
      summary: response.summary || null,
      plan: response.plan || null,
    };
  },
};
