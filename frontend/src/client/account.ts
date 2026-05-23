import { accountClient } from './connect';
import { toCamelCase } from '../adapters/dataAdapter';

export type { Account, BrokerCompany } from '../gen/api_pb';

export interface ConnectAccountResult {
  success: boolean;
  message: string;
}

export const accountApi = {
  list: async () => {
    const response = await accountClient.listAccounts({});
    const accounts = (response as any).accounts || [];
    return toCamelCase(accounts);
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
    const response: any = await accountClient.connectAccount({ id });
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
    const response: any = await accountClient.searchBroker({ 
      company,
      mtType: mtType || 'MT5',
    });
    return response.companies;
  },

  // 轻量探测账户是否具备交易权限（非投资者只读模式）。
  verifyTradePermission: async (id: string) => {
    const response: any = await (accountClient as any).verifyTradePermission({ id });
    return {
      hasTradePermission: Boolean(response?.hasTradePermission),
      isInvestor: Boolean(response?.isInvestor),
      verified: Boolean(response?.verified),
      message: String(response?.message || ''),
    };
  },

  // 用新密码做一次 Connect 测试，成功后覆盖库里存的密码并刷新 is_investor。
  updateTradingPassword: async (id: string, newPassword: string) => {
    const response: any = await (accountClient as any).updateTradingPassword({
      id,
      newPassword,
    });
    return {
      success: Boolean(response?.success),
      hasTradePermission: Boolean(response?.hasTradePermission),
      isInvestor: Boolean(response?.isInvestor),
      message: String(response?.message || ''),
    };
  },
};
