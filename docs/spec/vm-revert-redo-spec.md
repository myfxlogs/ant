# Spec：VM 返工批重新施工（D-REVERT-SCOPE-DRIFT-001）

> **状态**：🟦open（派工中，待 Devin IDE 施工 + Devin CLI 验收）
> **基线**：HEAD `564e6d03`（2026-08-26）
> **施工提示词**：`docs/audits/builder-handoff-vm-revert-redo-2026-08-26.md`
> **registry 条目**：`D-REVERT-SCOPE-DRIFT-001`（`docs/audits/tech-debt-registry.md:1953`）

## 1. 问题陈述

commit `830b2c79` 的 revert 实际范围远超 commit message 描述，把 `acaa86db` 引入的几乎所有 VM 返工工作（round 1-5，约 15 个 ID）都 revert 了。8 个 ID 的修复代码已不存在，registry 状态从"施工完成待复审"降级回"🟦open（待施工）"。

**影响**：VM（MQL→AST→Bytecode 执行引擎）的缓存完整性、交易上下文、编译器语义、timeseries 语义、运行时 fail-closed 增强全部回退到施工前状态。这些是策略执行引擎的核心安全不变量，缺失会导致：
- 缓存污染攻击（SourceHash 不校验 → 旧 bytecode 执行新源码）
- 交易上下文失真（订单缓存不失效 → OrderSelect 返回 stale 数据）
- 编译器语义丢失（MQL→IR 静默丢弃语义 → 策略行为偏离）
- timeseries 语义错误（iHighest/iLowest mode 不校验 → 错误极值）

## 2. 设计决策

### D1：不尝试恢复被 revert 的代码

**理由**：revert 后已有多个 commit 在其上构建（包括 D-REVERT-CLEANUP-001 的 122 文件删除），git revert 的 revert 理论上可行但会引入大量冲突且无法保证恢复完整。更安全的做法是基于 registry 中的原始修复 spec 重新施工。

**替代方案（否决）**：`git revert 830b2c79`——会撤销 revert，但后续 commit 的文件删除会冲突，且无法保证恢复到 `acaa86db` 的完整状态。

### D2：分 4 批施工，按安全优先级

**批次顺序**：
1. VM-CACHE-INTEGRITY-1/2（SourceHash）——缓存安全，攻击面最广
2. VM-TRADE-CONTEXT-1/2（交易上下文）——交易安全，直接影响下单
3. VM-COMPILER-SEMANTICS-1 + BT-FUNC-ENTRYPC-FWD（编译器）——编译正确性
4. VM-TIMESERIES-SEMANTICS-1 + VM-RUNTIME-FAILCLOSED-1（语义）——运行时语义

**理由**：缓存完整性是其他所有修复的基础（如果 bytecode 缓存可被污染，其他修复都可能被绕过）。交易上下文直接影响真金白银。编译器和语义问题影响策略行为正确性但不直接导致资金损失。

### D3：基于 registry spec 重新施工，不做架构变更

**理由**：registry 中的修复 spec 已经过 2-4 轮独立复审，设计是成熟的。本轮重新施工不应引入架构变更，只恢复被 revert 的实现。如有架构改进需求，单独走 ADR。

### D4：每批独立验收，不批量验收

**理由**：8 个 ID 涉及不同子系统，批量验收会增加复审负担且难以隔离问题。每批施工完成后独立验收，通过后再派下一批。

## 3. 验收标准

每批施工完成后，Devin CLI 独立复审：
1. **mutation RED→restore→GREEN**：每个关键修复必须有真实对抗证明
2. **门禁全绿**：`go build ./...` / `go test ./tools/mql2go/... -count=1` / `go test -race ./tools/mql2go/... -count=1` / `go test ./internal/connect/strategy -count=1` / `go test -race ./internal/connect/strategy -count=1` / `go vet ./...` / `check-file-lines --strict`（0 errors）/ `buf lint` / `git diff --check`
3. **file-lines**：新增文件不超限（300/450/800 红线）
4. **复用核对**：`bash scripts/cap.sh` 多关键词查重
5. **状态诚实**：施工方不得自标 ✅done，只标 `🟦open（施工完成，待独立复审）`

## 4. 不做

- 不做架构变更（D3）
- 不恢复被 revert 的代码（D1）
- 不批量验收（D4）
- 不改写历史审计事实
- 不部署（D-COMMIT-SCOPE-001 部署闸仍有效）
