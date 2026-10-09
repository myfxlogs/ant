// appsettings.go —— ENV-TO-PG-1：C 档业务旋钮白名单（→ system_config，DB-wins boot overlay）。
// 语义：boot 时 seed-once（env 非空→表，仅缺键种入）后 PG 唯一真相源，env 改值失效（残值告警）；
// 表无行 → env/default 兜底（config.Load 照旧供值，合法过渡路径）；DB 坏值 warn+保留 env/default
// （不置零不致命）。无热更新：必须在组件构造前调用（main.go bootstrap 链），顺序后移静默失效。
// 旋钮清单 = 派工单 docs/plan/2026-10-env-to-pg-consolidation.md C 档 + 同性质补齐
// （RATE_LIMIT_*、AI_DAILY_MAX_*——原则「业务配置唯一真相=PG」，见 ADR-0031）。
package config

import (
	"log"
	"os"
	"sort"
	"strconv"
)

// settingMeta C 档旋钮元数据：typ=管理面展示型（bool|number|string），set=typed setter。
type settingMeta struct {
	typ string
	set func(*Config, string) error
}

// systemSettingKeys C 档白名单。bool 解析与 getenvBool 同族（true/1/yes），但 DB 侧
// 非法值显式报错（warn+保留）而非静默 false——库里出现笔误不能静默翻转旋钮。
var systemSettingKeys = map[string]settingMeta{
	"REQUIRE_KYC":                             boolv(func(c *Config) *bool { return &c.RequireKYC }),
	"REQUIRE_DISCLAIMER":                      boolv(func(c *Config) *bool { return &c.RequireDisclaimer }),
	"REQUIRE_QUESTIONNAIRE":                   boolv(func(c *Config) *bool { return &c.RequireQuestionnaire }),
	"REQUIRE_EMAIL_VERIFICATION":              boolv(func(c *Config) *bool { return &c.RequireEmailVerification }),
	"CHAIN_MONITOR_ENABLED":                   boolv(func(c *Config) *bool { return &c.ChainMonitorEnabled }),
	"COOKIE_SECURE":                           boolv(func(c *Config) *bool { return &c.CookieSecure }),
	"RATE_LIMIT_ENABLED":                      boolv(func(c *Config) *bool { return &c.RateLimitEnabled }),
	"ALPHAFORGE_RISK_GATE_ENABLED":            boolv(func(c *Config) *bool { return &c.RiskGateEnabled }),
	"ALPHAFORGE_RISK_GATE_KILLSWITCH_DEFAULT": boolv(func(c *Config) *bool { return &c.RiskGateKillSwitch }),
	"ALPHAFORGE_RISK_GATE_AUTOTRADE_DEFAULT":  boolv(func(c *Config) *bool { return &c.RiskGateAutotradeEnabled }),
	"RATE_LIMIT_LOGIN_PER_MINUTE":             intv(func(c *Config) *int { return &c.RateLimitLoginPerMinute }),
	"AI_DAILY_MAX_SESSIONS":                   intv(func(c *Config) *int { return &c.AIDailyMaxSessions }),
	"AI_DAILY_MAX_TOKENS":                     intv(func(c *Config) *int { return &c.AIDailyMaxTokens }),
	"APP_URL":                                 strv(func(c *Config) *string { return &c.AppURL }),
	"WEBAUTHN_RP_ID":                          strv(func(c *Config) *string { return &c.WebAuthnRPID }),
	"WEBAUTHN_RP_ORIGIN":                      strv(func(c *Config) *string { return &c.WebAuthnRPOrigin }),
	"TRONGRID_GRPC_ENDPOINT":                  strv(func(c *Config) *string { return &c.TronGridGRPCEndpoint }),
	"AI_DAILY_COST_LIMIT_USD":                 strv(func(c *Config) *string { return &c.AIDailyCostLimitUSD }),
	"AI_MIN_BALANCE":                          strv(func(c *Config) *string { return &c.AIMinBalance }),
}

func boolv(g func(*Config) *bool) settingMeta {
	return settingMeta{"bool", func(c *Config, v string) error {
		switch v {
		case "true", "1", "yes":
			*g(c) = true
		case "false", "0", "no":
			*g(c) = false
		default:
			return strconv.ErrSyntax
		}
		return nil
	}}
}

func intv(g func(*Config) *int) settingMeta {
	return settingMeta{"number", func(c *Config, v string) error {
		n, err := strconv.Atoi(v)
		if err != nil {
			return err
		}
		*g(c) = n
		return nil
	}}
}

func strv(g func(*Config) *string) settingMeta {
	return settingMeta{"string", func(c *Config, v string) error {
		*g(c) = v
		return nil
	}}
}

// ApplySystemSettingsOverlay system_config 行覆写 cfg（boot overlay：DB>env>default）。
// 白名单键命中 → typed set + env 残值告警；DB 无行 → env/default 静默；非白名单行不进 rows
// （调用方 LoadAllSystemConfig 已全量取回，此处逐键白名单判定）。键序字典序（日志确定性）。
func ApplySystemSettingsOverlay(cfg *Config, rows map[string]string) {
	keys := make([]string, 0, len(systemSettingKeys))
	for k := range systemSettingKeys {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	seen := map[string]bool{}
	for _, k := range keys {
		v, ok := rows[k]
		if !ok {
			continue
		}
		seen[k] = true
		if err := systemSettingKeys[k].set(cfg, v); err != nil {
			log.Printf("[config] %s: invalid system_config value %q ignored (env/default kept)", k, v)
			continue
		}
		if envV := os.Getenv(k); envV != "" && envV != v {
			log.Printf("[config] %s: env value ignored, system_config row wins", k)
		}
		log.Printf("[config] %s (source=system_config)", k)
	}
	// 无行键打 source 行：运维一眼看清每个旋钮谁在供值（env/default）。
	for _, k := range keys {
		if seen[k] {
			continue
		}
		source := "default"
		if os.Getenv(k) != "" {
			source = "env"
		}
		log.Printf("[config] %s (source=%s)", k, source)
	}
}

// SystemSettingSeedKeys seed 迭代键表（字典序确定）。
func SystemSettingSeedKeys() []string {
	keys := make([]string, 0, len(systemSettingKeys))
	for k := range systemSettingKeys {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// SystemSettingKeyType 键的 value_type 标注（seed 落 system_config.value_type 列）。
func SystemSettingKeyType(key string) string {
	if m, ok := systemSettingKeys[key]; ok {
		return m.typ
	}
	return "text"
}

// IsSystemSettingKey 白名单判定（调用方对 system_config 既有非 C 档键静默跳过）。
func IsSystemSettingKey(key string) bool {
	_, ok := systemSettingKeys[key]
	return ok
}
