# 代码质量修复计划 · GLM 施工清单

> 来源：`docs/audits/foundation-code-quality.md`

## P0 · 文件拆分（阻断 CI）

- [ ] **F1** `backend/cmd/server/handlers.go` (462 行) → 按域拆为 `handlers_marketplace.go` + `handlers_strategy.go` + `handlers_user.go` + `handlers_admin.go`
- [ ] **F2** `backend/internal/service/account_service.go` (452 行) → 拆为 `account_crud.go` + `account_lifecycle.go` + `account_sync.go`
- [ ] **F3** `frontend/src/pages/marketplace/components/AutoGeneratePanel.tsx` (433 行) → 拆为 `AutoGeneratePanel.tsx` + `AutoGenerateProgress.tsx` + `AutoGenerateResult.tsx`
  - ⚠️ 此文件 GLM 施工中，完工后立即拆分

## P0 · 死代码清理

- [ ] **D1** 删除 30 个 `unused` 函数/变量/类型（golangci-lint 报告）
  - `golangci-lint run --fix` 不能自动删除 unused——需手动逐条确认后删除
  - **红线**：删除前确认不是接口实现的一部分（unparam 可能是接口要求）

## P0 · Error 检查

- [ ] **E1** 修复 20 个 `errcheck` 告警（未检查 error 返回值）
  - 优先处理：文件 I/O、网络调用、DB 操作的未检查 error
  - 低优先级：`fmt.Fprintf` 等输出操作的未检查 error

## P1 · 代码重复

- [ ] **DUP1** `backend/internal/marketplace/leaderboard.go` — 四种榜单查询逻辑提取公共 builder 函数
  - 改动 < 30 行

## P1 · 安全审查

- [ ] **SEC1** 审查 5 个 `gosec` 告警，逐一确认是误报或修复

## P2 · 杂项

- [ ] **M1** 修复 3 个 `unconvert` + 1 个 `misspell`
- [ ] **M2** 升级 Go 到 1.26（当前 go1.25）

---

## 执行约束

- 每个 P0 项做完 → 跑 `golangci-lint run` + `go build ./...` + `check-file-lines --strict`
- `unused` 删除：逐文件 commit，万一误删可单独 revert
- 不得新增 `//nolint` 注释
