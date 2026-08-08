# AlphaForge 知识库架构（第一性 · 最优解）

> **状态**：架构决策稿（2026-08-08，第一负责人定）。待用户确认后拆施工 spec。
> **一句话**：知识库 = 自我进化的基质；是"trust 护城河"的工程实体；底座 = **PG 单库 + pgvector（向量列）/类型化关系列/LISTEN/NOTIFY，零 JSONB**；原则 = **确定性优先，LLM 仅补语义缺口**。
> **关联**：`market-strategy-review.md`（trust=护城河）/ `mql-ea-compatibility-proposal.md` §6（自我进化闭环，原"虚空"）/ `mql-honesty-audit-spec.md` + `mql-decision-loop-spec.md`（已落的第一批真砖）/ FEAT-5（战绩衰减循环）/ ADR-0012（PG 单存储）。

---

## 0. 为什么必须有知识库（第一性）

- **自我发现是单向的**（发现→修→忘），**自我进化需要基质**（发现→**沉淀→复用**→复利）。无基质则 compat proposal §6 的循环空转 = "虚空建楼"。
- **知识库 = trust 护城河的本体**。战略面叫"信任/流动性护城河"，工程面就是"这个不断复利的知识库"。同一物。
- **复利知识是最防不住的资产**：对手抄得走代码/功能/UI，抄不走"哪些 EA 能跑、哪些修复、哪些策略真赚、用户要什么"。

## 1. 知识的四类（先界定"存什么"）

| 知识流 | 性质 | 来源（生产者）| 消费者 | 现状 |
|---|---|---|---|---|
| **K1 MQL 兼容知识** | 结构化事实（标识符→支持状态/severity/值/映射）| 诚实性分析、L0 修复、人工核验 | 编译器/VM（解析未知）、agent（生成避开不支持）、路线图 | 🟢 真砖但散（constants.go/api_registry/compat_fixes）|
| **K2 交易战绩** | 时序+聚合指标（per 策略/账户，hash 链防篡改）| 实盘+回测 | 衰减检测(FEAT-5)、市场展示、AI 迭代反馈 | 🟢 真砖（trade_records + live_performance）|
| **K3 需求信号** | 计数/事件（不支持特性被命中 N 次/M 用户）| 决策闭环 LEARN（被堵时记录）| 路线图优先级、K1 补全排序 | 🔴 **未捕获**（决策闭环 spec §4 待建）|
| **K4 AI 学习** | 语义/模式（策略特征→结果；相似策略检索）| agent-engine（生成→结果）、迭代 | agent 生成（避开失败模式/复制成功模式）| 🟡 agent memory 层在，未与 K1/K2 打通 |

## 2. 底座决策（最优 = 第一性一致）

### ✅ 选：**PG 单库 + 扩展**（pgvector + 类型化关系列 + LISTEN/NOTIFY + 递归 CTE）

| 能力 | 用途 | PG 手段 |
|---|---|---|
| 结构化事实（K1/K3）| 标识符→状态/值/计数，精确查找 | **类型化关系列** + 索引 |
| 语义相似（K4）| 策略模式相似检索 | **pgvector**（向量列，非 JSON）|
| 属性/谱系 | 修复规则/上下文/来源/版本链 | **类型化列 + 子表**（禁 JSONB）|
| 循环触发（push-first）| 新知识到达→触发消费者 | **LISTEN/NOTIFY** |
| 关系/谱系 | 策略版本/依赖链 | 递归 CTE |

> **禁 JSONB 的理由**：知识库要可查/可索引/可复利，JSONB 三点全削弱，且诱发 app 层 `json.Marshal`（违 CLAUDE.md 零 JSON 铁律，JSONB 豁免仅限"DB 管、不 Marshal"，知识库不该走这条险路）。知识=结构化数据→必须类型化列；真无法结构化的语义→pgvector 向量列。**全方案零 JSONB、零 app Marshal，无需豁免。**

### ❌ 否决项（+ 第一性理由）

| 否决 | 理由 |
|---|---|
| 独立向量库（Pinecone/Milvus/Weaviate）| 违反**单存储 ADR-0012**；运维 ×N；pre-revenue 无专职运维；当前向量量级（万级）pgvector 绰绰有余，向量库在**十亿级**才必拆（Y3+ 问题）|
| 图库（Neo4j）| 过度工程；PG 关系+递归 CTE 够用 |
| **LLM-as-KB**（把知识塞 LLM 上下文）| 非确定、不可复用、不复利、每次 token；**违反复利原则**。LLM 只补"无法结构化的语义缺口" |
| Polyglot（多库混搭）| 一致性/备份/同步成本 ×N，pre-revenue 致命 |
| 纯关系（无 pgvector）| 缺语义检索（K4 策略相似），AI 学习层做不起来 |

**结论**：PG-with-extensions 不是"凑合"，是**当前+中期最优**且**第一性一致**（单存储/push-first/no-app-JSON）。语义层到十亿级再拆向量库（远 post-revenue）。

## 3. 原则：**确定性优先**（你的 3c 普适化）

- **结构化/确定性知识为主**（K1/K3 + K4 的可结构化部分）：可复用、确定、零 token、复利。**这是主干。**
- **LLM 仅补语义缺口**：K4 中真无法结构化的（策略意图相似、失败原因推理）才用 LLM/embedding，且结果**尽量沉淀回结构化**（LLM 一次性产、KB 永久存），不让 LLM 成为运行时依赖。
- 推论：compat proposal 当年为"常量推断"引入 LLM 的设计，应**优先改成确定性查表**（K1 兜住绝大多数），LLM 退到兜底。

## 4. 架构：KB 服务层（in-process，复用既有模式）

```
生产者                          知识库（PG 单库）                   消费者
─────────                       ──────────────                     ─────────
诚实性分析 ──┐                 kb_compat_fact  ──→ 编译器/VM(L0 解析)
L0 修复规则 ─┤─write→        kb_compat_fix   ──→ agent(生成避开)
人工核验 ───┘                  kb_demand_signal──→ 路线图/补全排序
实盘/回测 ────write(既有)→     trade_records   ──→ 衰减检测/市场/AI 反馈
决策闭环 LEARN ─write→        live_performance
agent 结果 ───write→           kb_ai_learning  ──→ agent(相似检索/避坑)
                                  (pgvector)
                       ↘ LISTEN/NOTIFY (push) ↗ 触发对应消费者循环
```

- **KB Service**（`internal/knowledgebase/`）：read API（lookup/similarity/aggregate）+ write API（record fact/fix/demand/outcome/learning）。in-process 调用，proto only（跨进程时）。
- **统一即价值**：把 constants.go/api_registry/compat_fixes/trade_records/agent memory **收编成 KB 的域表/视图**，不再是散件孤岛。
- **写路径 push-first**：新知识写入 → NOTIFY → 对应消费者循环触发（衰减检测/agent 重训/路线图刷新），**禁轮询**。

## 5. 自我进化循环（穿过 KB 才算"实处"）

| 循环 | 检测→沉淀→复用→复利 | 现状 |
|---|---|---|
| **C1 兼容循环** | EA 命中未知→记 K3 需求+K1 盲区→有确定性修复则 L0 即时复用；无则攒需求→阈值触发补 K1→后续 EA 受益 | L0 已起跑；K3 捕获待建 |
| **C2 战绩循环** | 策略跑→K2 战绩累积→衰减检测→作者/AI 迭代→新版本→新战绩 | FEAT-5 部分建（衰减检测✅，迭代闭环✅）|
| **C3 Agent-RAG 循环**（⚠️ 非 0-token）| agent 生成前 RAG 检索 K1/K2/K4 → inject 进 prompt → 生成更优 → 结果写回 K4。**LLM 不学 KB（无状态）；复利在 KB+检索，不在模型权重**。每次 inject 花 token，漂移降不零→靠诚实管线重验兜底 | agent memory 是种；未接 K1/K2 + 未用 pgvector 相似 |

> **⚠️ 关键澄清（纠"AI 学习"误解，2026-08-08 用户追问）**：厂商 LLM 每次调用无状态、**不学习**我们的 KB。"越用越聪明"有两套、勿混：
> - **确定性复利（0-token/0-drift，皇冠 = 用户 3c）**：K1 兼容知识 + L0 修复，由**编译器/VM/Gate 消费，不经过 LLM**。这才是真正的 0-token/0-drift。
> - **Agent-RAG 复利（花 token，漂移降不零）**：agent 经 LLM 用 KB（RAG），有价值但非 0-token。
> **原则：最大化确定性层（能进 K1/L0 的不留给 LLM）；agent-RAG 只补确定性够不着的语义/生成。** 三循环"写回 KB+下次读 KB"= 复利的充要条件，也是验自我进化是否真落地的判据。

## 6. 第一性对账

| 项目约束 | 本方案 |
|---|---|
| 单存储（ADR-0012）| ✅ PG 单库，无新 DB |
| Push-first（禁轮询）| ✅ LISTEN/NOTIFY 触发循环 |
| 无 JSON（零容忍）| ✅ 全 proto 交换 + 纯类型化 PG 列 + pgvector 向量列；**零 JSONB、零 app Marshal，无需豁免** |
| Decimal 精度 | ✅ 战绩指标 decimal |
| 不碰用户资金/不自营 | ✅ 知识库只读+聚合，不交易 |
| 信任护城河 | ✅ KB = 护城河工程实体 |
| 复利/防抄 | ✅ 结构化知识累积，对手抄不走 |

## 7. 分阶段（落到实处路径）

| 阶段 | 内容 | 价值 |
|---|---|---|
| **P0 收编（统一散件）** | constants.go/api_registry/compat_fixes → `kb_compat_fact`/`kb_compat_fix` 域表 + KB Service read/write；trade_records/performance 挂为 K2 视图 | 消孤岛，成资产 |
| **P1 补 K3（需求捕获）** | 决策闭环 LEARN：被堵的不支持特性 → `kb_demand_signal` 计数；NOTIFY 路线图 | C1 闭环 |
| **P2 C1 兼容循环跑通** | L0 读 `kb_compat_fix` 即时复用（已部分）+ 需求阈值触发补 K1 | 越用越准 |
| **P3 C3 AI 学习（pgvector）** | `kb_ai_learning` + pgvector；agent 生成前相似检索（避坑/复制）| agent 越用越聪明 |
| **(C2 战绩循环已部分在 FEAT-5，补全接 KB)** | 衰减/迭代读写 K2 | 战绩复利 |

P0-P1 优先（统一 + 补 K3），P2-P3 跟进。**P0 是把现有散件升级成资产的关键一步，本身即高价值。**

## 8. 为什么这是最优解（收口）

1. **第一性一致**：单存储/push-first/确定性优先——无一处为"知识库"破例。
2. **最小代价最大复利**：不加 DB、不加运维，把 PG 已有能力（pgvector/JSONB/LISTEN）用到极致；每跑一个 EA/策略都让 KB 长一寸。
3. **统一散件**：compat/战绩/agent memory/需求 收成一个资产，而非多一个孤岛。
4. **可演进**：语义层到量级再拆向量库，路径预留，不锁死。
5. **战略对齐**：KB = trust 护城河本体 = 最防抄资产。把"VM 是信任技术根 + trust 是护城河 + 系统越用越聪明"三线收成一个工程实体。

---

> **✅ 已确认（2026-08-08 用户拍板）**：① 底座 PG+pgvector（否决独立向量库）；② 确定性优先（结构化主、LLM 补语义）；③ P0 先做收编散件。**P0 施工 spec 已出**：`docs/spec/kb-p0-consolidate-spec.md`（收编 constants.go/api_registry/compat_fixes → kb_* 域表 + KB Service + 内存缓存/LISTEN + 编译器经 KB 解析）。P1=K3 需求捕获、P3=agent-RAG(pgvector)。
