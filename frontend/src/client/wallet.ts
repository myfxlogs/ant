import { walletClient } from './connect';

export const walletApi = {
  getWallet: async () => {
    const response: any = await walletClient.getWallet({});
    return response.wallet;
  },

  listTransactions: async (page = 1, pageSize = 20) => {
    const response: any = await walletClient.listTransactions({ page, pageSize });
    return { transactions: response.transactions || [], total: response.total || 0 };
  },
};
