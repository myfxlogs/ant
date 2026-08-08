# KB-P0 质量加固 spec（3 项非阻断跟进）

> **背景**：KB-P0 已 ✅（部署+seed+接线+测试全过，审计方二审实测）。本 spec 是 KB-P0 复审时识别的 3 个**非阻断加固点**（好→更好），不阻塞主线。
> **关联**：`kb-p0-consolidate-spec.md`（P0 主体✅）/ `mql-pipeline-completion-spec.md`（K3/T5 来源）/ `knowledge-base-architecture.md`。
> **角色**：审计方出 spec；施工方实现+回填，不自行宣告 ✅，等审计方实测。

---

## 1. K3 DemandRecorder 传真实 userID（需求信号去重）

**问题**：`marketplace/quality.go:168` `s.demandRecorder.RecordDemandSignal(ctx, bs.Id, uuid.Nil)`——userID 用 `uuid.Nil` placeholder。但发布是认证作者调用，**ctx 里有真实 userID**（`interceptor.GetUserID(ctx)`）。uuid.Nil → `kb_demand_signal.user_count` 无法去重（= hit_count，膨胀）→ 需求排序失真（1 用户踩 100 次 ≠ 100 用户各 1 次）。

**任务**：`RecordDemandSignal` 调用前从 ctx 取真实 userID：`uid := interceptor.GetUserID(ctx)`（取不到→`uuid.Nil` 兜底，保留 graceful）。
- REUSE：`interceptor.GetUserID`（项目标准鉴权取 userID）。
- **对抗证明**：同用户对同一不支持函数触发 2 次 demand → `user_count=1`（distinct）；用 uuid.Nil 则 =2（错）。删去重→必红。

## 2. T5 CoverageChecker nil 时 Warn（安全门别静默 fail-open）

**问题**：`live_runner.go:129` `if cfg.Mode == "live" && s.coverageChecker != nil`——paper 跳过✓、clean 放行✓，但 **coverageChecker==nil + live 时静默跳过**（fail-open 无提示）。对安全门（防不可靠策略上真账户），silent fail-open 有风险：checker 漏注入→门失效且无信号。

**任务**：`mode=="live" && s.coverageChecker == nil` 分支加 `s.log.Warn("live coverage gate skipped: coverage checker not injected")`。fail-open 本身保留（项目优雅降级模式），但不能静默。
- **对抗证明**：nil checker + live 启动 → 日志含 warn；删 warn → 无日志（测试断言日志/或改 testdouble 验证）。

## 3. C1 复利 e2e 测试（覆盖真 PG LISTEN/NOTIFY 投递）

**问题**：`service_test.go` 的 C1 测试用 mock `loadFromDB`（覆盖逻辑：notify→refresh 接线），但**真 PG LISTEN/NOTIFY 投递未测**。LISTEN 掉线/channel 名错/连接断→缓存 stale，unit 测抓不到（silent break）。

**任务**：新增 `//go:build integration` e2e 测试——`RecordFact(新常量)` → **轮询等缓存刷新（超时，如 2s）** → 断言 `LookupConstant(新常量)` 命中（证真 NOTIFY→LISTEN→loadFromDB→cache 全链）。
- REUSE：现有 integration 测试模式（`//go:build integration`，真 PG）。
- **对抗证明**：故意破坏 NOTIFY（如改 channel 名/不发 notify）→ 缓存不刷新 → 轮询超时 → 测试红（证 e2e 真覆盖投递链）。

---

## 验收（审计方实测）
- **K3**：同用户 2 次触发 → user_count=1；对抗证明。
- **T5**：nil checker + live → warn 日志在。
- **C1 e2e**：integration 测试 RecordFact→轮询→LookupConstant 命中；破坏 NOTIFY→超时红。

## 完工回填（施工方）
1. `tech-debt-registry.md`：新增 `KB-HARDEN-1/2/3` 🟦→✅ + 对抗证明。
2. `handover-audit-plan.md` 变更日志。
3. 不自行宣告 ✅——等审计方实测。

---

> **审计方注**：3 项都不阻断 KB-P0 ✅，是质量加固（K3 信号准/T5 安全可见/C1 防silent-stale）。可与 K4(agent-RAG P3) 并行或在其前做。优先级：T5 Warn（安全可见，最小改动）> K3 userID（信号质量）> C1 e2e（耐久）。
