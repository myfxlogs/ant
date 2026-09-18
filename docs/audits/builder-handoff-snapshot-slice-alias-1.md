# 施工派工单 — SNAPSHOT-SLICE-ALIAS-1

> 设计 SSOT（Devin CLI 设计实查 2026-09-18，落档 commit 见 git log）。施工方只施工不决策；遇偏差停下上报。

## 0. 立项背景

QS-2.4-F2（registry 行 212）：retained 快照 slice 别名依赖跨文件 immutable 约定。设计实查把别名图从 registry 原载的 2 点扩到 **5 条边**——现存全部"安全"只因为每条写入路径都恰好 fresh-build，任一方未来加原地写即静默 race 且污染的是**持久态**（broker `latest` / cache `snapshots`），损坏跨事件残留。

## 1. 设计实查实证（已核实，勿重复审计）

### D1 别名图（全链路）

| # | 边 | 位置 | 说明 |
|---|---|---|---|
| E1 | retained↔delivery | `mthub/broker_types.go:143` `retained := *merged` 浅拷→`:150/:158` merged 投递全订阅者 | retained 与所有投递对象共享两 slice 底层数组 |
| E2 | retained↔replay | `Subscribe:170` / `SubscribeAll:194` `latest` **原指针**入队 | 新订阅者拿到 retained 本体（连浅拷都无） |
| E3 | merged↔旧 retained | `mergePositionSnapshot:94` `merged := *current`，incoming 非 PositionsAuthoritative 时保留 current 数组 | 旧 retained 在 `:147` 被覆盖后无 broker 侧残留——**不修** |
| E4 | cache retained↔delivery | `position_cache.go:120` `merged := *snap` 且 `snap.PositionsAuthoritative`（或无权威 current）时 merged 直存 snap 数组 | `:95-96`/`:124-125` 两分支已 fresh-copy，权威路径是缺口 |
| E5 | cache retained↔读者 | `GetSnapshot:159` / `GetFresh×3` 原指针返回存储对象 | 读者拿到 cache 内部态 |

### D2 现状安全性（已验证）

- 生产端 3 站全部 fresh-build 后发后即弃：`pipeline_callbacks.go:49/87`、`mutation_helpers.go:58`（`make`+append→Publish→不再触）。
- 消费端全只读：`backfillContextStrings`/`activeSessionToProto`/`snapshot_persister`/barrier listener。
- 测试无指针身份依赖（`broker_types_test.go` 全值断言）。
- `vm.cachedPositions` 是 VM 侧独立 slice，与本债无关。

### D3 裁定：边界不变量（非"仅钉注释"）

**不变量一句话**：`PositionSnapshotBroker` 与 `PositionCache` 永不向边界外暴露其 retained `Positions`/`PendingOrders` 底层数组。

- **拒绝仅钉注释**：约定横跨 mthub+strategy+server 5+ 文件，正是本债要消的隐患形态。
- **拒绝 copy-per-live-delivery**：live 投递是惯用 Go pub-sub 共享只读消息；N 订阅者×每事件拷贝不成比例（P3）。该残留面用 API 契约注释钉住（见 S4）。
- E3 不修：旧 retained 在 `:147` 后 broker 不可达，alias 只影响已投递对象（由契约覆盖）。

### D4 测试判别设计（确定性，无时序）

变异方向单一：删掉任一深拷贝点 → 对应隔离性断言观察到损坏 → RED。无需 race/时序。

## 2. 施工步骤

### S1 `mthub/broker_types.go` — retained/replay 私有化

新增 helper（文件底部或 mergePositionSnapshot 旁）：

```go
// cloneItems deep-copies a PositionSnapshotItem slice; nil stays nil.
func cloneItems(in []PositionSnapshotItem) []PositionSnapshotItem {
	if in == nil {
		return nil
	}
	out := make([]PositionSnapshotItem, len(in))
	copy(out, in)
	return out
}
```

- `Publish` `:143-146` 改为：
  ```go
  retained := *merged
  // SNAPSHOT-SLICE-ALIAS-1: retained state must never share mutable
  // backing arrays with objects delivered to subscribers.
  retained.Positions = cloneItems(merged.Positions)
  retained.PendingOrders = cloneItems(merged.PendingOrders)
  retained.UpdateTicket = 0
  retained.UpdateType = ""
  retained.UpdateMagic = 0
  ```
- `Subscribe` `:169-171` replay 处：`ch <- latest` → 发送浅拷+深拷切片的副本：
  ```go
  if latest := b.latest[accountID]; latest != nil {
      cp := *latest
      cp.Positions = cloneItems(latest.Positions)
      cp.PendingOrders = cloneItems(latest.PendingOrders)
      ch <- &cp
  }
  ```
- `SubscribeAll` `:192-197` 同型：`ch <- latest` → 同样副本化。

### S2 `position_cache.go` — put 存储私有化 + Get* 边界拷贝

- `put` `:120` 分支（financials-authoritative）：在 `:121` 的 `if` 块后补 `else`，覆盖两种现状缺口（`snap.PositionsAuthoritative` 直存 / 无权威 current 直存 snap 数组）：
  ```go
  merged := *snap
  if !snap.PositionsAuthoritative && current != nil && current.PositionsAuthoritative {
      // ... 既有 :124-125 fresh-copy 保持不动 ...
  } else {
      // SNAPSHOT-SLICE-ALIAS-1: stored slices must be cache-private —
      // the incoming snap is shared with other broker subscribers.
      merged.Positions = append([]mthub.PositionSnapshotItem(nil), snap.Positions...)
      merged.PendingOrders = append([]mthub.PositionSnapshotItem(nil), snap.PendingOrders...)
      // PositionsAuthoritative/provenance: keep snap's own values
      // (merged = *snap already carries them; do NOT overwrite).
  }
  ```
  注意 `:91` positions-only 分支（`:95-96` 已 fresh-copy）不动。
- 四个 Get 方法返回值改发私有副本。加 helper：
  ```go
  // snapshotCopy returns a copy whose Positions/PendingOrders are
  // cache-private; callers may freely mutate the returned copy.
  func snapshotCopy(s *mthub.PositionSnapshot) *mthub.PositionSnapshot {
      if s == nil {
          return nil
      }
      cp := *s
      cp.Positions = append([]mthub.PositionSnapshotItem(nil), s.Positions...)
      cp.PendingOrders = append([]mthub.PositionSnapshotItem(nil), s.PendingOrders...)
      return &cp
  }
  ```
  `GetSnapshot:159` `return c.snapshots[accountID]` → `return snapshotCopy(c.snapshots[accountID])`；`GetFreshFinancialSnapshot:178`/`GetFreshPositionSnapshot:202`/`GetFreshTradingSnapshot:231` 三处 `return snap, true` → `return snapshotCopy(snap), true`。

### S3 契约注释钉住（共享只读投递面）

`PositionSnapshot` 结构体 doc（`:13` 上方注释块）追加一行契约：
`// 契约：经 broker 投递的对象为共享只读；需改写请先自行拷贝（SNAPSHOT-SLICE-ALIAS-1）。`
`Subscribe`/`SubscribeAll` doc 各补一句相同语义。

### S4 新测试（broker_types_test.go + position_cache_test.go，全部确定性）

- `TestPositionSnapshotBroker_RetainedIsolation`：Publish 权威 snap → 收投递 ev → **改 `ev.Positions[0].Ticket=999`** → 晚订阅者 replay 必须仍为原值。无 S1-Publish 深拷 → RED。
- `TestPositionSnapshotBroker_ReplayIsolation`：Publish → Subscribe 收 replay → 改 `ev.Positions[0].Ticket=999` → 再 Subscribe 的 replay 必须为原值。无 S1-replay 深拷 → RED。
- `TestPositionCache_PutIsolation`：构造 `FinancialsAuthoritative+PositionsAuthoritative+CapturedAt+FinancialsSource` 齐全的 snap → `PutSnapshot` → 改原 `snap.Positions[0].Ticket` → `GetSnapshot` 必须为原值。无 S2-put 拷贝 → RED。
- `TestPositionCache_GetCopyIsolation`：`PutSnapshot` 同上 → `GetSnapshot` 改返回对象 item → 再 `GetSnapshot` 必须为原值。无 S2-Get 拷贝 → RED。

### S5 全量回归

mthub 包 + strategy 包全量（投递共享面消费者行为不变，仅隔离性增强）。

## 3. mutation 验收项（独立复审执行）

1. 删 `Publish` retained 两 cloneItems → T-Retained RED。
2. 删 `Subscribe` replay 副本化 → T-Replay RED。
3. 删 `put` else 拷贝 → T-Put RED。
4. 删 `snapshotCopy` 应用（GetSnapshot 还原原指针） → T-GetCopy RED。

## 4. 验收门（机检五件套+）

```bash
cd backend && go build ./...
go test ./internal/mthub/ ./internal/connect/strategy/
go test -race -count=3 ./internal/mthub/ ./internal/connect/strategy/
go vet ./internal/mthub/ ./internal/connect/strategy/
gofmt -l internal/mthub/broker_types.go internal/mthub/broker_types_test.go internal/connect/strategy/position_cache.go internal/connect/strategy/position_cache_test.go
go run ./tools/check-file-lines --strict
git diff --check
```

## 5. 边界/不做

- 不改 live 投递共享语义（copy-per-delivery 已否决，契约注释钉住）。
- 不动 `mergePositionSnapshot` 的 E3 carry-forward 别名（旧 retained 无残留）。
- 不改生产端三站（已 fresh-build 验证）。
- 不改 `GetFresh*` 的 freshness 判定逻辑，仅返回值副本化。
- 不碰 TickBroker/BarBroker/AccountStatusBroker（各自独立类型，无 slice 字段同类问题）。
- 勿部署、勿 push、禁 `--no-verify`。

## 6. 完成报告格式

`[施工完成:SNAPSHOT-SLICE-ALIAS-1] @<commit-hash>` + 机检五件套结果 + 4 测试名逐一列出。勿部署，停手等 Devin CLI 复审。
