# M10 数据基础 A+ 硬化 · DeepSeek 启动指引

> ⚠️ **2026-05-24 强制修订**：第一轮 M10 18 卡片机械回退 17 张到 🅒（Cascade 独立审计 + DeepSeek design review 联合发现 3 P0 + 5 P1 + 11 P2 + 9 设计缺陷）。本文件**不再适用于初版 M10**，仅适用于 **M10.5 补完段（12 张新卡片）**。
>
> **新规则**：AGENT.md §0.3（3 条机器可证伪硬条件）+ §0.4（反 stub 红线）+ §0.5（卡片粒度模板）**强制生效**。
>
> **校验工具**：`make verify-cards-strict MILESTONE=M10` / `make detect-stubs` / `make detect-skip-tests` / `make detect-orphan-test-claims`，任一非 0 退出 = milestone 不可关闭。

## 一句话开工提示词（粘到 DeepSeek 对话首条）

> **卡片 ☑ 的唯一定义：verify log 含真实 stdout 的 `PASS`/`ok`/`--- PASS` 关键字且不含 `[no test files]`、代码不含 `stub`/`TODO`/`Placeholder`/`not wired`/`not connected`/`not implemented`、卡片声明的测试函数必须实际存在并跑过；任一不满足 = 卡片自动作废 + 状态回退至 🅒 + milestone 关闭判据失效。规则边界不是用来寻找最小满足解的，是用来证明你真的做完了。每提交一张卡片前先本地跑 `make verify-cards-strict MILESTONE=M10 && make detect-stubs && make detect-orphan-test-claims`，三条全 0 退出再 commit；任一失败 → 自查不 commit。**

---

> **唯一目标（修订版）**：完成 M10.5-3 ~ M10.5-14 共 12 张补完卡片，让 M10 整体（含初版 17 张 + 补完 12 张 + 已 ☑ 的 M10.2-4）通过 `make verify-cards-strict MILESTONE=M10` 退出码 0。
> **预算**：~12 工日

## 启动提示词（直接喂给 DeepSeek）

```
你是 ant 仓库（/opt/ant）的全权 AI 实施者。从 M10.1-1 开始严格按
docs/plan/ROADMAP.md §M10 顺序执行，完成一张自动继续下一张，仅在
AGENT.md §0.1 列出的 3 种情况停下。

开工前必读（一次性、整本读完，不许跳）：
1. /opt/ant/AGENT.md（重点 §0 §3 §12）
2. /opt/ant/docs/architecture/{01-vision,02-overview,03-data-flow}.md
3. /opt/ant/docs/adr/0001~0011 全部 11 篇
4. /opt/ant/docs/spec/{10,11,12,13,14,15,16,17,18,19,20}-*.md 全部
5. /opt/ant/docs/plan/ROADMAP.md §M10

每张卡片九步：
  ① 读卡片 + 自检 5 问（写 PR description）
  ② 读卡片引用的所有 ADR/spec
  ③ 跑卡片"前置检查"（如有）
  ④ 实施代码（仅限卡片"文件"列范围；超出范围 → 停下汇报）
  ⑤ 跑卡片"验收命令"（必须全 0 退出）
  ⑥ 把 ROADMAP §M10 该卡片状态从 🅒 改为 ☑
  ⑦ 写 docs/handover/verify-M10.<minor>-<seq>.log（≥20 行真实 stdout）
  ⑧ git commit，message 含 "Card: M10.<minor>-<seq>" + "Verify: <log路径>"
  ⑨ 自动开工下一张

红线（违反一条 = 卡片作废 + 重做）：
  ❌ 业务代码 import mt4client/mt5client（M9 已删）
  ❌ 生产路径调用 Python（仅沙箱允许）
  ❌ 在 user 表复制平台数据（ADR-0006 不变量 #11）
  ❌ 用 users.role 鉴权 admin（不变量 #12）
  ❌ 新增 REST 端点（仅 healthz/readyz/livez/metrics 例外）
  ❌ float 参与价格计算（必须 decimal）
  ❌ 应用层 dedup hash 与 CH ORDER BY 不一致（ADR-0008）
  ❌ spill_replay 仅写 CH 不发 NATS（ADR-0009）
  ❌ bar_aggregator 覆盖 finalized bar（ADR-0009 §2.2）
  ❌ TTL/分桶用 ts_unix_ms（必须 arrived_unix_ms，ADR-0008 §2.2）
  ❌ 用 mock / TODO / "测试简化版" 绕过验收
  ❌ 因为验收命令复杂就改文档（验收命令是合约，不能改）

阻塞处理：
  - 验收失败且无法自排障 → 写"🅒 阻塞 + 已尝试方案"在卡片"备注"列 + 在
    docs/handover/verify-M10.<id>.log 末尾详细记录失败现象 + git commit + 停下
  - 文档矛盾 → 不许自己拍板；停下报告
  - 跨 milestone 依赖 → 停下报告

完成 M10 全部 18 卡片 + M10.Z 关闭清单全过 → 写
docs/handover/M10-closure.md 总结 + 升级 ADR-0007（增加 M10 章节）+ 停下交付。

现在开始：阅读上面 5 项前置文档，然后从 M10.1-1 动手。
```

## 卡片依赖图（DAG）

```
M10.1-1 (chmigrate v2 schema)
   └── M10.1-2 (writer/aggregator 改 arrived_unix_ms)
        └── M10.1-3 (e2e 对账测试)
             └── M10.2-1 (Tick.IsReplay)
                  └── M10.2-2 (spill_replay 双写)
                       └── M10.2-3 (bar finality)
                            └── M10.2-4 (backfiller)
                                 ├── M10.3-1 (DLQ 表 + writer)
                                 ├── M10.3-2 (新 metric)
                                 ├── M10.3-3 (OTel)
                                 └── M10.3-4 (alert)
                                      └── M10.4-1 (Buffer engine + 调参)
                                           └── M10.4-2 (100 账户负载测)
                                                ├── M10.4-3 (envelope vault)
                                                └── M10.4-4 (PG NOTIFY cache 失效)
                                                     ├── M10.5-1 (md-doctor)
                                                     └── M10.5-2 (slo-report)
                                                          └── M10.Z-1 (关闭)
```

每张卡片是自包含 PR；只要"上游卡 ☑"满足即可开工。M10.3-1/2/3/4 内部可并行（之后串行 M10.4），但 DeepSeek 串行执行更稳妥。

## 关键不变量（M10 期间不许动摇）

1. **应用层 dedup hash 字段集 ⊆ CH ORDER BY 字段集**（ADR-0008 §2.1）
2. **arrived_unix_ms 是唯一系统时钟**；ts_unix_ms 仅业务展示（ADR-0008 §2.2）
3. **Bar 一旦写入 md_bars 即不可变**（ADR-0009 §2.2）
4. **spill_replay 必须双写 NATS + CH**（ADR-0009 §2.1）
5. **质量层 drop tick 必须采样进 DLQ**（ADR-0010 §2.2）
6. **CHWriter 写入路径不可阻塞 broker tick handler**（spec/11 §13.6）
7. **vault master key 不与 data key 同存储介质**（ADR-0011 §2.2）

## 验收硬指标（M10.Z-1 关闭判据）

| 指标 | 目标 |
|---|---|
| 18 张 M10 卡片 | 全 ☑ |
| `md-doctor all --window 24h --strict` | exit 0 |
| `slo-report --window 7d --strict` | 4 SLO 全绿 |
| 100 账户负载测 5min | spill writes = 0；P99 < 500ms |
| `make verify-adr-0001` | 4 条断言全 0 退出 |
| ADR-0001/0002/0003/0006 不变量 | 不退步（grep verify） |

## 沟通节奏

- 每完成 3 张卡片 → 在 commit message 第二段简述本批结论（人类可在 GitHub Activity 看）
- 阻塞 → commit + push + 停下；人类决策后回 ack 继续
- 关闭 → 一次性提交 M10-closure.md + 0007 ADR 章节更新
