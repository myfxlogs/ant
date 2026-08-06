# P0 施工 Spec 03：交易字段完整性不变量（恒等类，Stage 1.2）

> ADR-0028 §4.2 防线 B 恒等类——trades 字段的物理边界。3 个纯函数，和已落地的 `checkVolumeInvariant`/`checkCapitalConservation` 同模式。不需 bars 数据（只遍历 `result.Trades` 字段）。

## 现状代码（必读）

- 文件：`backend/internal/connect/strategy/backtest_worker_vm.go`
- `buildBacktestResponse` 已有 `checkVolumeInvariant` + `checkCapitalConservation`（纯函数返回 `*BlindSpot`，调用处 `if bs := ...; bs != nil { resp.Risk.IsReliable=false; resp.BlindSpots=append(...) }`）。**本任务加 3 个并列同模式函数，不动现有两个。**
- 相关类型（已核对）：
  - `backtest.Trade`：`EntryPrice/ExitPrice decimal.Decimal`、`Side sdk.PositionSide`、`EntryTime/ExitTime time.Time`
  - `sdk.PositionSide` 合法值：`sdk.SideBuy`、`sdk.SideSell`（其余为非法）
  - 返回 `*antv1.BlindSpot`（Id/Category/Severity/Description），`interp.SeverityFatal`

## 任务：3 个纯函数 + 接入

1. **`checkPricePositive(result *backtest.Result) *antv1.BlindSpot`**：每笔 `EntryPrice > 0` 且 `ExitPrice > 0`。任一 ≤0 → BlindSpot（Id `non_positive_price`）。空 Trades → nil（vacuously）。
2. **`checkSideValid(result *backtest.Result) *antv1.BlindSpot`**：每笔 `Side == sdk.SideBuy || Side == sdk.SideSell`。任一非法 → BlindSpot（Id `invalid_side`）。空 Trades → nil。
3. **`checkTimeOrder(result *backtest.Result) *antv1.BlindSpot`**：每笔 `!EntryTime.After(ExitTime)`（即 EntryTime ≤ ExitTime，开仓不晚于平仓）。任一 EntryTime > ExitTime → BlindSpot（Id `time_order_violation`）。**EntryTime == ExitTime 合法（≤）**。空 Trades → nil。
4. **接入 `buildBacktestResponse`**：紧跟 `checkCapitalConservation` 之后，3 个并列 `if`（同模式）。

每个 BlindSpot：`Category:"invariant"`, `Severity:interp.SeverityFatal`, Description 用中文说清违反了什么。

## 测试（必须带，每个函数）

- **正例**：合法 trades → nil。
- **负例**：违反 → BlindSpot，字段正确。
- **边界**（每个函数自行补全）：
  - 价格：零价格、负价格、EntryPrice 正但 ExitPrice 零、极小正价格（应通过）。
  - 方向：SideBuy/SideSell 通过、其他值（如 0、99）违反。
  - 时间：EntryTime==ExitTime 通过、EntryTime>ExitTime 违反、正常 EntryTime<ExitTime 通过。
  - 共同：空 Trades（nil）、单笔、多笔（验证遍历全部，含违规在中间/末尾）。
- **集成**：`buildBacktestResponse` 接入（合法时这几个 Id 不出现；某个违反时对应 BlindSpot 出现 + IsReliable=false）。

## 约束

- 纯函数（与现有两个同模式），**不改 checkVolumeInvariant/checkCapitalConservation**、不重构无关、不硬编码、不 mock。
- `go build ./...` + `go test ./internal/connect/strategy/` 全绿。
- 评审会做对抗验收（spec 未明说输入 + 测试覆盖），具体 case 不预先公布。

## 验收（评审）

- build + test 全绿。
- 读实现：3 个纯函数判断对（>0 / 合法 side / !After）、空 Trades vacuously、接入对（紧跟 capital 之后、不动现有）。
- 对抗 case（不预先公布）。
- 集成测试必须有。
- 不过度设计、无无关改动。
