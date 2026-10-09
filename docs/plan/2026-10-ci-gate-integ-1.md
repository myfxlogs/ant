# CI-GATE-INTEG-1：integration tag 编译门禁 + diff-check 固化进 CI（Devin CLI 设计 SSOT 2026-10-09，v2 修订版）

## 立项背景

FILE-SPLIT-T2 独立复审实证 `-tags integration` build-broken 逃逸默认门禁。**v2 修订（方案自审抓出范围低估）**：`go vet -tags integration ./...` 全仓实测 **4 包断**，非初版估的 1 包——v1 只列 connect/strategy `ptr` 撞名，漏了 3 处 API 漂移腐化测试（长期无 tag 编译所致，恰是门禁存在的意义）。

## 设计 SSOT

- S1 必须先修全部 4 处存量断，否则新门禁一上线 main 即红。
- vet 而非 build：`go build` 不编译 `_test.go`，撞名/API 漂移只有 `vet -tags integration` 能抓。
- diff --check 需 base 参照：`actions/checkout` 默认 fetch-depth=1，要 `fetch-depth: 0` + 显式 `origin/main...HEAD` 三点 diff。

## 施工步骤

### S1 修全仓 `-tags integration` 存量断（4 包）

**S1a `internal/connect/strategy/strategy_crud_integration_test.go`（INTEG-TAG-STRATEGY-PTR-1）**
- 删第 22 行包级 `func ptr[T any](v T) *T { return &v }`（与源文件导入的 `alphaforge/internal/pkg/ptr` 包撞名）。
- import 块加 `"alphaforge/internal/pkg/ptr"`。
- 全部 `ptr("...")` 调用（实测 7 处：:146/:147/:148/:231/:232/:233/:338，均字符串字面量）改 `ptr.Str("...")`——复用 `internal/pkg/ptr` 的 `Str`，不引入新 helper。

**S1b `internal/agent/generator_e2e_test.go:100`**
- 实测 drift：测试调 `NewGenerator(aiSvc, zap.NewNop(), NewProfiler(...), NewInterpreter(...), cache, memStore, hooks, settingsStore, ...)` 为旧签名；现签名（`internal/agent/generator.go:48`）：`NewGenerator(aiSvc *systemai.Service, log *zap.Logger, cache *LLCache, memory *MemoryStore, mkt repository.MarketDataStore, btRepo *repository.BacktestRunRepository, dbExec func(ctx, sql, args) error, dbQuery func(ctx, sql, args) (string, error)) *Generator`。
- 修法：按现签名重排实参——`NewProfiler`/`NewInterpreter`/`hooks`/`settingsStore` 已不在签名（Generator 内部自持），`mkt`/`btRepo` 若测试路径未触达可传 nil，`dbExec`/`dbQuery` 传返回 nil 错误的桩 func。落点前读 `NewGenerator` 构造体确认 nil 安全（若构造期解引用则建最小 fake），并通读该测试实际调用的 Generator 方法确认未触达 nil 依赖。

**S1c `internal/connect/admin/admin_reset_password_integration_test.go:85`**
- 实测 drift：`NewAdminUserServer(adminRepo, resetRepo, log)` 旧签名；现签名（`admin_user_handler.go:32`）六参 `(repo, resetRepo, walletSvc *service.WalletService, acctSvc *usersvc.AccountNumberService, deletionSvc *service.UserDeletionService, log)`。
- 修法：该测试只调 `ResetUserPassword`——先读该方法路径确认不触达 wallet/acct/deletion 三 service，确认后传 nil；若触达则构造最小实例。

**S1d `internal/connect/user/account_handler_integration_test.go:205`**
- 实测 drift：`acct.GetBalance()` 现返回 `string`（proto `balance` 为 decimal-string，`account_entity.pb.go:187`），测试 `!= 10000.0` float 比较编译失败。
- 修法：`!= "10000"`（若服务端写实格式为 `"10000.00"` 则按实——先跑通读真实返回）；同函数 `t.Logf` 的 `balance=%.2f` 配 string 实参会触发 vet printf 检查，改 `%s`。

### S2 `.github/workflows/ci.yml` backend-lint job（行 28-49）

- `actions/checkout@v4` 步加 `with: fetch-depth: 0`。
- `go vet` 步后插入两步：
  ```yaml
      - name: go vet (integration tag)
        run: go vet -tags integration ./...
      - name: git diff --check
        run: git fetch origin main --depth=1 && git diff --check origin/main...HEAD
  ```
- push-to-main 事件 `origin/main...HEAD` 三点语法安全（merge-base=HEAD 时空 diff 通过，即门禁只对 PR diff 实质生效——有意为之）。

### S3 自查

- `go vet -tags integration ./...` 全仓净零（4 处修后无其他存量断——v2 已全仓实扫，0 额外）。
- `git diff --check` 本分支净。

## 验收门禁

- S1 后 `go vet -tags integration ./...` 全仓 0 错；4 包 `go test -tags integration -list '.*'` 全部可枚举；`go test -count=1` 触及包绿
- `go build ./...`、`go vet ./...`、gofmt/diff --check 净、check-file-lines 0 ERROR
- ci.yml YAML 可解析（`python3 -c "import yaml; yaml.safe_load(open('.github/workflows/ci.yml'))"`）
- mutation：临时在任一 `_test.go` 复声明撞名 → `vet -tags integration` 本地复红（证明门禁链有效）

勿部署，停手等 Devin CLI 复审。
