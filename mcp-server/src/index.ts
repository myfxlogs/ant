import { McpServer } from '@modelcontextprotocol/sdk/server/mcp.js';
import { StdioServerTransport } from '@modelcontextprotocol/sdk/server/stdio.js';
import { z } from 'zod';

const baseUrl = (process.env.ANTTRADER_CONNECT_URL || 'http://127.0.0.1:8080').replace(/\/+$/, '');
const token = process.env.ANTTRADER_ACCESS_TOKEN || '';

const server = new McpServer({ name: 'anttrader-mcp-server', version: '0.1.0' });
const services = {
  account: 'antrader.AccountService',
  job: 'antrader.JobService',
  market: 'antrader.MarketService',
  marketRegime: 'antrader.MarketRegimeService',
  strategy: 'antrader.StrategyService',
  strategyAsset: 'antrader.StrategyAssetService',
  strategyExperiment: 'antrader.StrategyExperimentService',
} as const;

function jsonText(data: unknown) {
  return { content: [{ type: 'text' as const, text: JSON.stringify(data, null, 2) }] };
}

async function callConnect(service: string, method: string, body: Record<string, unknown>) {
  const response = await fetch(`${baseUrl}/${service}/${method}`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: JSON.stringify(body),
  });
  const text = await response.text();
  let data: unknown = text;
  if (text) {
    try {
      data = JSON.parse(text);
    } catch {
      data = text;
    }
  }
  if (!response.ok) {
    throw new Error(`ConnectRPC ${service}/${method} failed: ${response.status} ${text}`);
  }
  return data;
}

function pageInput(input: { limit?: number; offset?: number }, defaultLimit = 20) {
  return { limit: input.limit ?? defaultLimit, offset: input.offset ?? 0 };
}

async function callJson(service: string, method: string, body: Record<string, unknown>) {
  return jsonText(await callConnect(service, method, body));
}

server.registerTool('whoami', {
  title: 'Who am I',
  description: 'Return current MCP backend target and whether an access token is configured.',
  inputSchema: {},
}, async () => jsonText({ baseUrl, authenticated: token.length > 0 }));

server.registerTool('list_accounts', {
  title: 'List accounts',
  description: 'List accounts visible to the authenticated user.',
  inputSchema: {},
}, async () => callJson(services.account, 'ListAccounts', {}));

server.registerTool('list_symbols', {
  title: 'List symbols',
  description: 'List market symbols for an account.',
  inputSchema: { accountId: z.string() },
}, async ({ accountId }) => callJson(services.market, 'GetSymbols', { accountId }));

server.registerTool('get_quotes', {
  title: 'Get quotes',
  description: 'Get quotes for symbols through backend MarketService.',
  inputSchema: { accountId: z.string(), symbols: z.array(z.string()) },
}, async ({ accountId, symbols }) => callJson(services.market, 'GetQuotes', { accountId, symbols }));

server.registerTool('list_templates', {
  title: 'List strategy templates',
  description: 'List strategy templates visible to the user.',
  inputSchema: {},
}, async () => callJson(services.strategy, 'ListTemplates', {}));

server.registerTool('get_template', {
  title: 'Get strategy template',
  description: 'Get a single strategy template by ID.',
  inputSchema: { id: z.string() },
}, async ({ id }) => callJson(services.strategy, 'GetTemplate', { id }));

server.registerTool('submit_backtest', {
  title: 'Submit backtest',
  description: 'Submit a backend-authorized template backtest job.',
  inputSchema: {
    accountId: z.string(),
    templateId: z.string(),
    symbol: z.string(),
    timeframe: z.string(),
    initialCapital: z.number().optional(),
  },
}, async input => callJson(services.strategy, 'RunTemplateBacktest', {
  accountId: input.accountId,
  templateId: input.templateId,
  symbol: input.symbol,
  timeframe: input.timeframe,
  initialCapital: input.initialCapital ?? 10000,
}));

server.registerTool('get_backtest_job', {
  title: 'Get job',
  description: 'Get a generic job by ID.',
  inputSchema: { jobId: z.string() },
}, async ({ jobId }) => callJson(services.job, 'GetJob', { jobId }));

server.registerTool('detect_market_regime', {
  title: 'Detect market regime',
  description: 'Detect market regime through backend MarketRegimeService.',
  inputSchema: { accountId: z.string(), symbol: z.string(), timeframe: z.string(), count: z.number().optional() },
}, async input => callJson(services.marketRegime, 'DetectMarketRegime', {
  accountId: input.accountId,
  symbol: input.symbol,
  timeframe: input.timeframe,
  count: input.count ?? 120,
}));

server.registerTool('submit_strategy_experiment', {
  title: 'Submit strategy experiment',
  description: 'Submit a strategy parameter experiment through backend StrategyExperimentService.',
  inputSchema: {
    baseTemplateId: z.string(),
    parameterSpace: z.record(z.string(), z.unknown()),
    searchMethod: z.string().optional(),
    maxCandidates: z.number().optional(),
    objective: z.string().optional(),
  },
}, async input => callJson(services.strategyExperiment, 'SubmitStrategyExperiment', {
  baseTemplateId: input.baseTemplateId,
  parameterSpace: input.parameterSpace,
  searchMethod: input.searchMethod ?? 'grid',
  maxCandidates: input.maxCandidates ?? 12,
  objective: input.objective ?? 'balanced',
  idempotencyKey: `mcp-${Date.now()}`,
}));

server.registerTool('list_strategy_experiments', {
  title: 'List strategy experiments',
  description: 'List strategy experiments visible to the authenticated user.',
  inputSchema: { limit: z.number().optional(), offset: z.number().optional() },
}, async input => callJson(services.strategyExperiment, 'ListStrategyExperiments', pageInput(input)));

server.registerTool('list_experiment_candidates', {
  title: 'List experiment candidates',
  description: 'List candidates for a strategy experiment.',
  inputSchema: { experimentId: z.string() },
}, async input => callJson(services.strategyExperiment, 'ListExperimentCandidates', {
  experimentId: input.experimentId,
}));

server.registerTool('list_strategy_assets', {
  title: 'List strategy assets',
  description: 'List strategy assets visible to the authenticated user.',
  inputSchema: { limit: z.number().optional(), offset: z.number().optional() },
}, async input => callJson(services.strategyAsset, 'ListStrategyAssets', pageInput(input)));

server.registerTool('get_strategy_asset', {
  title: 'Get strategy asset',
  description: 'Get a strategy asset by ID.',
  inputSchema: { assetId: z.string() },
}, async input => callJson(services.strategyAsset, 'GetStrategyAsset', {
  assetId: input.assetId,
}));

server.registerTool('clone_strategy_asset', {
  title: 'Clone strategy asset',
  description: 'Clone a strategy asset to a user-owned strategy template draft.',
  inputSchema: { assetId: z.string(), name: z.string().optional() },
}, async input => callJson(services.strategyAsset, 'CloneStrategyAsset', {
  assetId: input.assetId,
  name: input.name ?? 'MCP cloned strategy asset',
}));

await server.connect(new StdioServerTransport());
