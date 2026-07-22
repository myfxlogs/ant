import { depositClient } from './connect';
import type {
  GetDepositAddressResponse,
  ListMyDepositsResponse,
  ListManualReviewDepositsResponse,
  ListDepositAddressesResponse,
  ImportDepositAddressesResponse,
  ListPendingSignBundlesResponse,
  PendingSignBundleEntry,
  ExportUnsignedSweepBundleResponse,
  ExportBatchUnsignedSweepBundleResponse,
  ImportSignedSweepBundleResponse,
  GetSweepDashboardResponse,
  SweepDashboardEntry,
  BuildUndelegateOnlyBundleResponse,
  ImportXpubResponse,
} from '../gen/ant/v1/deposit_pb';

export type { PendingSignBundleEntry, SweepDashboardEntry };

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

  listDepositAddresses: async (params?: { page?: number; pageSize?: number; status?: string }) => {
    const msg = await depositClient.listDepositAddresses({
      page: params?.page || 1,
      pageSize: params?.pageSize || 20,
      status: params?.status || '',
    }) as ListDepositAddressesResponse;
    return {
      addresses: msg.addresses || [],
      total: msg.total || 0,
      availableCount: msg.availableCount || 0,
    };
  },

  importDepositAddresses: async (batchData: Uint8Array) => {
    const msg = await depositClient.importDepositAddresses({ batchData }) as ImportDepositAddressesResponse;
    return {
      imported: msg.imported || 0,
      skipped: msg.skipped || 0,
    };
  },

  // Admin: Sweep APIs
  listPendingSignBundles: async () => {
    const msg = await depositClient.listPendingSignBundles({}) as ListPendingSignBundlesResponse;
    return msg.bundles as PendingSignBundleEntry[];
  },

  exportUnsignedSweepBundle: async (depositAddressId: string) => {
    const msg = await depositClient.exportUnsignedSweepBundle({ depositAddressId }) as ExportUnsignedSweepBundleResponse;
    return msg.unsignedBundle;
  },

  exportBatchUnsignedSweepBundle: async (depositAddressIds: string[]) => {
    const msg = await depositClient.exportBatchUnsignedSweepBundle({ depositAddressIds }) as ExportBatchUnsignedSweepBundleResponse;
    return msg.unsignedBundle;
  },

  importSignedSweepBundle: async (signedBundle: Uint8Array) => {
    const msg = await depositClient.importSignedSweepBundle({ signedBundle }) as ImportSignedSweepBundleResponse;
    return { batchId: msg.batchId, broadcastComplete: msg.broadcastComplete };
  },

  getSweepDashboard: async (page = 1, pageSize = 20) => {
    const msg = await depositClient.getSweepDashboard({ page, pageSize }) as GetSweepDashboardResponse;
    return {
      addresses: msg.addresses as SweepDashboardEntry[],
      total: msg.total,
      totalUnswept: msg.totalUnswept,
      threshold: msg.threshold,
    };
  },

  buildUndelegateOnlyBundle: async (depositAddressIds: string[]) => {
    const msg = await depositClient.buildUndelegateOnlyBundle({ depositAddressIds }) as BuildUndelegateOnlyBundleResponse;
    return msg.unsignedBundle;
  },

  importXpub: async (xpubExport: Uint8Array) => {
    const msg = await depositClient.importXpub({ xpubExport }) as ImportXpubResponse;
    return {
      xpub: msg.xpub,
      fingerprint: msg.fingerprint,
      fingerprintVerified: msg.fingerprintVerified,
    };
  },
};
