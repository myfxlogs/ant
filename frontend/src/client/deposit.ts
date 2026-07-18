import { depositClient } from './connect';
import type { GetDepositAddressResponse, ListMyDepositsResponse, ListManualReviewDepositsResponse } from '../gen/ant/v1/deposit_pb';

export type { Deposit } from '../gen/ant/v1/deposit_pb';

export const depositApi = {
  getDepositAddress: async () => {
    const msg = await depositClient.getDepositAddress({}) as GetDepositAddressResponse;
    return {
      address: msg.address || '',
      network: msg.network || 'TRC20',
    };
  },

  listMyDeposits: async (page = 1, pageSize = 20) => {
    const msg = await depositClient.listMyDeposits({ page, pageSize }) as ListMyDepositsResponse;
    return { deposits: msg.deposits || [], total: msg.total || 0 };
  },

  // Admin APIs
  listManualReviewDeposits: async (params?: { page?: number; pageSize?: number }) => {
    const msg = await depositClient.listManualReviewDeposits({
      page: params?.page || 1,
      pageSize: params?.pageSize || 20,
    }) as ListManualReviewDepositsResponse;
    return { deposits: msg.deposits || [], total: msg.total || 0 };
  },
};
