# 运维部署手册

## 1. 当前入口

本手册是当前运维部署入口。容器操作以 `docs/ops/container-restart-checklist.md` 为准。

## 2. 服务组成

Docker Compose 编排主要服务：

- `frontend`
- `backend`
- `strategy-service`
- `postgres`
- `redis`

## 3. 常用操作

```bash
docker compose ps
docker compose logs --tail=200 backend
docker compose logs --tail=200 frontend
docker compose up -d --build backend frontend
```

## 4. 健康检查

```bash
curl -fsS http://localhost:8012/health
curl -fsS http://localhost:8012/
```

## 5. 注意事项

- 不通过删除数据库卷解决业务问题。
- 不在未验证编译和测试前重启生产容器。
- 业务接口变更必须通过 ConnectRPC 契约和生成代码同步完成。

## 6. 生产部署（`43.255.29.23` + Cloudflare 隧道）

对外 **浏览器入口** 为 Cloudflare 隧道域名 **`https://trader.myfxlogs.org`**（HTTPS）。前端构建时若把 API 写成 **`http://`** 会在 HTTPS 页触发 **Mixed Content** 被拦截，须通过 `VITE_API_URL` / `VITE_STREAM_URL` 写入与隧道一致的 **`https://trader.myfxlogs.org`**（无尾部斜杠）。

### 6.1 SSH 登录（运维，连源站）

| 项 | 值 |
|----|-----|
| 地址 | `43.255.29.23` |
| 端口 | **`33953`**（非 22） |
| 用户 | `root` |

**本机密钥**建议放在仓库根目录 **`./ssh/id_ed25519`**（目录 **`ssh/`** 已在 **`.gitignore`** 中忽略，**禁止**把私钥提交进 Git）。若使用 `~/.ssh/config` 别名（示例：`Host ant` → `HostName 43.255.29.23`、`Port 33953`、`User root`、`IdentityFile …`），登录：

```bash
ssh ant
```

未配置别名时，在**仓库根**执行（`-i` 按本机路径修改；Windows 可为 `D:/antssh/id_ed25519`）：

```bash
ssh -i ./ssh/id_ed25519 -p 33953 -o IdentitiesOnly=yes root@43.255.29.23
```

登录后再执行下文 **Docker** 命令（在服务器上的仓库路径内）。

### 6.2 前端构建与 Compose（隧道域名）

在**服务器** `/opt/anttrader`（或等价路径）执行：

```bash
cd /opt/anttrader
VITE_API_URL=https://trader.myfxlogs.org VITE_STREAM_URL=https://trader.myfxlogs.org docker compose build frontend
VITE_API_URL=https://trader.myfxlogs.org VITE_STREAM_URL=https://trader.myfxlogs.org docker compose up -d --build backend frontend
```

- 用户验证：浏览器打开 **`https://trader.myfxlogs.org`**。  
- 源站健康（可选）：`ssh` 登录后 `curl -fsS http://127.0.0.1:8012/health`（容器映射 **`8012:8080`** 不变；隧道将公网 HTTPS 转到源站由你侧 Cloudflare / Nginx 配置）。

### 6.3 更换隧道域名或直连 IP 调试

若隧道域名变更，只改构建命令中的 `VITE_API_URL` / `VITE_STREAM_URL` 后 **`build frontend`**。  
**仅**内网用 `http://IP:8012` 调试、且外网仍走 HTTPS 时：**不要**用 `http://…` 写入 `VITE_*` 打生产包，应使用与真实用户相同的 **https** 基址，或临时去掉 `prod-host` 叠加仅用同源（见 `.env.example` 说明）。
