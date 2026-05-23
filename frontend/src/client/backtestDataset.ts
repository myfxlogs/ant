import { backtestDatasetClient } from './connect';
import { create } from '@bufbuild/protobuf';
import type { Timestamp } from '@bufbuild/protobuf/wkt';
import {
  CreateFrozenBacktestDatasetRequestSchema,
  DeleteBacktestDatasetRequestSchema,
  ListBacktestDatasetsRequestSchema,
} from '../gen/backtest_dataset_pb';

const datasetClient = backtestDatasetClient;

function toTimestamp(d?: Date): Timestamp | undefined {
  if (!d) return undefined;
  const ms = d.getTime();
  const seconds = Math.floor(ms / 1000);
  const nanos = (ms % 1000) * 1_000_000;
  return { seconds: BigInt(seconds), nanos } as unknown as Timestamp;
}

function toDateMaybe(v: any): Date | undefined {
  if (!v) return undefined;
  if (typeof v?.toDate === 'function') {
    const d = v.toDate();
    if (d instanceof Date && !Number.isNaN(d.getTime())) return d;
  }
  const seconds = v?.seconds;
  const nanos = v?.nanos;
  if (seconds !== undefined) {
    const secNum = typeof seconds === 'bigint' ? Number(seconds) : Number(seconds);
    const nanoNum = nanos !== undefined ? Number(nanos) : 0;
    const ms = secNum * 1000 + Math.floor(nanoNum / 1_000_000);
    const d = new Date(ms);
    if (!Number.isNaN(d.getTime())) return d;
  }
  if (typeof v === 'string' || typeof v === 'number') {
    const d = new Date(v);
    if (!Number.isNaN(d.getTime())) return d;
  }
  return undefined;
}

export interface BacktestDatasetView {
  id: string;
  accountId: string;
  symbol: string;
  timeframe: string;
  from?: Date;
  to?: Date;
  count: number;
  frozen: boolean;
  createdAt?: Date;
}

function toView(d: any): BacktestDatasetView {
  return {
    id: d.id,
    accountId: d.accountId,
    symbol: d.symbol,
    timeframe: d.timeframe,
    from: toDateMaybe(d.from),
    to: toDateMaybe(d.to),
    count: Number(d.count ?? 0),
    frozen: !!d.frozen,
    createdAt: toDateMaybe(d.createdAt),
  };
}

export const backtestDatasetApi = {
  list: async (params: { accountId?: string; symbol?: string; timeframe?: string; limit?: number; offset?: number }) => {
    const msg = create(ListBacktestDatasetsRequestSchema, {
      accountId: params.accountId,
      symbol: params.symbol,
      timeframe: params.timeframe,
      limit: params.limit ?? 50,
      offset: params.offset ?? 0,
    });
    const resp: any = await datasetClient.listBacktestDatasets(msg as any);
    return (resp.datasets || []).map(toView);
  },

  createFrozen: async (params: { accountId: string; symbol: string; timeframe: string; from: Date; to: Date }) => {
    const msg = create(CreateFrozenBacktestDatasetRequestSchema, {
      accountId: params.accountId,
      symbol: params.symbol,
      timeframe: params.timeframe,
      from: toTimestamp(params.from),
      to: toTimestamp(params.to),
    });
    const resp: any = await datasetClient.createFrozenBacktestDataset(msg as any);
    return { datasetId: resp.datasetId as string };
  },

  delete: async (datasetId: string) => {
    const msg = create(DeleteBacktestDatasetRequestSchema, {
      datasetId,
    });
    await datasetClient.deleteBacktestDataset(msg as any);
  },
};
