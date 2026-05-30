import { timestampDate } from '@bufbuild/protobuf/wkt';
import type { Timestamp } from '@bufbuild/protobuf/wkt';
import type { AIAgentDefinition } from '../../gen/ant/v1/ai_agent_pb';
import type { ConversationMessage as ProtoConversationMessage, ConversationSummary as ProtoConversationSummary } from '../../gen/ant/v1/ai_conversation_pb';

export function protoDate(ts: Timestamp | undefined): Date {
  return ts ? timestampDate(ts) : new Date();
}

export function toConversationRole(role: string): 'user' | 'assistant' | 'system' {
  if (role === 'user' || role === 'assistant' || role === 'system') return role;
  return 'user';
}

export interface ChatResult {
  message: string;
  suggestions: string[];
}

export interface ConversationSummary {
  id: string;
  title: string;
  messageCount: number;
  createdAt: Date;
  updatedAt: Date;
}

export interface ConversationMessage {
  id: string;
  role: 'user' | 'assistant' | 'system';
  content: string;
  createdAt: Date;
}

export interface ConversationDetail extends ConversationSummary {
  messages: ConversationMessage[];
}

export interface AIAgentDefinitionView {
  id: string;
  agentKey: string;
  type: string;
  name: string;
  identity: string;
  inputHint: string;
  enabled: boolean;
  position: number;
  providerId: string;
  modelOverride: string;
}

export function toAgentView(a: AIAgentDefinition): AIAgentDefinitionView {
  return {
    id: a.id || '',
    agentKey: a.agentKey || '',
    type: a.type || '',
    name: a.name || '',
    identity: a.identity || '',
    inputHint: a.inputHint || '',
    enabled: !!a.enabled,
    position: typeof a.position === 'number' ? a.position : 0,
    providerId: a.providerId || '',
    modelOverride: a.modelOverride || '',
  };
}

export function mapConversationSummary(c: ProtoConversationSummary): ConversationSummary {
  return {
    id: c.id,
    title: c.title || 'Untitled',
    messageCount: c.messageCount || 0,
    createdAt: protoDate(c.createdAt),
    updatedAt: protoDate(c.updatedAt),
  };
}
