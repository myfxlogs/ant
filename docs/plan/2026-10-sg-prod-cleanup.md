# TASK: sg 生产机开发残留清理（SG-PROD-CLEANUP-1）— 派单给 Zcode

> **立项背景**：sg（192.229.85.35，root SSH，开发机 `~/.ssh/config` Host sg）曾兼做开发机；按
> 机器分工约束（constraints.md §Deployment，2026-10-09 定），sg 转纯生产部署机，开发残留
> 全面清理让出资源。实拍：磁盘 58G/用 40G/69%；Docker 可回收镜像 ~3.6G；/root/.cache/go-build
> 2.3G；/opt/ant/frontend/node_modules 490M（docker 构建，host 不需要）；stale 仓库副本
> /opt/antt(218M)+/opt/anttrader(167M)；/opt/chrome 339M（无进程）；/root/.npm 177M。
> **本文档 = 设计 SSOT**。坐标基于 2026-10-09 Devin CLI 对 sg 的实拍证据，执行时每步先复核再动手。
>
> **业主裁决（2026-10-09）**：sg 跑 **ant + arb 双项目生产**，arb 整机（含 arb-postgres-test）
> 是生产资产一律不碰；/opt/binance-proxy 保留；agent 残留目录先盘点报告**不删**；node/npm
> 工具链保留（最优解：清缓存不卸本体——docker 构建不需要但运维面可能用，本体占用小）。

## 目标与约束

- **目标**：释放 ~4.8G+ 磁盘 + 清除开发期残留；生产服务零中断、零数据损失。
- **纪律**：每步先跑 guard 复核（证据不符即跳过该步并在报告标注 `SKIP:<原因>`）；破坏性命令
  只许按本文精确路径执行，不得扩散（禁 `rm -rf /opt/*` 类通配、禁 `docker system prune -a`、
  **禁 `docker volume prune`**）；全程不停 ant/arb 任何容器、不改 crontab、不改 systemd 单元。
- **执行环境**：全部命令在 sg 上以 root 执行（`ssh sg`）。

## 红线清单（任何步骤不得触碰）

| 资产 | 说明 |
|------|------|
| compose 项目 `ant` 全部容器/在役镜像/volume/network | alphaforge-backend/frontend/postgres/nats/redis/umami/prometheus/alertmanager/uptime-kuma |
| compose 项目 `arb` 全部容器/volume | **arb-core、arb-postgres、arb-postgres-test 均为 arb 生产资产（业主裁决）** |
| `/opt/ant` | 生产仓库（仅允许 S5 清 `frontend/node_modules`）；含 `/opt/ant/backups`（cron 每日备份落点）、`.git`、`.env` |
| `/opt/arb`、`/opt/arb-backups`（1.4G 生产备份）、`/opt/cex-proxy` + `cex-proxy.service`（arb 生产 :8443）、`/opt/binance-proxy`（业主保留） | arb 生产链 |
| `/usr/local/go`、`/usr/bin/go`、`/usr/bin/node`、`/usr/bin/npm` | `/usr/local/go` 是 arb digest cron 依赖；node/npm 保留（业主裁决=只清缓存） |
| `/root/go/pkg`（arb cron `go run` 模块缓存）、`/root/.zcode`（施工 agent 本体）、`/root/.ssh`、`/root/.docker`、crontab 全部条目、全部 systemd 单元 | 生产/运维依赖 |
| `docker volume` 一切对象 | 禁 prune；arbpgsql/ant 数据卷在册 |

## 施工步骤（S1–S8 串行，逐步先 guard 后执行）

### S1 基线盘点（只读，证据留档）
- `df -h /`、`docker system df`、`docker ps -a`、`git -C /opt/ant status --short && git -C /opt/ant rev-parse HEAD`、
  `git -C /opt/ant fetch origin && git -C /opt/ant rev-parse origin/main`。
- 落点：报告记录 before 值；**若 /opt/ant 工作区脏**（有未提交/未跟踪文件）→
  `git -C /opt/ant stash push -u -m "sg-cleanup-<date>"`（可恢复，禁 reset/checkout 覆盖），
  报告 stash ref；若 `HEAD != origin/main` → 同步属部署动作，本单不做，仅报告。

### S2 Docker 可回收项（~3.7G）
- `docker builder prune -f`；`docker image prune -f`（**只清 dangling，禁 `-a`**）。
- guard：`docker images -f dangling=true` 先列清单入报告；运行中 11 容器镜像不得受影响
  （dangling prune 语义保证，仍复核 `docker ps` 全 healthy）。

### S3 Go 构建缓存（~2.3G）
- guard：`du -sh /root/.cache/go-build` 确认是 go-build；`/usr/local/go/bin/go env GOCACHE`
  应指向该路径。**只许** `/usr/local/go/bin/go clean -cache`（regrows，arb digest cron
  下次运行仅变慢不失败）；禁动 `/root/go/pkg`（模块缓存，删了 arb cron 要重新下载依赖）。

### S4 npm 缓存（~177M）
- `rm -rf /root/.npm/_cacache`（npm 缓存体）；`/root/.npm` 目录本体及配置保留。

### S5 ant 仓库内 dev 残留（~490M+）
- guard：`ls /opt/ant/frontend/node_modules >/dev/null` 且确认 compose frontend 走
  `NODE_BUILD_IMAGE` docker 构建（`/opt/ant/docker-compose.yml` frontend.build.args 已实拍）。
- `rm -rf /opt/ant/frontend/node_modules`；**保留** `/opt/ant/frontend/dist`（疑似部署产物）、
  `/opt/ant/backups`、`.git`、`.env`。
- 附带：`find /opt/ant -name node_modules -o -name .next -o -name dist -prune` 列其他构建残留
  入报告（只报告，除 frontend/node_modules 外不删）。

### S6 stale 仓库副本（~385M）：/opt/antt、/opt/anttrader
- guard（每目录独立判定）：`ls -d /opt/<d>/.git` 存在则 `git -C /opt/<d> status --porcelain`
  与 `git -C /opt/<d> log --branches --not --remotes`——**有任何未提交改动或未推送 commit →
  该目录 STOP，tar.gz 到 /root/cleanup-quarantine/ 并在报告标 `SKIP:dirty`**；
  `ls /opt/anttrader/backups` 若有文件列出入报告（确认是陈旧备份再删）。
- 两目录 guard 全过才 `rm -rf /opt/antt /opt/anttrader`（逐个删，不许合并通配）。

### S7 dev 工具残留：/opt/chrome（339M）
- guard：`pgrep -f chrome` 空 + `grep -rl chrome /etc/systemd/system/ 2>/dev/null` 空 +
  `grep -rn "opt/chrome" /opt/ant/scripts /opt/arb/scripts crontab 2>/dev/null` 空 → `rm -rf /opt/chrome`。

### S8 日志与临时文件（~200M）
- `journalctl --vacuum-size=50M`；`find /tmp -mtime +7 -delete`（先 `find /tmp -mtime +7 | head`
  入报告）；`/var/log/*.gz` 与 `*.log.*` 轮替旧档删除（先 `ls -la` 入报告，保留当前代+上一代）。
- **不改** `/var/log/ant-backup.log`、`/var/log/docker-builder-prune.log` 当前文件（可 truncate 零长？——不，保留原样，只删轮替档）。

### S9 盘点不删项（报告专用，禁止删除）
- `/root/.claude`（340M）、`/root/.codex`（1.5M）、`/root/.cursor`（876K）、`/root/.local`（865M）、
  `/root/go/bin`（346M）：`du -sh <dir>/*` 各列 top 构成入报告（业主裁决「先搞清楚情况」——
  本轮只盘点，处置待决策）。
- `/root/go/bin` 另查：列二进制名 → `crontab -l`、`grep -r <name> /opt/arb /opt/ant/scripts`、
  `systemctl list-units | grep <name>` 全空才在报告标「可候选删除」（**本轮仍不删**）。

### S10 终态验证
- `df -h /` after 对比；`docker ps` 11 容器全 Up healthy；`docker exec alphaforge-backend wget -qO- localhost:8080/healthz` 或等价健康探针；
  arb 侧 `docker inspect arb-core --format {{.State.Status}}`=running、`systemctl is-active cex-proxy`=active；
  `crontab -l | wc -l` 与 S1 基线一致（零改动）；`git -C /opt/ant status` clean（或 stash ref 记录）。

## 验收

- before/after `df -h` 释放量 ≥4G；`docker system df` reclaimable 收敛。
- 双项目零中断：ant 11 容器 healthy 不断；arb-core/arb-postgres/arb-postgres-test/cex-proxy 状态不变；crontab 零 diff。
- 报告含：每步 guard 输出、删除清单+体积、S9 盘点明细、S1 stash/偏离项（若有）、SKIP 项原因。
- 报告格式：`[施工完成:SG-PROD-CLEANUP-1]` + 证据块。

## 边界/不做

- 不做部署（不 pull/build/up）；不动 ant/arb 任何运行物与数据卷；不删 agent 盘点目录（S9）；
  不碰 systemd/crontab/防火墙/sshd；不清理 arb 项目内部任何文件（arb 侧自清是 arb 的事）。
- 发现与实拍不符（容器清单变了、目录已不在、guard 失败）→ 该步 SKIP 并报告，不即兴扩大范围。

> 勿做范围外动作，完成报证据等 Devin CLI 复审。禁 `--no-verify`，禁 force-push，禁 `docker system prune -a`/`volume prune`。
