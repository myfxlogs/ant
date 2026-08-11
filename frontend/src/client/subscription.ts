import { subscriptionClient } from './connect';
import type { ListPlansResponse, GetMySubscriptionResponse, SubscribePlanResponse, CancelSubscriptionResponse, ChangePlanResponse, GetUsageSummaryResponse, Plan, ListBoundAccountsResponse, BoundAccount } from '../gen/ant/v1/subscription_pb';

export type { BoundAccount } from '../gen/ant/v1/subscription_pb';

export const subscriptionApi = {
  listPlans: async () => {
    const msg = await subscriptionClient.listPlans({}) as ListPlansResponse;
    return (msg.plans || []) as Plan[];
  },

  getMySubscription: async () => {
    const msg = await subscriptionClient.getMySubscription({}) as GetMySubscriptionResponse;
    return {
      subscription: msg.subscription || null,
      plan: msg.plan || null,
    };
  },

  subscribe: async (planName: string, billingCycle: string, autoRenew: boolean) => {
    const msg = await subscriptionClient.subscribe({ planName, billingCycle, autoRenew }) as SubscribePlanResponse;
    return {
      subscription: msg.subscription || null,
      transactionId: msg.transactionId || '',
      amountCharged: msg.amountCharged || '0',
      balanceAfter: msg.balanceAfter || '0',
    };
  },

  cancelSubscription: async () => {
    const msg = await subscriptionClient.cancelSubscription({}) as CancelSubscriptionResponse;
    return { subscription: msg.subscription || null };
  },

  changePlan: async (newPlanName: string, billingCycle: string) => {
    const msg = await subscriptionClient.changePlan({ newPlanName, billingCycle }) as ChangePlanResponse;
    return {
      subscription: msg.subscription || null,
      transactionId: msg.transactionId || '',
      amountCharged: msg.amountCharged || '0',
      balanceAfter: msg.balanceAfter || '0',
    };
  },

  getUsageSummary: async () => {
    const msg = await subscriptionClient.getUsageSummary({}) as GetUsageSummaryResponse;
    return {
      summary: msg.summary || null,
      plan: msg.plan || null,
    };
  },

  listBoundAccounts: async () => {
    const msg = await subscriptionClient.listBoundAccounts({}) as ListBoundAccountsResponse;
    return {
      accounts: (msg.accounts || []) as BoundAccount[],
      maxAccounts: msg.maxAccounts || 0,
    };
  },

  unbindAccount: async (mtAccountId: string) => {
    await subscriptionClient.unbindAccount({ mtAccountId });
  },
};
