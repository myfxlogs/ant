# 知识库 P0 施工 spec — 收编散件成资产（K1 兼容知识 + L0 修复）

> **依据**：`docs/design/knowledge-base-architecture.md`（架构已 ✅ 用户确认 2026-08-08：底座 PG+pgvector / 确定性优先 / P0 收编）。
> **P0 范围**：把 K1（MQL 兼容知识）+ L0（确定性修复）从**散件**（constants.go / api_registry / compat_fixes）**收编成统一 KB 域表 + KB Service**，让兼容知识成为**可查/可复利/单一源**的资产，并让编译器/VM 经 KB 解析（C1 确定性复利的落地）。
> **非 P0**（后续 phase）：K3 需求捕获（P1）、K4/pgvector agent-RAG（P3）、C2 战绩循环已在 FEAT-5。
> **角色**：审计方出 spec；施工方实现+回填，不自行宣告 ✅，等审计方实测。

---

## 1. 目标
- 散件 → 统一：`constants.go`（~428 常量）+ `api_registry.go`（supported/unsupported 函数）+ `compat_fixes`（L0 修复）→ 收编进 KB 域表，单一源。
- 编译器/VM 符号解析**优先查 KB**（确定性、0-token、复利），KB 命中即用。
- 新增兼容知识**只进 KB**（不再散落 constants.go），可累积。

## 2. Schema（纯类型化列，零 JSONB）

```sql
-- K1: 兼容事实（常量/函数/指标的 supported 状态 + 值/映射）
CREATE TABLE kb_compat_fact (
    id BIGSERIAL PRIMARY KEY,
    identifier TEXT NOT NULL,              -- "clrGreen" / "iCustom" / "OrderSend"
    kind TEXT NOT NULL,                    -- constant | function | indicator
    status TEXT NOT NULL,                  -- supported | unsupported | partial
    severity TEXT NOT NULL DEFAULT 'info', -- fatal | warning | info（消费侧用）
    value_text TEXT,                       -- 常量值（文本）/映射目标名
    value_numeric NUMERIC,                 -- 常量值（数值）
    mapping_target TEXT,                   -- 别名指向（clrGreen→Green）
    source TEXT NOT NULL DEFAULT 'seed',   -- seed | manual | auto-verified
    verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (identifier, kind)
);
CREATE INDEX idx_kb_compat_fact_id ON kb_compat_fact(identifier);

-- L0: 确定性修复（问题 pattern → 透明变换规则）
CREATE TABLE kb_compat_fix (
    id BIGSERIAL PRIMARY KEY,
    pattern TEXT NOT NULL,                 -- 触发标识/pattern
    fix_type TEXT NOT NULL,                -- alias | rename | normalize
    resolution_target TEXT NOT NULL,       -- 确定性解析目标
    source TEXT NOT NULL DEFAULT 'manual',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (pattern)
);
```

## 3. KB Service（`internal/knowledgebase/`）

- **Read**（热路径，**禁 per-constant 查 PG**）：启动时把 `kb_compat_fact`+`kb_compat_fix` 全量载入**内存缓存**；`LookupConstant(name)`/`LookupFunction(name)`/`LookupFix(pattern)` 走缓存。变更经 **LISTEN/NOTIFY** 失效缓存（push-first，禁轮询）。
- **Write**：`RecordFact(...)`/`RecordFix(...)` → INSERT kb_* + `pg_notify`。新增兼容知识唯一入口。
- 跨进程交换全 proto（零 app Marshal）；数值 decimal。

## 4. 迁移 + 接线（去风险：seed + 回退）
1. **Seed**：一次性脚本把 `constants.go`（常量）+ `api_registry.go`（supported/unsupported 函数 + severity）+ `compat_fixes`（L0）→ 灌入 `kb_compat_fact`/`kb_compat_fix`（source=seed）。
2. **接线**：编译器/VM 符号解析改为**先查 KB Service 缓存** → 命中则用（值/映射/severity）。
3. **回退（去风险）**：KB 未命中时**回退现有 `constants.go`/`api_registry`**（保证迁移期零回归）；后续 cleanup 再删回退。
4. 新增兼容知识只走 `RecordFact/RecordFix`（不再改 constants.go）。

## 5. REUSE（施工方 `bash scripts/cap.sh`）
`constants.go`/`api_registry.go`/`compat_fixes`（seed 源）、`classifySeverity`/`SeverityForBuiltin`（severity 一致）、`pg_notify`/LISTEN 模式（同 WatchSchedules/decay_monitor）、decimal。**禁**：新 DB（用 PG）、JSONB（用类型化列）。

## 6. 验收（审计方实测）
- `kb_compat_fact` 行数 ≈ constants.go 常量数 + api_registry 函数数；seed 完整。
- 编译器解析 `clrGreen`/`MODE_SENKOU_A`/未知指标 → 经 KB 缓存命中，行为同 HONESTY-1/2 修复后（盲区数一致，无回归）。
- `RecordFact` 写一条新常量 → NOTIFY → 缓存刷新 → 编译器立即可解析（**C1 确定性复利实证**：不重启、不加常量到 Go、0 LLM）。
- 性能：编译期 per-constant 不查 PG（走缓存）；冷启动载入 <1s。

## 7. 对抗证明
- 删 KB 缓存查询 → 回退 constants.go → 仍工作但新知识不复利（证 KB 是复利入口）。
- RecordFact 后不经 NOTIFY → 缓存不刷新 → 新常量解析失败（证 push-first 缺一不可）。

## 8. 完工回填纪律（施工方）
1. `tech-debt-registry.md` 新增 `KB-P0` 🟦→✅ + 对抗证明 + 性能数据。
2. `handover-audit-plan.md` 变更日志。
3. 不自行宣告 ✅——等审计方实测（重点验：seed 完整、缓存经 NOTIFY 刷新、编译经 KB 无回归、0 per-constant PG 查询）。

---

> **审计方注**：P0 把散件升级成资产（单一源 + 可查 + 可复利），是 KB 从"概念"变"实物"的关键一步。去风险靠 seed+回退（迁移期零回归）。完成后 C1 确定性复利闭环成立（新增兼容知识→KB→编译器即时复用，0 token），为 P1（K3 需求捕获）/P3（agent-RAG）铺路。
