package config

import (
	"testing"
)

// appsettings_test.go —— ENV-TO-PG-1：C/D 档白名单 overlay 单测。
// 语义锚：DB-wins、坏 DB 值 warn+保留、无行键 env/default 兜底、非白名单行跳过。

func TestSystemSettingsBoolSetterMatrix(t *testing.T) {
	cases := []struct {
		v    string
		want bool
		ok   bool
	}{
		{"true", true, true}, {"1", true, true}, {"yes", true, true},
		{"false", false, true}, {"0", false, true}, {"no", false, true},
		{"TRUE", false, false}, {"banana", false, false}, {"", false, false},
	}
	for _, tc := range cases {
		cfg := &Config{CookieSecure: true}
		err := systemSettingKeys["COOKIE_SECURE"].set(cfg, tc.v)
		if tc.ok && err != nil {
			t.Fatalf("value %q: unexpected err %v", tc.v, err)
		}
		if !tc.ok && err == nil {
			t.Fatalf("value %q: expected error, got nil", tc.v)
		}
		if tc.ok && cfg.CookieSecure != tc.want {
			t.Fatalf("value %q: got %v want %v", tc.v, cfg.CookieSecure, tc.want)
		}
		if !tc.ok && cfg.CookieSecure != true {
			t.Fatalf("value %q: invalid value must not mutate field", tc.v)
		}
	}
}

func TestSystemSettingsIntSetter(t *testing.T) {
	cfg := &Config{RateLimitLoginPerMinute: 10}
	if err := systemSettingKeys["RATE_LIMIT_LOGIN_PER_MINUTE"].set(cfg, "25"); err != nil || cfg.RateLimitLoginPerMinute != 25 {
		t.Fatalf("valid int: got %d err %v", cfg.RateLimitLoginPerMinute, err)
	}
	if err := systemSettingKeys["RATE_LIMIT_LOGIN_PER_MINUTE"].set(cfg, "abc"); err == nil || cfg.RateLimitLoginPerMinute != 25 {
		t.Fatalf("bad int must error and keep field: %d err %v", cfg.RateLimitLoginPerMinute, err)
	}
}

func TestOverlayDBWinsOverEnv(t *testing.T) {
	t.Setenv("COOKIE_SECURE", "false")
	cfg := Load() // env→false
	ApplySystemSettingsOverlay(cfg, map[string]string{"COOKIE_SECURE": "true"})
	if !cfg.CookieSecure {
		t.Fatal("system_config row must win over env (DB-wins)")
	}
}

func TestOverlayNoRowKeepsEnv(t *testing.T) {
	t.Setenv("APP_URL", "https://example.org")
	cfg := Load()
	ApplySystemSettingsOverlay(cfg, map[string]string{}) // 无行
	if cfg.AppURL != "https://example.org" {
		t.Fatalf("no-row key must keep env value, got %q", cfg.AppURL)
	}
}

func TestOverlayInvalidDBValueKeepsEnvDefault(t *testing.T) {
	t.Setenv("CHAIN_MONITOR_ENABLED", "true")
	cfg := Load()
	ApplySystemSettingsOverlay(cfg, map[string]string{"CHAIN_MONITOR_ENABLED": "banana"})
	if !cfg.ChainMonitorEnabled {
		t.Fatal("invalid DB value must be ignored, env value kept")
	}
}

func TestOverlayForeignRowsSkipped(t *testing.T) {
	// system_config 既有非 C 档键（reconcile/sweep 族）不得触碰 cfg。
	cfg := Load()
	before := cfg.AppURL
	ApplySystemSettingsOverlay(cfg, map[string]string{
		"reconcile_shortage_threshold": "999", "sweep_threshold": "x", "deposit_xpub": "xpub",
	})
	if cfg.AppURL != before {
		t.Fatal("foreign rows must not mutate config")
	}
	if !IsSystemSettingKey("REQUIRE_KYC") || IsSystemSettingKey("reconcile_shortage_threshold") {
		t.Fatal("whitelist predicate broken")
	}
}

func TestPlatformSecretsOverlay(t *testing.T) {
	cfg := &Config{JWTSecret: "env-fallback", SMTPHost: "env-smtp"}
	ApplyPlatformSecretsOverlay(cfg, map[string]string{
		"JWT_SECRET":    "db-secret",
		"SMTP_HOST":     "db-smtp",
		"unknown_key_x": "zzz", // 非白名单行静默跳过
	})
	if cfg.JWTSecret != "db-secret" || cfg.SMTPHost != "db-smtp" {
		t.Fatal("secrets overlay must set whitelisted fields")
	}
}

func TestSeedKeysDeterministicAndTyped(t *testing.T) {
	ks := SystemSettingSeedKeys()
	if len(ks) == 0 {
		t.Fatal("seed keys empty")
	}
	for i := 1; i < len(ks); i++ {
		if ks[i-1] >= ks[i] {
			t.Fatalf("seed keys not sorted: %s >= %s", ks[i-1], ks[i])
		}
	}
	if got := SystemSettingKeyType("REQUIRE_KYC"); got != "bool" {
		t.Fatalf("REQUIRE_KYC type = %s, want bool", got)
	}
	if got := SystemSettingKeyType("AI_DAILY_MAX_TOKENS"); got != "number" {
		t.Fatalf("AI_DAILY_MAX_TOKENS type = %s, want number", got)
	}
	if got := SystemSettingKeyType("APP_URL"); got != "string" {
		t.Fatalf("APP_URL type = %s, want string", got)
	}
	// D 档 seed 键覆盖计划 D 清单核心键
	wantSecrets := []string{"JWT_SECRET", "SMTP_PASSWORD", "MTAPI_TOKEN", "TRONGRID_API_KEY", "TRONSCAN_API_KEY"}
	have := map[string]bool{}
	for _, k := range PlatformSecretSeedKeys() {
		have[k] = true
	}
	for _, k := range wantSecrets {
		if !have[k] {
			t.Fatalf("platform secret seed keys missing %s", k)
		}
	}
	if IsPlatformSecretKey("DEPOSIT_XPUB") || IsPlatformSecretKey("UMAMI_APP_SECRET") {
		t.Fatal("禁区键不得入 D 档白名单（xpub 既有语义/umami compose 期）")
	}
}
