import { walletClient, adminUserClient } from './connect';

export const walletApi = {
  /** Get wallet for the current user, or for a specific user (admin only). */
  getWallet: async (userId?: string) => {
    const response: any = await walletClient.getWallet({ userId: userId || '' });
    return response.wallet;
  },

  /** List transactions for the current user, or for a specific user (admin only). */
  listTransactions: async (page = 1, pageSize = 20, userId?: string) => {
    const response: any = await walletClient.listTransactions({ page, pageSize, userId: userId || '' });
    return { transactions: response.transactions || [], total: response.total || 0 };
  },

  /** Admin: adjust a user's wallet balance (positive=add, negative=deduct). */
  adjustBalance: async (userId: string, amount: string, description: string) => {
    const response: any = await walletClient.adjustBalance({ userId, amount, description });
    return response.wallet;
  },

  /** Admin: search users by email or account number. */
  searchUsers: async (search: string) => {
    const response: any = await adminUserClient.listUsers({ page: 1, pageSize: 20, search });
    return (response.users || []) as Array<{
      id: string;
      email: string;
      nickname?: string;
      accountNumber?: string;
    }>;
  },
};
