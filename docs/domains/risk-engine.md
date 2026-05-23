# RiskEngine 统一风控

## 1. 目标

`RiskEngine` 是手动交易、自动交易和策略调度的统一风控决策入口，输出 `RiskDecision`。

## 2. 决策结构

`RiskDecision` 包含：

- `decision`：`allow` 或 `reject`。
- `allowed`：是否允许执行。
- `source`：`manual`、`auto`、`schedule`。
- `code`：稳定原因编码。
- `reason`：展示或日志原因。
- `retryable`：是否可重试。
- `context`：必要上下文。

## 3. 自动交易校验

自动交易风控包括：

- 账户禁用拒绝。
- 投资者账户拒绝。
- symbol 空值拒绝。
- volume 非正数、NaN、Inf 拒绝。
- 持仓数为负拒绝。
- balance/equity 为负、NaN、Inf 拒绝。
- 按配置检查最大持仓、最大手数、每日亏损、最大回撤。

## 4. 调度风控

调度运行前通过 RiskEngine 统一返回 allow/reject 决策。拒绝原因应写入执行日志或调度错误字段。

## 5. 前端边界

前端不得重新实现风控计算或交易决策，只展示后端返回的 `RiskDecision`、原因编码和汇总状态。
