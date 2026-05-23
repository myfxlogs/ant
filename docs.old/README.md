# docs 导航

> ant 量化交易平台文档体系 · 最后更新 2026-05-23（M0.2）

## 快速入口

| 你是… | 读这个 |
|--------|--------|
| 新 Agent 第一天 | [AGENT-RUNBOOK](../docs/tasks/AGENT-RUNBOOK.md) → [ADR 索引](adr/README.md) |
| 实施卡片 | [ROADMAP](../docs/plan/ROADMAP.md) → 对应 milestone 卡片表 |
| 查架构 | [ADR 索引](adr/README.md) → 按主题找对应 ADR |
| 查验收 | [handover/](handover/) `RS-final-verify.log` |

## 五大区

### 1. domain — 领域设计

| 文档 | 状态 | 说明 |
|---|---|---|
| `docs/domains/` | 🚧 待建立 | 订单状态机、风控规则、symbol 体系等 |
| `docs/接口与数据流架构约定.md` | ✅ 现行 | 接口形态与数据流约定 |

### 2. plan — 计划与路线图

| 文档 | 状态 |
|---|---|
| `docs/plan/ROADMAP.md` | ✅ M0.1-M6 完整路线图 |
| `docs/AlfQ功能迁移计划.md` | 📖 参考蓝本（只读，非约束） |
| `docs/进度/` | 🚧 进度跟踪 |

### 3. adr — 架构决策记录

| 文档 | 状态 |
|---|---|
| `docs/adr/README.md` | ✅ 12 篇 ADR 全部 Accepted，按主题索引 |
| `docs/adr/0001-*.md` … `0012-*.md` | ✅ 7 节标准格式 |

### 4. audit — 审查与整改

| 文档 | 状态 |
|---|---|
| `docs/audit/DESIGN-REVIEW-2026-05.md` | ✅ 145 项 finding 完整清单 |
| `docs/_archive/remediation-2026-05/` | 📦 14 文件已归档（2026-05-23） |

### 5. runbook — 执行手册

| 文档 | 状态 |
|---|---|
| `docs/tasks/AGENT-RUNBOOK.md` | ✅ 执行入口 |
| `docs/handover/RS-final-verify.log` | ✅ M0.1 验收日志 |

## 专项设计

| 文档 | 说明 |
|---|---|
| `docs/专项设计/` | 辩论流程、SSE 配置、边缘网关等专项 |

## 文档约定

- 中文为主，技术术语保留英文
- 状态标记：✅ 现行 · 🚧 施工中 · 📖 参考 · 📦 已归档
- 冲突处理：ADR > ROADMAP > AGENT.md > 其他
