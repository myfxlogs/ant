import { depositClient } from './connect';
import type { GetDepositInfoResponse, CreateDepositResponse, ListMyDepositsResponse, ListDepositsResponse, ApproveDepositResponse, RejectDepositResponse } from '../gen/ant/v1/deposit_pb';

export type { DepositRequest } from '../gen/ant/v1/deposit_pb';

export const depositApi = {
  getDepositInfo: async () => {
    const res = await depositClient.getDepositInfo({});
    const msg = res.msg as GetDepositInfoResponse;
    return {
      receivingAddress: msg.receivingAddress || '',
      network: msg.network || 'TRC20',
      exchangeRate: msg.exchangeRate || '1',
    };
  },

  createDeposit: async (amount: string, txHash?: string) => {
    const res = await depositClient.createDeposit({ amount, txHash: txHash || '' });
    return (res.msg as CreateDepositResponse).deposit;
  },

  listMyDeposits: async (page = 1, pageSize = 20) => {
    const res = await depositClient.listMyDeposits({ page, pageSize });
    const msg = res.msg as ListMyDepositsResponse;
    return { deposits: msg.deposits || [], total: msg.total || 0 };
  },

  // Admin APIs
  listDeposits: async (params?: { page?: number; pageSize?: number; status?: string }) => {
    const res = await depositClient.listDeposits({
      page: params?.page || 1,
      pageSize: params?.pageSize || 20,
      status: params?.status || '',
    });
    const msg = res.msg as ListDepositsResponse;
    return { deposits: msg.deposits || [], total: msg.total || 0 };
  },

  approveDeposit: async (depositId: string, reviewNote?: string) => {
    const res = await depositClient.approveDeposit({ depositId, reviewNote: reviewNote || '' });
    return (res.msg as ApproveDepositResponse).deposit;
  },

  rejectDeposit: async (depositId: string, reviewNote?: string) => {
    const res = await depositClient.rejectDeposit({ depositId, reviewNote: reviewNote || '' });
    return (res.msg as RejectDepositResponse).deposit;
  },
};
