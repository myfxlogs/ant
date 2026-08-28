# 施工提示词 — FIX-2026-08-28-DATA-TRUTH-1-RECONCILIATION-CONVERGENCE

> **设计 SSOT**: `docs/spec/fix-2026-08-28-data-truth-1-reconciliation-convergence.md`
> **base commit**: `c0f53813`
> **scope**: 单 commit，改 `reconciliation.go` + `service_orders.go`（新增 `ImportBrokerOrder`）+ 新建 1 测试文件
> **不做**: 不改 ADR-0013、不改 `FetchOpenedOrders`/`FetchOrderHistory` RPC 调用、不改事件驱动架构、不部署

## 立项背景

**触发**: registry DATA-TRUTH-1 — `orders` 表与 broker 双向不一致，reconciliation 只检测不收敛。

**证据链**:
- `reconciliation.go:189-196` ghost（broker 有 ant 无）仅 `log.Warn` 从不补写 → 永不收敛（违反 ADR-0013 §2.3 "broker 有 PG 无 → INSERT"）
- `reconciliation.go:174-177` orphan 仅在 `state==SUBMITTED` 时修 → 其他非终态永不修复
- `reconciliation.go:125` broker 24h 窗口 vs `:143-147` ant 全量查询 → 结构性假 orphan（实测 129 条 warn/账户/轮）

**决策（Devin CLI 定稿）**: Q1=A（加 24h 下界）/Q2=A（ghost 自动补写）/Q3=A（修复器，符合 ADR-0013）

**约束**: 
- ghost 补写用新 SQL `ON CONFLICT (mt_account_id, ticket) DO NOTHING`（不能用 `OmsWriter.InsertOrder` 的 `ON CONFLICT (id) DO NOTHING`，后者 ticket 用 `hashToNegative` 占位）
- 已平仓 ghost 订单写入 `trade_records` 复用 `TradeRecordRepository.Create`（含 hash chain）
- orphan 修复扩展到所有非终态（不只是 SUBMITTED）

## S1: ant 侧查询加 24h 下界（根因 C）

**文件**: `backend/internal/mthub/reconciliation.go:143-147`

```go
// before (:143)
rows, err := r.pg.Query(ctx, `
    SELECT ticket, state FROM orders WHERE mt_account_id = $1::uuid
    UNION ALL
    SELECT ticket, 'CLOSED' FROM trade_records WHERE account_id = $1::uuid
`, accountID)

// after
cutoff := Clk.Now().Add(-24 * time.Hour)
rows, err := r.pg.Query(ctx, `
    SELECT ticket, state FROM orders WHERE mt_account_id = $1::uuid AND created_at >= $2
    UNION ALL
    SELECT ticket, 'CLOSED' FROM trade_records WHERE account_id = $1::uuid AND close_time >= $2
`, accountID, cutoff)
```

**不改**: broker 侧查询（`:125` 已是 24h 窗口）、scan 代码、`brokerTickets` map 构建逻辑。

## S2: ghost 自动补写（根因 A）

**文件**: `backend/internal/mthub/reconciliation.go:189-196`

```go
// before (:189)
for ticket := range brokerTickets {
    if _, exists := antTickets[ticket]; !exists {
        r.log.Warn("reconciliation: ghost order (broker has, ant missing)",
            zap.String("accountID", accountID),
            zap.Int64("ticket", ticket))
        ghosts++
    }
}

// after
for ticket, br := range brokerTickets {
    if _, exists := antTickets[ticket]; !exists {
        if r.svc != nil {
            if err := r.svc.ImportBrokerOrder(ctx, accountID, br); err != nil {
                r.log.Error("reconciliation: ghost import failed",
                    zap.String("accountID", accountID),
                    zap.Int64("ticket", ticket), zap.Error(err))
            } else {
                repaired++
            }
        }
        r.log.Warn("reconciliation: ghost order (broker has, ant missing)",
            zap.String("accountID", accountID),
            zap.Int64("ticket", ticket))
        ghosts++
    }
}
```

**注意**: `brokerTickets` 是 `map[int64]*OrderRecord`（`:130`），`br` 已可用，无需额外查找。

## S3: 新增 ImportBrokerOrder 方法

**文件**: `backend/internal/mthub/service_orders.go`（新增方法）

```go
// ImportBrokerOrder writes a broker-side OrderRecord into the OMS orders table
// and (if closed) into trade_records. Used by reconciliation to converge ghost
// orders (broker has, ant missing). Idempotent via ON CONFLICT DO NOTHING.
func (s *MtHubService) ImportBrokerOrder(ctx context.Context, accountID string, br *OrderRecord) error {
    if s.omsWriter == nil || br == nil {
        return nil
    }
    // 1. Insert into orders table with real broker ticket + broker state.
    state := brokerOrderStateToOMS(br.State)
    if state == "" {
        state = OMSStateReconciling
    }
    orderID := uuid.New().String()
    _, err := s.omsWriter.Pool().Exec(ctx, `
        INSERT INTO orders (id, mt_account_id, platform, ticket, symbol, order_type, volume, price, stop_loss, take_profit, state, magic_number)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
        ON CONFLICT (mt_account_id, ticket) DO NOTHING
    `, orderID, accountID, platform(accountID, s.hub), br.Ticket, br.SymbolRaw, int16(br.OrderType),
        br.Volume, br.OpenPrice, br.StopLoss, br.TakeProfit, string(state), br.Magic)
    if err != nil {
        return fmt.Errorf("import broker order: insert orders: %w", err)
    }
    // 2. If closed (has CloseTime + ClosePrice), also write to trade_records.
    if !br.CloseTime.IsZero() && br.ClosePrice.GreaterThan(decimal.Zero) {
        // TradeRecordRepository.Create handles hash chain + ON CONFLICT DO NOTHING.
        // MtHubService needs access to tradeRecordRepo — see S3a below.
        // ... (S3a wires the repo)
    }
    return nil
}
```

**S3a: MtHubService 加 tradeRecordRepo 字段**

`MtHubService`（`service.go:46`）当前无 `tradeRecordRepo` 字段。新增：
```go
type MtHubService struct {
    // ...existing fields...
    tradeRecordRepo *repository.TradeRecordRepository // NEW: for ImportBrokerOrder ghost writes
}
```

`NewMtHubService` 签名加 `tradeRecordRepo *repository.TradeRecordRepository` 参数（可 nil）。`cmd/server` 装配点传 repo。

**S3b: OmsWriter 暴露 Pool()**

`OmsWriter`（`oms_writer.go:88`）的 `pool` 是私有字段。新增 `Pool() *pgxpool.Pool` getter，供 `ImportBrokerOrder` 用。或者 `ImportBrokerOrder` 作为 `OmsWriter` 的方法（更内聚）。

**推荐**: `ImportBrokerOrder` 作为 `MtHubService` 方法，`OmsWriter` 加 `Pool()` getter（最小改动）。

**S3c: 已平仓 ghost 写入 trade_records**

```go
// 在 ImportBrokerOrder 内，br.CloseTime 非 zero 时
if !br.CloseTime.IsZero() && br.ClosePrice.GreaterThan(decimal.Zero) {
    // 查 user_id（trade_records 需要）
    var userID uuid.UUID
    err := s.omsWriter.Pool().QueryRow(ctx,
        `SELECT user_id FROM mt_accounts WHERE id = $1::uuid`, accountID).Scan(&userID)
    if err != nil {
        return fmt.Errorf("import broker order: lookup user_id: %w", err)
    }
    rec := &model.TradeRecord{
        UserID:     userID,
        AccountID:  uuid.MustParse(accountID),
        Ticket:     br.Ticket,
        Symbol:     br.SymbolRaw,
        OrderType:  br.OrderTypeString(),  // 复用现有 helper（order_types.go:87）
        Volume:     br.Volume,
        OpenPrice:  br.OpenPrice,
        ClosePrice: br.ClosePrice,
        Profit:     br.Profit,
        Swap:       br.Swap,
        Commission: br.Commission,
        OpenTime:   br.OpenTime,
        CloseTime:  br.CloseTime,
        StopLoss:   br.StopLoss,
        TakeProfit: br.TakeProfit,
        MagicNumber: int(br.Magic),
        Platform:   platform(accountID, s.hub),
    }
    if s.tradeRecordRepo != nil {
        if err := s.tradeRecordRepo.Create(ctx, rec); err != nil {
            return fmt.Errorf("import broker order: insert trade_record: %w", err)
        }
    }
}
```

**注意**: `ScheduleID` 设为 nil（ghost 订单不知道归属 schedule）。`br.OrderTypeString()` 是现有 helper（`order_types.go:87`，返回 "BUY"/"SELL"/"BUY_LIMIT" 等，与 `trade_records.order_type` 列兼容）。

## S4: orphan 修复扩展到所有非终态（根因 B）

**文件**: `backend/internal/mthub/reconciliation.go:174-177`

```go
// before (:174)
if antState == string(OMSStateSubmitted) && r.svc != nil {
    r.repairOrder(ctx, accountID, ticket, OMSStateFailed)
    repaired++
}

// after
if isNonTerminalOMSState(antState) && r.svc != nil {
    r.repairOrder(ctx, accountID, ticket, OMSStateFailed)
    repaired++
}
```

**新增 helper**（`reconciliation.go` 内或 `oms_writer.go`）:
```go
// isNonTerminalOMSState returns true if the state is not a terminal state.
// Terminal states: FILLED, CANCELLED, FAILED, EXPIRED, REJECTED, SLIPPAGE_REJECTED.
func isNonTerminalOMSState(state string) bool {
    switch OMSState(state) {
    case OMSStateFilled, OMSStateCancelled, OMSStateFailed, OMSStateExpired,
        OMSStateRejected, OMSStateSlippageRejected:
        return false
    }
    return true
}
```

**注意**: orphan 是"ant 有 broker 无"——broker 确认不存在，ant 侧应标记为 FAILED（无论当前什么非终态）。终态订单不修（已是终态，无需修）。

## T1-T4: 对抗证明

**新建文件**: `backend/internal/mthub/reconciliation_convergence_test.go`

**T1: `TestReconciliation_AntQueryHasTimeBound`**（S1 守卫）
```go
func TestReconciliation_AntQueryHasTimeBound(t *testing.T) {
    content := mustReadFile(t, "internal/mthub/reconciliation.go")
    idx := strings.Index(content, "func (r *ReconciliationLoop) reconcileAccount")
    if idx < 0 {
        t.Fatal("reconcileAccount not found")
    }
    body := content[idx:]
    if !strings.Contains(body, "created_at >=") {
        t.Fatal("ant orders query must have created_at >= time bound (S1)")
    }
    if !strings.Contains(body, "close_time >=") {
        t.Fatal("ant trade_records query must have close_time >= time bound (S1)")
    }
}
```

**T2: `TestReconciliation_GhostAutoImports`**（S2 守卫）
```go
func TestReconciliation_GhostAutoImports(t *testing.T) {
    content := mustReadFile(t, "internal/mthub/reconciliation.go")
    idx := strings.Index(content, "for ticket, br := range brokerTickets")
    if idx < 0 {
        t.Fatal("ghost loop not found — must iterate brokerTickets with br")
    }
    body := content[idx:]
    if !strings.Contains(body, "ImportBrokerOrder") {
        t.Fatal("ghost loop must call ImportBrokerOrder (S2)")
    }
}
```

**T3: `TestReconciliation_OrphanRepairsAllNonTerminal`**（S4 守卫）
```go
func TestReconciliation_OrphanRepairsAllNonTerminal(t *testing.T) {
    content := mustReadFile(t, "internal/mthub/reconciliation.go")
    if !strings.Contains(content, "isNonTerminalOMSState") {
        t.Fatal("must use isNonTerminalOMSState (S4)")
    }
    // 断言不再硬编码 OMSStateSubmitted
    idx := strings.Index(content, "if antState == string(OMSStateSubmitted)")
    if idx >= 0 {
        // 检查这行是否在 orphan 分支（broker missing）——如果是就是未修
        t.Fatal("orphan repair must use isNonTerminalOMSState, not hardcoded SUBMITTED (S4)")
    }
}
```

**T4: `TestImportBrokerOrder_UsesTicketConflict`**（S3 守卫）
```go
func TestImportBrokerOrder_UsesTicketConflict(t *testing.T) {
    content := mustReadFile(t, "internal/mthub/service_orders.go")
    idx := strings.Index(content, "func (s *MtHubService) ImportBrokerOrder")
    if idx < 0 {
        t.Fatal("ImportBrokerOrder not found (S3)")
    }
    body := content[idx:]
    if !strings.Contains(body, "ON CONFLICT (mt_account_id, ticket) DO NOTHING") {
        t.Fatal("ImportBrokerOrder must use ON CONFLICT (mt_account_id, ticket) DO NOTHING (S3, not ON CONFLICT (id))")
    }
}
```

**helper（内联定义）**:
```go
package mthub

import (
    "os"
    "path/filepath"
    "strings"
    "testing"
)

func mustReadFile(t *testing.T, relPath string) string {
    t.Helper()
    wd, err := os.Getwd()
    if err != nil {
        t.Fatalf("getwd: %v", err)
    }
    // wd is .../backend/internal/mthub → backend root is ../..
    backendRoot := filepath.Join(wd, "..", "..")
    data, err := os.ReadFile(filepath.Join(backendRoot, relPath))
    if err != nil {
        t.Fatalf("read %s: %v", relPath, err)
    }
    return string(data)
}
```

## 对抗证明 RED→restore→GREEN

**RED S1**: 将 `created_at >= $2` / `close_time >= $2` 改回无下界 → T1 RED。
**RED S2**: 将 `ImportBrokerOrder` 调用注释掉 → T2 RED。
**RED S3**: 将 `ON CONFLICT (mt_account_id, ticket) DO NOTHING` 改回 `ON CONFLICT (id) DO NOTHING` → T4 RED。
**RED S4**: 将 `isNonTerminalOMSState` 改回 `antState == string(OMSStateSubmitted)` → T3 RED。
**restore**: 逐项恢复 → 全 4 测试 GREEN。

## 验收门禁

```bash
cd backend
go build ./...
go vet ./...
go test ./internal/mthub/...
go test -race -count=3 ./internal/mthub/...
go run ./tools/check-file-lines --strict   # 0 errors
git diff --check                             # clean
```

## 红队自审 A-F

- **A 架构**: 复用 `brokerOrderStateToOMS`/`TradeRecordRepository.Create`（含 hash chain）；`ImportBrokerOrder` 用新 SQL 避开 `OmsWriter.InsertOrder` 的 id-based 冲突；无逆向依赖
- **B 实现**: S1 加时间下界是最简方案；S2 ghost 补写是 ADR-0013 §2.3 要求；S3 新增方法而非改 `InsertOrder` 签名；S4 helper 函数最简
- **C 洁净**: check-lines 0 errors / 无死代码 / 无 TODO / 无调试残留
- **D 正确性**: T1-T4 对抗证明；ghost 补写幂等（ON CONFLICT DO NOTHING）；orphan 修复扩展到所有非终态；24h 窗口对称
- **E 合规**: 符合 ADR-0013 §2.3 "broker 有 PG 无 → INSERT"
- **F 文档**: registry/STATE/handover 同步

## 收工

更新 `docs/audits/tech-debt-registry.md`（`DATA-TRUTH-1` + `DATA-TRUTH-1-DESIGN-AUDIT` 条目状态）+ `docs/handoff/STATE.md` + `docs/audits/handover-audit-plan.md`（追加变更日志）。

**勿部署，停手等 Devin CLI 复审。**
