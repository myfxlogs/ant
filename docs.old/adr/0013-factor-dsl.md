# ADR-0013: 因子 DSL 规范

**Status**: Accepted  
**Date**: 2026-05-23  
**Deciders**: 架构决策  
**Supersedes**: None  
**关联文档**: `docs/spec/factor-dsl.md` · `backend/internal/factor/dsl/` · `research/ant_research/factor/dsl/`

## 背景

ant 当前策略信号生成依赖用户编写的 Python 代码在 `strategy-service` 沙箱中执行。这带来三个问题：

1. **安全攻面**：RestrictedPython + AST 白名单是黑名单式防御，历史上多次被绕过
2. **AI 生成损耗**：自然语言 → Python 代码 → 沙箱 exec，两次翻译损耗大
3. **回测/实盘漂移**：`indicators.py` 硬编码 230 行算子，与 AI 生成的策略描述语义不一致

alfq 的因子 DSL 方案解决了这些问题：受限文法 → 零安全攻面；NL → DSL 字符串（无中间代码）；Go/Python 双引擎严格对齐。

## 决策

ant 引入因子 DSL 作为策略信号的**唯一生产表达方式**：

1. **语法**：采用 EBNF 定义的受限表达式语法，支持算术/逻辑/比较/三元运算 + 函数调用
2. **算子**：移植 15 个内置算子（SMA/EMA/RSI/MACD/ATR/BB/ZScore/Rank/Corr/Cov 等）
3. **编译模型**：DSL 表达式 → Lexer → Parser → AST → Compiler → Op，Op 支持逐 bar 增量求值
4. **双引擎**：Go 实现用于生产实盘，Python 实现用于研究回测，语义严格对齐

## 后果

### 正面

- 零安全攻面：DSL 是纯数据，无代码执行路径
- AI 自然语言可直接映射到 DSL 表达式（无中间 Python）
- 回测/实盘一致：Go 和 Python 引擎同一份 AST/算子实现
- 热更新：DSL 表达式可动态加载，无需重启服务

### 负面

- 表达能力有限：不支持循环/变量/自定义函数
- 迁移成本：现有 Python 策略需要转换为 DSL
- 算子扩展需同时修改 Go 和 Python 两端

### 风险缓解

- `migrate_legacy_strategy.py` 工具自动转换可迁移策略
- `ValidateExpression()` 校验器在编译时拦截非法表达式
- Go/Python 对齐测试：100 表达式 × 1000 bar，误差 < 1e-9

## 实现

- Go DSL 引擎：`backend/internal/factor/dsl/`（14 文件，~2000 行，22 个单测全通过）
- Python DSL 引擎：`research/ant_research/factor/dsl/`（待移植）
- 文档：`docs/spec/factor-dsl.md`（EBNF 语法 + 算子表 + 对齐约定）

## 备选方案

| 方案 | 优点 | 缺点 | 决定 |
|------|------|------|------|
| **DSL（选中）** | 安全、AI 友好、可热更新 | 表达能力有限 | ✅ |
| 保留 Python 沙箱 | 图灵完备 | 安全攻面、漂移风险 | ❌ 降级为研究专用 |
| Lua/Lox 嵌入式脚本 | 比 Python 轻量 | 仍需沙箱、生态弱 | ❌ 不解决安全问题 |
| 全 SQL 因子 | 数据库原生 | 性能差、无增量求值 | ❌ |
