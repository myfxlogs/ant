import { depositClient } from './connect';
import type { GetDepositInfoResponse, CreateDepositResponse, ListMyDepositsResponse, ListDepositsResponse, ApproveDepositResponse, RejectDepositResponse } from '../gen/ant/v1/deposit_pb';

export type { DepositRequest } from '../gen/ant/v1/deposit_pb';

export const depositApi = {
  getDepositInfo: async () => {
    const msg = await depositClient.getDepositInfo({}) as GetDepositInfoResponse;
    return {
      receivingAddress: msg.receivingAddress || '',
      network: msg.network || 'TRC20',
      exchangeRate: msg.exchangeRate || '1',
    };
  },

  createDeposit: async (amount: string, txHash?: string) => {
    return (await depositClient.createDeposit({ amount, txHash: txHash || '' }) as CreateDepositResponse).deposit;
  },

  listMyDeposits: async (page = 1, pageSize = 20) => {
    const msg = await depositClient.listMyDeposits({ page, pageSize }) as ListMyDepositsResponse;
    return { deposits: msg.deposits || [], total: msg.total || 0 };
  },

  // Admin APIs
  listDeposits: async (params?: { page?: number; pageSize?: number; status?: string }) => {
    const msg = await depositClient.listDeposits({
      page: params?.page || 1,
      pageSize: params?.pageSize || 20,
      status: params?.status || '',
    }) as ListDepositsResponse;
    return { deposits: msg.deposits || [], total: msg.total || 0 };
  },

  approveDeposit: async (depositId: string, reviewNote?: string) => {
    return (await depositClient.approveDeposit({ depositId, reviewNote: reviewNote || '' }) as ApproveDepositResponse).deposit;
  },

  rejectDeposit: async (depositId: string, reviewNote?: string) => {
    return (await depositClient.rejectDeposit({ depositId, reviewNote: reviewNote || '' }) as RejectDepositResponse).deposit;
  },
};
