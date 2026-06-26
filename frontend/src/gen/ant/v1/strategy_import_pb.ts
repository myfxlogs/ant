// @generated — placeholder until buf generate runs with protoc-gen-es.
// Types mirror strategy_import.proto messages exactly.

import { ServiceType } from "@bufbuild/protobuf";

export const StrategyImportService: ServiceType = {
  typeName: "ant.v1.StrategyImportService",
  methods: {
    analyzeCode: { name: "AnalyzeCode", I: AnalyzeCodeRequest, O: AnalyzeCodeResponse, kind: 1 /* unary */ },
    generateCode: { name: "GenerateCode", I: GenerateCodeRequest, O: GenerateCodeResponse, kind: 1 },
    importStrategy: { name: "ImportStrategy", I: ImportStrategyRequest, O: ImportStrategyResponse, kind: 1 },
  },
} as ServiceType;

// ── AnalyzeCode ──────────────────────────────────────────────────

export interface AnalyzeCodeRequest {
  sourceCode: string;
  sourceName: string;
  sourceLang: string;
}

export interface AnalyzeCodeResponse {
  strategyName: string;
  mqlVersion: string;
  coverageScore: number;
  totalBlocks: number;
  recognizedBlocks: number;
  params: ParamField[];
  groups: ParamGroupInfo[];
  executionKind: string;
  blindSpots: BlindSpotItem[];
  entryRulesCount: number;
  exitRulesCount: number;
  sizingKind: string;
  riskChecksCount: number;
  indicatorNames: string[];
}

export interface ParamField {
  name: string;
  label: string;
  paramType: string;
  defaultValue: string;
  group: string;
  groupOrder: number;
  min?: number;
  max?: number;
  step?: number;
  options: ParamOption[];
}

export interface ParamOption {
  value: string;
  label: string;
}

export interface ParamGroupInfo {
  name: string;
  label: string;
  order: number;
  fieldCount: number;
}

export interface BlindSpotItem {
  id: string;
  location: string;
  category: string;
  severity: string;
  description: string;
  handling: string;
  userActionRequired: boolean;
}

// ── GenerateCode ─────────────────────────────────────────────────

export interface GenerateCodeRequest {
  sourceCode: string;
  sourceName: string;
  sourceLang: string;
  paramOverrides: Record<string, string>;
}

export interface GenerateCodeResponse {
  pythonCode: string;
  codeLines: number;
  compiles: boolean;
  qualityGateFailures: string[];
}

// ── ImportStrategy ───────────────────────────────────────────────

export interface ImportStrategyRequest {
  sourceCode: string;
  sourceName: string;
  sourceLang: string;
  paramOverrides: Record<string, string>;
  workspaceId?: string;
}

export interface ImportStrategyResponse {
  strategyId: string;
  strategyName: string;
  pythonCode: string;
  coverageScore: number;
  blindSpots: BlindSpotItem[];
}
