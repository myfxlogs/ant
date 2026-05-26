import { accountClient } from './connect';
import { toCamelCase } from '../adapters/dataAdapter';

export type { Account, BrokerCompany } from '../gen/ant/v1/api_pb';

export interface ConnectAccountResult {
  success: boolean;
  message: string;
}

export const accountApi = {
  list: async () => {
    const response = await accountClient.listAccounts({});
    return toCamelCase(response.accounts);
  },

  get: async (id: string) => {
    const response = await accountClient.getAccount({ id });
    return toCamelCase(response);
  },

  create: async (data: {
    login: string;
    password: string;
    mtType: string;
    brokerCompany: string;
    brokerServer: string;
    brokerHost: string;
  }) => {
    return await accountClient.createAccount({
      login: data.login,
      password: data.password,
      mtType: data.mtType,
      brokerCompany: data.brokerCompany,
      brokerServer: data.brokerServer,
      brokerHost: data.brokerHost,
    });
  },

  update: async (params: {
    id: string;
    brokerCompany?: string;
    brokerServer?: string;
    brokerHost?: string;
    isDisabled?: boolean;
  }) => {
    return await accountClient.updateAccount({
      id: params.id,
      brokerCompany: params.brokerCompany,
      brokerServer: params.brokerServer,
      brokerHost: params.brokerHost,
      isDisabled: params.isDisabled,
    });
  },

  delete: async (id: string) => {
    await accountClient.deleteAccount({ id });
  },

  connect: async (id: string): Promise<ConnectAccountResult> => {
    const response = await accountClient.connectAccount({ id });
    return {
      success: response.success,
      message: response.message,
    };
  },

  disconnect: async (id: string) => {
    await accountClient.disconnectAccount({ id });
  },

  reconnect: async (id: string) => {
    await accountClient.reconnectAccount({ id });
  },

  searchBroker: async (company: string, mtType?: string) => {
    const response = await accountClient.searchBroker({
      company,
      mtType: mtType || 'MT5',
    });
    return response.companies;
  },

  // Lightweight probe to check whether the account has trade permissions
  // (not investor read-only mode).
  verifyTradePermission: async (id: string) => {
    const response = await accountClient.verifyTradePermission({ id });
    return {
      hasTradePermission: response.hasTradePermission,
      isInvestor: response.isInvestor,
      verified: response.verified,
      message: response.message,
    };
  },

  // Test-connect with a new password, then persist it and refresh is_investor.
  updateTradingPassword: async (id: string, newPassword: string) => {
    const response = await accountClient.updateTradingPassword({
      id,
      newPassword,
    });
    return {
      success: response.success,
      hasTradePermission: response.hasTradePermission,
      isInvestor: response.isInvestor,
      message: response.message,
    };
  },
};
