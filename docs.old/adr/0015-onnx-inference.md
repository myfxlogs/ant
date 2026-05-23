# ADR-0015: ONNX 推理通道

**Status**: Accepted  
**Date**: 2026-05-23  
**Deciders**: 架构决策  
**Supersedes**: None  
**关联**: `backend/internal/quantengine/` · `research/ant_research/model/` · `backend/migrations/097_strategy_models.up.sql`

## 背景

ant 的策略信号生成从 Python 沙箱切换到 DSL 后，DSL 可以覆盖大部分技术指标类策略，但机器学习类策略（LightGBM/XGBoost/神经网络）无法用 DSL 表达。需要一个模型推理通道：

1. 用户在 research 环境训练模型 → 导出 ONNX
2. 生产环境加载 ONNX → 逐 bar 推理 → 输出信号
3. 模型版本可管理、可回滚

## 决策

引入 ONNX Runtime 作为 ML 模型推理引擎，构建 `quantengine` 模块：

### 架构

```
research (Python)                    production (Go)
┌──────────────────┐                ┌──────────────────────┐
│ trainer.py       │                │ quantengine/          │
│   LightGBM/skl   │   ONNX export  │   onnx_runtime.go     │
│        ↓         │ ─────────────> │   runner.go           │
│ exporter.py      │    model.onnx  │   runtime.go          │
│   → MinIO/local  │                │   signal_oms_bridge.go│
└──────────────────┘                └──────────┬───────────┘
                                               │
                                               ▼
                                         OMS Executor
                                        (M2 state machine)
```

### 模型管理

`strategy_models` 表（PG）：

```sql
id, strategy_id, version, onnx_uri, inputs (JSONB), outputs (JSONB), owner_user_id, active
```

- `onnx_uri`: MinIO 路径或本地文件路径
- `inputs`/`outputs`: JSONB 描述模型输入输出 schema
- `active`: 同一策略可注册多版本，仅 active=true 的生效

### 信号 → OMS 审计

推理输出的 buy/sell 信号通过 `signal_oms_bridge` 转换为 OMS 订单，附带 `signal_id` 字段用于审计追踪。

## 后果

### 正面

- 支持任意 ML 模型（LightGBM/XGBoost/sklearn/PyTorch → ONNX）
- 模型版本管理（多版本共存、回滚）
- 推理与交易链路隔离（signal → OMS 桥接，可独立测试）
- Go 原生推理（无 Python GIL、无沙箱风险）

### 负面

- ONNX Runtime 依赖（CGO，需 `libonnxruntime.so`）
- 仅支持 ONNX 导出的模型（部分 sklearn 自定义转换器不兼容）
- 模型输入特征需与 DSL 因子对齐（需约定特征顺序）

### 风险缓解

- `trainer.py` 内建 `assert_onnx_exportable()` 校验
- `exporter.py` 导出时记录 input/output schema 到 `strategy_models.inputs/outputs`
- 推理前校验：输入维度必须与模型声明一致

## 实现

- `backend/internal/quantengine/onnx_runtime.go`: ONNX Runtime CGO 绑定
- `backend/internal/quantengine/runner.go`: 逐 bar 推理循环
- `backend/internal/quantengine/runtime.go`: 模型加载/卸载/热更新
- `backend/internal/quantengine/signal_oms_bridge.go`: 信号 → OMS 转换
- `research/ant_research/model/trainer.py`: 训练入口（LightGBM/sklearn）
- `research/ant_research/model/exporter.py`: ONNX 导出 + MinIO 上传
- `backend/migrations/097_strategy_models.up.sql`: PG 模型注册表

## 备选方案

| 方案 | 优点 | 缺点 | 决定 |
|------|------|------|------|
| **ONNX Runtime** | 跨框架、Go 原生、业界标准 | CGO 依赖 | ✅ |
| Python 子进程推理 | 灵活 | GIL、进程管理、延迟高 | ❌ |
| TensorFlow Serving | 成熟 | 太重、需要独立服务 | ❌ |
| 纯 Go 实现（GoML） | 零 CGO | 模型覆盖少、需重新训练 | ❌ |
