# P0 施工 Spec 01：成交手数 >0 不变量 + assessRisk 可信度闸门

> 来源：ADR-0028 §4.2 防线 B 的第一个恒等类不变量。背景：xianhua 事故中，手数=0 的回测被报告为 SUCCEEDED（假成功）。本任务实现"成交手数>0"判决型不变量，并作为 assessRisk 的前置可信度闸门。

## 现状代码（必读，自行定位精确插入点）

- **文件**：`backend/internal/connect/strategy/backtest_worker_vm.go`
- **函数**：`buildBacktestResponse(result *backtest.Result, cfg backtest.Config, params backtestParams, vmRunner *mql2go.VMRunner) (*antv1.ExecuteBacktestResponse, ...)`
- 该函数现状：构造 `resp`（含 Metrics / EquityCurve / Trades / ExecutionAssumptions），随后 `resp.Risk = assessRisk(result.Metrics)`，再 `resp.BlindSpots = attachBlindSpots(...)`。
- 相关类型（已核对，准确）：
  - `backtest.Result.Trades` 是 `[]backtest.Trade`（`strategy/backtest/types.go`）
  - `backtest.Trade.Volume` 是 `github.com/shopspring/decimal.Decimal`
  - `resp.Risk` 是 `*antv1.ExecuteRiskAssessment`，含字段 `IsReliable bool`
  - `resp.BlindSpots` 是 `[]*antv1.BlindSpot`（字段：`Id / Category / Severity / Description`）
  - `assessRisk` 在 `backtest_worker_helpers.go`，返回 `*ExecuteRiskAssessment`（Score/Level/IsReliable/warnings）。其 `IsReliable` 现仅看"交易数>=10"。

## 任务

1. **实现"成交手数 >0"恒等类不变量**：遍历 `result.Trades`，每笔 `Volume` 必须严格 `> 0`。
2. **作为 assessRisk 的前置可信度闸门**（ADR-0028 §4.4）：若存在任何 `Volume <= 0` 的交易（违反），则结果标记不可信：
   - `resp.Risk.IsReliable = false`
   - `resp.BlindSpots` 增加一条：`Id` 如 `"zero_volume_trade"`、`Severity` 用 fatal 级、`Description` 说明"存在手数<=0的交易，回测结果不可信"。
   - 逻辑顺序：本不变量检查应在 assessRisk 评分产出之后、最终返回之前生效（即覆盖 IsReliable）。
3. **恒等类语义**：无交易时本规则不适用（vacuously true，不应误报）。

## 测试要求（必须带，新建测试文件）

覆盖三类：
- **正例**：所有交易 Volume > 0 → 不变量通过，本规则不将 IsReliable 置 false。
- **负例**：存在 Volume == 0 的交易 → 不变量触发，IsReliable=false， BlindSpot 出现且内容正确。
- **边界**：自行补充你认为必要的边界 case（评审会检查你补了哪些、是否充分）。

## 约束

- 不准硬编码（如为通过测试写死 return）。
- 不准用 mock 代替真实遍历。
- **最小改动**：不自作主张加其他不变量、不重构无关代码、不引入新依赖。
- 必须 `go build ./...` 与 `go test ./internal/connect/strategy/` 通过。

## 验收（评审标准）

- `go build ./...` + `go test ./internal/connect/strategy/` 全绿。
- 读实现：真遍历、判断正确（严格 >0）、IsReliable/BlindSpot 正确设置、恒等类语义对（无交易不误报）。
- **对抗验收**：评审会构造 spec 未明说的输入跑你的代码 + 读你的测试看覆盖度（具体 case 不预先公布）。
- 无过度设计、无无关改动、无 `//nolint`。
