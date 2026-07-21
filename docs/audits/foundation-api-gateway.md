# 地基审计 · api-gateway

## 1. 核心设计替代方案

**当前**：ConnectRPC + SSE。52 个 gRPC 服务，335 个 RPC。14 个 Connect handler 目录。禁止 REST（除 healthz）。

**替代方案 A**：REST API。

不选。双向流（SSE）在 REST 里需要 WebSocket——项目禁止 WebSocket。ConnectRPC 支持 server-stream，且 proto 定义本身是类型安全的 API 合约。选择正确。

**替代方案 B**：GraphQL。

不选。量化交易 API 不是"前端需要什么字段"的问题——是"策略引擎产生 proto 结构体、前端消费 proto 结构体"。GraphQL 的灵活查询在这个场景是额外复杂度，无收益。

**结论**：✅ 架构最优。ConnectRPC + proto 是最适合量化交易平台的 API 方案。

## 2. 上下游契约

**上游**：所有后端模块通过 Connect handler 暴露 RPC。proto 定义在 `proto/ant/v1/`。

**下游**：frontend 通过 ConnectRPC client 调用。SSE 流用于实时数据推送。

**隐患 A**：335 个 RPC 中有多少是"写了一次再没人用过"的？API 膨胀会导致维护负担——新增一个字段，要追 N 个 handler 是否受影响。**需要 API 使用频率统计。**

**隐患 B**：proto 定义是手动维护还是从代码生成？52 个 service 如果手动维护，容易产生"实现改了但 proto 没改"的不一致。

**隐患 C**：SSE 连接数管理。每个用户可能同时打开多个 SSE 流（行情、订单、通知）。当前没有连接数限制——高并发下可能压垮服务器。

## 3. 已知架构债务

| 债务 | 严重度 | 方案 |
|------|--------|------|
| API 膨胀（335 RPC） | 🟡 | 统计每个 RPC 的调用频率，低频的标记 deprecated 或合并 |
| SSE 连接数无上限 | 🟡 | 按用户限制并发 SSE 连接数（如每人最多 5 个流） |
| proto 与实现一致性 | 🟢 | 自动化——`buf generate` 保证生成代码和 proto 同步。手动维护风险低 |

## 4. 总评

架构最优。问题不在设计——在治理。335 个 RPC 是"功能驱动增长"的自然结果，但需要周期性清理。SSE 连接数限制是上线前必须加的保护。
