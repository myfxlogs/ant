import { creditClient, adminCreditClient } from './connect';
import type { CreditTransaction } from '../gen/ant/v1/credit_pb';

export const creditApi = {
  getBalance: async (): Promise<{ balance: string; frozenBalance: string }> => {
    const resp = await creditClient.getCreditBalance({});
    return {
      balance: resp.balance,
      frozenBalance: resp.frozenBalance,
    };
  },

  listTransactions: async (page = 1, pageSize = 20): Promise<{ transactions: CreditTransaction[]; total: bigint }> => {
    const resp = await creditClient.listCreditTransactions({ page, pageSize });
    return {
      transactions: resp.transactions,
      total: resp.total,
    };
  },

  // Admin APIs
  addCredits: async (userId: string, amount: string, description: string): Promise<string> => {
    const resp = await adminCreditClient.addCredits({ userId, amount, description });
    return resp.newBalance;
  },

  refundCredits: async (userId: string, amount: string, description: string): Promise<string> => {
    const resp = await adminCreditClient.refundCredits({ userId, amount, description });
    return resp.newBalance;
  },

  listAllTransactions: async (page = 1, pageSize = 20, userId?: string, txType?: string) => {
    const resp = await adminCreditClient.listAllCreditTransactions({ page, pageSize, userId: userId || '', txType: txType || '' });
    return {
      transactions: resp.transactions,
      total: resp.total,
    };
  },
};
