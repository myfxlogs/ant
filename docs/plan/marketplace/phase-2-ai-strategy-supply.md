> ⚠️ 已迁移至 docs/blocks/strategy-marketplace/plans/phase-2-ai-strategy-supply.md。此文件保留为兼容旧引用。

# Phase 2 · AI 策略供给 · 落地排期清单

> 权威依据：`docs/roadmaps/strategy-marketplace.md` Phase 2 + ADR-0024（Agent-Native 策略平台）
> 冲突时以本文 + ADR-0024 为准。
> 本清单串联 4 个子系统：agent-engine → mql-compiler → backtest-engine → marketplace。

## 0. 目标与边界（执行前必读）

- **达成即最优**：用户用自然语言描述策略需求 → AI 生成源码 → 编译 → 回测 → 达标自动上架。全流程 SSE 推送进度，失败时返回具体原因。
- **第一性原则**：
  - Agent 思考在 Python（pandas/numpy/optuna 生态），执行在 Go VM（性能差距 200x，per-bar RPC 不可行）。
  - 回测在 Agent 循环的内层——每次迭代只有一次 RPC 往返（提交源码 + 取回结果）。
  - 编译器（MQL → IR → Bytecode → VM）是确定性管道，不做"AI 直接写 Go 代码"的捷径（不可验证、不可回测）。
- **合规**：proto-only、decimal 计价、无 REST、文件行数、复用优先。

---

## 1. 依赖图

```
2.0 (Agent Gateway 协议) ──── 所有模块的前置
    │
    ├─→ 2.1 (AI 一键生成+上架) ──── 核心管线
    │       ├─→ 2.1a (GenerateStrategy RPC — SSE 进度)
    │       ├─→ 2.1b (Agent Gateway 串联)
    │       ├─→ 2.1c (前端 AutoGeneratePanel)
    │       └─→ 2.1d (上架前质量校验 — 复用 Phase 1.2)
    │
    ├─→ 2.2 (批量生成队列) ──── 依赖 2.1 管线
    │
    └─→ 2.3 (参数模板) ──── 依赖 2.1 管线
```

---

## 模块 2.0 · Agent Gateway 协议定义（全 Phase 2 前置）

**前置**：`bash scripts/cap.sh agent generate strategy sse stream` → 确认现有 agent-engine 能力。

### 协议设计

- [ ] **2.0a Agent 策略生成协议**

  **文件**：`proto/ant/v1/agent_strategy_generation.proto` (new)

  ```protobuf
  syntax = "proto3";
  package ant.v1;

  // ── Request: user describes what they want ──
  message GenerateAndPublishRequest {
    string description = 1;           // 自然语言需求描述
    string asset_class = 2;           // 目标资产类别
    repeated string symbols = 3;      // 目标品种
    string timeframe = 4;             // 目标时间周期
    string risk_level = 5;            // conservative / moderate / aggressive
    string strategy_type = 6;         // 可选：trend_following / mean_reversion / breakout / arbitrage / auto
    bool auto_publish = 7;            // 回测达标后是否自动上架（默认 true）
    string title_override = 8;        // 可选：自定义策略标题
    string language = 9;              // en / zh-cn
  }

  // ── Response: SSE stream ──
  message GenerateAndPublishEvent {
    string stage = 1;                 // 当前阶段
    // stage 枚举值:
    //   "generating"    — AI 正在生成策略源码
    //   "compiling"     — mql-compiler 编译中
    //   "backtesting"   — backtest-engine 回测中
    //   "evaluating"    — 质量门槛校验中
    //   "publishing"    — 上架中
    //   "completed"     — 完成，result 字段有内容
    //   "failed"        — 失败，error 字段有原因

    string message = 2;               // 用户可读的状态消息
    double progress = 3;              // 0.0 ~ 1.0 估算进度

    // populated at stage=completed
    string strategy_id = 4;           // 生成的策略 ID
    string publish_id = 5;            // 上架 ID（如果 auto_publish=true）
    BacktestSnapshot backtest = 6;    // 回测结果摘要

    // populated at stage=failed
    string error_stage = 7;           // 在哪一步失败的
    string error_detail = 8;          // 失败原因（中文/英文，取决于 language）
    bool retryable = 9;               // 用户是否可以重试
  }

  // Add to MarketplaceService:
  // rpc GenerateAndPublish(GenerateAndPublishRequest)
  //     returns (stream GenerateAndPublishEvent);
  ```

  **设计决策**：
  - 用单个 SSE stream RPC 承载全流程——每阶段推进一个 event，前端实时渲染进度条。
  - `error_stage` + `error_detail` 精确定位失败位置（生成/编译/回测/上架），用户知道该改什么。
  - `retryable` 标记——编译失败可重试（改需求描述），回测不达标不可重试（需要改策略逻辑）。

### Agent Gateway 接口定义

- [ ] **2.0b Agent Gateway 内部接口**

  **文件**：`backend/internal/connect/gateway/ai_gateway_handler.go`（已有，扩展）

  ```go
  // AgentGateway 是 Python Agent 层与 Go 编译执行层的桥梁。
  // 现有接口（保留）：
  //   - GenerateCode(description) → (source_code, language)
  //   - CompileAndBacktest(source_code, params) → (backtest_result)
  //
  // 新增接口：
  type AutoGenerateRequest struct {
      Description  string
      AssetClass   string
      Symbols      []string
      Timeframe    string
      RiskLevel    string
      StrategyType string // "" = auto
      Language     string
  }

  type AutoGenerateResult struct {
      StrategyID   string
      SourceCode   string
      Language     string  // "mql4" | "mql5" | "python"
      BacktestSnapshot *antv1.BacktestSnapshot
      QualityPassed bool
      QualityViolations []string
      DurationMs   int64
  }

  // AgentGateway.AutoGenerate(ctx, req) → (<-chan AutoGenerateProgress, error)
  type AutoGenerateProgress struct {
      Stage    string
      Message  string
      Progress float64
      Error    string  // non-empty = failed
      Result   *AutoGenerateResult
  }
  ```

  **架构决策 — 为什么不在 Python Agent 层做发布？**

  发布涉及 DB 写入、钱包操作、权限校验。这些必须在 Go API 层完成（Python Agent 层不持有 DB 连接、不持有钱包密钥）。所以发布步骤由 Go `marketplace_handler_autogen.go` 在收到 Agent Gateway 的 `completed` 结果后执行。

  ```
  Python Agent (策略生成)
      │ ConnectRPC
      ▼
  Go Agent Gateway (编译调度)
      │ in-process
      ▼
  mql-compiler (编译)
      │ in-process
      ▼
  backtest-engine (回测)
      │ in-process
      ▼
  Go marketplace handler (质量校验 + 发布)
  ```

- **Gate 2.0**：proto 编译通过 (`buf generate`) + `go build ./...`。

---

## 模块 2.1 · AI 一键生成 + 自动上架

### 后端 — RPC 实现

- [ ] **2.1a `GenerateAndPublish` SSE Handler**

  **文件**：`backend/internal/connect/marketplace/marketplace_handler_autogen.go` (new)

  **核心逻辑**（伪代码）：

  ```go
  func (s *MarketplaceServer) GenerateAndPublish(
      req *connect.Request[antv1.GenerateAndPublishRequest],
      stream *connect.ServerStream[antv1.GenerateAndPublishEvent],
  ) error {
      ctx := stream.Context()
      userID := auth.MustGetUserID(ctx)

      send := func(stage, msg string, progress float64) { ... }
      sendEvent := func(ev *antv1.GenerateAndPublishEvent) { ... }

      // Stage 1: AI 生成
      send("generating", "AI 正在生成策略代码...", 0.1)
      progressCh, err := s.agentGateway.AutoGenerate(ctx, AutoGenerateRequest{...})
      if err != nil {
          sendEvent(failedEvent("generating", err.Error(), true))
          return nil
      }

      // Stage 2-4: 实时转发 Agent Gateway 的进度
      var result *AutoGenerateResult
      for p := range progressCh {
          switch p.Stage {
          case "compiling":
              send("compiling", p.Message, 0.3+p.Progress*0.3)
          case "backtesting":
              send("backtesting", p.Message, 0.6+p.Progress*0.3)
          case "evaluating":
              send("evaluating", p.Message, 0.9)
          case "completed":
              result = p.Result
          case "failed":
              sendEvent(failedEvent(p.Stage, p.Error, isRetryable(p.Error)))
              return nil
          }
      }

      // Stage 5: 质量校验（Go 侧二次校验）
      send("evaluating", "正在校验回测质量...", 0.92)
      violations, _ := s.marketplace.ValidateBacktestQuality(ctx, result.BacktestSnapshot)
      if len(violations) > 0 && !req.AutoPublish {
          sendEvent(completedWithViolations(result, violations))
          return nil
      }

      // Stage 6: 自动上架
      if req.AutoPublish {
          send("publishing", "正在上架到策略市场...", 0.95)
          publishParams := PublishParams{
              UserID: userID,
              StrategyID: result.StrategyID,
              Title: or(req.TitleOverride, generateTitle(req.Description)),
              Description: req.Description,
              PriceModel: PriceModelFree,  // AI 生成策略默认免费，用户可手动改价
              PriceAmount: "0",
              AssetClass: req.AssetClass,
              Symbols: req.Symbols,
              Timeframe: req.Timeframe,
              RiskLevel: req.RiskLevel,
              BacktestSnapshotProto: mustMarshal(result.BacktestSnapshot),
          }
          publishID, err := s.marketplace.Publish(ctx, publishParams)
          if err != nil {
              sendEvent(failedEvent("publishing", err.Error(), true))
              return nil
          }
          sendEvent(completedEvent(result.StrategyID, publishID, result.BacktestSnapshot))
      }

      send("completed", "策略生成完成", 1.0)
      return nil
  }
  ```

  **红线**：
  - 整个 handler 不得持有 DB 事务跨 stage——每个 stage 独立，失败不污染后续。
  - `stream` 的每个 `Send` 必须 check `ctx.Err()`——用户关闭页面时立即终止。
  - Agent Gateway 超时设为 5 分钟（生成+编译+回测 的正常上限）。

- [ ] **2.1b Agent Gateway 串联实现**

  **文件**：`backend/internal/connect/gateway/ai_gateway_handler.go`（扩展）

  **新增方法**：`AutoGenerate(ctx, req) → (<-chan AutoGenerateProgress, error)`

  **逻辑**：
  1. 调用 Python Agent 的 `generate_strategy` RPC（传入自然语言描述 + 参数约束）
  2. Agent 返回源码后，调用 `mql-compiler` 编译（`compile_mql.go` 或 `compile_py.go`）
  3. 编译成功 → 调用 `backtest-engine` 执行回测
  4. 回测完成 → 反序列化 `BacktestSnapshot`
  5. 通过 channel 返回结果

  **关键边界处理**：
  - 编译失败：错误消息里包含编译错误的具体行号和内容（Agent 可用于自我修正）
  - 回测超时：3 分钟硬超时（部分策略含死循环），超时后 kill subprocess
  - 并发控制：全局 semaphore（默认 4）限制同时进行的 AI 生成任务，防止资源耗尽

  **文件**：`backend/internal/connect/gateway/autogen_rate_limiter.go` (new)

  ```go
  // Per-user rate limit: 10 requests per hour.
  // Global concurrency: 4 simultaneous AutoGenerate calls.
  type AutoGenerateLimiter struct {
      userLimits sync.Map  // userID → *rate.Limiter
      globalSem  chan struct{}
  }
  ```

### 前端实现

- [ ] **2.1c `AutoGeneratePanel` 组件**

  **文件**：`frontend/src/pages/marketplace/components/AutoGeneratePanel.tsx` (new)

  **UI 结构**：

  ```
  ┌─────────────────────────────────────────────┐
  │  🤖 AI 策略生成                               │
  │                                              │
  │  描述你的策略需求:                             │
  │  ┌─────────────────────────────────────────┐ │
  │  │ "在 EURUSD H1 上做趋势跟踪，用 EMA 交叉   │ │
  │  │  信号，止损 50 点，止盈 100 点..."       │ │
  │  └─────────────────────────────────────────┘ │
  │                                              │
  │  资产类别: [forex ▼]  品种: [EURUSD ✕]       │
  │  时间周期: [H1 ▼]     风险: [moderate ▼]     │
  │  策略类型: [auto ▼]                          │
  │                                              │
  │  ☑ 回测达标后自动上架到策略市场               │
  │                                              │
  │  [🚀 开始生成]                               │
  │                                              │
  │  ── 生成进度 ────────────────────────────     │
  │  ✅ 代码生成 (12s)                            │
  │  ✅ 编译通过 (3s)                             │
  │  ⏳ 回测中... 60%                            │
  │  ⏳ 质量评估                                  │
  │  ⏳ 上架                                      │
  └─────────────────────────────────────────────┘
  ```

  **状态管理**：
  - 使用 `useServerStream` hook（复用现有 SSE 流模式）订阅 `GenerateAndPublish` 流。
  - 每收到 event 更新 stage 列表 + 进度条。
  - 成功时展示策略预览卡片 + "查看策略详情" / "修改定价" 按钮。
  - 失败时展示错误原因 + "重新生成" / "修改需求" 按钮。

- [ ] **2.1d 完成后 — 策略预览 + 定价编辑**

  **文件**：`frontend/src/pages/marketplace/components/AutoGeneratePanel.tsx`（续）

  **生成成功后的操作**：
  1. 展示策略预览卡片（标题、回测指标、实盘/回测对比预告）
  2. "修改定价"按钮 → 弹窗设置 price_model 和 price_amount（默认免费）
  3. "立即发布"按钮 → 调用 `SetStrategyPricing` + 跳转策略详情页
  4. "在策略库中查看"按钮 → 跳转 `StrategyWorkspace`

- **Gate 2.1**：`go build ./...` + `buf generate` + 前端编译通过 + 端到端手动测试（一个真实的 AI 生成请求）。

---

## 模块 2.2 · 批量策略生成队列

- [ ] **2.2a 待生成任务表**

  **文件**：`backend/migrations/xxx_auto_generated_strategies.up.sql` (new)

  ```sql
  CREATE TABLE auto_generation_tasks (
      id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      asset_class VARCHAR(20) NOT NULL,
      symbol VARCHAR(20) NOT NULL,
      timeframe VARCHAR(10) NOT NULL,
      strategy_type VARCHAR(30) NOT NULL,  -- trend_following / mean_reversion / breakout / arbitrage
      risk_level VARCHAR(15) NOT NULL DEFAULT 'moderate',
      status VARCHAR(20) NOT NULL DEFAULT 'pending',
      -- pending → generating → compiling → backtesting → awaiting_review → published / rejected
      strategy_id UUID,                    -- assigned after generation
      result_backtest_snapshot BYTEA,       -- proto BacktestSnapshot
      quality_passed BOOLEAN,
      error_message TEXT,
      created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
      started_at TIMESTAMPTZ,
      finished_at TIMESTAMPTZ
  );
  CREATE INDEX idx_autogen_tasks_status ON auto_generation_tasks(status);
  ```

  **设计决策**：每个组合一条任务记录，支持断点续传（重启后从 `pending` 或中断的 stage 继续）。

- [ ] **2.2b 任务生产者 — 扫描品种×周期×类型空间**

  **文件**：`backend/internal/marketplace/batch_generator.go` (new)

  **逻辑**：
  1. 每周执行一次（或 Admin 手动触发）。
  2. 从系统配置读取覆盖范围：`marketplace.batch.symbols`（如 `["EURUSD","GBPUSD","USDJPY","BTCUSD","XAUUSD"]`）、`marketplace.batch.timeframes`（如 `["M15","H1","H4","D1"]`）、`marketplace.batch.strategy_types`（如 `["trend_following","mean_reversion","breakout"]`）。
  3. 笛卡尔积生成任务，跳过高频失败组合（如同品种同周期同类型过去 7 天失败过 3 次）。
  4. INSERT INTO `auto_generation_tasks` (status='pending')。

- [ ] **2.2c 任务消费者 — PG NOTIFY 驱动，禁止轮询**

  **文件**：`backend/internal/marketplace/batch_generator.go`（续）

  **架构约束**：🔴 本项目禁止 timer/polling。消费者由 PG NOTIFY 唤醒，非定时扫描。

  **逻辑**：
  1. 任务生产者（2.2b）在 INSERT 后发送 `NOTIFY auto_generation_task_ready`。
  2. 消费者启动时用 `pg.Listen` 订阅该 channel。
  3. 收到 NOTIFY → 取一条 `SELECT ... FROM auto_generation_tasks WHERE status='pending' LIMIT 1 FOR UPDATE SKIP LOCKED`。
  4. 取到任务 → 调用 `AgentGateway.AutoGenerate` → 更新 task 状态 → 完成后再次检查是否有 pending 任务（处理积压）。
  5. 无 pending 任务 → 等待下一个 NOTIFY（阻塞在 `pg.Conn.WaitForNotification`）。
  6. 限制每日消费数（`marketplace.batch.daily_limit`，默认 20）。

  **设计决策**：PG NOTIFY 丢失不致命——生产者写入后发 NOTIFY，如果 NOTIFY 丢失（极少发生），Admin 可通过 `TriggerBatchGeneration` RPC 手动唤醒消费者。

- [ ] **2.2d Admin 审核面板**

  **文件**：`backend/internal/connect/marketplace/marketplace_handler_autogen.go`
  **新增 RPC**：
  - `ListAutoGenTasks(status, offset, limit)` → 待审核列表
  - `ApproveAutoGenTask(task_id, price_model, price_amount)` → 批量批准 → 自动 Publish
  - `RejectAutoGenTask(task_id, reason)` → 拒绝

  **前端**：Admin Dashboard 新增"AI 生成策略审核"模块。列表展示每个任务的品种×周期×类型、回测摘要、质量达标状态。支持全选 + 批量批准。

- **Gate 2.2**：`go build ./...` + `buf generate` + 单元测试（任务生产者、消费者、去重逻辑）。

---

## 模块 2.3 · 策略参数模板

- [ ] **2.3a 模板表**

  **文件**：`backend/migrations/xxx_strategy_parameter_templates.up.sql` (new)

  ```sql
  CREATE TABLE strategy_parameter_templates (
      id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      template_key VARCHAR(50) UNIQUE NOT NULL,  -- "trend_following_ema_cross"
      template_type VARCHAR(30) NOT NULL,         -- trend_following / mean_reversion / breakout / arbitrage
      name_i18n JSONB NOT NULL DEFAULT '{}',      -- {"en":"EMA Crossover Trend Following","zh-cn":"EMA交叉趋势跟踪"}
      description_i18n JSONB NOT NULL DEFAULT '{}',
      parameters JSONB NOT NULL,                   -- 参数定义 schema（见下方）
      default_risk_level VARCHAR(15) DEFAULT 'moderate',
      icon VARCHAR(50),                           -- 前端图标标识
      sort_order INT DEFAULT 0,
      enabled BOOLEAN DEFAULT true,
      created_at TIMESTAMPTZ NOT NULL DEFAULT now()
  );

  -- 预置种子数据
  INSERT INTO strategy_parameter_templates (template_key, template_type, name_i18n, parameters) VALUES
  ('trend_ema_cross', 'trend_following',
   '{"en":"EMA Crossover Trend","zh-cn":"EMA 双均线交叉趋势"}',
   '{"fast_period": {"type":"int","min":5,"max":50,"default":12,"label":"快线周期"},
     "slow_period": {"type":"int","min":20,"max":200,"default":26,"label":"慢线周期"},
     "stop_loss_pips": {"type":"int","min":10,"max":200,"default":50,"label":"止损点数"},
     "take_profit_pips": {"type":"int","min":20,"max":500,"default":100,"label":"止盈点数"}}'),
  ('mean_rev_bollinger', 'mean_reversion',
   '{"en":"Bollinger Band Mean Reversion","zh-cn":"布林带均值回归"}',
   '{"period": {"type":"int","min":10,"max":50,"default":20,"label":"布林带周期"},
     "std_dev": {"type":"float","min":1.0,"max":3.0,"default":2.0,"label":"标准差倍数"},
     "stop_loss_pips": {"type":"int","min":10,"max":200,"default":30,"label":"止损点数"}}');
  ```

- [ ] **2.3b RPC 定义**

  **文件**：`proto/ant/v1/marketplace_service.proto`
  **新增**：

  ```protobuf
  message StrategyParameterTemplate {
    string id = 1;
    string template_key = 2;
    string template_type = 3;
    string name = 4;           // 根据 Accept-Language 返回对应语言
    string description = 5;
    string parameters_schema = 6;  // JSON string，便于前端动态渲染表单
    string default_risk_level = 7;
    string icon = 8;
  }

  message ListStrategyTemplatesRequest {
    string template_type = 1;  // 可选筛选
  }
  message ListStrategyTemplatesResponse {
    repeated StrategyParameterTemplate templates = 1;
  }

  message GenerateFromTemplateRequest {
    string template_id = 1;
    string parameters_json = 2;  // 用户填写的参数值 {"fast_period":15,...}
    string symbol = 3;
    string timeframe = 4;
    bool auto_publish = 5;
  }
  // Response: stream GenerateAndPublishEvent（复用 2.1 的流）

  // Add to MarketplaceService:
  // rpc ListStrategyTemplates(ListStrategyTemplatesRequest) returns (ListStrategyTemplatesResponse);
  // rpc GenerateFromTemplate(GenerateFromTemplateRequest) returns (stream GenerateAndPublishEvent);
  ```

  **设计决策**：`GenerateFromTemplate` 复用 `GenerateAndPublishEvent` 流——前端不需要区分"自由描述"和"模板生成"，统一处理。

- [ ] **2.3c 前端模板选择器**

  **文件**：`frontend/src/pages/marketplace/components/AutoGeneratePanel.tsx`（扩展）

  **两种入口**：
  1. "从模板开始" —— 卡片网格展示可用模板，点击进入参数填写表单。
  2. "自由描述" —— 文本框输入自然语言（已有 2.1c）。

  **模板选择器 UI**：
  ```
  ┌─────────────────────┐  ┌─────────────────────┐
  │ 📈 EMA 双均线交叉     │  │ 📉 布林带均值回归     │
  │ 趋势跟踪 · 稳健型     │  │ 均值回归 · 中等风险   │
  │                      │  │                      │
  │ 参数: 快线/慢线/止损  │  │ 参数: 周期/标准差/止损 │
  │ [选择此模板]         │  │ [选择此模板]         │
  └─────────────────────┘  └─────────────────────┘
  ```

  选择模板后 → 参数填写表单（根据 `parameters_schema` 动态渲染 InputNumber + Slider）→ "开始生成"按钮。

- **Gate 2.3**：`go build ./...` + `buf generate` + 前端编译通过。

---

## Phase 2 完成检验

全部模块通过后：

```bash
go build ./...
buf generate
go test ./internal/marketplace/...
go test ./internal/connect/marketplace/...
go test ./internal/connect/gateway/...
cd frontend && npm run build
bash scripts/gen_capability_map.sh
```

**关键验收场景**：
1. 自然语言描述 → AI 生成策略 → 自动回测 → 达标上架（完整管线）
2. 自然语言描述 → 编译失败 → 前端显示具体编译错误（行号+内容）
3. 自然语言描述 → 回测不达标 → 前端显示不达标指标
4. 批量生成任务：Admin 触发 → 20 个任务排队消费 → 审核面板批准/拒绝
5. 从模板生成：选模板 → 填参数 → AI 填充 → 回测 → 上架
6. 并发限制：第 5 个同时生成请求被限流（返回 rate_limit 错误）
7. 用户关闭页面 → SSE 流正确处理 context cancel
