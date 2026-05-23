# 精简重构 Batch 1 实施准则

## 1. 文档目标

本文档是交付后第一批低风险精简重构的实施依据。

Batch 1 不做大规模架构迁移，不新增业务能力，只处理已经确认的低风险维护成本问题。

目标：

- 降低重复装配代码。
- 消除静默吞错。
- 明确占位接口状态。
- 清理前端历史兼容残留。
- 保持当前生产行为稳定。

## 2. 硬性边界

本批次必须遵守：

- 不改变浏览器到后端的业务通信方式，仍统一使用 ConnectRPC。
- 不新增业务 REST 接口。
- 不把业务计算、统计计算、风控计算、交易决策、指标聚合、AI 调度、回测与策略执行移动到前端。
- 不触碰 MT4/MT5 外部生成代码。
- 不做全量 `domain/infra` 目录迁移。
- 不做 proto 结构重拆，除非验证发现生成链路必须调整。
- 除 Markdown 外，任何代码、脚本、proto、生成文件单文件不得超过 800 行。

## 3. 本批次范围

### 3.1 后端 Server/Container 去重复

目标文件：

- `backend/internal/server/server.go`
- `backend/internal/app/container.go`

当前问题：

- `Container` 已经持有大量 repo/service/worker/manager。
- `Server` 仍重复铺开多数字段。
- 后续新增服务时需要在多个结构体中同步字段，维护成本高。

实施原则：

- `Server` 应尽量变薄。
- `Server` 可以持有 `container *app.Container`。
- 只有启动、停止、HTTP 服务、必要生命周期对象保留在 `Server` 字段。
- 不改变 service 构造顺序。
- 不改变 ConnectRPC 注册行为。
- 不改变 worker 启停顺序。

验收标准：

- `server.New` 行为等价。
- `Server.Start` / `Server.Stop` 行为等价。
- 后端测试通过。
- 容器重建后健康检查通过。

### 3.2 策略服务静默吞错治理

目标文件：

- `strategy-service/app/memory.py`

当前问题：

- 存在 `except Exception: pass`。
- 非关键路径允许不阻断主流程，但不应完全静默。

实施原则：

- 引入模块级 logger。
- 持久化失败、加载失败应记录 warning。
- 不改变 memory 查询和记录的对外返回契约。
- 不让 memory 辅助能力失败影响回测主流程。

验收标准：

- 不再新增或保留无理由的 `except Exception: pass`。
- strategy-service 可启动。
- 后端调用策略服务的主流程不受影响。

### 3.3 策略服务 external 占位接口状态明确化

目标文件：

- `strategy-service/app/routes/external.py`
- `strategy-service/app/schemas.py`

当前问题：

- macro 接口已经返回 `reserved/not_configured`。
- news 接口返回空列表，但状态表达不够明确。

实施原则：

- 如果保留 `news`，必须显式返回 `status=reserved`、`provider_status=not_configured`。
- 不把占位接口描述成已完成外部数据能力。
- 不新增真实第三方调用。

验收标准：

- 响应结构清楚表达未配置状态。
- 文档或字段名不会误导使用者。

### 3.4 前端 axios 历史残留清理

目标文件：

- `frontend/src/utils/error.ts`
- `frontend/package.json`
- `frontend/package-lock.json`

当前问题：

- 前端业务通信已经收敛到 ConnectRPC。
- `utils/error.ts` 仍引用 `AxiosError` 类型。
- 如果没有真实 axios 调用，应移除 axios 依赖。

实施原则：

- 先确认 `frontend/src` 无 axios 真实调用。
- `getErrorMessage` 应改为基于 `unknown` 的通用错误解析。
- 支持 ConnectRPC 错误、普通 `Error`、含 `message` 字段对象、历史 `response.data.message` 结构。
- 不降低用户可见错误信息质量。

验收标准：

- `frontend/src` 不再 import axios。
- 如无其他依赖需要，`frontend/package.json` 移除 axios。
- `npm --prefix frontend run lint` 通过。
- `npm --prefix frontend run build` 通过。

## 4. 明确不纳入 Batch 1 的事项

以下事项推迟到后续批次：

- 大规模迁移 `backend/internal/service` 到 `backend/internal/domain`。
- 拆分 `StrategyTemplatePage.tsx`、`Summary.tsx`、`SystemAI.tsx` 等前端大页面。
- 重写 strategy-service 为 ConnectRPC/gRPC。
- 重构交易执行主链路。
- 重构调度运行主循环。
- 继续拆分已经低于 800 行的 proto 文件。

## 5. 实施顺序

建议按以下顺序执行：

1. 清理前端 axios 残留。
2. 修复 strategy-service memory 静默吞错。
3. 明确 external news 占位状态。
4. 精简后端 `Server` 重复字段。
5. 统一运行验证。

原因：

- 前三项改动小，便于快速建立绿色验证基线。
- `Server` 去重复涉及后端装配，放在最后更容易定位问题。

## 6. 每步变更要求

每个子任务必须满足：

- 先读现状，再改代码。
- 只改任务相关文件。
- 不做顺手重构。
- 不引入新依赖，除非文档先说明必要性。
- 不删除当前生产必需文件。
- 不创建新的兼容层。

## 7. 验证清单

Batch 1 完成后必须运行：

```bash
python3 scripts/check-file-lines.py
npm --prefix frontend run lint
npm --prefix frontend run build
cd backend && go test ./...
make proto
git diff --check
docker compose up -d --build backend frontend
docker compose ps
curl -fsS http://localhost:8012/health
curl -fsSI http://localhost:8012
```

验收结果必须记录：

- 行数检查结果。
- 前端 lint/build 结果。
- 后端测试结果。
- proto 生成结果。
- 容器健康状态。
- 前端首页 HTTP 状态。

## 8. 回滚原则

任一子任务出现以下情况，应回滚该子任务：

- 需要改变生产行为才能通过测试。
- 需要扩大重构范围到核心交易链路。
- 需要新增业务 REST。
- 需要移动业务计算到前端。
- 需要修改 MT4/MT5 外部生成代码。

回滚后应先更新本文档或总方案，再重新实施。
