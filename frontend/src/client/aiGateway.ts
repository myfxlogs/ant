import { create } from '@bufbuild/protobuf';
import { aiGatewayClient } from './connect';
import {
  ListSystemModelsRequestSchema,
  GetTokenUsageRequestSchema,
} from '../gen/ant/v1/ai_gateway_pb';
import {
  ListProvidersRequestSchema,
  UpdateProviderRequestSchema,
  ListModelsRequestSchema,
  UpsertModelRequestSchema,
  DeleteModelRequestSchema,
} from '../gen/ant/v1/ai_gateway_pb';

export interface SystemModelInfo {
  id: string;
  providerId: string;
  modelName: string;
  displayName: string;
  pricePer1mInput: string;
  pricePer1mOutput: string;
}

export interface TokenUsageInfo {
  featureTokens: Record<string, number>;
  records: Array<{
    id: string;
    paidBy: string;
    providerId: string;
    modelName: string;
    feature: string;
    inputTokens: number;
    outputTokens: number;
    cost: string;
  }>;
}

export interface AIProviderInfo {
  id: string;
  providerId: string;
  name: string;
  baseUrl: string;
  enabled: boolean;
  hasApiKey: boolean;
}

export interface AIModelConfigInfo {
  id: string;
  modelName: string;
  displayName: string;
  pricePer1mInput: string;
  pricePer1mOutput: string;
  enabled: boolean;
  sortOrder: number;
}

export const aiGatewayApi = {
  // ── User-facing ──
  listSystemModels: async (): Promise<SystemModelInfo[]> => {
    const req = create(ListSystemModelsRequestSchema, {});
    const resp = await aiGatewayClient.listSystemModels(req);
    return (resp.models || []).map(m => ({
      id: m.id,
      providerId: m.providerId,
      modelName: m.modelName,
      displayName: m.displayName || m.modelName,
      pricePer1mInput: m.pricePer1mInput,
      pricePer1mOutput: m.pricePer1mOutput,
    }));
  },

  getTokenUsage: async (): Promise<TokenUsageInfo> => {
    const req = create(GetTokenUsageRequestSchema, {});
    const resp = await aiGatewayClient.getTokenUsage(req);
    return {
      featureTokens: resp.featureTokens || {},
      records: (resp.records || []).map(r => ({
        id: r.id,
        paidBy: r.paidBy,
        providerId: r.providerId,
        modelName: r.modelName,
        feature: r.feature,
        inputTokens: r.inputTokens,
        outputTokens: r.outputTokens,
        cost: r.cost || '0',
      })),
    };
  },

  // ── Admin ──
  listProviders: async (): Promise<AIProviderInfo[]> => {
    const req = create(ListProvidersRequestSchema, {});
    const resp = await aiGatewayClient.listProviders(req);
    return (resp.providers || []).map(p => ({
      id: p.id,
      providerId: p.providerId,
      name: p.name,
      baseUrl: p.baseUrl,
      enabled: p.enabled,
      hasApiKey: p.hasApiKey,
    }));
  },

  updateProvider: async (params: {
    id: string;
    name?: string;
    baseUrl?: string;
    apiKey?: string;
    enabled?: boolean;
  }): Promise<void> => {
    const req = create(UpdateProviderRequestSchema, {
      id: params.id,
      name: params.name,
      baseUrl: params.baseUrl,
      apiKey: params.apiKey,
      enabled: params.enabled,
    });
    await aiGatewayClient.updateProvider(req);
  },

  listModels: async (providerId: string): Promise<AIModelConfigInfo[]> => {
    const req = create(ListModelsRequestSchema, { providerId });
    const resp = await aiGatewayClient.listModels(req);
    return (resp.models || []).map(m => ({
      id: m.id,
      modelName: m.modelName,
      displayName: m.displayName || '',
      pricePer1mInput: m.pricePer1mInput,
      pricePer1mOutput: m.pricePer1mOutput,
      enabled: m.enabled,
      sortOrder: m.sortOrder,
    }));
  },

  upsertModel: async (params: {
    id?: string;
    providerId: string;
    modelName: string;
    displayName?: string;
    pricePer1mInput: string;
    pricePer1mOutput: string;
    enabled?: boolean;
    sortOrder?: number;
  }): Promise<string> => {
    const req = create(UpsertModelRequestSchema, {
      id: params.id,
      providerId: params.providerId,
      modelName: params.modelName,
      displayName: params.displayName,
      pricePer1mInput: params.pricePer1mInput,
      pricePer1mOutput: params.pricePer1mOutput,
      enabled: params.enabled,
      sortOrder: params.sortOrder,
    });
    const resp = await aiGatewayClient.upsertModel(req);
    return resp.id;
  },

  deleteModel: async (id: string): Promise<void> => {
    const req = create(DeleteModelRequestSchema, { id });
    await aiGatewayClient.deleteModel(req);
  },
};
