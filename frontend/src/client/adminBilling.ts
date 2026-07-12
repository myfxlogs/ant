import { adminBillingClient } from './connect';

export type { AdminSubscriptionDetail, PlanRevenue, AdminWalletTransactionDetail } from '../gen/ant/v1/admin_billing_pb';

export const adminBillingApi = {
  listSubscriptions: async (params?: { page?: number; pageSize?: number; plan?: string; status?: string }) => {
    const response: any = await adminBillingClient.listSubscriptions({
      page: params?.page || 1,
      pageSize: params?.pageSize || 20,
      plan: params?.plan || '',
      status: params?.status || '',
    });
    return {
      subscriptions: response.subscriptions || [],
      total: Number(response.total) || 0,
    };
  },

  getRevenueSummary: async () => {
    const response: any = await adminBillingClient.getRevenueSummary({});
    return {
      plans: response.plans || [],
      totalMonthlyRevenue: response.totalMonthlyRevenue || '0',
      totalRevenue: response.totalRevenue || '0',
    };
  },

  listWalletTransactions: async (params?: { page?: number; pageSize?: number; txType?: string; userId?: string }) => {
    const response: any = await adminBillingClient.listAdminWalletTransactions({
      page: params?.page || 1,
      pageSize: params?.pageSize || 20,
      txType: params?.txType || '',
      userId: params?.userId || '',
    });
    return {
      transactions: response.transactions || [],
      total: Number(response.total) || 0,
    };
  },
};
