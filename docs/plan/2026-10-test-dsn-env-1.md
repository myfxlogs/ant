# TEST-DSN-ENV-1 施工单 —— 测试硬编码 DSN 清债（含生产密码出仓）

## 立项背景（证据链）

CI-RESTORE-1 独立复审（2026-10-09 Devin CLI [角色:决策]）留债：全量 `go test ./...` 3 败=`internal/service/bound_account_svc_test.go` 硬编码 `postgres://alphaforge:alphaforge@localhost:5432/alphaforge`（be831d5d 引入）。复审扩查抓出**更严重存量**：`internal/knowledgebase/demand_test.go:17` 硬编码 `postgres://ant:QxhrPqrizFg0iTWNOnabaFvv@localhost:5433/ant` ——该密码与 sg 生产 `POSTGRES_PASSWORD` 实值一致，**生产凭据明文入 git**。

## 设计 SSOT

本仓测试 DB 约定（实证两处）：`TEST_PG_DSN` env 优先，回落 `postgres://ant:ant@localhost:5432/ant?sslmode=disable`；连接失败 `t.Skipf` 跳过而非 FAIL——且必须 `pool.Ping` 后才 Skipf（pgxpool.New 惰性连接，不 Ping 则 skip 永不触发、坏 DSN 变 FAIL——这正是 bound_account 三测挂掉的机制）。

## 边界/不做

- 只动下表 3 个文件；`internal/connect/**` 一组 `localhost:5433` 拼装 DSN 已是 env 驱动（DB_USER/DB_PASSWORD/DB_NAME+回落），不纳入。
- 不改测试断言、不删测试；只改取池 helper。
- **密码轮换不在本单**：生产 ant 库密码已入 git 史，须业主裁决轮换+CI 同步（registry 已登记 PROD-PG-PASSWORD-IN-GIT，单独跟进）。

## S1. 逐文件改造（3 文件）

| 文件 | 行 | 改法 |
|---|---|---|
| `internal/service/bound_account_svc_test.go` | 175-181 `testPool` | DSN 改 `os.Getenv("TEST_PG_DSN")` 空则回落 `postgres://ant:ant@localhost:5432/ant?sslmode=disable`；`pgxpool.New` 后加 `pool.Ping` 失败→`pool.Close()`+`t.Skipf`（照 `demandTestPool` 形态） |
| `internal/knowledgebase/demand_test.go` | 17 `demandTestPool` | 同上——**硬编码生产密码必须出源码**；Ping/Skipf 已有保留 |
| `internal/repository/strategy_run_task4_test.go` | 16 `dsn :=` | 同样 env+回落改造；若其后无 Ping 守卫补 `pool.Ping`→Skipf |

改动均为同包 helper 内部，测试函数零改动。

## T1–T3. 门禁（先红后绿实证）

- **T1 无 DB 环境（默认验证环境）**：`unset TEST_PG_DSN; go test ./internal/service/ ./internal/knowledgebase/ ./internal/repository/` —— 3 个原 FAIL 测试须 SKIP（`--- SKIP`/`ok` 但含 skip 行），包级非零退出消失；`strategy_run_task4` 原状即 pass/skip 不回归。
- **T2 有 DB**：`TEST_PG_DSN=postgres://ant:ant@localhost:5432/ant?sslmode=disable go test ./internal/service/` —— 若本机存在 ant 库则 3 测真跑绿（当前开发机默认无此库则 SKIP 亦可接受，如实记录）。
- **T3 mutation**：把 helper 回落 DSN 改坏（`ant:nope@localhost:59999/nope`）→ `TEST_PG_DSN` unset 时测试须 **SKIP 不是 FAIL**；恢复。
- 机检五件套：build✓ vet✓ gofmt 触碰文件净✓ check-file-lines 0 ERROR✓ `git diff --check`✓；`grep -rn 'QxhrPqrizFg0iTWNOnabaFvv' .` 全仓归零（含其他文件若有）。

## 完工回报

`[施工完成:TEST-DSN-ENV-1] @<commit>` + T1/T2/T3 输出 + 密码串 grep 归零证据。

## 固定尾部

勿部署，勿 push 生产，停手等 Devin CLI 复审。禁 `--no-verify`。
