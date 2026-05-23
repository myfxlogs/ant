# 回测系统边界

## 1. 当前状态

本文件是当前回测系统领域入口。历史回测文档不再作为实施依据。

## 2. 职责边界

- 策略服务负责策略校验、执行沙箱和回测计算。
- 后端负责回测任务、运行状态、结果入库、成本模型和 ConnectRPC 对外契约。
- 前端只展示回测运行状态、指标、交易列表和后端返回的成功/终态字段。

## 3. 状态权威

- 回测运行状态以后端 `BacktestRun` 为权威。
- `is_terminal`、`is_succeeded` 由后端计算。
- 前端不得根据状态字符串自行推导终态或成功态。

## 4. 后续拆分方向

- `backtest-architecture.md`
- `backtest-data-model.md`
- `backtest-metrics.md`
- `backtest-cost-model.md`
