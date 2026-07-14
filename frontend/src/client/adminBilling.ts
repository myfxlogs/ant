import { adminBillingClient } from './connect';
import type { ListAdminSubscriptionsResponse, GetRevenueSummaryResponse, ListAdminWalletTransactionsResponse } from '../gen/ant/v1/admin_billing_pb';

export type { AdminSubscriptionDetail, PlanRevenue, AdminWalletTransactionDetail } from '../gen/ant/v1/admin_billing_pb';

export const adminBillingApi = {
  listSubscriptions: async (params?: { page?: number; pageSize?: number; plan?: string; status?: string }) => {
    const msg = await adminBillingClient.listSubscriptions({
      page: params?.page || 1,
      pageSize: params?.pageSize || 20,
      plan: params?.plan || '',
      status: params?.status || '',
    }) as ListAdminSubscriptionsResponse;
    return {
      subscriptions: msg.subscriptions || [],
      total: Number(msg.total) || 0,
    };
  },

  getRevenueSummary: async () => {
    const msg = await adminBillingClient.getRevenueSummary({}) as GetRevenueSummaryResponse;
    return {
      plans: msg.plans || [],
      totalMonthlyRevenue: msg.totalMonthlyRevenue || '0',
      totalRevenue: msg.totalRevenue || '0',
    };
  },

  listWalletTransactions: async (params?: { page?: number; pageSize?: number; txType?: string; userId?: string }) => {
    const msg = await adminBillingClient.listAdminWalletTransactions({
      page: params?.page || 1,
      pageSize: params?.pageSize || 20,
      txType: params?.txType || '',
      userId: params?.userId || '',
    }) as ListAdminWalletTransactionsResponse;
    return {
      transactions: msg.transactions || [],
      total: Number(msg.total) || 0,
    };
  },
};
