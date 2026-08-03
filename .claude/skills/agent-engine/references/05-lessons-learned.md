# 05 — 教训与反模式（每天都应该重读）

> 今天的错明天不要再犯。

## 铁律：Claude Code 怎么做，我们就怎么做

当遇到设计决策时，先问自己：**Claude Code 是怎么做的？** 它在生产环境已验证，不要自己发明。

| 我们犯的错 | Claude Code 的做法 | 修复 |
|-----------|-------------------|------|
| 在 system prompt 里列举所有工具+使用规则 | 工具描述里写清楚何时用，prompt 极短 | 工具 description 承载智能，prompt 回退到 6 行 |
| `reasoning_effort: "low"` 压制成默认 | 不设 reasoning_effort，信任 `tool_choice: auto` | 删除该行 |
| 工具描述冗长但缺 "何时用" 指导 | 描述精准回答 "什么时候调这个工具" | 每个描述加 use-when 语义 |
| 忘记注册工具（`read_kline` 只在 Chat agent） | 所有可用工具都在 Agent 注册表里 | 加入 `buildPythonToolRegistry` |

## 反模式检查清单

动工前逐条过一遍：

- [ ] 提示词是否超过 10 行？→ 太长，把指令移到工具描述里
- [ ] 是否在改 `reasoning_effort` / `temperature` / `max_tokens` 来"修复"模型行为？→ 根因在别处，别调这些
- [ ] 新工具是否只在某个 agent 注册了，另一个没加？→ Generator 和 Chat 各自维护工具表
- [ ] 代码在系统提示里列举工具名？→ 删掉。工具描述会自己解释
- [ ] `[TOOL:]` 示例在提示/描述里？→ 删掉。原生 function calling 不需要，文本回退是通用的

## 正确模式

### 工具描述模板

```
"Description: "做什么。什么时候用。不要什么时候用。"
```

好例子：
```
"读取K线数据并返回市场分析。当你需要了解当前市场状况、分析行情趋势、查看价格形态时调用。"
```

坏例子（信息不足）：
```
"读取K线数据。"
```

坏例子（信息过载）：
```
"读取K线数据并返回市场分析。包含：总bar数、日期范围、当前价格、EMA20、EMA50、
趋势方向、近期高低价、波动率、最近10根K线OHLC。调用: [TOOL: read_kline ...]"
```

### 提示词模板

```
你是 X。选合适的工具。

## 规则
- 语义歧义 → 问。装饰性歧义 → 默认+注释。
- 改代码前先读当前代码。
```

不超过 6 行。

### 工具注册检查清单

新工具上线时，确认两端都有：
- [ ] `buildPythonToolRegistry` (Generator, `agent_tools.go`)
- [ ] `toolStream` switch case (`generator_agent.go`) — 需要前端流事件的
- [ ] Chat agent 路径如果需要（`tool_registry.go` 或 `strategy_plan_handler.go`）

## 今天踩过的坑

1. **conversationId 管线断裂** — 5 个 bug 排成一串（Create UUID 不匹配 → SetConversationRepo 没调 → UTF-8 字节截断 → ResolveSession 自动建空白行 → useWorkspaceSession 遗留代码）。修完一条，下一条才暴露。教训：端到端测试每次都要从头到尾走一遍。

2. **prompt 膨胀** — 加了又删，删了又缩。教训：看到 prompt 超过 10 行，停下来问"能不能写到工具描述里？"

3. **sed 破坏 Go 代码** — 用 sed 改 Go 导致语法错误、重复行、缺失行。教训：能用 Edit 工具就用，Edit 失败就读文件再 Write 全量重写。

4. **reasoning_effort 盲设** — 没查 Claude Code 怎么处理就直接设 low。教训：改 API 参数前先看 Claude Code 源码/配置。
