# ADR-0031 · env 收口 PG：业务配置唯一真相=PG（system_config + platform_secrets 双轨）

- **状态**：Accepted（施工完成待独立复审——ENV-TO-PG-1）
- **日期**：2026-10-09
- **来源**：业主令（2026-10-09 派单，ARB 项目 D-368/D-515/D-564 同类收口审计结论平移）；设计 SSOT = `docs/plan/2026-10-env-to-pg-consolidation.md`
- **涉及功能块**：`backend/internal/config`、`backend/cmd/server`（boot 链）、`backend/internal/repository`、`backend/internal/secrets`
- **关联**：ADR-0011（secrets 信封加密）、ADR-0026 R5（deposit xpub 指纹锚）

## 1. 背景

`.env` + `config.Load()` 集中读 40+ 键、散落 `os.Getenv` ~16 处，无统一 overlay/seed 机制：配置改值需改 .env 重启、秘密件明文躺主机 env 文件、无门禁防新增散读。ant 已具备承载面：`system_config`（平台 KV，管理面可改，28 行）+ `internal/secrets`（AES-256-GCM purpose 制）。

## 2. 决策

1. **四档分类**（清单即验收表，逐键落位见派工单）：
   - **A 引导件**（留 env）：DB_*/NATS_*/REDIS_*/PORT/SPILL_DIR/GEOIP_DB_PATH/ANT_KEY_DIR/ANT_MASTER_KEY(_FILE)/SENTRY_*/OTEL_*/MTAPI_MT4|MT5_HOST/TEST_*
   - **B 构建/部署期**（留 env，PG 救不了）：VITE_MENU_MODE/NODE_BUILD_IMAGE/ANT_FRONTEND_PORT/NATS_USER|PASSWORD（compose 插值）/UMAMI_APP_SECRET（喂 umami 侧容器，后端零读取）
   - **C 业务旋钮**（→ `system_config`，21 键）：REQUIRE_*(4)/CHAIN_MONITOR_ENABLED/COOKIE_SECURE/RATE_LIMIT_*(2)/APP_URL/WEBAUTHN_RP_ID|ORIGIN/TRONGRID_GRPC_ENDPOINT/ALPHAFORGE_RISK_GATE_*(3)/AI_*(4)
   - **D 秘密件**（→ `platform_secrets` 密文轨，10 键）：JWT_SECRET/SMTP_*(6)/MTAPI_TOKEN/TRONGRID_API_KEY/TRONSCAN_API_KEY
2. **seed-once**：boot 时 env 非空值逐键 `ON CONFLICT DO NOTHING` 入库（已有行永不被 env 覆盖）；此后 PG 唯一真相源，env 改值失效（残值告警日志）。seed 先于任何消费点。
3. **DB-wins overlay**：boot 读 `system_config` 全表 + 解密 `platform_secrets`，typed 覆写 `config.Config` 字段；DB 坏值 warn+保留 env/default（不置零不致命）；D 档解密失败 boot fatal（错主钥/篡改=配置不可用，fail-closed）。
4. **`config.Load()` 保留全部键 env+default 供值**（零语义漂移），仅作 overlay 前的过渡兜底；boot 链：`connectPostgres → newSecretsClient → seedAndOverlayConfig → Validate → initInfrastructure(pool, secClient)`。
5. **加密 purpose**：新增 `secrets.PurposePlatformSecret`（HKDF info），密文存 `platform_secrets.value_enc BYTEA`（migration 282）。
6. **防回潮门禁**：`scripts/check-env-reads.sh` + `scripts/env-allowlist.txt`（文件级白名单=bootstrap 清单），pre-commit 强制；AGENTS.md 红线补「业务配置唯一真相=PG」。

## 3. 禁区（语义保留）

- **DEPOSIT_XPUB(+FINGERPRINT) 不入本轨**：维持 `xpub_audit.go`「DB 优先 env 回退」；指纹 env-only（ADR-0026 R5 防替换锚，DB 被改也无法伪造指纹）。
- **不改任何键的业务默认值语义**：defaults 留在 `config.Load()`；AI 阈值解析分支逐语义平移（正十进制/非正整数/坏串三路）。
- B 档不硬搬 PG（搬了不生效）。

## 4. 后果

- +：改配置不再动 .env/重启镜像文件；秘密件明文不落主机文件（首次 seed 后可从 .env 删除）；新增散读被门禁拦截。
- −：boot 多两次轻量查询（system_config 全表 + platform_secrets 全表，行数 ~50，一次性成本）。
- −：无热更新（与现状一致——RiskGate 三键实拍为 boot-only 读，运行期零写入，无语义降级；AI 配额运行期热更走既有 agent_managed_settings 层，不受影响）。
- C 档键种入 `system_config` 后管理面可见可改（admin_visible=TRUE，value_type 按 bool/number/string 标注）；改后需重启生效（boot overlay 语义，管理面已按此心智使用）。

## 5. 验收锚

- `grep -rn "os.Getenv\|LookupEnv" backend/internal backend/cmd`（非 _test）仅剩白名单 5 文件；
- 删除 .env C/D 键后行为不变（PG 已 seed）；门禁拦非白名单新增；既有测试绿。
- 明细与 mutation 证据见 `docs/audits/tech-debt-registry.md` ENV-TO-PG-1 行。
