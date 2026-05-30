import { timestampDate } from '@bufbuild/protobuf/wkt';
import type { Timestamp } from '@bufbuild/protobuf/wkt';
import { create } from '@bufbuild/protobuf';
import { AIAgentDefinitionSchema, type AIAgentDefinition } from '../../gen/ant/v1/ai_agent_pb';
import type { ConversationMessage as ProtoConversationMessage, ConversationSummary as ProtoConversationSummary } from '../../gen/ant/v1/ai_conversation_pb';
import type { WorkflowRunSummary as ProtoWorkflowRunSummary, WorkflowStep as ProtoWorkflowStep } from '../../gen/ant/v1/ai_workflow_entity_pb';

export type { AIReport } from '../../gen/ant/v1/ai_pb';

function protoDate(ts: Timestamp | undefined): Date {
  return ts ? timestampDate(ts) : new Date();
}

function toConversationRole(role: string): 'user' | 'assistant' | 'system' {
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

export interface WorkflowRunSummary {
  id: string;
  title: string;
  status: string;
  createdAt: Date;
  updatedAt: Date;
  stepCount: number;
}

export interface WorkflowStep {
  id: string;
  runId: string;
  key: string;
  title: string;
  status: string;
  input: string;
  output: string;
  error: string;
  durationMs: number;
  createdAt: Date;
}

export interface WorkflowRunDetail {
  run: WorkflowRunSummary;
  steps: WorkflowStep[];
  contextJson: string;
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

export function mapWorkflowRunSummary(r: ProtoWorkflowRunSummary): WorkflowRunSummary {
  return {
    id: r.id,
    title: r.title || '',
    status: r.status || '',
    createdAt: protoDate(r.createdAt),
    updatedAt: protoDate(r.updatedAt),
    stepCount: r.stepCount || 0,
  };
}

export function mapWorkflowStep(s: ProtoWorkflowStep): WorkflowStep {
  return {
    id: s.id,
    runId: s.runId,
    key: s.key,
    title: s.title || '',
    status: s.status || '',
    input: s.input || '',
    output: s.output || '',
    error: s.error || '',
    durationMs: Number(s.durationMs || 0n),
    createdAt: protoDate(s.createdAt),
  };
}

export function viewToAgentDefinition(a: AIAgentDefinitionView): AIAgentDefinition {
  return create(AIAgentDefinitionSchema, {
    id: a.id,
    agentKey: a.agentKey,
    type: a.type,
    name: a.name,
    identity: a.identity,
    inputHint: a.inputHint,
    enabled: a.enabled,
    position: a.position,
    providerId: a.providerId || '',
    modelOverride: a.modelOverride || '',
  });
}
