# K线获取：账户感知（broker 优先）落地方案

生成时间: 2026-06-01
问题: 用户有多个交易账号，每个账号 broker/symbol/K线不同。切换账号、切换 symbol 时拿不到 K 线。
结论: **后端整条"按账户会话向 broker 实时取 K 线"的链路已实现，前端接错了 RPC。** 核心修复为前端改调正确 RPC，后端零改动；可选补一个分页字段。
适用对象: 交给 AI 按步骤落地。

---

## 0. 根因（已核实）

存在两条 K 线路径，前端用错了那条。

### 路径 A — 前端当前在用（ClickHouse 单一数据源，账户无关）
```
marketApi.getKlines → marketClient.getKlines → MarketService.GetKlines
   → MarketDataRepository.GetKlines（ClickHouse）
```
- 文件: `frontend/src/client/market.ts` `getKlines`；`backend/internal/connect/marketplace/market_handler.go:36`
- **`GetKlinesRequest` proto 没有 account_id 字段**（仅 canonical/broker/period/from/to/limit）。
- 后果:
  - 切换交易账号 → 后端完全忽略，K 线不变。
  - ClickHouse 只含 tick 管道预采集过的品种 → 未采集的 symbol = 空图。

### 路径 B — 已实现但前端从未调用（按账户会话向 broker 取，broker 优先 + ClickHouse 兜底）
```
tradingClient.priceHistory → MtHubServer.PriceHistory
   → MtHubService.PriceHistory → exec.FetchPriceHistory
      → MT4: QuoteHistory RPC   (backend/internal/mdgateway/adapter/mt4/orders.go:301，真实现)
      → MT5: PriceHistory RPC   (backend/internal/mdgateway/adapter/mt5/orders.go:345，真实现)
```
- 文件: `backend/internal/connect/system/mthub_service_extra.go:37`（`PriceHistory` handler）
- `PriceHistoryRequest` 有 `account_id / canonical / period / limit`。
- 生成的 TS 客户端 `priceHistory` 已存在于 `frontend/src/gen/ant/v1/mthub_service_pb.ts`，但 `client/market.ts` **从未调用**。
- handler 逻辑: `account_id` 非空且属于该用户 → 走 broker 会话取；为空或取到 0 条 → 回落 ClickHouse。

> 用户的真实场景（账号各自 broker、symbol、K线不同）正是路径 B 的设计目的：用该账号自己的 broker 会话取，broker 有什么 symbol 就能取，取不到再兜底。

### 周期格式无需映射（已核实）
后端 `periodSeconds()`（`mthub_service_extra.go:23`）接受 `1m/5m/15m/30m/1h/4h/1d/1w`，与前端 `timeframe` 完全一致；broker 侧 `M1/H1` 转换由适配器 `periodToMT4TF()` 内部完成。前端无需改动周期字符串。

---

## 1. 落地步骤

### 步骤 1（必做，核心修复）：前端 `getKlines` 切到 `priceHistory`

文件: `frontend/src/client/market.ts`

将现有 `getKlines`（调 `marketClient.getKlines`，尝试 symbol 变体）替换为调用账户感知的 `priceHistory`：

```ts
import { marketClient, tradingClient } from './connect';
import { create } from '@bufbuild/protobuf';
import { PriceHistoryRequestSchema } from '@/gen/ant/v1/mthub_service_pb';
import type { OHLCV } from '../gen/ant/v1/mthub_service_pb';
import type { Timestamp } from '@bufbuild/protobuf/wkt';

// getKlines 改为账户感知（broker 优先 + 后端 ClickHouse 兜底）。
getKlines: async (params: {
  symbol: string;
  timeframe: string;
  count?: number;
  before?: number;       // 见步骤 2：向左加载更多
  accountId?: string;
}): Promise<KlineData[]> => {
  // 账户感知是前提：没有 accountId 时无法定位 broker 会话。
  if (!params.accountId) return [];

  const req: Record<string, unknown> = {
    accountId: params.accountId,
    canonical: params.symbol,
    period: params.timeframe,           // 格式天然对齐
    limit: params.count ?? 300,
  };
  if (params.before) {
    // 步骤 2 落地后启用：取该时间点之前的更早 bar。
    req.to = { seconds: BigInt(params.before), nanos: 0 };
  }

  const resp: any = await tradingClient.priceHistory(
    create(PriceHistoryRequestSchema, req),
  );
  const bars = (resp.bars || []) as OHLCV[];
  return bars.map((bar) => ({
    time: toUnixSeconds(bar.openTime),
    open: Number(bar.open ?? '0'),
    high: Number(bar.high ?? '0'),
    low: Number(bar.low ?? '0'),
    close: Number(bar.close ?? '0'),
    volume: Number(bar.volume ?? 0),
  }));
},
```

注意:
- `PriceChart.tsx` 已经在传 `accountId` 给 `marketApi.getKlines`（见 `PriceChart.tsx:74,182`），无需改图表组件的调用签名。
- `resolveSymbol` 仍为 passthrough；broker symbol 直接使用。
- 路径 A 的 symbol 后缀变体（`m`、`.`）逻辑可移除——broker 返回的就是该 broker 的真实 symbol，由 `SymbolList`/`SymbolPicker` 提供，不需要猜测变体。

### 步骤 2（必做，保住"向左滚动加载更多"）：给 PriceHistory 增加 before/to 分页

现状: `PriceHistoryRequest` 只有 `limit`，handler 用 `from = now - limit*periodSeconds` 算窗口，**无 before/to**，因此 `PriceChart` 的 load-more（向左滚动加载更早 bar）会失效。

改动点:

1. proto: 在 `PriceHistoryRequest` 增加可选 `to`（向后兼容）。定位包含该 message 的 proto 文件（`mthub_service.proto`，对应 `gen/ant/v1/mthub_service_pb.ts` 的 `PriceHistoryRequest`，当前字段 `account_id=1, canonical=2, period=3, limit=4`）：
```proto
message PriceHistoryRequest {
  string account_id = 1;
  string canonical = 2;
  string period = 3;
  int32 limit = 4;
  optional google.protobuf.Timestamp to = 5;   // 取此时间之前的 bar（向左分页）
}
```
2. `buf generate` 重新生成 Go + TS。
3. handler `MtHubServer.PriceHistory`（`mthub_service_extra.go:37`）：用 `to` 作为窗口右界。
```go
now := time.Now().Unix()
to := now
if m.To != nil {
    to = m.To.AsTime().Unix()
}
from := to - int64(limit)*periodSeconds(period)
bars, _ = s.svc.PriceHistory(ctx, m.AccountId, m.Canonical, period, from, to, int(limit))
```
   ClickHouse 兜落分支也用 `to` 传入 `GetKlines` 的时间上界，保持一致。
4. 适配器已支持 `from/to/count`（`FetchPriceHistory(ctx, symbol, period, from, to, count)`），无需改。

> 备选（不改 proto）: load-more 仍走路径 A（ClickHouse `marketClient.getKlines` 带 `to`），初始/实时走路径 B。代价是混用两套数据，**不推荐**（可能出现 broker 与 ClickHouse 价格/对齐不一致）。

### 步骤 3（必做，保证 broker 路径生效）：选中账户时确保已连接

broker 路径要求 `hub.Get(accountID)` 有活跃会话；未连接时后端**静默回落 ClickHouse**（可能为空）。

改动点: 在选择账户的入口（如 workspace `handleAccountChange`、`PriceChart` 所在页面的账户切换）确保触发 `accountApi.connect(accountId)`（或复用 `useAccount().connectAccount`）。可在切换后异步连接，连接成功再刷新 K 线。

验收口径: 选一个已绑定但未连接的账户 → 自动连接 → 图表显示该 broker 的 K 线。

---

## 2. 验收标准

- **切账号**: 在图表/workspace 切换不同 broker 的交易账号，K 线随之变为该账号 broker 的数据（不再固定不变）。
- **切 symbol**: 选择该账号 broker 提供的任意 symbol（含 ClickHouse 未预采集的），都能显示 K 线。
- **未连接兜底**: 账号未连接时不报错，回落 ClickHouse（可能为空，配合步骤 3 自动连接）。
- **加载更多**: 向左滚动可加载更早的 bar（依赖步骤 2）。
- **多周期**: 1m/5m/15m/30m/1h/4h/1d/1w 均正常。

---

## 3. 影响面与风险

- **影响组件**: `PriceChart.tsx` 被 `Market`、`Trading`、`StrategyWorkspace`(Chart Tab) 共用。切换数据源对三处同时生效——需三处都回归验证。
- **性能**: 5s 轮询改为打 broker 会话；建议轮询保持小 `count`，后续可改用 tick SSE 合成最新 bar 替代轮询。
- **broker 限频**: 频繁 QuoteHistory/PriceHistory 可能受 broker 限频；可在后端对 (account,symbol,period) 做短时缓存（如最新窗口缓存 3-5s）。
- **proto 向后兼容**: 步骤 2 新增 `to` 为 optional，不破坏既有调用。

---

## 4. 文件改动清单（速查）

| 步骤 | 文件 | 改动 |
|------|------|------|
| 1 | `frontend/src/client/market.ts` | `getKlines` 改调 `tradingClient.priceHistory`，移除 symbol 变体猜测 |
| 2 | `proto .../mthub_service.proto` | `PriceHistoryRequest` 增 `optional Timestamp to = 5`；`buf generate` |
| 2 | `backend/internal/connect/system/mthub_service_extra.go` | `PriceHistory` 用 `to` 作窗口右界（broker + ClickHouse 兜底分支） |
| 3 | 账户选择入口（`StrategyWorkspacePage.handleAccountChange` 等） | 切账户时触发 `connectAccount` |

---

## 5. 关键事实备注（供实施者核对，避免走错路）

- 不要去新实现 MT 的 QuoteHistory/PriceHistory——**已实现**（mt4/orders.go:301、mt5/orders.go:345）。
- 不要给 `MarketService.GetKlines` 加 account_id——那条是 ClickHouse 路径，正确的账户感知 RPC 是 `MtHubService.PriceHistory`。
- 周期字符串不需要前后端映射层——后端与适配器已处理。
- invariant #8: 取 K 线经 mthub → adapter，禁止前端或业务层直调 mt4client/mt5client。
