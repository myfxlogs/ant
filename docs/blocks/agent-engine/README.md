# agent-engine — Agent 引擎

> AI 策略生成、盲区桥接、三层记忆、回测反馈、参数优化。

## 代码位置

```
backend/internal/agent/         ← 27 文件：Generator、MemoryStore、Bridge、tool registry
backend/internal/ai/            ← 60+ 文件：回测反馈、蒙特卡洛、walkforward、优化器(DE/TPE/KDE)
```

## 关键设计

- Agent 循环：观察→思考→行动(write_strategy)→编译→回测→分析→迭代
- 盲区桥接：MQL 有盲区→LLM 翻译到 Python 子集→编译验证→VM 回测确认（最多 3 次重试）
- 三层记忆（ADR-0025）：全局领域知识 / 用户策略模板 / Agent 经验
- Python 思考（pandas/numpy/optuna），Go VM 执行（性能差距 200x）

## 依赖

```
mql-compiler(编译) → agent-engine
backtest-engine(回测验证) → agent-engine
```

## 被依赖

```
agent-engine → api-gateway(SSE 流式输出)
agent-engine → strategy-marketplace(AI 生成策略 + 迭代优化)
```
## 关联文档
- [spec/26-ai-strategy-generation.md](spec/26-ai-strategy-generation.md)
