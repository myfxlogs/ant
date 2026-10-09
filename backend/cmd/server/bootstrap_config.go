package main

// bootstrap_config.go —— ENV-TO-PG-1：boot 三步（ADR-0031 / 派工单
// docs/plan/2026-10-env-to-pg-consolidation.md）。
//
//	connectPostgres → newSecretsClient → seedAndOverlayConfig → Validate → initInfrastructure
//
// seed-once：C 档 env 值→system_config、D 档 env 值→platform_secrets（仅缺键种入，
// ON CONFLICT DO NOTHING——已有行永不被 env 覆盖）。此后 PG 唯一真相源，env 改值失效。
// overlay：DB>env>default，DB 行 typed 覆写 cfg（C 档坏值 warn+保留；D 档解密失败 boot
// fatal——错主钥/篡改=配置不可用）。值语义：C 档值可进日志；D 档值永不进日志。
// 顺序红线：必须先于任何 config 消费点（组件构造在 main.go initInfrastructure 之后）。

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"alphaforge/internal/config"
	"alphaforge/internal/repository"
	"alphaforge/internal/secrets"
)

// connectPostgres 从 initInfrastructure 前移（ENV-TO-PG-1）：seed/overlay 需要 pool，
// 必须先于其余基建装配。DSN 全部来自 A 档引导件 env。
func connectPostgres(cfg *config.Config, log *zap.Logger) *pgxpool.Pool {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName, cfg.DBSSLMode)
	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Fatal("pg parse config failed", zap.Error(err))
	}
	if cfg.DBMaxConns > 0 {
		poolCfg.MaxConns = int32(cfg.DBMaxConns)
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		log.Fatal("pg connect failed", zap.Error(err))
	}
	log.Info("pg pool configured", zap.Int32("max_conns", poolCfg.MaxConns))
	return pool
}

// newSecretsClient 从 initInfrastructure 前移：D 档 seed（加密）与 overlay（解密）需要。
func newSecretsClient(cfg *config.Config, log *zap.Logger) secrets.Client {
	if mk := cfg.AntMasterKey; mk != "" {
		secClient, err := secrets.New(mk, 1)
		if err != nil {
			log.Fatal("secrets: cannot create client from ANT_MASTER_KEY", zap.Error(err))
		}
		log.Info("secrets: client initialized")
		return secClient
	}
	log.Fatal("ANT_MASTER_KEY is required — generate one with: go run cmd/ant-vault/main.go")
	return nil
}

// systemSettingDescriptions C 档 seed 行的管理面描述（value_type 由 config.SystemSettingKeyType 落列）。
var systemSettingDescriptions = map[string]string{
	"REQUIRE_KYC":                             "登录/提现 KYC 强制闸（env REQUIRE_KYC 收口）",
	"REQUIRE_DISCLAIMER":                      "免责声明确认闸（env REQUIRE_DISCLAIMER 收口）",
	"REQUIRE_QUESTIONNAIRE":                   "投资者问卷闸（env REQUIRE_QUESTIONNAIRE 收口）",
	"REQUIRE_EMAIL_VERIFICATION":              "登录前邮箱验证闸（env REQUIRE_EMAIL_VERIFICATION 收口）",
	"CHAIN_MONITOR_ENABLED":                   "TRON 充值监听循环闸（env CHAIN_MONITOR_ENABLED 收口）",
	"COOKIE_SECURE":                           "refresh_token Secure 标志（env COOKIE_SECURE 收口）",
	"RATE_LIMIT_ENABLED":                      "登录限流总闸（env RATE_LIMIT_ENABLED 收口）",
	"ALPHAFORGE_RISK_GATE_ENABLED":            "风控闸总开关（env ALPHAFORGE_RISK_GATE_ENABLED 收口）",
	"ALPHAFORGE_RISK_GATE_KILLSWITCH_DEFAULT": "风控急停默认值（boot 读；env ALPHAFORGE_RISK_GATE_KILLSWITCH_DEFAULT 收口）",
	"ALPHAFORGE_RISK_GATE_AUTOTRADE_DEFAULT":  "自动交易默认授权（boot 读；env 收口）",
	"RATE_LIMIT_LOGIN_PER_MINUTE":             "登录限流每分钟次数（env 收口）",
	"AI_DAILY_MAX_SESSIONS":                   "AI 每用户每日会话上限（env AI_DAILY_MAX_SESSIONS 收口）",
	"AI_DAILY_MAX_TOKENS":                     "AI 每用户每日 token 上限（env 收口）",
	"APP_URL":                                 "邮件链接公开基址（env APP_URL 收口）",
	"WEBAUTHN_RP_ID":                          "WebAuthn Relying Party ID（env 收口）",
	"WEBAUTHN_RP_ORIGIN":                      "WebAuthn RP Origin（env 收口）",
	"TRONGRID_GRPC_ENDPOINT":                  "TronGrid gRPC 端点（env 收口）",
	"AI_DAILY_COST_LIMIT_USD":                 "平台 AI 日成本熔断阈值 USD（env 收口；运行期 agent_managed_settings 可热更）",
	"AI_MIN_BALANCE":                          "系统代付 AI 最低余额门槛 USD（env 收口；运行期热更层可覆写）",
}

// seedAndOverlayConfig seed-once + DB-wins overlay（语义见文件头）。
// 错误返回非 nil 表示配置不可用（fail-closed）——调用方（main）boot fatal。
func seedAndOverlayConfig(ctx context.Context, pool *pgxpool.Pool, secClient secrets.Client, cfg *config.Config) error {
	adminRepo := repository.NewAdminRepository(pool)

	// ── C 档 seed：env 非空 → system_config 仅缺键种入 ──
	for _, k := range config.SystemSettingSeedKeys() {
		v := envValue(k)
		if v == "" {
			continue
		}
		if err := adminRepo.InsertSystemConfigIfAbsent(ctx, k, v,
			config.SystemSettingKeyType(k), systemSettingDescriptions[k]); err != nil {
			return fmt.Errorf("seed system_config %s: %w", k, err)
		}
	}

	// ── D 档 seed：env 非空 → 加密入 platform_secrets 仅缺键种入（值永不日志） ──
	for _, k := range config.PlatformSecretSeedKeys() {
		v := envValue(k)
		if v == "" {
			continue
		}
		enc, err := secClient.Encrypt(ctx, secrets.PurposePlatformSecret, []byte(v))
		if err != nil {
			return fmt.Errorf("seal platform secret %s: %w", k, err)
		}
		if err := adminRepo.InsertPlatformSecretIfAbsent(ctx, k, enc); err != nil {
			return fmt.Errorf("seed platform_secrets %s: %w", k, err)
		}
	}

	// ── C 档 overlay：system_config 全表 → 白名单键 DB-wins 覆写 ──
	rows, err := adminRepo.LoadAllSystemConfig(ctx)
	if err != nil {
		return fmt.Errorf("load system_config: %w", err)
	}
	config.ApplySystemSettingsOverlay(cfg, filterSystemSettingRows(rows))

	// ── D 档 overlay：解密覆写（解密失败=错主钥/篡改 → 配置不可用） ──
	encSecrets, err := adminRepo.LoadPlatformSecrets(ctx)
	if err != nil {
		return fmt.Errorf("load platform_secrets: %w", err)
	}
	if len(encSecrets) > 0 {
		plain := make(map[string]string, len(encSecrets))
		for k, enc := range encSecrets {
			pt, err := secClient.Decrypt(ctx, secrets.PurposePlatformSecret, enc)
			if err != nil {
				return fmt.Errorf("decrypt platform secret %s (wrong ANT_MASTER_KEY or tampered row?): %w", k, err)
			}
			plain[k] = string(pt)
		}
		config.ApplyPlatformSecretsOverlay(cfg, plain)
	}
	return nil
}

// filterSystemSettingRows 只放行 C 档白名单键（system_config 另有 reconcile/sweep 等既有键，
// 不在本 overlay 管辖，混入会触发无谓告警）。
func filterSystemSettingRows(rows map[string]string) map[string]string {
	out := make(map[string]string, len(rows))
	for k, v := range rows {
		if config.IsSystemSettingKey(k) {
			out[k] = v
		}
	}
	return out
}

// envValue seed 读 env 的唯一入口（变量键名——门禁按文件白名单放行本文件）。
func envValue(key string) string {
	return os.Getenv(key)
}
