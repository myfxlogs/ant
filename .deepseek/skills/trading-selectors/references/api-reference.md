# API Reference

## Account List

**Endpoint**: `POST /antrader.ResearchDatasetService/ListAccounts`

Uses the research client:
```ts
import { researchClient } from '@/modules/research/client/research';
const r = await researchClient.dataset.listAccounts({});
```

### Response (`ListResearchAccountsResponse`)

Proto: `antrader.ListResearchAccountsResponse`

| Field | Type | Description |
|-------|------|-------------|
| `accounts` | `AccountSummary[]` | List of user's accounts |

### AccountSummary

Proto: `antrader.AccountSummary`

| Field | Type | Description |
|-------|------|-------------|
| `accountId` | `string` | UUID of the trading account |
| `broker` | `string` | Broker name (e.g. "Exness Technologies Ltd") |
| `accountNumber` | `string` | Account login number |
| `isDisabled` | `bool` | Whether the account is disabled |
| `isConnected` | `bool` | Whether the account is currently connected |

Only show non-disabled accounts (`!a.isDisabled`).

## Symbol List (Live Broker)

**Method**: `marketApi.getSymbols(accountId: string)`

```ts
import { marketApi } from '@/client/market';
const list = await marketApi.getSymbols(accountId);
```

This calls `MarketService.GetSymbols` which connects to the broker (MT4/MT5)
via the account's active connection to retrieve available trading instruments.

### Return Type: `SymbolInfo[]`

```ts
interface SymbolInfo {
  symbol: string;
  description?: string;
  currency?: string;
  digits?: number;
  tickSize?: number;
  tickValue?: number;
  contractSize?: number;
  minLot?: number;
  maxLot?: number;
  lotStep?: number;
}
```

### Processing for Select Options

```ts
const seen = new Set<string>();
const opts = (list || [])
  .map((s: any) => String(s?.symbol || '').trim())
  .filter((v: string) => v)
  .filter((v: string) => { if (seen.has(v)) return false; seen.add(v); return true; })
  .map((v: string) => ({ value: v, label: v }));
```

This deduplicates symbols and converts to Ant Design Select `options` format.

## Symbol List (Database - DO NOT USE for pickers)

**Endpoint**: `POST /antrader.ResearchDatasetService/ListSymbols`

```
⚠️ DO NOT use this for user-facing symbol pickers.
This queries `kline_data` (static historical cache), not live broker data.
```

Request: `{ accountId: string }`
Response: `{ symbols: string[] }`

## Build Dataset

**Endpoint**: `POST /antrader.ResearchDatasetService/BuildDataset`

```ts
const now = new Date();
const yearAgo = new Date(Date.now() - 365 * 86400 * 1000);
await researchClient.dataset.buildDataset({
  accountId: string,       // UUID
  kind: 1 | 2 | 3,        // SINGLE=1, PANEL=2, UNIVERSE=3
  symbol: string,          // Trading symbol (for SINGLE)
  primaryTf: string,       // e.g. "H1"
  from: { $type: 'google.protobuf.Timestamp', seconds: BigInt(...), nanos: 0 },
  to: { $type: 'google.protobuf.Timestamp', seconds: BigInt(...), nanos: 0 },
  minBars: number,
  canonicalClock: 'utc',
});
```

### Optional
- `symbols?: string[]` — for PANEL datasets
- `universeId?: string` — for UNIVERSE datasets
- `secondaryTfs?: string[]`
- `hypothesisId?: string`
- `costProfile?: CostProfile`
