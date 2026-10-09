// appsecrets.go —— ENV-TO-PG-1：D 档秘密件白名单（→ platform_secrets 密文轨，ANT_MASTER_KEY
// AES-256-GCM 封装，见 ADR-0031）。明文仅存在于 boot 内存与密文解包瞬间：禁日志、禁错误串、
// 禁管理面回显。语义同 C 档：seed-once（env 非空→密文行，仅缺键种入）后 PG 唯一真相源；
// 表无行 → env 兜底（config.Load 照旧）。解密失败=配置不可用 → 调用方 boot fatal（fail-closed）。
// 排除键（禁区保留）：DEPOSIT_XPUB(+FINGERPRINT)（xpub_audit.go DB 优先 env 回退语义 +
// 指纹 env-only 防替换锚，ADR-0026 R5）；UMAMI_APP_SECRET（compose 期喂 umami 侧容器，
// 后端零读取——搬 PG 不生效）。
package config

import (
	"log"
	"sort"
)

// platformSecretKeys D 档白名单：env 键 → Config 字段。SMTP_PORT/HOST 虽非高敏，随 SMTP
// 凭据族整组入密文轨（避免管理面明文露出邮件拓扑）。
var platformSecretKeys = map[string]func(*Config, string){
	"JWT_SECRET":       func(c *Config, v string) { c.JWTSecret = v },
	"SMTP_HOST":        func(c *Config, v string) { c.SMTPHost = v },
	"SMTP_PORT":        func(c *Config, v string) { c.SMTPPort = v },
	"SMTP_USER":        func(c *Config, v string) { c.SMTPUser = v },
	"SMTP_PASSWORD":    func(c *Config, v string) { c.SMTPPassword = v },
	"SMTP_FROM":        func(c *Config, v string) { c.SMTPFrom = v },
	"SMTP_TO":          func(c *Config, v string) { c.SMTPTo = v },
	"MTAPI_TOKEN":      func(c *Config, v string) { c.MtapiToken = v },
	"TRONGRID_API_KEY": func(c *Config, v string) { c.TrongridAPIKey = v },
	"TRONSCAN_API_KEY": func(c *Config, v string) { c.TronscanAPIKey = v },
}

// ApplyPlatformSecretsOverlay 解密后的密钥集覆写 cfg（DB-wins）。值永不进日志。
func ApplyPlatformSecretsOverlay(cfg *Config, secrets map[string]string) {
	keys := make([]string, 0, len(secrets))
	for k := range secrets {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		set, ok := platformSecretKeys[k]
		if !ok {
			continue // 非白名单行静默跳过（值不可日志）
		}
		set(cfg, secrets[k])
		// 只打键名与来源，不打值。
		log.Printf("[config] %s (source=platform_secrets)", k)
	}
}

// PlatformSecretSeedKeys seed 迭代键表（字典序确定）。
func PlatformSecretSeedKeys() []string {
	keys := make([]string, 0, len(platformSecretKeys))
	for k := range platformSecretKeys {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// IsPlatformSecretKey D 档白名单判定（门禁/调用方用）。
func IsPlatformSecretKey(key string) bool {
	_, ok := platformSecretKeys[key]
	return ok
}
