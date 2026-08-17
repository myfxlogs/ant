# Builder Handoff — STREAM-FREEZE-1（SSE 数据流冻结自愈）

> 日期：2026-08-17 ｜ 审计方：Claude Code ｜ 施工方：Windsurf
> 根因与证据链见 `docs/audits/tech-debt-registry.md` 条目 `STREAM-FREEZE-1`（必读）。

## 0. 背景一句话

用户报告：浏览器跑一段时间后数据流冻结（无最新报价、无持仓更新），状态仍显示"已连接"，必须手动刷新才恢复。审计方已定论根因 = **前端 SSE 客户端无 stale 检测，半开/僵尸连接（睡眠/唤醒、网络切换、NAT 过期、CF 隧道静默重置）既不 yield 也不 reject → 永不重连**。后端 15s ping 一直在发，前端只是把它当普通事件丢弃，没有转成"连接健康度"信号。

## 1. 任务（5 个，一个 scope：SSE 流自愈）

> **用户业务验收标准（2026-08-17 用户确认，最高优先）**：账户 connected → 页面数据必须永远活着；任何网络扰动（睡眠/唤醒、切换网络、CF 重置、后端重启）后**自动恢复、零手动刷新**；恢复期间 UI 诚实显示"重连中"，绝不显示假的"已连接"+冻结数据。物理边界：网络真断时无数据可推，承诺的是"检测 ≤45s（online 事件触发时秒级）+ 自动恢复 + 诚实状态"。

### Task A0（基础设施）— 共享 stream watchdog helper

**新建** `frontend/src/client/streamWatchdog.ts`：

```ts
export interface StreamWatchdog {
  touch(): void;        // 收到任何事件（含心跳）时调用
  start(): void;
  stop(): void;         // 组件卸载/isAborted 时清理，零泄漏
  checkNow(): void;     // 立即执行一次 staleness 检查（online/visibilitychange 用）
}
export function createStreamWatchdog(opts: {
  staleThresholdMs: number;
  onStale: () => void;          // 触发方：abort 当前流 + 重连 + 上报 UI 状态
  checkIntervalMs?: number;     // 默认 5000
}): StreamWatchdog;
```

实现要点：
- 内部 `lastEventAt`，`touch()` 刷新；interval 检查 `Date.now() - lastEventAt > staleThresholdMs` → `onStale()`。
- `start()` 时重置 `lastEventAt`（新连接宽限 = 阈值本身，防刚建立就误杀）。
- **全局事件钩子**（helper 内注册一次/模块级，或导出 `armGlobalRecheck(fn)`）：`window.addEventListener('online', ...)` 和 `document.addEventListener('visibilitychange', ...)`（visible 时）→ 调用所有活跃 watchdog 的 `checkNow()`。理由：睡眠唤醒/网络恢复时浏览器发 `online`，秒级触发重连，不等 45s 周期；后台 tab timer 冻结，回前台补检。
- `onStale` 触发时 `console.warn('[stream] stale detected, forcing reconnect')`（观测性）。

### Task A（P1 核心）— 三处流客户端接 watchdog

**位置与阈值**：
- `frontend/src/client/stream.ts` `subscribeEvents`（:59）→ 45_000（3 × 后端 15s ping）
- `frontend/src/client/stream.ts` `subscribeUserSummary`（:140）→ 90_000（3 × 30s keepalive）
- `frontend/src/client/sharedStream.ts` `startSharedStream`（:20）→ Task B 上线后 45_000；B 未上前 120_000 防误杀（当前后端无心跳）

**接入方式**：`for await` 每个事件 `watchdog.touch()`（含 ping/heartbeat——它们是健康度信号）；`onStale` → `abortController.abort()` + 立即 `runStream(0)`（**复用现有重连路径，勿另写第二套**）；同时通过 `onError`/`onStale` 回调通知 StreamProvider。

### Task A2（P1）— Live 页 watchActive 接 watchdog（用户点名场景）

**位置**：`frontend/src/pages/strategy/LiveStrategyPage.tsx:50-74`（`watchActive` while 循环）

**问题**：后端 20s heartbeat（LIVE-SSE-HEARTBEAT 已做）前端 `if (event.heartbeat) continue` 只跳过不利用——**无 stale 检测**，僵尸连接时 `for await` 永挂 → while 卡死永不重连 → `streamError` 不触发 → UI 连"重连中"横幅（:196）都不显示，纯静默冻结。**用户"策略在服务器跑着但前端显示掉了"即此流**。

**修复**：同 Task A 接 watchdog（阈值 60_000 = 3 × 20s 心跳）；heartbeat 事件也 `touch()`；`onStale` → `ctrl.abort()`（现 while 循环的 catch/重连路径自动接管）+ `setStreamError(true)`（诚实显示横幅）。
**顺手统一（可选但推荐）**：`:97` `watchSchedules` 的 90s 无条件硬重置（`Promise.race` + 90s timer）替换为同款 watchdog（后端有 NOTIFY 推送；若后端无心跳则保留硬重置并注释原因）——watchdog 是硬重置的严格改进（健康流不断连）。若不动，红队自审须回答为何保留双模式。

### Task B（P2）— 后端补两个流的心跳

**位置**：`backend/internal/connect/system/stream_handlers_extra.go`
- `SubscribeOrderUpdates`（:20）— select 循环加 15s ticker 发空事件（`&antv1.OrderUpdateEvent{}`）
- `SubscribeProfitUpdates`（:68）— 同上，发 `&antv1.ProfitUpdateEvent{}`

**先例（照抄模式）**：`runSummaryKeepaliveOnly`（同文件 :144-157）+ `runEventLoop` 的 keepalive case（`stream_handler.go:140/169`）。
**前端兼容**：心跳空事件到达消费端（`bridgeStreamEvents`/`bridgeProfitEvents`）必须跳过不写 store（检查 `accountId` 或关键字段为空即 return）。sharedStream 路径同样处理。
**测试注入**：镜像 `heartbeatInterval` 可注入字段模式（先例：`strategy_active_handlers.go` WatchActiveStrategies，registry LIVE-SSE-HEARTBEAT 条目）。

### Task C（P3）— StreamProvider 清理顺序 + 诚实状态

**位置**：`frontend/src/providers/StreamProvider.tsx:110-114`

1. `onError` 回调里 `unsubEventsRef.current = null` **之前**先调用 `unsubEventsRef.current?.()`——否则 cap 前挂起的 backoff `setTimeout` 可能在 Provider 已建新流后复活旧流 → 双流重复事件。summary 的 unsubscribe 顺序同样确认。
2. **诚实状态（信任关键）**：新增 `onStale` 通道（stream.ts 回调），watchdog 触发时立即 `setConnectionState('connecting')`——恢复期间用户看到"重连中"，而不是停在 'connected' 的假象。当前 `isConnected` 只由收事件置 true，僵尸时永不回落。


### Task A（P1 核心）— 前端 stale watchdog

**位置**：
- `frontend/src/client/stream.ts` — `subscribeEvents`（:59-112）、`subscribeUserSummary`（:140-193）
- `frontend/src/client/sharedStream.ts` — `startSharedStream`（:20-77）

**要求**：
1. 每个 runStream 实例维护 `lastEventAt = Date.now()`；`for await` 每收到一个事件（**含 ping**）就刷新。
2. watchdog `setInterval`（5s 检查周期，与现有重连 timer 独立）：`Date.now() - lastEventAt > staleThreshold` →
   - `console.warn('[stream] stale detected, forcing reconnect')`
   - `abortController.abort()` + 立即 `runStream(0)`（复用现有重连路径，勿另写第二套重连）
   - 通过 `onError`/`onStale` 通知上层（StreamProvider 置 `connectionState='connecting'`）
3. 阈值常量集中定义（文件顶部，带注释说明依据）：
   - `subscribeEvents`：45_000（3 × 后端 15s ping）
   - `subscribeUserSummary`：90_000（3 × 30s keepalive，见 `stream_handlers_extra.go:180`）
   - sharedStream（Task B 完成后）：45_000；B 未上线前用 120_000 防误杀（当前后端无心跳）
4. `visibilitychange` → `document.visibilityState === 'visible'` 时立即执行一次 staleness 检查（后台 tab timer 被冻结，回前台补检）。
5. 组件卸载/isAborted 时清理 watchdog interval（零泄漏）。
6. **注意**：连接刚建立、尚未收到首事件前，watchdog 不得立即触发（建立时重置 `lastEventAt`，宽限 = 阈值本身即可）。

### Task B（P2）— 后端补两个流的心跳

**位置**：`backend/internal/connect/system/stream_handlers_extra.go`
- `SubscribeOrderUpdates`（:20）— select 循环加 15s ticker 发空事件（`&antv1.OrderUpdateEvent{}`）
- `SubscribeProfitUpdates`（:68）— 同上，发 `&antv1.ProfitUpdateEvent{}`

**先例（照抄模式）**：`runSummaryKeepaliveOnly`（同文件 :144-157）+ `runEventLoop` 的 keepalive case（`stream_handler.go:140/169`）。
**前端兼容**：心跳空事件到达 `dispatchStreamEvent` 后 payload case 匹配但字段全零——消费端（`bridgeStreamEvents`/`bridgeProfitEvents`）必须跳过空事件不写入 store（检查 `accountId` 或关键字段为空即 return）。sharedStream 路径同样处理。
**测试注入**：镜像 `heartbeatInterval` 可注入字段模式（先例：`strategy_active_handlers.go` WatchActiveStrategies，registry LIVE-SSE-HEARTBEAT 条目）。

### Task C（P3）— StreamProvider 清理顺序

**位置**：`frontend/src/providers/StreamProvider.tsx:110-114`

`onError` 回调里 `unsubEventsRef.current = null` **之前**先调用 `unsubEventsRef.current?.()`——否则 cap 前挂起的 backoff `setTimeout` 可能在 Provider 已建新流后复活旧流 → 双流重复事件。同时确认 summary 的 unsubscribe 顺序一致。

## 2. Reuse Preflight 结论（已由审计方执行）

- `REUSE: runSummaryKeepaliveOnly 心跳模式 @ backend/internal/connect/system/stream_handlers_extra.go:144`
- `REUSE: runEventLoop keepalive ticker @ backend/internal/connect/system/stream_handler.go:140`
- `REUSE: heartbeatInterval 可注入测试字段 @ backend/internal/connect/strategy/strategy_active_handlers.go（LIVE-SSE-HEARTBEAT 先例）`
- `REUSE: isLikelyStreamTransportFailure / 重连 backoff @ frontend/src/client/stream.ts:83`
- `REUSE: Live 页 watchActive 心跳事件处理（`if (event.heartbeat) continue`）@ frontend/src/pages/strategy/LiveStrategyPage.tsx:59`
- `NEW: 前端 stale watchdog helper（已搜 watchdog/stale/heartbeat/keepalive，cap.sh + 代码层均无命中）` → Task A0 新建 `streamWatchdog.ts`，Task A/A2 各消费方复用同一 helper（**勿三处各写一套**）

## 3. 对抗证明（必做，断言级）

1. **Task A0**：单测 helper——fake timer 推进 > 阈值无 touch → onStale 触发；touch 后推进 → 不触发；stop 后 → 不触发（零泄漏）。
2. **Task A**：测试用 fake timer 推进 >45s 无事件 → 断言 abort 被调用 + runStream 重启（如重新调 subscribeEvents factory 计数 >1）。**删 watchdog 检查行 → 测试 RED**（流死不恢复）。
3. **Task A2**：同模式测 watchActive（mock stream 挂起无事件 → 推进 >60s → 断言 ctrl.abort 被调用 + 循环重连 + `streamError` 置 true）。**删 watchdog 接入 → RED**。
4. **Task B**：heartbeatInterval 注入 100ms，mock stream → 2s 内收到空事件 GREEN；**删 ticker case → RED**（超时无心跳）。
5. **Task C**：触发 onError 后断言旧 unsubscribe 被调用（mock 计数）。

## 4. 显式红队自审（施工方完工前必答，写入回填）

1. watchdog 误杀：慢网络下 ping 延迟 >阈值（45s）的容忍度（阈值是否需放宽/自适应）？
2. 重连风暴：watchdog 触发 + backoff 重连叠加，是否会造成并发双连接？（abort 必须先于新连接建立）
3. `online`/`visibilitychange` 全局钩子：多 watchdog 并发 checkNow 的幂等；非浏览器环境（SSR/test）的守卫。
4. 空心跳事件被 store 消费的路径逐一排查（positionsMap 写入、profit patch、TanStack setQueryData、Live 页 schedules/activeStrategies）？
5. Task B 空 `OrderUpdateEvent{}` 经过 `toCamelCase` 后是否触发前端 switch case 的意外分支？
6. watchdog interval 与 isAborted 竞态（清理时正在触发）？
7. subscribeEvents 的 `transportFailStreak` 与 watchdog 触发的交互（watchdog 重连算不算 transport failure）？
8. Live 页 watchActive：watchdog 触发 abort 后，现 while 循环的 catch 是否按预期接管并走 2s 重连？（不要把 watchdog 重连做成第三套逻辑）
9. 后台 tab：长时间切后台再回前台，`online` 与 `visibilitychange` 可能同时触发 → 是否重复重连？
10. StreamProvider `onStale` → `setConnectionState('connecting')` 后，若重连成功收首事件 → 恢复 'connected'，两条路径的状态机是否闭合？

## 5. 验收标准（审计方实测）

1. **复现自愈（账户页）**：开账户详情页 → devtools Network offline ≥60s（或系统睡眠唤醒）→ 恢复 → **不刷新页面**，报价/持仓数据 ≤60s 内自动恢复且 `connectionState` 先显示重连再恢复 connected。若走 `online` 事件路径，恢复应更快（秒级）。
2. **复现自愈（Live 页，用户点名场景）**：Live 页开策略 tab → offline 或睡眠唤醒 → 恢复 → **不刷新**，策略列表/状态恢复更新，期间显示"Connection interrupted, reconnecting…"横幅（`streamError`），**绝不静默冻结**。
3. 对抗证明独立删行复测（5/5 RED）。
4. 门禁：`tsc --noEmit` 0err / vitest 全绿 / `npm run build` 绿 / `go build ./...` 绿 / `go run ./tools/check-file-lines --strict` 0 ERROR。
5. 部署：后端 `docker compose build backend && docker compose up -d backend`；前端 `docker cp frontend/dist/. alphaforge-frontend:/usr/share/nginx/html/ && docker exec alphaforge-frontend nginx -s reload`。**部署验证到入口响应头层**（QC-CACHE-LEAK 教训）。

## 6. 回填纪律（不做 = 任务判失败）

1. registry `STREAM-FREEZE-1`：`🟦open → ✅done`（标日期）+ 真实根因/修复/对抗证明/测试结果。
2. `handover-audit-plan.md` 变更日志加一行。
3. 不自行宣告完成——等审计方核对 + 实测。

## 7. 范围约束

One task = one scope：只动 SSE 自愈链——`frontend/src/client/streamWatchdog.ts`（新建）、`stream.ts`、`sharedStream.ts`、`StreamProvider.tsx`、`LiveStrategyPage.tsx`、`backend/internal/connect/system/stream_handlers_extra.go` + 对应测试。不顺手重构、不改无关重连逻辑、不动 broker/handler 业务语义。

## 8. 一句话开工版（给用户的转述）

前端 SSE 流缺"僵尸连接"自愈：账户页 + Live 页共 5 处流都加统一 watchdog（>N 秒无事件强制重连，网络恢复秒级触发）+ 后端 2 个流补心跳 + 断连时 UI 诚实显示"重连中"——账户在线数据永远保活，网络波动自动恢复，无需刷新。
