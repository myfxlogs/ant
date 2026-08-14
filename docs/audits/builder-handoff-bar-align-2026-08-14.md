# 施工交接：BAR-ALIGN（P0，回填器 bar open_ts 未取整 → 历史数据损坏 → 策略不信号/回测崩）

> **审计方根因定论 2026-08-14。** 这是"策略实盘/回测不产生信号"的**真正根因**——不是执行链坏，是历史 bar 数据损坏。最高优先。
>
> **铁律**：`scripts/verify-adversarial.sh` 自验删行必红 + commit + 部署 + 回填 registry + 不自行宣告完成。**数据清理是破坏性操作，先备份/在事务里做**。

---

## 根因（审计方实证）

回填器把 mtapi 历史 bar 转 mdtick.Bar 时，**open_ts 没取整到周期边界**：
- `backend/internal/mdgateway/adapter/mt4/price_history.go:86` `convertMT4Bars`：`OpenTsUnixMs: t.UnixMilli()`
- `backend/internal/mdgateway/adapter/mt5/price_history.go:92` `convertMT5Bars`：`OpenTsUnixMs: t.UnixMilli()`

mtapi 返回的 bar `Time` 带亚秒精度（如 1784977500.385s），`t.UnixMilli()` 直接存 → **1784977500385**（非 5m 对齐，偏移 10-744ms）。live 聚合器是对的（`bucket*periodMs`，对齐），所以**只有回填的历史 bar 坏**。

## 实证损坏规模

md_bars 里非对齐脏 bar（open_ts % periodMs != 0）：BTCUSDm 1m=**47968**、5m=**9595**、15m=3198、1h=799；全库 5m 非对齐 **34592** 条，跨 2 账户，从 57 天前到近期。account_id 多为空（回填数据）。

## 影响（P0）

1. **回测崩**：`backtest_worker_vm.go:91 engine.Run` 抛 `bars are not chronologically ordered at index 487`（脏 bar 乱序）→ 回测失败。
2. **策略不信号**：live/回测加载历史 context（含脏 bar）→ 指标（MACD 等）算在乱序/错误 bar 上 → 永不触发 → 0 信号。**这就是用户报告"策略不产生信号"的根因。**
3. 执行链本身没问题（组件测试过、报价数据在流）——是**输入数据坏了**。

## 修复（两部分）

### A. 代码修复（mt4 + mt5 convert 函数）
把 open_ts 取整到周期边界：
```go
// convertMT4Bars / convertMT5Bars 里：
pm := mdtick.PeriodMs(period)
openMs := t.UnixMilli()
openMs = openMs - (openMs % pm)   // 向下取整到周期边界（镜像 live 聚合器 bucket*Ms）
// 然后
OpenTsUnixMs:  openMs,
CloseTsUnixMs: openMs + pm,
```
**mt4 + mt5 都改**（两处 convert 函数）。

### B. 数据清理（破坏性，先备份）
脏 bar 取整会产生重复（唯一约束 `idx_md_bars_unique(broker,canonical,period,open_ts,close_ts)`）。两种：
1. **删除所有非对齐脏 bar + 重新回填**（推荐，干净）：`DELETE FROM md_bars WHERE open_ts_unix_ms % <periodMs> != 0`（按各周期算），然后用修复后的回填器重拉历史。
2. 或迁移脚本：取整 + 去重（保留每对齐边界一条）。
**先 `CREATE TABLE md_bars_backup_20260814 AS SELECT * FROM md_bars` 备份**，事务里做，验完再提交。

## 对抗证明

- **代码**：mock mtapi 返回 bar Time 带亚秒（如 1784977500385）→ 旧代码存 1784977500385（非对齐，RED）；新代码存 1784977500000（对齐，GREEN）。`verify-adversarial.sh` 验 convert 函数。
- **数据**：清理后 `SELECT COUNT(*) FROM md_bars WHERE period='5m' AND open_ts_unix_ms%300000!=0` = 0（GREEN）。
- **端到端**：清理+修复+重部署后，跑一个回测（之前崩的那个）→ 不再报 "not chronologically ordered" + 正常出结果；live MACD 策略正常产生信号。

## 红队自审
- [ ] mt4 + mt5 都改（漏一个 = 该平台仍脏）。
- [ ] 取整用**向下取整**（`openMs - openMs%pm`），不是四舍五入。
- [ ] close_ts 也用对齐后的 openMs+pm（不要用原 t.UnixMilli()+pm，否则 close 也不对齐）。
- [ ] 数据清理前**备份**，事务内做，验 COUNT=0 再提交。
- [ ] 清理后**重新回填**（否则历史数据空，回测/live context 不够）。
- [ ] 重部署后重启策略 run（让它重载干净的 context）。
