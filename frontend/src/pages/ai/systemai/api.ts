import { create } from '@bufbuild/protobuf'
import { systemAIClient } from '@/client/connect'
import {
  ListSystemAIConfigsRequestSchema,
} from '@/gen/system_ai_query_pb'
import {
  UpdateSystemAIConfigRequestSchema,
  UpdateSystemAISecretRequestSchema,
} from '@/gen/system_ai_update_pb'
import {
  DiscoverSystemAIModelsRequestSchema,
  ValidateSystemAIConnectionRequestSchema,
} from '@/gen/system_ai_probe_pb'
import type { AIConfig } from './model'

export async function listSystemAIConfigs(): Promise<{ items: AIConfig[] }> {
  const r = await systemAIClient.listSystemAIConfigs(create(ListSystemAIConfigsRequestSchema, {}))
  return {
    items: r.items.map((it) => ({
      provider_id: it.providerId,
      name: it.name,
      base_url: it.baseUrl,
      organization: it.organization,
      models: it.models || [],
      default_model: it.defaultModel,
      temperature: it.temperature,
      timeout_seconds: it.timeoutSeconds,
      max_tokens: it.maxTokens,
      purposes: it.purposes || [],
      primary_for: it.primaryFor || [],
      enabled: it.enabled,
      has_secret: it.hasSecret,
      updated_at: it.updatedAt,
    })),
  }
}

export async function updateSystemAIConfig(providerId: string, payload: Record<string, unknown>) {
  await systemAIClient.updateSystemAIConfig(create(UpdateSystemAIConfigRequestSchema, {
    providerId,
    name: String(payload.name || ''),
    baseUrl: String(payload.base_url || ''),
    organization: String(payload.organization || ''),
    models: (payload.models as string[]) || [],
    defaultModel: String(payload.default_model || ''),
    temperature: Number(payload.temperature || 0),
    timeoutSeconds: Number(payload.timeout_seconds || 0),
    maxTokens: Number(payload.max_tokens || 0),
    purposes: (payload.purposes as string[]) || [],
    primaryFor: (payload.primary_for as string[]) || [],
    enabled: Boolean(payload.enabled),
  }))
  return { provider_id: providerId }
}

export async function updateSystemAISecret(providerId: string, secret: string) {
  await systemAIClient.updateSystemAISecret(create(UpdateSystemAISecretRequestSchema, { providerId, secret }))
  return { provider_id: providerId, secret_updated: true }
}

export async function clearSystemAISecret(providerId: string) {
  await systemAIClient.updateSystemAISecret(create(UpdateSystemAISecretRequestSchema, { providerId, secret: '' }))
  return { provider_id: providerId, secret_updated: false }
}

export async function discoverSystemAIModels(providerId: string) {
  const r = await systemAIClient.discoverSystemAIModels(create(DiscoverSystemAIModelsRequestSchema, { providerId }))
  return { provider_id: r.providerId, models: r.models, default_model: r.defaultModel }
}

export async function validateSystemAI(providerId: string) {
  const r = await systemAIClient.validateSystemAIConnection(create(ValidateSystemAIConnectionRequestSchema, { providerId }))
  return { provider_id: r.providerId, ok: r.ok, model_count: r.modelCount }
}
