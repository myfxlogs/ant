# 施工方标准操作流程（Builder SOP）

> **谁读**：施工方 agent（Windsurf Cascade / Cursor / Codex / 任何执行修复的 agent）。
> **怎么用**：每次接到修复任务，按 Phase 0→1→2→3 执行。AGENTS.md 的「协作模式」段是本文的精简版（自动注入），本文是详细版（按需读）。
> **关联**：`AGENTS.md`（技术约束 + 精简协作纪律）、`docs/audits/tech-debt-registry.md`（债务总账 = 你的工作台）、`docs/audits/handover-audit-plan.md`（审计全局进度）。

---

## 0. 心智模型：为什么有这套流程

本项目是**碰钱系统**（策略市场，实盘战绩公开）。CLAUDE.md 的核心约束：「执行语义永不全自动改」「因困难而妥协最优解视为违规」。所以分工：

- **审计方（Claude Code）**：只读、验证、记录、出 spec。它代码级定位根因，但不直接改（保持独立判断 + 省 token）。
- **施工方（你）**：动手改。但你的自由度被这套流程约束——**不是给你一个 bug 你自由发挥，而是给你一个已定位的根因 + 验收标准，你精准修 + 如实回填**。

**铁律**：发现可以自动（防线抓 bug），修复必须受约束（人/流程把关）。你的每次施工都要**可追溯、可验收、可同步**。

---

## Phase 0 ｜ 动工前对账（不读就动工 = 失职）

### 0.1 必读链（按顺序）

1. **`docs/audits/tech-debt-registry.md`** → 找到你的任务条目（如 `BT-6`），读全：
   - **根因**（审计方的定位，可能标"假设"——你的首要任务之一是验证/修正它）
   - **位置**（`文件:行号`）
   - **状态**（应为 `🟦open`）
   - **修复方向**（审计方的建议，非命令——你可以证伪）
   - **验收标准**（怎么算修完）
2. **`docs/audits/handover-audit-plan.md`** → 知道这条管线审计到哪、有哪些相邻发现（避免你改 A 碰坏 B）。
3. **`AGENTS.md` / `CLAUDE.md`** → 技术约束（ConnectRPC-only / 禁 json / Decimal / push-first / 确定性 / file-lines 红线）。

### 0.2 根因优先（Root-Cause-First，CLAUDE.md 最高优先级）

动工前必须查历史，**禁止看到问题就重写**：

```bash
git log --all --oneline -- <出问题的文件路径>     # 该文件所有变更历史
git blame <文件> -L <行>,<行>                     # 出问题的代码是谁、哪个 commit、为什么引入
git show <commit> --stat                          # 读原 commit message + 关联 ADR/spec，理解原始设计意图
```

判断：
- **后续修改引入的 bug** → 精准修复那个变更，保留原始设计。
- **有意移除的功能** → 先在 registry 条目下讨论是否恢复，不擅自恢复。
- **从未实现过** → 才可从零写（且 registry 条目应已说明）。

> **红线**：把以前的**最优解**替换成"我觉得更好的"新实现——尤其当原实现有 ADR/spec/复杂边界处理时——视为违规。但如果原实现确实非最优（违反 CLAUDE.md Part 0.1），替换它不违规，**前提是已通过 git log/blame 理解了原设计**。

### 0.3 复用核对（Reuse Preflight，CLAUDE.md 强制）

动工任何**新 file/function** 前：

```bash
bash scripts/cap.sh <动词>          # 查能力，多换几个词（动词 + 别名）
bash scripts/cap.sh <符号名>
```

PR 描述里逐条给结论：`REUSE: <symbol> @ <file:line>`（复用现成）或 `NEW: 无现成能力（已搜：<关键词>）`（确认真空白）。**缺少 `REUSE:`/`NEW:` = 该任务判失败。**

---

## Phase 1 ｜ 施工纪律

### 1.1 最小改动 + 风格对齐

- 改动范围 = 任务条目定义的范围。**one task = one scope**，不顺手重构无关代码（CLAUDE.md 禁 cross-scope）。
- 匹配周围代码的命名、注释密度、惯用法。新代码要"读起来像周围代码写的"。
- file-lines 硬红线：Go >450 行、TS >375 行必须拆分。检查：`cd backend && go run ./tools/check-file-lines --strict`（0🔴）。

### 1.2 测试（正 / 负 / 边界）

每个修复必须带测试，三类齐全：
- **正例**：合法情况通过（确保你没误伤正常路径）。
- **负例**：构造违反/触发 bug 的输入，确认被抓/被修（这是回归守护的核心）。
- **边界**：容差临界、空值、并发、重试。

### 1.3 确定性优先（回测/编译器/任何有序 pipeline）

- ❌ 禁止 `time.Now()` 生成测试 bar timestamp（违反 spec 21 §10 Determinism Contract）。用固定 epoch：`time.Date(2024, 1, 1, 0, i, 0, 0, time.UTC)`。
- ❌ 禁止裸遍历 Go map 处理有序依赖（编译器/链接器）。有前向引用 → 两遍编译（Pass 1 预注册，Pass 2 编译体）；无前向引用但需确定性 → 排序 key 后遍历。Go map 迭代随机性是 **per-invocation**。
- ❌ 禁止未固定 seed 的随机（`rand.Seed(time.Now())`）。用固定 seed 或 clock-derived。
- 详见 `AGENTS.md`「MQL2GO VM Pitfalls」（通用规律，非 mql2go 专属）。

### 1.4 对抗证明（关键，否则测试无效）

修完，**删掉你修复的关键一行**，跑测试，证明它**必红**。如果删了还绿，说明你的测试根本没覆盖这个修复 = 测试无效 = 任务未完成。

> BT-6 范例：删掉两遍编译的 Pass 1 预注册，`TestParamPipeline_FloatDefaultParam` 必然偶发 FAIL（volume=0）。这就是对抗证明。

### 1.5 构建检查（完工前必过）

```bash
go build ./...                                       # 必须通过
cd backend && go run ./tools/check-file-lines --strict   # 0🔴（🟡🟢 通过）
bash scripts/gen_capability_map.sh                   # 若新增/改了能力，刷新 docs/CAPABILITIES.md
```

---

## Phase 2 ｜ 进度回填（这是"同步"的核心，不做 = 未完成）

完工后**必须**回填以下文档。这是让审计方/下一个 agent 跟你同步的唯一手段——**代码改了但文档没回填 = 等于没做**（别人不知道你干了什么）。

### 2.1 回填 `docs/audits/tech-debt-registry.md`（必做）

找到你的任务条目：
- **状态列**：`🟦open` → `✅done`（标日期，如 `✅done（2026-08-07 修复）`）。
- **条目末尾追加**（不要删原内容，追加）：
  - **真实根因**：你查到的实际根因。**若与审计方假设不同，如实写明并指出差异**（这是高价值信息——审计方假设可能偏，你深挖到的真根因纠偏了它）。
  - **修复方式**：改了哪些文件/函数，核心改动一句话。
  - **对抗证明**：删哪一行测试必红。
  - **测试结果**：量化（如"50 次连跑 0 失败"、"build + 测试绿、file-lines 0🔴"）。
- **只改状态列 + 追加补充**。不删条目、不改审计方的事实陈述（保留决策轨迹）。

### 2.2 沉淀 pitfall（若是普遍规律，必做）

如果你的修复揭示了一个**普遍规律**（不只这一个 bug，而是一类陷阱），写进 `AGENTS.md` / `CLAUDE.md` 的对应 Pitfalls 段，成永久约束：
- BT-6 范例：map 迭代非确定 → 写进「MQL2GO VM Pitfalls」+ 通用规则"编译器/链接器禁止裸遍历 map 处理有序依赖"。
- 这防止下一个 agent 重犯同类错。

### 2.3 回填 `docs/audits/handover-audit-plan.md` 变更日志（必做）

文件末尾「变更日志」加一行：日期 + 改了什么 + 根因（一句话）。

### 2.4 发现新 gap → 新增条目（不要塞现有条目）

施工中若发现**新的、registry 里没有的问题**，在 registry 新增一条（`🟦open` + 根因/位置/修复方向）。不要把它塞进你正在修的条目（会让那条变臃肿、状态语义混乱）。

### 2.5 禁止碎片化（单一事实源）

- ❌ 禁止新建并行进度文档（另起 `progress.md` / `my-notes.md` / `fix-log.md` 之类）。所有进度只在 registry + 审计计划 + memory 三层。
- ❌ 禁止把进度记在代码注释里当文档（注释是给读代码的人，不是进度账本）。

---

## Phase 3 ｜ 交接验收

### 3.1 汇报格式

完工汇报给审计方，包含：
1. **改了哪些文件**（列表）。
2. **真实根因**（一句话）。
3. **修复方式**（核心改动）。
4. **测试结果 + 对抗证明**（量化 + 删哪行必红）。
5. **回填了哪些文档**（registry 条目状态 / pitfall 段 / 审计日志）。

### 3.2 不越权验收

- **你不自行宣告"完成"**。你回填 `✅done` 是"施工方自评完成"，但**最终 `✅done` 的权威性来自审计方核对状态 + 实测**。
- 审计方可能：实测你的修复、核对根因描述、质疑你的修复方式。这是正常的质量门，不是不信任。
- 状态语义：`❓待核` = 记录过未对账当前代码 / `🟦open` = 已核验仍存在 / `✅done` = 已修且经审计方验收。

---

## 红线（违反 = 任务判失败）

| ❌ 红线 | 为什么 |
|---|---|
| 不读 registry/审计计划就动工 | 会重复造轮子、偏离已定位的根因 |
| 不 `git log`/`blame` 就重写 | 丢失历史 bug 修复、设计退化 |
| 只改代码不回填 registry 状态 | 别人无法同步，等于没做 |
| 新建并行进度文档 | 碎片化事实源，信息散落 |
| "标记 legacy / 沉默代替修复 / 回退代替重新生成" | CLAUDE.md 明令禁止的快捷方式 |
| 自行宣告完成 | 越权验收，绕过质量门 |
| 顺手改无关代码 | one task = one scope，cross-scope 违规 |
| 用 `time.Now()` / 未固定 seed / 裸 map 迭代 | 确定性违规，制造 flaky |

---

## 验收 checklist（施工方自检，汇报前过一遍）

- [ ] git log/blame 理解了原设计，确认是精准修复而非重写
- [ ] cap.sh 复用核对，PR 有 REUSE:/NEW: 结论
- [ ] 测试正/负/边界齐全
- [ ] 对抗证明：删关键行测试必红
- [ ] `go build ./...` 通过
- [ ] `check-file-lines --strict` 0🔴
- [ ] registry 条目状态 🟦→✅ + 追加真实根因/修复方式/对抗证明/测试结果
- [ ] 普遍 pitfall 已沉淀进 AGENTS.md/CLAUDE.md
- [ ] 审计计划变更日志加了一行
- [ ] 没新建并行文档
- [ ] 没自行宣告完成（等审计方验收）

---

## 标准回填范例（BT-6，照这个学）

**审计方原始条目**（`tech-debt-registry.md` BT-6，节选）：
> 参数链端到端测试 flaky + volume=0 偶发复现。flaky 根因：`makeE2EBars` 用 `time.Now().Add()` → 违反 Determinism Contract... 修复方向：① `makeE2EBars` 固定 timestamp；② flaky 消除后定位 volume=0 偶发根因（VM 参数注入 extern vs input...）。状态 `🟦open`。

**施工方回填后**（状态 + 追加，原内容保留）：
> ...状态：`✅done（2026-08-07 修复：根因 = ir.Funcs 是 map[string]*FuncDef，compile.go:51 遍历该 map 编译用户函数，Go map 迭代序非确定 → 当 caller(CheckForOpen) 先于 callee(LotsOptimized) 编译时，compileCall 在 bc.Funcs 找不到 callee → 落入 "unknown function" 盲区 → 返回值静默替换为 NoneVal(=0) → OrderSend(...,LotsOptimized(),...) = OrderSend(...,0,...) → volume=0。修复：两遍编译——第一遍预注册所有用户函数 entry PC（emit OP_ENTER_FUNC + 写 bc.Funcs），第二遍编译函数体。前向引用在第一遍后全部可解析。50 次连跑 0 失败。compile.go 单文件改动。注：makeE2EBars 的 time.Now() timestamp 问题仍存在（确定性违规），但非本次根因——指标计算只用 OHLCV 不用绝对时间。makeE2EBars 固定 timestamp 作为后续 cleanup。）`

**这个范例体现了所有要点**：
- ✅ 状态 🟦→✅ + 标日期
- ✅ **真实根因纠偏了审计方假设**（审计方猜 time.Now + extern/input，真根因是 map 迭代）——如实写明
- ✅ 修复方式具体（compile.go 两遍编译）
- ✅ 对抗证明 + 测试结果量化（50 次 0 失败）
- ✅ 保留审计方原内容（不删），追加而非覆盖
- ✅ 附带发现（makeE2EBars time.Now 仍存，作为后续 cleanup 记录，不混入本次修复）

**对应的 pitfall 沉淀**：`AGENTS.md`「MQL2GO VM Pitfalls」加了一条「Go map 迭代序非确定 → 用户函数前向引用静默返回 0」+ 通用规则。

---

## 常见错误（Windsurf Cascade 等 agentic 工具的典型毛病）

| 毛病 | 对策（本 SOP 的哪条拦它） |
|---|---|
| Cascade 爱顺手改一堆无关代码 | Phase 1.1「one task = one scope」+ 红线 |
| 改完就完，不记文档 | Phase 2「完工必须回填，否则判失败」 |
| 不查历史就重写整个模块 | Phase 0.2「根因优先」+ 红线 |
| 测试用 time.Now/随机，flaky 假绿 | Phase 1.3「确定性优先」+ 红线 |
| 自行宣告"已修复完成" | Phase 3.2「不越权验收」 |
| 新建 my-fix-log.md 记自己的进度 | Phase 2.5「禁止碎片化」+ 红线 |
