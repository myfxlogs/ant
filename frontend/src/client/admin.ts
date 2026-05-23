import { adminAccountClient, adminConfigClient, adminLogClient, adminSystemClient, adminTradingClient, adminUserClient } from './connect';

// Note: getDashboard is defined in AdminUserService, not AdminSystemService

export type { DashboardStats } from '../gen/admin_dashboard_pb';
export type { AdminLog } from '../gen/admin_log_pb';
export type { AccountWithUser } from '../gen/admin_account_pb';
export type { UserWithAccounts } from '../gen/admin_user_entity_pb';
export type { TradingSummary } from '../gen/admin_trading_summary_pb';
export type { SystemConfig } from '../gen/admin_config_pb';

export type UserListParams = {
  page?: number;
  pageSize?: number;
  search?: string;
  status?: string;
  role?: string;
};

export type CreateUserRequest = {
  username: string;
  email: string;
  password: string;
  role?: string;
};

export type UpdateUserRequest = {
  username?: string;
  email?: string;
  role?: string;
  status?: string;
};

export type AccountListParams = {
  page?: number;
  pageSize?: number;
  userId?: string;
  search?: string;
  status?: string;
  mtType?: string;
};

export type LogListParams = {
  page?: number;
  pageSize?: number;
  userId?: string;
  action?: string;
  startDate?: string;
  endDate?: string;
  module?: string;
  actionType?: string;
};


export const adminApi = {
  getDashboard: async () => {
    return await adminUserClient.getDashboard({});
  },

  getDashboardStats: async () => {
    return adminApi.getDashboard();
  },

  listLogs: async (params?: LogListParams) => {
    const response: any = await adminLogClient.listLogs({
      page: params?.page || 1,
      pageSize: params?.pageSize || 10,
      userId: params?.userId,
      action: params?.action,
      startDate: params?.startDate,
      endDate: params?.endDate,
      module: params?.module,
      actionType: params?.actionType,
    });
    return {
      logs: response.logs,
      total: response.total,
    };
  },

  getLogs: async (params?: LogListParams) => {
    return adminApi.listLogs(params);
  },

  exportLogs: async (params?: { userId?: string; action?: string }): Promise<Blob> => {
    const response: any = await adminLogClient.exportLogs({
      userId: params?.userId,
      action: params?.action,
    });
    return new Blob([new Uint8Array(response.data)], { type: 'application/octet-stream' });
  },

  listUsers: async (params?: UserListParams) => {
    const response: any = await adminUserClient.listUsers({
      page: params?.page || 1,
      pageSize: params?.pageSize || 10,
      search: params?.search,
      status: params?.status,
      role: params?.role,
    });
    return {
      users: response.users,
      total: response.total,
    };
  },

  getUsers: async (params?: UserListParams) => {
    return adminApi.listUsers(params);
  },

  createUser: async (data: CreateUserRequest) => {
    return await adminUserClient.createUser({
      username: data.username,
      email: data.email,
      password: data.password,
      role: data.role,
    });
  },

  updateUser: async (id: string, data: UpdateUserRequest) => {
    return await adminUserClient.updateUser({
      id,
      username: data.username,
      email: data.email,
      role: data.role,
      status: data.status,
    });
  },

  deleteUser: async (id: string) => {
    await adminUserClient.deleteUser({ id });
  },

  disableUser: async (id: string) => {
    await adminUserClient.disableUser({ id });
  },

  enableUser: async (id: string) => {
    await adminUserClient.enableUser({ id });
  },

  resetUserPassword: async (id: string) => {
    const response: any = await adminUserClient.resetUserPassword({ id });
    return { newPassword: response.newPassword };
  },

  listAccounts: async (params?: AccountListParams) => {
    const response: any = await adminAccountClient.listAccountsAdmin({
      page: params?.page || 1,
      pageSize: params?.pageSize || 10,
      userId: params?.userId,
      search: params?.search,
      status: params?.status,
      mtType: params?.mtType,
    });
    return {
      accounts: response.accounts,
      total: response.total,
    };
  },

  getAccounts: async (params?: AccountListParams) => {
    return adminApi.listAccounts(params);
  },

  freezeAccount: async (id: string) => {
    await adminAccountClient.freezeAccount({ id });
  },

  unfreezeAccount: async (id: string) => {
    await adminAccountClient.unfreezeAccount({ id });
  },

  getTradingSummary: async (params?: { startDate?: string; endDate?: string }) => {
    return await adminTradingClient.getTradingSummary({
      startDate: params?.startDate,
      endDate: params?.endDate,
    });
  },

  listConfigs: async () => {
    const response: any = await adminConfigClient.listConfigs({});
    return response.configs;
  },

  getConfig: async () => {
    return adminApi.listConfigs();
  },

  setConfig: async (key: string, params: { value: string; description?: string }) => {
    await adminConfigClient.setConfig({
      key,
      value: params.value,
      description: params.description,
    });
  },

  updateConfig: async (key: string, value: string) => {
    return adminApi.setConfig(key, { value });
  },

  toggleConfigEnabled: async (key: string, enabled: boolean) => {
    await adminConfigClient.toggleConfigEnabled({
      key,
      enabled,
    });
  },

  healthCheck: async () => {
    return await adminSystemClient.healthCheck({});
  },

  getMetrics: async () => {
    return await adminSystemClient.getMetrics({});
  },

  resolveAlert: async (alertId: string) => {
    await adminSystemClient.resolveAlert({ alertId });
  },

  clearCache: async () => {
    await adminSystemClient.clearCache({});
  },

  invalidateCache: async (tags: string[]) => {
    await adminSystemClient.invalidateCache({ tags });
  },
};
