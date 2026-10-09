# TASK: env 收口 PG（alphaforge）— 派单给 Zcode

> 来源：ARB 项目同类收口审计结论平移（2026-10-09）。
> 原则：**env 只留「连库之前就需要的引导件」，其余业务信息全部进 PG**。
> ant 现状已具备承载面：`global_settings`(16 行)/`system_config`(28)/`system_ai_configs`/`risk_configs` + `internal/repository/*_settings.go` 仓储层 + `xpub_audit.go` 已有「env→DB 回退」先例模式。

## 现状盘点（审计实拍，开工前再复核一遍勿凭印象）

- `.env` 17 键；`internal/config/config.go` 一处集中 `getenv*` 40+ 键；散落 `os.Getenv` ~16 处
- **无统一 overlay/seed 机制**（ARB 侧 D-368 已有 DB>env>default overlay 可参考移植）

## 三层分类（逐键归位，清单即验收表）

### A. 引导件 —— 留 env（注释注明理由）
`DB_HOST/DB_PORT/DB_USER/DB_PASSWORD/DB_NAME/DB_SSLMODE/DB_MAX_CONNS`、`NATS_URL`、`REDIS_*`、`PORT`、`SPILL_DIR`、`GEOIP_DB_PATH`、`ANT_KEY_DIR`、`ANT_MASTER_KEY(_FILE)`（解密封件钥匙，不能入库）、`TEST_*`、`SENTRY_*`、`OTEL_*`

### B. 构建/部署期 —— 留 env（PG 救不了：前端 bake 与 compose 期变量）
`VITE_MENU_MODE`、`NODE_BUILD_IMAGE`、`ANT_FRONTEND_PORT`

### C. 业务旋钮 → PG（global_settings / system_config 择其一，按该表既有语义归位）
`REQUIRE_KYC`、`REQUIRE_DISCLAIMER`、`REQUIRE_QUESTIONNAIRE`、`REQUIRE_EMAIL_VERIFICATION`、`CHAIN_MONITOR_ENABLED`、`COOKIE_SECURE`、`APP_URL`、`WEBAUTHN_RP_ID/ORIGIN`、`TRONGRID_GRPC_ENDPOINT`、`ALPHAFORGE_RISK_GATE_*`(3)、`AI_DAILY_COST_LIMIT_USD`、`AI_MIN_BALANCE`

### D. 秘密件 → 加密存储（ANT_MASTER_KEY 加密体系，禁明文落 global_settings）
`JWT_SECRET`、`SMTP_HOST/PORT/USER/PASSWORD/FROM/TO`、`MTAPI_TOKEN`、`TRONGRID_API_KEY`、`TRONSCAN_API_KEY`、`DEPOSIT_XPUB`+`DEPOSIT_XPUB_FINGERPRINT`、`UMAMI_APP_SECRET`

## 机制要求

1. **seed-once**：boot 时 env 值 → PG（仅缺键种入），此后 PG 唯一真相源、env 改名失效；seed 必须先于任何读 config 的消费点
2. **调用点改造**：`config.go` 的 `getenv*` 改读 settings/secrets 仓储；散落 `os.Getenv` 全部收口
3. **门禁（防回潮，含其他席位）**：新增脚本/CI 检查——`internal/` + `cmd/` 中新增 `os.Getenv`/`os.LookupEnv` 仅允许落在白名单文件（bootstrap 清单），其余拒绝；AGENTS.md 补一行钦定「业务配置唯一真相=PG」
4. **热更语义自查**：`ALPHAFORGE_RISK_GATE_KILLSWITCH` 类急停键——确认现状是 boot-only 读（迁 PG 语义等价）还是运行期轮询读（若是则迁 PG 后应保持可热翻转，别降级成重启生效）

## 禁区

- secrets 禁明文落 global_settings/system_config（走加密轨）
- B 档构建期键不要硬搬 PG（搬了也不生效）
- `DEPOSIT_XPUB` 现有「DB 优先 env 回退」语义保留（xpub_audit.go 注释已是先例）
- 不改任何键的业务默认值语义

## 验收

- A/B/C/D 四档清单逐键落位可核；`grep -rn "os.Getenv\|LookupEnv" backend/internal backend/cmd` 仅剩引导件
- 删除 .env 中 C/D 档键后服务行为不变（PG 已 seed）
- 门禁拦得住：测试提交一个非白名单 `os.Getenv` → fail
- 既有测试绿

## 参考实现（ARB 侧 /opt/arb 现成可抄）

- `internal/config/app_config_keys.go` — 白名单注册表+typed setter（boolv/intv/f64/durv/strv/listv/setv）
- `internal/config/app_config_overlay.go` — DB>env>default boot overlay + env 残值告警
- `internal/store/pgstore/app_secrets.go` — 加密 seed 径
- `scripts/check-env-reads.sh` + `scripts/env-allowlist.txt` — 门禁三层闸+ratchet 白名单
