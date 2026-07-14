import { depositClient } from './connect';

export type { DepositRequest } from '../gen/ant/v1/deposit_pb';

export const depositApi = {
  getDepositInfo: async () => {
    const response: any = await depositClient.getDepositInfo({});
    return {
      receivingAddress: response.receivingAddress || '',
      network: response.network || 'TRC20',
      exchangeRate: response.exchangeRate || '1',
    };
  },

  createDeposit: async (amount: string, txHash?: string) => {
    const response: any = await depositClient.createDeposit({ amount, txHash: txHash || '' });
    return response.deposit;
  },

  listMyDeposits: async (page = 1, pageSize = 20) => {
    const response: any = await depositClient.listMyDeposits({ page, pageSize });
    return { deposits: response.deposits || [], total: response.total || 0 };
  },

  // Admin APIs
  listDeposits: async (params?: { page?: number; pageSize?: number; status?: string }) => {
    const response: any = await depositClient.listDeposits({
      page: params?.page || 1,
      pageSize: params?.pageSize || 20,
      status: params?.status || '',
    });
    return { deposits: response.deposits || [], total: response.total || 0 };
  },

  approveDeposit: async (depositId: string, reviewNote?: string) => {
    const response: any = await depositClient.approveDeposit({ depositId, reviewNote: reviewNote || '' });
    return response.deposit;
  },

  rejectDeposit: async (depositId: string, reviewNote?: string) => {
    const response: any = await depositClient.rejectDeposit({ depositId, reviewNote: reviewNote || '' });
    return response.deposit;
  },
};
