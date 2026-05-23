import { createClient } from '@connectrpc/connect';
import { LogService } from '../gen/log_pb';
import { transport } from './transport';
import type { ConnectionLog as RpcConnectionLog } from '../gen/log_connection_pb';
import type { ExecutionLog as RpcExecutionLog } from '../gen/log_execution_pb';
import type { OrderHistoryRecord as RpcOrderHistoryRecord } from '../gen/log_order_pb';
import type { OperationLog as RpcOperationLog } from '../gen/log_operation_pb';
import type { ScheduleRunLog } from '../gen/log_schedule_pb';

const logClient = createClient(LogService, transport);

export type { ScheduleRunLog } from '../gen/log_schedule_pb';

export const logApi = {
  getConnectionLogs: async (params: {
    page?: number;
    pageSize?: number;
    accountId?: string;
    status?: string;
    startDate?: string;
    endDate?: string;
  }): Promise<{ logs: RpcConnectionLog[]; total: number }> => {
    const response = await logClient.getConnectionLogs({
      page: params.page || 1,
      pageSize: params.pageSize || 20,
      accountId: params.accountId || '',
      status: params.status || '',
      startDate: params.startDate || '',
      endDate: params.endDate || '',
    });
    return {
      logs: response.logs,
      total: response.total,
    };
  },

  getExecutionLogs: async (params: {
    page?: number;
    pageSize?: number;
    accountId?: string;
    scheduleId?: string;
    symbol?: string;
    status?: string;
    startDate?: string;
    endDate?: string;
  }): Promise<{ logs: RpcExecutionLog[]; total: number }> => {
    const response = await logClient.getExecutionLogs({
      page: params.page || 1,
      pageSize: params.pageSize || 20,
      accountId: params.accountId || '',
      scheduleId: params.scheduleId || '',
      symbol: params.symbol || '',
      status: params.status || '',
      startDate: params.startDate || '',
      endDate: params.endDate || '',
    });
    return {
      logs: response.logs,
      total: response.total,
    };
  },

  getOrderHistory: async (params: {
    page?: number;
    pageSize?: number;
    accountId?: string;
    scheduleId?: string;
    symbol?: string;
    startDate?: string;
    endDate?: string;
  }): Promise<{ orders: RpcOrderHistoryRecord[]; total: number }> => {
    const response = await logClient.getOrderLogHistory({
      page: params.page || 1,
      pageSize: params.pageSize || 20,
      accountId: params.accountId || '',
      symbol: params.symbol || '',
      startDate: params.startDate || '',
      endDate: params.endDate || '',
      scheduleId: params.scheduleId || '',
    });
    return {
      orders: response.orders,
      total: response.total,
    };
  },

  getOperationLogs: async (params: {
    page?: number;
    pageSize?: number;
    module?: string;
    action?: string;
    resourceType?: string;
    resourceId?: string;
    startDate?: string;
    endDate?: string;
  }): Promise<{ logs: RpcOperationLog[]; total: number }> => {
    const response = await logClient.getOperationLogs({
      page: params.page || 1,
      pageSize: params.pageSize || 20,
      module: params.module || '',
      action: params.action || '',
      resourceType: params.resourceType || '',
      resourceId: params.resourceId || '',
      startDate: params.startDate || '',
      endDate: params.endDate || '',
    });
    return {
      logs: response.logs,
      total: response.total,
    };
  },

  getScheduleRunLogs: async (params: {
    page?: number;
    pageSize?: number;
    scheduleId: string;
  }): Promise<{ logs: ScheduleRunLog[]; total: number }> => {
    const response = await logClient.getScheduleRunLogs({
      page: params.page || 1,
      pageSize: params.pageSize || 20,
      scheduleId: params.scheduleId || '',
    });
    return {
      logs: response.logs,
      total: response.total,
    };
  },
};
