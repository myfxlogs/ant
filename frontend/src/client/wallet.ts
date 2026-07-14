import { walletClient, adminUserClient } from './connect';
import type { GetWalletResponse, ListWalletTransactionsResponse, AdjustBalanceResponse } from '../gen/ant/v1/wallet_pb';
import type { ListUsersResponse } from '../gen/ant/v1/admin_user_pb';

export const walletApi = {
  /** Get wallet for the current user, or for a specific user (admin only). */
  getWallet: async (userId?: string) => {
    return (await walletClient.getWallet({ userId: userId || '' }) as GetWalletResponse).wallet;
  },

  /** List transactions for the current user, or for a specific user (admin only). */
  listTransactions: async (page = 1, pageSize = 20, userId?: string) => {
    const msg = await walletClient.listTransactions({ page, pageSize, userId: userId || '' }) as ListWalletTransactionsResponse;
    return { transactions: msg.transactions || [], total: msg.total || 0 };
  },

  /** Admin: adjust a user's wallet balance (positive=add, negative=deduct). */
  adjustBalance: async (userId: string, amount: string, description: string) => {
    return (await walletClient.adjustBalance({ userId, amount, description }) as AdjustBalanceResponse).wallet;
  },

  /** Admin: search users by email or account number. */
  searchUsers: async (search: string) => {
    const msg = await adminUserClient.listUsers({ page: 1, pageSize: 20, search }) as ListUsersResponse;
    return (msg.users || []) as Array<{
      id: string;
      email: string;
      nickname?: string;
      accountNumber?: string;
    }>;
  },
};
