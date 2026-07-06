import { accountClient } from './connect';
import { toCamelCase } from '../adapters/dataAdapter';
import type { Account } from '@/types/account';

export interface ConnectAccountResult {
  success: boolean;
  message: string;
}

const NUMERIC_FIELDS: (keyof Account)[] = [
  'balance', 'credit', 'equity', 'margin', 'freeMargin', 'marginLevel',
  'profit', 'profitPercent', 'leverage', 'method',
];

function coerceAccountNumbers(raw: Account): Account {
  const out = { ...raw };
  for (const key of NUMERIC_FIELDS) {
    const v = out[key];
    if (typeof v === 'string') {
      const n = Number(v);
      (out as Record<string, unknown>)[key] = Number.isFinite(n) ? n : 0;
    }
  }
  return out;
}

function coerceAccountList(raw: Account[]): Account[] {
  return raw.map(coerceAccountNumbers);
}

export const accountApi = {
  list: async (): Promise<Account[]> => {
    const response = await accountClient.listAccounts({});
    return coerceAccountList(toCamelCase<Account[]>(response.accounts));
  },

  get: async (id: string): Promise<Account> => {
    const response = await accountClient.getAccount({ id });
    return coerceAccountNumbers(toCamelCase<Account>(response));
  },

  create: async (data: {
    login: string;
    password: string;
    mtType: string;
    brokerCompany: string;
    brokerServer: string;
    brokerHost: string;
  }): Promise<Account> => {
    const response = await accountClient.createAccount({
      login: data.login,
      password: data.password,
      mtType: data.mtType,
      brokerCompany: data.brokerCompany,
      brokerServer: data.brokerServer,
      brokerHost: data.brokerHost,
    });
    return coerceAccountNumbers(toCamelCase<Account>(response));
  },

  update: async (params: {
    id: string;
    brokerCompany?: string;
    brokerServer?: string;
    brokerHost?: string;
  }): Promise<Account> => {
    const response = await accountClient.updateAccount({
      id: params.id,
      brokerCompany: params.brokerCompany,
      brokerServer: params.brokerServer,
      brokerHost: params.brokerHost,
    });
    return coerceAccountNumbers(toCamelCase<Account>(response));
  },

  delete: async (id: string, password?: string) => {
    await accountClient.deleteAccount({ id, password: password || '' });
  },

  connect: async (id: string): Promise<ConnectAccountResult> => {
    const response = await accountClient.connectAccount({ id });
    const camel = toCamelCase(response);
    return {
      success: camel.success,
      message: camel.message,
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
    return toCamelCase(response.companies);
  },

  // Lightweight probe to check whether the account has trade permissions
  // (not investor read-only mode).
  verifyTradePermission: async (id: string) => {
    const response = await accountClient.verifyTradePermission({ id });
    const camel = toCamelCase(response);
    return {
      hasTradePermission: camel.hasTradePermission,
      isInvestor: camel.isInvestor,
      verified: camel.verified,
      message: camel.message,
    };
  },

  // Test-connect with a new password, then persist it and refresh is_investor.
  updateTradingPassword: async (id: string, newPassword: string, oldPassword?: string) => {
    const response = await accountClient.updateTradingPassword({
      id,
      newPassword,
      oldPassword: oldPassword || '',
    });
    const camel = toCamelCase(response);
    return {
      success: camel.success,
      hasTradePermission: camel.hasTradePermission,
      isInvestor: camel.isInvestor,
      message: camel.message,
    };
  },
};
