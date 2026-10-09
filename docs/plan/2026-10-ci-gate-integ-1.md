# CI-GATE-INTEG-1：integration tag 编译门禁 + diff-check 固化进 CI（Devin CLI 设计 SSOT 2026-10-09）

## 立项背景

FILE-SPLIT-T2 独立复审实证两处存量 `-tags integration` build-broken 逃逸默认门禁：`connect/system` events 残余层撞名（已裁决删除，`02e73db1`）、`connect/strategy` `ptr` 重名（registry `INTEG-TAG-STRATEGY-PTR-1`）。同时 `git diff --check` 仅靠施工自觉。规则靠文档传递迟早再漏——固化进 CI 一次免疫。

## 设计 SSOT

- S1 必须先修 `ptr` 撞名，否则新门禁一上线 main 即红。
- vet 而非 build：`go build` 不编译 `_test.go`，撞名只有 `vet -tags integration` 或 `test -tags integration` 能抓。
- diff --check 需 base 参照：`actions/checkout` 默认 fetch-depth=1，要 `fetch-depth: 0` + 显式 `origin/main...HEAD` 三点diff。

## 施工步骤

### S1 修 INTEG-TAG-STRATEGY-PTR-1（`backend/internal/connect/strategy/strategy_crud_integration_test.go`）

- 删第 22 行包级 `func ptr[T any](v T) *T { return &v }`（与源文件导入的 `alphaforge/internal/pkg/ptr` 包撞名）。
- import 块加 `"alphaforge/internal/pkg/ptr"`。
- 全部 `ptr("...")` 调用（实测 7 处：:146/:147/:148/:231/:232/:233/:338，均为字符串字面量）改 `ptr.Str("...")`——复用 `internal/pkg/ptr` 的 `Str`，符合 cap.sh 复用纪律，不引入新 helper。
- 验收：`go vet -tags integration ./internal/connect/strategy` 净；`go test -tags integration -list '.*' ./internal/connect/strategy` 可枚举。

### S2 `.github/workflows/ci.yml` backend-lint job（行 28-49）

- `actions/checkout@v4` 步加 `with: fetch-depth: 0`。
- `go vet` 步后插入两步：
  ```yaml
      - name: go vet (integration tag)
        run: go vet -tags integration ./...
      - name: git diff --check
        run: git fetch origin main --depth=1 && git diff --check origin/main...HEAD
  ```
- 对 push 到 main 的事件 `origin/main...HEAD` 三点语法安全（merge-base=HEAD 时空 diff 通过）。

### S3 自查

- `go vet -tags integration ./...` 全仓仅 connect/strategy 存量断——S1 修后全仓净零。
- `git diff --check` 本分支净。

## 验收门禁

- S1 后 `go vet -tags integration ./...` 全仓 0 错、`go test -tags integration -list '.*' ./internal/connect/strategy` 可枚举、`go test -count=1 ./internal/connect/strategy` 绿
- `go build ./...`、`go vet ./...`、gofmt/diff --check 净、check-file-lines 0 ERROR
- ci.yml YAML 语法可解析（`python3 -c "import yaml; yaml.safe_load(open('.github/workflows/ci.yml'))"` 或等效）
- mutation：临时在任一 `_test.go` 复声明撞名 → `vet -tags integration` 本地复红（证明门禁链有效）

勿部署，停手等 Devin CLI 复审。
