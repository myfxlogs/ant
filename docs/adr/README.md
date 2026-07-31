# ADR 索引（v2）

> v1 ADR 已归档至 `docs.old/adr/`。
> v2 维护：MT 重写（0001–0005）+ C2C 归属（0006）+ M7 复盘（0007）+ M10 硬化（0008–0011）+ EA 替代（0020–0021）。

| ID | 标题 | 状态 |
| --- | --- | --- |
| 0001 | MT 基础完全重写（路线 B） | Partially superseded by 0012 |
| 0002 | ClickHouse 作为时序存储 | Superseded by 0012 |
| 0003 | mtapi 直连，不再二次包装 | Accepted |
| 0004 | Tick 去重与质量分级 | Partially superseded by 0012 |
| 0005 | CircuitBreaker + Spill 故障恢复 | Partially superseded by 0012 |
| 0006 | 平台共享层 vs 用户私有层（C2C 架构）| Accepted |
| 0007 | M7-M9 执行回顾：B 方案叙事与实际结果的偏离 | Accepted |
| 0008 | 存储层去重键对齐 + 时间轴纪律 | Accepted |
| 0009 | Spill Replay 双写 + Bar 不可变性 + 历史回填 | Partially superseded by 0012 |
| 0010 | SLO + Alert + DLQ + Trace 框架 | Partially superseded by 0012 |
| 0011 | 容量调优 + Vault 轮换 + Normalizer 缓存失效 | Partially superseded by 0012 |
| 0012 | 统一回测/实盘代码路径 | Accepted |
| 0013 | 订单状态机 + 崩溃恢复 + 幂等性 | Accepted |
| 0014 | 持仓级风控 | Accepted |
| 0015 | 仿真交易（Paper Trading）| Accepted |
| 0016 | Bar 修订级联处理 | Partially superseded by 0012 |
| 0017 | AI 会话记忆 + 意图澄清 + 回测反馈 | Accepted |
| 0018 | 信号→执行延迟 SLO | Accepted |
| 0019 | M11 前端架构（增量重构，React/Zustand/TanStack Query） | Accepted |
| 0020 | EA 完全替代：统一 Strategy SDK + 双实现 Broker | Superseded by 0021 |
| 0021 | 策略运行时从 Python 迁移到 Go | Partially superseded by 0023 |
| 0022 | MQL 盲区架构 — 静态分析 + 运行时追踪 + 致命阻断 | Accepted |
| 0023 | AST 树遍历解释器 + MQL 源码为唯一真实来源 | Accepted |
| 0024 | Agent-Native 策略平台 — 双前端编译 + Go 进程内 Agent | Accepted |
| 0025 | Agent-Native 交互体验与自我进化设计 | Accepted |
| 0026 | HD 钱包充值系统 — 每用户独立地址 + 自动到账确认 | Proposed |
| 0027 | 策略模块前端重构 — Gallery + Detail + Workspace feature-slice | Implemented |

## 编号规则

- 单调递增，不复用、不删除
- 文件名 `NNNN-<kebab-slug>.md`
- 状态：`Proposed | Accepted | Rejected | Superseded | Implemented`
- Superseded 的 ADR 在 header 注明 superseded by NNNN

## 模板

```markdown
# ADR-NNNN · <标题>

- **状态**：Accepted | Proposed | Rejected | Superseded by NNNN
- **日期**：YYYY-MM-DD
- **决策者**：<name>
- **关联 spec**：docs/spec/...

## 1. 背景
（为什么需要这个决策）

## 2. 决策
（一段话说清楚做什么）

## 3. 备选方案
| 方案 | 优点 | 缺点 | 否决理由 |

## 4. 后果
- 正面：
- 负面：
- 中性：

## 5. 实施约束
（具体的代码/接口/schema 约束）

## 6. 验证方式
（如何证明决策落地）
```
