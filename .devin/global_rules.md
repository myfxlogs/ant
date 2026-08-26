# AlphaForge Devin Project Rules

## Source of truth

- `AGENTS.md` is the contract SSOT (唯一真相源); `CLAUDE.md` / `.windsurfrules` are entry shells that load `@AGENTS.md`.
- `docs/handoff/STATE.md` is the lightweight handover payload (T0, ≤20KB); `docs/audits/tech-debt-registry.md` is the authoritative open-gap ledger (T2, unlimited).
- `docs/audits/handover-audit-plan.md` is the append-only cross-session handover log.
- `docs/constraints.md` (技术约束) / `docs/pitfalls.md` (坑库) / `docs/项目定位.md` (业务方向+导航) are T1 knowledge docs.
- Treat builder self-reports as claims to verify, not as acceptance evidence.
- For every engineering task, invoke `/ant-workflow` before exploring or editing.

## Non-negotiable collaboration rules

- Separate audit and construction: an implementation pass does not self-declare `✅done`; Devin must perform an independent audit after evidence and may then mark acceptance. Historical `⚠️待Claude复审` entries mean `待独立复审`; they do not require Claude specifically when Devin is the assigned auditor.
- One task ID, one scope; preserve unrelated user changes and never silently broaden the diff.
- New sessions start by reading `AGENTS.md` (contract SSOT) + `docs/handoff/STATE.md` (current state) + relevant registry entry; read history before rewriting behavior.
- Every fix needs a real adversarial proof: remove or mutate the critical behavior, the relevant test must fail, then restore it.
- Construction must stop at the repository's pending-review status after updating registry/handover/AGENTS; do not deploy or push unless the user explicitly requests that external action. Devin is the independent auditor in this workflow and is responsible for the final `✅done` decision.
- Never use destructive recovery (`rm -rf`, `git reset --hard`, `git clean`, force-push, history rewrite) without explicit confirmation for that exact action.

## Tooling discipline

- Prefer Devin's `read`, `grep`, and `find_file_by_name` tools for file reads, content search, and path discovery; do not use shell `cat`, `head`, `tail`, `grep`, `find`, or `wc` as substitutes.
- When a shell command is necessary and `rtk` is available, use the RTK wrapper; do not pipe commands merely to truncate output—split the operation or use the specialized tool.
- Parallelize independent reads/searches; never parallelize dependent edits, tests, or writes to the same file.

## Reporting

Report decisions, evidence, changed files, verification results, remaining gaps, and deployment status; do not return a menu of technical options when the repository and rules determine the answer.
