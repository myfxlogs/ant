---
description: Modify or debug the real-time MQL code check / save-time audit / CodeMirror lint integration
---

# MQL Code Audit Feature

## Overview

- Real-time code check: 800ms debounce in `CodeEditorArea` calls `strategyVersionApi.checkCode(code)`.
- Save-time audit: `handleSave` passes `compileAudit: true` to `updateCode`.
- Editor lint: CodeMirror `lintGutter` + `setDiagnostics` (NOT `linter()` + compartment).

## Backend

- `CheckCode` RPC: `backend/internal/connect/strategy/code_check_handler.go`
  - `compileAndAudit(source)` compiles MQL → IR → Bytecode and returns diagnostics + blind spots.
  - Compile errors are encoded in the response (`CompileSuccess=false`); the handler never returns a Go error.
  - `compileErrorBlindSpot(err)` helper lives in `strategy_import_handler.go` and is shared with `AnalyzeImportCode`.
- `UpdateStrategyCode` enhanced to accept `compile_audit` and persist `coverage_score` via `ImportedStrategyRepository.UpdateCoverageScore`.
- Proto: `proto/ant/v1/strategy_runtime.proto` (`CheckCodeRequest` / `CheckCodeResponse`, `UpdateStrategyCodeRequest` / `UpdateStrategyCodeResponse`).

## Frontend

- `StrategyCodeEditor` (`frontend/src/components/strategy/StrategyCodeEditor.tsx`):
  - Integrates `@codemirror/lint` with `lintGutter` and `setDiagnostics`.
  - `Diagnostic` interface: `{ message; severity: 'error' | 'warning' | 'info' }` (no line numbers from compiler).
  - `toCMDiagnostics()` maps to CodeMirror `from:0, to:0`.
- `CodeEditorArea` (`frontend/src/pages/strategy/components/workspace/CodeEditorArea.tsx`):
  - 800ms debounce `checkCode`.
  - `blindSpotsToDiagnostics()` converts `BlindSpot[]` + compile error to diagnostics.
  - Audit status bar below editor.
- Client split: `frontend/src/client/strategy_version.ts` extracted from `strategy.ts` for ESLint max-lines.

## Architecture decisions

- `CheckCode` is high-frequency and only returns diagnostics + coverage; `AnalyzeImportCode` is low-frequency and also extracts params/groups/rules. Keep them separate.
- `setDiagnostics` direct dispatch is simpler than `linter()` + `compartment.reconfigure()`.
- `compileAndAudit` returns `*CheckCodeResponse` (not `(*Response, error)`) because compile errors are user-facing data.
- Only call `UpdateCoverageScore` when `CompileSuccess=true`.

## Deployment

```bash
docker builder prune -f
docker compose build backend && docker compose up -d backend
npm run build
docker cp frontend/dist/. alphaforge-frontend:/usr/share/nginx/html/
docker exec alphaforge-frontend nginx -s reload
```
