---
name: ant-workflow
description: Execute AlphaForge's rigorous Claude-style audit, construction, red-team, handover, and acceptance workflow for repository engineering tasks.
argument-hint: "[task-id or request]"
triggers:
  - user
  - model
allowed-tools:
  - read
  - grep
  - glob
  - exec
  - edit
---

# AlphaForge Engineering Workflow

This skill ports the project's Claude Code collaboration method to Devin. It is the default workflow for every engineering task, including debugging, feature work, refactors, code review, and acceptance review.

## 0. Choose the role and mode before touching code

Classify the request:

- **Audit/review/acceptance**: act read-only on source code; do not modify production code or tests. You may append audit evidence to the registry and handover documents. In this Devin workflow, Devin is the independent auditor and owns the final acceptance decision.
- **Construction/implementation**: implement the requested task only. Do not self-accept it; stop at the repository's pending-review status after evidence and document updates.
- **Planning/design**: investigate and discuss the decision; do not edit implementation files until the plan is approved or the user explicitly asks execution.
- **Commit/push/deploy**: treat as separate external side-effect phases. Never infer permission from "finished" or "ready"; ask/require explicit user request.

If the request names a task ID, use that ID as the scope boundary. If no ID exists, search the registry before inventing a new one.

## 1. Mandatory session bootstrap

Before exploration:

1. Read `AGENTS.md` and `CLAUDE.md`.
2. Read the relevant sections of `docs/audits/handover-audit-plan.md` and `docs/audits/tech-debt-registry.md`.
3. Inspect `git status`, recent history, and the current diff; preserve pre-existing user changes.
4. Identify whether the task is `❓待核`, `🟦open`, `✅done`, or `⚠️待Claude复审`.
5. Read only the relevant block docs and referenced handoff; do not create parallel progress documents.

For a new task, establish the root cause, invariant, affected pipeline, scope exclusions, adversarial proof, and verification gates in the registry or existing task document before implementation.

## 2. Root-cause-first investigation

When behavior disappeared, regressed, or changed:

1. Run `git log --all --oneline -- <path>`.
2. Run `git blame` on the suspected lines.
3. Read the introducing commit, ADR, spec, or handover context.
4. Decide whether the behavior was lost, intentionally removed, or never implemented.
5. Prefer a precise repair of the responsible change; do not rewrite from scratch without historical evidence.

Trace the full pipeline instead of stopping at the symptom:

`source/event → adapter → broker/gateway → cache/store → handler/runner → proto/SSE → frontend/VM`.

At every boundary ask: does it connect, does it preserve semantics, does it fail closed, and is it authoritative?

## 3. Reuse preflight before new code

Before creating a file, function, handler, collector, RPC path, or component:

- Run `bash scripts/cap.sh <verb-or-symbol>` with multiple useful aliases.
- Search neighboring implementations and tests with the repository search tools.
- Record the result in the implementation/PR report as `REUSE: <symbol>@<file:line>` or `NEW: searched <keywords>, no existing capability`.
- Reuse existing abstractions unless the code proves they are insufficient.

Do not add a dependency or framework without verifying the repository already uses it and checking the package manifest.

## 4. Construction protocol

For implementation tasks:

1. Write a failing, behavior-level test when the project has test infrastructure.
2. Implement the smallest correct change within the task scope.
3. Preserve comments and local conventions; do not add `TODO`, `HACK`, `legacy`, or silent fallback shortcuts.
4. Keep external APIs ConnectRPC/proto/SSE-only; do not introduce REST, WebSocket, JSON application exchange, or polling where a push source exists.
5. Use `decimal.Decimal` for Go prices and monetary values; never use `float64` for price calculations.
6. Keep Go files under 450 lines and TypeScript files under 375 lines; split by semantic responsibility before crossing the hard limit.
7. Do not modify unrelated user changes, generated files outside the affected proto, security policy, package-manager security controls, or deployment state.

For concurrent code, use channels/conditions/context cancellation for test synchronization; do not use probabilistic sleeps to make a test pass.

## 5. Red-team proof is mandatory

After implementation, stop trusting the implementation and attack it:

- Mutate or remove every critical line claimed by the task.
- Prefer a real integration path over a helper-only assertion.
- Verify the relevant test turns RED with an assertion or compile failure for the intended reason.
- Restore the mutation immediately and rerun GREEN.
- Test negative, empty, nil, stale, mixed-magic, wrong-type, timeout, concurrent, and partial-authority cases as relevant.
- If deleting a line leaves tests green, the test is not an adversarial proof; add or strengthen the test before completion.

Use an isolated worktree for risky mutations when possible. Never use destructive git cleanup to make the worktree convenient.

## 6. Verification gates

Run the narrowest relevant gates first, then the package/project gates appropriate to the scope:

- Target behavior tests.
- `go test` and `go test -race` for affected Go packages.
- `go build ./...`.
- `go run ./tools/check-file-lines --strict`.
- `buf lint`/proto generation checks for proto changes.
- `tsc --noEmit`, frontend tests, and `npm run build` for frontend/proto UI changes.
- Report pre-existing unrelated failures separately; never call a failed gate green.
- Run `git diff --check` and inspect the final file list for cross-scope churn.

A builder claim such as "all gates green" is not evidence until Devin independently runs the applicable gates.

## 7. Three-layer handover discipline

Construction completion must update, immediately and truthfully:

1. `docs/audits/tech-debt-registry.md`: task status, actual root cause, fix, adversarial proof, tests, remaining risks.
2. `docs/audits/handover-audit-plan.md`: one append-only dated change-log entry.
3. `AGENTS.md` only for a reusable project-wide pitfall; otherwise do not add noise.

Keep `✅done` rows and changelog entries; never delete or rewrite historical evidence. Construction stops at the repository's pending-review status and does not declare final acceptance.

Devin performs the independent audit after construction. Audit completion may update the registry to `✅done` only after independently verifying code, behavior, adversarial RED/GREEN, scope, and gates. If blocked, add a new append-only audit finding and leave the task `🟦open`.

## 8. Final report format

Always report:

- **Decision**: pass, conditional pass, or not accepted.
- **Root cause / intent**: what the change is actually solving.
- **Evidence**: exact tests, gate commands, mutation results, and file references.
- **Scope**: files changed and unrelated changes preserved.
- **Risks/gaps**: unresolved architecture, production, or deployment validation.
- **Next action**: the smallest concrete rework or explicit external approval needed.

Use clickable `<ref_file ... />` and `<ref_snippet ... />` citations when naming repository files. Do not return a menu of options when the repository and rules determine the decision.

## 9. Safety and side effects

Never commit, push, deploy, send external messages, alter production data, or perform irreversible cleanup unless the user explicitly requests that exact action. Build/test locally first. Deployment, when explicitly approved, must follow the repository's documented Docker path; do not use host-built binaries copied into containers.
