import { webauthnClient } from './connect';
import type {
  BeginRegistrationResponse,
  FinishRegistrationResponse,
  ListCredentialsResponse,
  CredentialEntry,
  BeginWithdrawalResponse,
  FinishWithdrawalResponse,
  ListWithdrawalsResponse,
  WithdrawalEntry,
  CancelWithdrawalResponse,
  AddWhitelistAddressResponse,
  ListWhitelistAddressesResponse,
  WhitelistEntry,
  RemoveWhitelistAddressResponse,
  ExportCredentialListResponse,
  ExportWhitelistResponse,
} from '../gen/ant/v1/webauthn_pb';

export type { CredentialEntry, WithdrawalEntry, WhitelistEntry };

export const webauthnApi = {
  beginRegistration: async (name: string) => {
    const msg = await webauthnClient.beginRegistration({ name }) as BeginRegistrationResponse;
    return msg.credentialCreationOptions;
  },

  finishRegistration: async (credentialResponse: Uint8Array, name: string) => {
    const msg = await webauthnClient.finishRegistration({
      credentialResponse,
      name,
    }) as FinishRegistrationResponse;
    return { credentialId: msg.credentialId, name: msg.name };
  },

  listCredentials: async () => {
    const msg = await webauthnClient.listCredentials({}) as ListCredentialsResponse;
    return msg.credentials as CredentialEntry[];
  },

  removeCredential: async (credentialId: string) => {
    await webauthnClient.removeCredential({ credentialId });
  },

  beginWithdrawal: async (amount: string, destAddress: string) => {
    const msg = await webauthnClient.beginWithdrawal({ amount, destAddress }) as BeginWithdrawalResponse;
    return {
      challenge: msg.challenge,
      nonce: msg.nonce,
      withdrawalId: msg.withdrawalId,
    };
  },

  finishWithdrawal: async (withdrawalId: string, assertion: Uint8Array, credentialId: string) => {
    const msg = await webauthnClient.finishWithdrawal({
      withdrawalId,
      assertion,
      credentialId,
    }) as FinishWithdrawalResponse;
    return {
      withdrawalId: msg.withdrawalId,
      status: msg.status,
      unsignedBundle: msg.unsignedBundle,
    };
  },

  listWithdrawals: async (page = 1, pageSize = 20) => {
    const msg = await webauthnClient.listWithdrawals({ page, pageSize }) as ListWithdrawalsResponse;
    return {
      withdrawals: msg.withdrawals as WithdrawalEntry[],
      total: msg.total,
    };
  },

  cancelWithdrawal: async (withdrawalId: string) => {
    const msg = await webauthnClient.cancelWithdrawal({ withdrawalId }) as CancelWithdrawalResponse;
    return msg.status;
  },

  addWhitelistAddress: async (address: string, label: string) => {
    const msg = await webauthnClient.addWhitelistAddress({ address, label }) as AddWhitelistAddressResponse;
    return msg.status;
  },

  listWhitelistAddresses: async () => {
    const msg = await webauthnClient.listWhitelistAddresses({}) as ListWhitelistAddressesResponse;
    return msg.addresses as WhitelistEntry[];
  },

  removeWhitelistAddress: async (id: string) => {
    const msg = await webauthnClient.removeWhitelistAddress({ id }) as RemoveWhitelistAddressResponse;
    return msg.status;
  },

  exportCredentialList: async () => {
    const msg = await webauthnClient.exportCredentialList({}) as ExportCredentialListResponse;
    return { credentials: msg.credentials, exportedAtMs: msg.exportedAtMs };
  },

  exportWhitelist: async () => {
    const msg = await webauthnClient.exportWhitelist({}) as ExportWhitelistResponse;
    return { whitelistProto: msg.whitelistProto, exportedAtMs: msg.exportedAtMs };
  },
};
