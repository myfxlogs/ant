# 上线前实施计划 — 2026-08-02

> 基于 `pre-launch-assessment.md` 的三个 Gap 展开，每个 Gap 拆分到文件级、函数级。

---

## Gap 0: 端到端用户路径验证

### 目标

用一个真实账号走完完整闭环，每一步记录结果。任何一步失败 = 对应功能不可上线。

### 验证路径

```
注册 → 绑 MT 账户 → 进 workspace → Agent 生成策略 → 回测 → 
上架 marketplace → 另一账号购买 → 查看分账到账
```

### 验证步骤（8 步）

| 步 | 操作 | 验收标准 | 可能失败点 |
|----|------|----------|-----------|
| 1 | admin@1.com 登录，进 /dashboard | 看到 Account Overview 卡片数据正确 | Dashboard 统计、SSE 连接 |
| 2 | 进入已绑账户 fcca3414 的详情页，查看持仓/订单/净值曲线 | 净値曲线有数据点、持仓列表正确 | SSE profit stream、PG 数据 |
| 3 | 进入 workspace，选 EURUSD/H1，用 Agent 生成一个简单的均线交叉策略 | Agent 返回策略代码 + 回测结果，编译通过 | Agent loop、write_strategy 工具 |
| 4 | 回测页查看结果：净値曲线、交易列表、胜率/夏普 | 曲线有数据、交易数 > 0、指标正确 | VM Runner、SimBroker、metrics |
| 5 | 将策略上架 marketplace，设置价格 | 上架成功，marketplace 列表可见 | marketplace publish API |
| 6 | 用另一个账号（如 1735319355@qq.com）登录，浏览 marketplace，购买该策略 | 购买成功，钱包扣款，策略出现在 workspace | purchase flow、wallet deduction |
| 7 | 回到 admin@1.com，查看分账是否到账 | provider wallet 余额增加，settlement 记录出现 | settlement 流程 |
| 8 | AI 分析：进入 account report 页面，生成 AI 报告 | 报告生成成功，钱包产生 ai_usage 扣费 | AI billing（已修复） |

### 不通过的处理

- 任一步失败 → 记录到本文档 §验证结果，标注失败原因
- 代码修复后重新跑该步，直到通过
- 全部 8 步通过 → Gap 0 关闭

### Step 9: 关键步骤转 Playwright E2E 测试

通过手动验证的步骤，将可自动化的部分转为 Playwright spec（项目已有 `tests/e2e/` 19 个 spec）：

| 步骤 | 自动化方式 |
|------|-----------|
| 登录 + Dashboard 数据正确 | Playwright: 登录 → 等待 SSE 推送 → 断言 Account Overview 卡片有数据 |
| 策略生成 + 回测 | Playwright: 输入 prompt → 等待 Agent 返回 → 断言回测结果显示 |
| 购买 + 钱包扣款 | Playwright: 浏览 marketplace → 购买 → 断言钱包余额变化 |

> 不追求全自动化——Agent 生成质量和 MT 连接依赖外部服务。只自动化不依赖外部状态的步骤。

**新文件**: `tests/e2e/happy-path-pre-launch.spec.ts`

### 预计时间: 4 小时（2h 手动 + 2h Playwright）

---

## Gap 1: Agent 策略生成质量基准

### 目标

建立可重复运行的 Agent 质量测试套件。每次改 prompt 或工具链后自动重跑。上线前达标基准。

### 实施步骤

#### Step 1: 编写测试用例定义文件

**新文件**: `backend/internal/agent/quality_benchmark_test.go` (build tag: `benchmark`)

定义 10 个策略需求，覆盖不同难度和风格。每个需求跑 5 次（LLM 非确定性）：

```
难度 简单(3): 单均线交叉、RSI超买超卖、布林带突破
难度 中等(4): MACD+均线、ADX趋势过滤、多时间框架、网格交易
难度 困难(3): 波动率自适应、多品种联动、事件驱动
```

每个需求为一个 struct：
```go
type BenchmarkCase struct {
    Name        string // "simple_ma_cross"
    Difficulty  string // "easy" | "medium" | "hard"  
    Prompt      string // 用户输入的自然语言策略需求
    MinTrades   int    // 回测至少产生的交易数（>0 表示策略有信号）
    CheckFunc   func(t *testing.T, result BenchmarkResult) // 额外检查
}
```

#### Step 2: 实现基准测试 Runner

```go
func TestAgentQualityBenchmark(t *testing.T) {
    for _, tc := range benchmarkCases {
        t.Run(tc.Name, func(t *testing.T) {
            // 1. 调用 Agent 生成策略（复用 GatewayServer.SubmitStrategy）
            // 2. 检查编译是否通过
            // 3. 检查回测是否完成
            // 4. 检查回测交易数 >= MinTrades
            // 5. 运行 CheckFunc（如有）
            // 6. 记录结果到 benchmark_results 表
        })
    }
}
```

#### Step 3: 建立基准指标

| 指标 | 目标 | 测量方式 |
|------|------|----------|
| 编译通过率 | ≥ 90% | `compile_success / total_runs` |
| 回测完成率 | ≥ 80% | `backtest_success / total_runs` |
| 策略有信号率 | ≥ 70% | `cases_with_trades > 0 / total_cases` |
| 平均回测夏普 | ≥ 0 | `avg(sharpe_ratio)` — 不要求赚钱，但不能全亏 |

#### Step 4: 不达标时的处理流程

1. 分析失败案例：编译失败 → 看错误类型分布（语法错误 vs 平台 SDK 用法错误 vs prompt 误解）
2. 调整 prompt（`internal/ai/locale_agent_en.go` 或工具 schema）
3. 重跑基准测试
4. 如果连续 3 轮调整不达标 → 升级为架构问题（可能需要改工具链或 LLM provider）

### 涉及文件

| 文件 | 动作 | 说明 |
|------|------|------|
| `internal/agent/quality_benchmark_test.go` | 新建 | 基准测试用例 + Runner |
| `internal/agent/benchmark_cases.go` | 新建 | 20 个测试用例定义 |
| `internal/ai/locale_agent_en.go` | 可能修改 | Prompt 调整（如果不达标） |
| `internal/agent/agent_tools_write.go` | 可能修改 | 工具 Schema 调整 |

### 运行方式

**不入 CI**——基准测试依赖外部 LLM（非确定性、有成本）。作为**上线前手动检查清单**：

```bash
# 上线前执行一次，结果写入 docs/benchmarks/agent-quality-YYYY-MM-DD.md
go test -tags=benchmark -count=1 -run TestAgentQualityBenchmark ./internal/agent/ | tee docs/benchmarks/agent-quality-$(date +%F).md
```

每次改 prompt 或工具链后手动重跑，对比历史结果。

### 涉及文件

| 文件 | 动作 | 说明 |
|------|------|------|
| `internal/agent/quality_benchmark_test.go` | 新建 | 基准测试 Runner |
| `internal/agent/benchmark_cases.go` | 新建 | 10 个测试用例定义 |
| `internal/ai/locale_agent_en.go` | 可能修改 | Prompt 调整 |
| `internal/agent/agent_tools_write.go` | 可能修改 | 工具 Schema 调整 |
| `docs/benchmarks/` | 新建目录 | 存放历史基准结果 |

### 预计时间: 2 天（1 天写框架 + 1 天调 prompt）

---

## Gap 2: Marketplace 资金链路集成测试

### 目标

购买/退款/订阅/试用四条核心资金路径各至少一个 happy-path + 一个 error-path 集成测试。
使用真实 PG + Redis（复用 `idempotency_integration_test.go` 的 `getTestPG` / `getTestRedis` 模式）。

### 实施步骤

#### 2.1 购买流程测试

**新文件**: `backend/internal/marketplace/purchase_integration_test.go`

```go
// TestPurchase_HappyPath: 用户 A 购买用户 B 的策略
//  1. 创建 strategy asset（seller）
//  2. 创建 buyer + seller wallet（含初始余额）
//  3. 调用 Purchase API
//  4. 断言：buyer 余额减少 purchase_price
//  5. 断言：seller 钱包增加 provider_amount（purchase_price - platform_fee）
//  6. 断言：platform 钱包增加 platform_fee
//  7. 断言：purchase 记录 status='completed'
//  8. 断言：buyer 获得策略授权（user_subscriptions 有记录）

// TestPurchase_InsufficientBalance: 余额不足
//  1. buyer 余额 < purchase_price
//  2. 调用 Purchase API
//  3. 断言：返回 InsufficientBalance 错误
//  4. 断言：buyer 余额不变、无 subscription 记录

// TestPurchase_DuplicateIdempotent: 幂等性
//  1. 成功购买一次
//  2. 用相同 clientID 再次购买
//  3. 断言：返回 idempotent replay，不重复扣款
```

#### 2.2 退款流程测试

**新文件**: `backend/internal/marketplace/refund_integration_test.go`

```go
// TestRefund_HappyPath: 购买后退款
//  1. 完成一次购买
//  2. 调用 Refund API（在退款窗口内）
//  3. 断言：buyer 余额恢复 purchase_price
//  4. 断言：seller 被 debited provider_amount
//  5. 断言：platform 被 debited platform_fee
//  6. 断言：buyer 策略授权被撤销

// TestRefund_WindowExpired: 退款窗口已过
//  1. 创建购买记录（settles_at 已过去）
//  2. 调用 Refund API
//  3. 断言：返回错误（退款窗口已过）
```

#### 2.3 订阅流程测试

**新文件**: `backend/internal/marketplace/subscription_integration_test.go`

```go
// TestSubscribe_HappyPath: 免费策略订阅
//  1. 创建免费策略
//  2. 调用 Subscribe API
//  3. 断言：subscription 记录创建、total_subscribers 正确

// TestSubscribe_PaidStrategy: 付费策略订阅（等价于购买）
//  1. 创建付费策略
//  2. 调用 Subscribe API
//  3. 断言：等同于 Purchase 流程
```

#### 2.4 金额精度验证

每个涉及金额计算的测试加精度断言：

```go
// 分账精度：providerAmt + platformFee = purchaseAmount
assert.True(t, providerAmt.Add(platformFee).Equal(purchaseAmount),
    "providerAmt(%s) + platformFee(%s) != purchaseAmount(%s)",
    providerAmt, platformFee, purchaseAmount)
```

#### 2.5 幂等性验证

所有资金操作（购买/退款/分账）验证幂等性：

```go
// 同一 clientID 重放 3 次，余额只变一次
for i := 0; i < 3; i++ {
    err := svc.Purchase(ctx, req) // same req.ClientID
    if i == 0 { assert.NoError(t, err) }
    if i > 0 { assert.ErrorIs(t, err, ErrIdempotentReplay) }
}
balanceAfter := getWalletBalance(t, buyerID)
assert.Equal(t, expectedBalance, balanceAfter)
```

#### 2.4 分账流程测试（补 A-002 CRITICAL 回归）

**文件**: `internal/marketplace/settlement_integration_test.go`

```go
// TestSettlement_HappyPath: 购买后自动分账
//  1. 完成一次购买（产生 frozen settlement）
//  2. 调用 SettleExpired API
//  3. 断言：provider wallet 增加 provider_amount
//  4. 断言：platform wallet 增加 platform_fee
//  5. 断言：settlement 状态 = 'settled'
//  6. 断言：provider_amount + platform_fee = purchase_amount（精度验证）

// TestSettlement_PartialFailure: 批次中部分失败
//  1. 创建 3 笔购买（其中 1 笔的 provider wallet 不存在）
//  2. 调用 SettleExpired
//  3. 断言：2 笔成功、1 笔失败（failedIDs 包含失败项）
//  4. 断言：成功项的 provider 收到钱、失败项保持 frozen
```

### 涉及文件

| 文件 | 动作 | 说明 |
|------|------|------|
| `internal/marketplace/purchase_integration_test.go` | 新建 | 购买 happy + error + 幂等 |
| `internal/marketplace/refund_integration_test.go` | 新建 | 退款 happy + 窗口过期 |
| `internal/marketplace/subscription_integration_test.go` | 新建 | 订阅 happy + 付费升级 |
| `internal/marketplace/settlement_integration_test.go` | 新建 | **分账 happy + 部分失败（A-002 回归）** |

### 验收标准

- `go test -tags=integration -count=1 ./internal/marketplace/` 全部 PASS
- 每条路径至少 1 个 happy + 1 个 error case
- 金额断言精确到 `decimal.Equal`

### 预计时间: 2 天

---

## Gap 3: GoExecutor 移除

### 目标

切断所有生产路径中 Go subprocess 执行策略的入口，统一走 Bytecode VM。
保留 `go_executor.go` 文件但标记为禁止编译，以备回滚。

### 实施步骤

#### Step 0: VM vs GoExecutor 对比验证（先验证再切）

取 10 个现有策略（Go 语法的高覆盖率策略），分别用 GoExecutor 和 Bytecode VM 跑回测，
用 `strategy/backtest/parity.go` 的 diff 工具对比结果：

```bash
# 对每个策略:
# 1. GoExecutor 回测 → 保存结果 A
# 2. VM 回测 → 保存结果 B
# 3. 对比 A vs B：净値曲线差异 < 0.1%、交易列表一致
```

验收标准：
- 10/10 策略对比通过（净値曲线差异 < 0.1%）
- 如有差异 → 先修 VM 兼容性，再切
- 全部通过 → 进入 Step 1

#### Step 1: 切断生产注入点

**文件**: `cmd/server/handlers_strategy.go`

删除 `GoExecutor` 的 setter 注入（约 line 55）：
```go
// 删除此行:
// server.SetGoExecutor(goExecutor)
```

#### Step 2: 修改策略执行路由

**文件**: `internal/connect/strategy/strategy_execution_handler.go`

在所有 `isGoStrategy(code)` 分支，改为走 VM 路径：
```go
// 改前:
if isGoStrategy(code) && s.goExecutor != nil {
    return s.goExecutor.Execute(...)
}
// 改后:
// GoExecutor removed per Gap 3 — all strategies route through VM
return s.executeViaVM(...)
```

#### Step 3: 保留文件但禁止编译

**文件**: `internal/connect/strategy/go_executor.go`

加 `//go:build ignore`：
```go
//go:build ignore
// +build ignore

// go_executor.go retained for emergency rollback — not compiled in production.
```

#### Step 4: Code Assist 路径检查

检查 `internal/connect/ai/code_assist_*.go` 是否有代码执行路径：
```bash
grep -rn "go run\|exec\.Command\|os\.Create.*\.go\|ioutil\.WriteFile.*\.go" internal/connect/ai/
```
如有 → 同样切到 VM。

#### Step 5: 回归验证

```bash
# 确认编译通过（GoExecutor 已排除）
go build ./...

# 确认所有策略执行走 VM
go test -short ./internal/connect/strategy/ 
go test -short ./strategy/backtest/

# 手工验证：workspace 中运行现有策略，确认回测结果与之前一致
```

### 涉及文件

| 文件 | 动作 | 说明 |
|------|------|------|
| `cmd/server/handlers_strategy.go` | 删除 line 55 注入 | 切断入口 |
| `internal/connect/strategy/strategy_execution_handler.go` | 修改 4 处 `isGoStrategy` 分支 | 路由到 VM |
| `internal/connect/strategy/go_executor.go` | 加 `//go:build ignore` | 保留但不编译 |

### 验收标准

- `go build ./...` 通过（GoExecutor 不参与编译）
- Workspace 中现有策略回测结果与移除前一致
- `go_executor.go` 仍存在于 repo 中（可回滚）

### 预计时间: 8 小时（4h 对比验证 + 4h 代码修改）

---

## 总时间线

```
Gap 0:  端到端验证  4h   ← 2h 手动 + 2h Playwright
Gap 3:  GoExecutor  8h   ← 4h VM 对比验证 + 4h 代码修改
Gap 1:  Agent 基准  2d   ← 1d 框架 + 1d 调 prompt（不入 CI）
Gap 2:  资金链测试  2d   ← 购买/退款/订阅/分账 4 个文件
```

总计: 约 6 个工作日。
