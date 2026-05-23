# ADR-0004 Python 沙箱进程模型

## Status
**Accepted** (2026-05-23) — 按推荐方向 B (fork-per-execution) 落地；M3 实施

## Context
strategy-service 是 ant 的核心安全边界。当前实现（`@/opt/ant/strategy-service/app/engine/sandbox.py`）：
- 三层防御：① AST 白名单（禁 import / dunder / open / eval / ...）；② RestrictedPython 字节码限制；③ Curated globals（仅暴露 numpy + math + 内置指标）
- iptables 出口拒绝（`docker-entrypoint.sh`）但失败静默（`|| true`）
- **关键问题**：所有用户代码在同一 uvicorn 进程内执行，多用户共享同一 Python 解释器。

风险：
- **R1** 死循环 / 内存爆炸：单用户拖垮整服务
- **R2** RestrictedPython 历史 CVE 表明 AST 沙箱可能被绕过
- **R3** 多用户代码共享 sys.modules / 全局状态污染
- **R4** iptables 静默失败 → egress 控制可能未启用

## Options
- **A. 保持 in-process**（现状）：性能最佳；但 R1-R4 需逐项加固（资源限额、AST 加固、显式探测 NET_ADMIN 失败 fail-fast）
- **B. fork-per-execution**（subprocess + resource limits + seccomp）：每次执行 fork 子进程，setrlimit + prlimit 控制 CPU/内存/FD；OS 级隔离强
- **C. nsjail 子进程**：B 的加强版，独立 namespace + cgroups；运维复杂度高
- **D. WASM 运行时**（wasmtime / Pyodide）：跨语言通用沙箱；性能/生态待评估

## Decision
**B (fork-per-execution)**：在 in-process 沙箱外层加 OS 级硬隔离；保留现有 AST 三层作为防御纵深；M3 实施。

## Consequences（按 B 推演）
- **+** 单用户故障不影响其他；OS 级 setrlimit 杜绝资源耗尽
- **+** seccomp 白名单系统调用，防绕过
- **−** 每次执行 fork 开销（数 ms）；高频回测可能压力增大 → 引入 worker pool
- **−** 调试更难：子进程崩溃信息需结构化上抛

## Related
- DESIGN-REVIEW SEC-4, MN-06
- AGENT.md 安全红线
- `@/opt/ant/strategy-service/app/engine/sandbox.py:1-50`

## History
- 2026-05-23 Proposed

- 2026-05-23 Accepted（按推荐方向落地，详见 Decision）
