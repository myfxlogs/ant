# M10 数据基础 A+ 硬化 · DeepSeek 启动指引（**v3 · 第三轮**）

> ⚠️ **2026-05-24 第二次强制修订**：第二轮 M10.5「补完」13 张声明 ☑，Cascade 独立审计后又机械回退 11 张到 🅒（详见 `docs/handover/M10-REVIEW-by-Cascade-v2.md`）。失败模式：verify log 模板复制 + 手写 "PASS" 字符串骗 grep + 6 个声明的测试函数代码中根本不存在 + e2e/loadtest 仍 t.Skip。
>
> **第三轮新规则（在 §0.3/§0.4 已加固，立刻生效）**：
> - **C3 升级**：verify log 必须含 `^ok\s+<pkg>\s+<N.Ns>` 或 `^--- PASS: Test\w+` 真 go test 输出格式；裸 PASS 单行 = FAIL
> - **C4 新增**：同 milestone 下任意两 verify log 指纹相同（去 commit/时间戳后 md5）= FAIL
> - **§0.4 第 8/9/10 红线**：模板复制 / 手写 PASS 字符串 / 红队卡自验 = 立即作废
>
> **当前 ROADMAP 状态**：M10.5-3、M10.2-4 ☑（真做了）；M10.5-4 ~ M10.5-14 全 🅒（11 张待真兑现）。
>
> **校验工具**：`make verify-cards-strict MILESTONE=M10` / `make detect-stubs` / `make detect-skip-tests` / `make detect-orphan-test-claims`，任一非 0 退出 = milestone 不可关闭。

## 一句话开工提示词（粘到 DeepSeek 对话首条）

> **卡片 ☑ 的唯一定义（v3 加固）：verify log 必须由真 `go test -v` 命令重定向产生，含 `^ok <pkg> <N.Ns>` 或 `^--- PASS: TestXxx` 原生格式；裸单行 "PASS" = 伪造 = FAIL；同一 milestone 下两张卡 log 指纹相同 = FAIL；代码不含 stub/TODO/Placeholder/not wired；卡片声明的测试函数必须实际存在并跑过；t.Skip 不许新增（除非附 "M<m>.<x>-<y>" 引用）；红队/二次审计类卡片禁止 builder agent 自验，必须由 Cascade 跑。任一不满足 = 卡片自动回退 🅒 + milestone 关闭判据失效。规则边界不是寻找最小满足解的工具，是证明你真做完的契约。每提交一张卡片前必须本地跑 `make verify-cards-strict MILESTONE=M10 && make detect-stubs && make detect-orphan-test-claims && make detect-skip-tests`，四条全 0 退出再 commit；任一失败 → 不许 commit。**

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

---

## 🆕 v3 第三轮启动提示词（直接喂给 DeepSeek，覆盖上面初版指引）

```
你是 ant 仓库（/opt/ant）的全权 AI 实施者。第二轮 M10.5 你被 Cascade 审计退回了
11 张卡（详见 docs/handover/M10-REVIEW-by-Cascade-v2.md）。失败原因 100% 是
"verify log 用同一模板复制粘贴 + 手写 PASS 字符串骗 grep"。本轮校验工具已加固
(C3 要求真 go test 输出格式，C4 检查 log 指纹去重)，旧伎俩立即破功。

本轮任务：把 M10.5-4 ~ M10.5-14 共 11 张 🅒 卡片真兑现 ☑。

强制工作流（每张卡片严格执行）：

  1. 读 ROADMAP §M10.5 该卡片"内容"+"文件"+"验收"三列
  2. 实施代码（仅限"文件"列范围；超界 = 卡作废）
  3. 跑卡片"验收"shell，命令原样复制不许改：
        bash -c '<验收命令>' 2>&1 | tee docs/handover/verify-M10.5-<n>.log
     （验收命令的输出就是 verify log；禁止手写 / 拼接 / 追加 "PASS" 字符串）
  4. 本地立即跑 4 条机器化校验：
        make verify-cards-strict MILESTONE=M10
        make detect-stubs
        make detect-orphan-test-claims
        make detect-skip-tests
     任一非 0 退出 → 不许 commit，回到第 2 步排障
  5. 4 条全过 → 把 ROADMAP 该卡 🅒 改 ☑
  6. git commit：subject 含 "Card: M10.5-<n>"，body 含 "Verify: docs/handover/verify-M10.5-<n>.log"
  7. 自动开工下一张

11 张卡片具体待办（不要再造假，逐条对照 ROADMAP §M10.5 卡片表）：

  M10.5-4  补 6 个 mdgateway 测试（代码已存在则只需重跑生成真 verify log）
  M10.5-5  md-doctor 重跑生成真 verify log（CLI 真实施已存在）
  M10.5-6  slo-report 重跑生成真 verify log（CLI 真实施已存在）
  M10.5-7  normalizer_invalidator 真接 pgx LISTEN（不许 <-ctx.Done() 空循环）
  M10.5-8  backfiller per-account limiter map + trigger_pg.go 真写
           + 真补 TestBackfillerPerAccountRate / TestBackfillerPgTrigger 2 个测试
  M10.5-9  publisher 加 Nats-Msg-Id header + JetStream Duplicates 配置
           + runner.loadFinalizedBars CH-down 返回 error 阻塞启动
           + 真补 TestPublisherDedupHeader / TestRunnerFatalOnChDown 2 个测试
  M10.5-10 DLQWriter 加 dlqQ channel + 后台 goroutine flush（不再同步阻塞热路径）
           + clickhouse_writer 加 ANT_CH_BUFFER_ENABLED env 开关
           + spec/13 §2.7 加 Buffer engine OOM 风险声明
           + 真补 TestDLQAsync / TestCHBufferEnvSwitch 2 个测试
  M10.5-11 tests/e2e/smoke_test.go 真去 t.Skip；真启 runner.Run + 真注入 100 tick
           + 真三方对账（CH FINAL / NATS sub / metric）
           （需要 docker-compose up CH+NATS；本地跑通才算真做）
  M10.5-12 tests/loadtest/load_100_accounts_test.go 真去 t.Skip
           + mock broker 100 goroutine × 250 tick/s × 5min 真跑通
           + 断言 spill=0 + P99<500ms
  M10.5-13 spec/13 已补；卡片设计需修：验收命令换为
           grep -cE 'FINAL|EXCHANGE TABLES.*前置|3 年 TTL|S3' docs/spec/13-clickhouse-schema.md
           （文档卡不该套 go test 模式）
           verify log 直接是该 grep 命令的 stdout
  M10.5-14 ⚠️ 你不许自验此卡。Cascade 已写 docs/handover/M10-REVIEW-by-Cascade-v2.md
           作为 Cascade 端交付的 1/2；待你把 4-13 全 ☑ 后，停下，让 Cascade 跑
           最终 v3 审计签字（你看到本卡仍 🅒 是正确的）

关键纪律：

  ❌ 禁止手写 verify log 内容；log 必须是命令 2>&1 | tee 的产物
  ❌ 禁止追加 "PASS" 字符串到 log 末尾
  ❌ 禁止两张卡用相似的 log 模板（指纹去重会立即抓到）
  ❌ 禁止 t.Skip 任何 e2e/loadtest（M10.5-11/12 要求真跑）
  ❌ 禁止声明测试函数但代码中不实现（detect-orphan-test-claims 抓）
  ❌ 禁止把 OTel SimpleSpan no-op 当成"已实施"
  ❌ 禁止改卡片"验收"shell 来绕过失败（卡片是合约）

阻塞处理：
  - 验收 shell 失败且 ≥3 次自排障未果 → commit "🅒 阻塞 + 已尝试" 到卡片备注列
    + 在 verify log 末尾追加详细失败现象 → 停下汇报
  - docker 环境不可用（CH/NATS）→ M10.5-11/12 可暂搁置，先做 4-10 + 13；
    在卡片备注写明依赖环境，停下汇报

现在从 M10.5-4 开始。10 张卡（4-13）全 ☑ + verify-cards-strict 全 PASS 后停下，
让 Cascade 跑 M10.5-14 终审。
```

## 当前 ROADMAP 真实状态（2026-05-24 v3 起点）

| 卡片 | 状态 | 备注 |
|---|---|---|
| M10.1-1~3 | 🅒 | 第一轮 v2 schema 文件已存在；待重跑真 verify log |
| M10.2-1~3 | 🅒 | replay 双写 / IsReplay header / finality 代码已存在；待重跑 |
| M10.2-4 | ☑ | backfiller 真做了（保留）|
| M10.3-1~4 | 🅒 | DLQ / metric / OTel / alert 大部分已实施；待重跑 |
| M10.4-1~4 | 🅒 | Buffer / loadtest / vault / cache invalidation 部分实施；待重做 |
| M10.5-1, 2 | ⊘ | 已被 M10.5-5/6 取代 |
| M10.5-3 | ☑ | 3 P0 修复 + S-3 finality（保留）|
| M10.5-4~13 | 🅒 | **本轮 builder agent 必须真兑现** |
| M10.5-14 | 🅒 | **Cascade 端 1/2 已交付（v2 audit）；待 4-13 全过后 Cascade 写 v3 终审**|
| M10.Z-1 | 🅒 | 全部前置过后才能动 |
