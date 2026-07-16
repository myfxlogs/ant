import { agentGatewayClient } from './connect';
import type { ListMemoryResponse } from '../gen/ant/v1/agent_gateway_pb';
import { create } from '@bufbuild/protobuf';
import {
  ListMemoryRequestSchema,
  SaveUserTemplateRequestSchema,
  DeleteUserTemplateRequestSchema,
  DeleteAgentExperienceRequestSchema,
  type TemplateScope,
} from '../gen/ant/v1/agent_gateway_pb';

export async function listMemory(): Promise<ListMemoryResponse> {
  return await agentGatewayClient.listMemory(
    create(ListMemoryRequestSchema, {}),
  );
}

export async function saveUserTemplate(
  name: string,
  content: string,
  scope?: TemplateScope,
): Promise<boolean> {
  const res = await agentGatewayClient.saveUserTemplate(
    create(SaveUserTemplateRequestSchema, { name, content, scope }),
  );
  return res.success;
}

export async function deleteUserTemplate(id: string): Promise<boolean> {
  const res = await agentGatewayClient.deleteUserTemplate(
    create(DeleteUserTemplateRequestSchema, { id }),
  );
  return res.success;
}

export async function deleteAgentExperience(id: string): Promise<boolean> {
  const res = await agentGatewayClient.deleteAgentExperience(
    create(DeleteAgentExperienceRequestSchema, { id }),
  );
  return res.success;
}
