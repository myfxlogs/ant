# ConnectRPC 接口风格指南

## 1. 基本原则

- 浏览器到后端的业务通信统一使用 ConnectRPC。
- 不新增业务 REST 接口。
- REST 仅允许健康检查、静态资源、必要反向代理或明确标记的第三方兼容入口。
- 业务计算、统计、风控、交易决策、策略状态判断由后端或策略服务完成。

## 2. Proto 设计

- 每个 proto 文件按单一领域拆分，避免把多个业务域塞进一个文件。
- message 命名使用业务语义，不暴露数据库表结构作为外部契约。
- response 中应提供前端展示所需的后端权威字段，避免前端二次推导。
- 状态类字段由后端给出明确枚举或布尔辅助字段。

## 3. 后端实现

- Connect handler 放在 `backend/internal/connect`。
- 路由注册统一通过 transport/connect 注册模块进入 server。
- handler 只做鉴权、参数转换、服务调用和错误映射。
- 业务逻辑放在 `internal/service` 或领域服务中。

## 4. 前端调用

- 前端 client 放在 `frontend/src/client`。
- 页面不得直接拼接业务 REST URL。
- 页面只渲染后端返回字段，轻量 UI 状态可以留在前端。

## 5. 错误处理

- 后端返回稳定错误 code 或原因字段。
- 前端只做展示和 i18n 映射，不反向推断业务状态。
