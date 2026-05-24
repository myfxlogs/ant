# ADR 索引（v2）

> v1 ADR 已归档至 `docs.old/adr/`。
> v2 维护：MT 重写（0001–0005）+ C2C 归属（0006）+ M7 复盘（0007）+ M10 硬化（0008–0011）。

| ID | 标题 | 状态 |
|---|---|---|
| 0001 | MT 基础完全重写（路线 B） | Accepted |
| 0002 | ClickHouse 作为时序存储 | Accepted |
| 0003 | mtapi 直连，不再二次包装 | Accepted |
| 0004 | Tick 去重与质量分级 | Accepted |
| 0005 | CircuitBreaker + Spill 故障恢复 | Accepted |
| 0006 | 平台共享层 vs 用户私有层（C2C 架构）| Accepted |
| 0007 | M7-M9 执行回顾：B 方案叙事与实际结果的偏离 | Accepted |
| 0008 | 存储层去重键对齐 + 时间轴纪律 | Accepted |
| 0009 | Spill Replay 双写 + Bar 不可变性 + 历史回填 | Accepted |
| 0010 | SLO + Alert + DLQ + Trace 框架 | Accepted |
| 0011 | 容量调优 + Vault 轮换 + Normalizer 缓存失效 | Accepted |

## 编号规则

- 单调递增，不复用、不删除
- 文件名 `NNNN-<kebab-slug>.md`
- 状态：`Proposed | Accepted | Rejected | Superseded`
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
