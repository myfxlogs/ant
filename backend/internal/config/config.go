package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/nats-io/nats.go"
)

// Config holds all application configuration.
//
// ENV-TO-PG-1 分档（ADR-0031 / docs/plan/2026-10-env-to-pg-consolidation.md）：
//   - A 档引导件（连库前必需：DB/NATS/Redis/PORT/目录/主钥/可观测）与 B 档构建期键：唯一来源 env。
//   - C 档业务旋钮 / D 档秘密件：Load() 仍按 env+default 供值（过渡兜底），boot 时 seed-once 入
//     PG（system_config / platform_secrets）后 DB-wins overlay 覆写（cmd/server bootstrap_config.go）
//     ——此后 PG 唯一真相源，env 改值失效。
//   - 禁区：DEPOSIT_XPUB(+FINGERPRINT) 维持 xpub_audit.go「DB 优先 env 回退」+指纹 env-only 锚。
type Config struct {
	// D6-A: Risk Gate
	RiskGateEnabled          bool
	RiskGateKillSwitch       bool
	RiskGateAutotradeEnabled bool

	// Database (PostgreSQL)
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string
	DBMaxConns int

	// NATS
	NATSURL string

	// Redis
	RedisHost     string
	RedisPort     string
	RedisPassword string

	// Secrets / crypto
	AntMasterKey     string
	AntKeyDir        string
	AntMasterKeyFile string

	// JWT
	JWTSecret string

	// HTTP server
	Port string

	// Data pipeline
	SpillDir    string
	GeoIPDBPath string

	// Rate limiting
	RateLimitLoginPerMinute int
	RateLimitEnabled        bool

	// Jurisdictional gate flags
	RequireKYC           bool
	RequireDisclaimer    bool
	RequireQuestionnaire bool

	// MTAPI
	MtapiToken   string // optional mtapi gateway token for account connection tests (D 档→platform_secrets)
	MtapiMT4Host string // mtapi gRPC gateway host for MT4 broker search (A 档引导件——服务发现留 env)
	MtapiMT5Host string // mtapi gRPC gateway host for MT5 broker search (A 档引导件)

	// AI gateway quotas (C 档→system_config；运行期另有 agent_managed_settings 热更层覆写)
	AIDailyMaxSessions  int
	AIDailyMaxTokens    int
	AIDailyCostLimitUSD string // decimal 字符串（历史语义接受 "45.5" 形态）
	AIMinBalance        string // decimal 字符串

	// SMTP (email notifications)
	SMTPHost     string
	SMTPPort     string
	SMTPUser     string
	SMTPPassword string
	SMTPFrom     string
	SMTPTo       string // comma-separated admin email addresses

	// AppURL is the public base URL for email links (e.g. https://alfq.org)
	AppURL string

	// RequireEmailVerification blocks login until email is verified (enable after SMTP is configured)
	RequireEmailVerification bool

	// Tron chain monitoring (USDT deposit)
	TrongridAPIKey       string
	TronscanAPIKey       string
	TronGridGRPCEndpoint string // e.g. grpc.trongrid.io:50051
	// ChainMonitorEnabled gates the TRON deposit monitor loop (paused while
	// the TRON track is descoped — avoids TronGrid rate-limit noise).
	ChainMonitorEnabled bool

	// HD wallet deposit (ADR-0026)
	DepositXpub            string
	DepositXpubFingerprint string

	// WebAuthn withdrawal authorization (ADR-0026 Phase E)
	WebAuthnRPID     string
	WebAuthnRPOrigin string

	// Cookie security: set Secure flag on refresh_token cookies.
	// Default true for production; set false only for local dev without TLS.
	CookieSecure bool
}

// Load reads all configuration from environment variables with defaults.
func Load() *Config {
	return &Config{
		DBHost:     getenv("DB_HOST", "postgres"),
		DBPort:     getenv("DB_PORT", "5432"),
		DBUser:     getenv("DB_USER", "ant"),
		DBPassword: getenv("DB_PASSWORD", "ant"),
		DBName:     getenv("DB_NAME", "ant"),
		DBSSLMode:  getenv("DB_SSLMODE", "disable"),
		// Default pgxpool MaxConns = max(4, NumCPU) is too small for the
		// push-first pipeline which holds several long-lived PG LISTEN/NOTIFY
		// connections plus one per active SSE stream. Those permanent listeners
		// exhaust the pool and every HTTP request blocks on Acquire. 25 leaves
		// ample headroom for listeners + concurrent request handling.
		DBMaxConns: getenvInt("DB_MAX_CONNS", 25),

		NATSURL: getenv("NATS_URL", nats.DefaultURL),

		RedisHost:     getenv("REDIS_HOST", "redis"),
		RedisPort:     getenv("REDIS_PORT", ""),
		RedisPassword: getenv("REDIS_PASSWORD", ""),

		AntMasterKey:     getenv("ANT_MASTER_KEY", ""),
		AntKeyDir:        getenv("ANT_KEY_DIR", ""),
		AntMasterKeyFile: getenv("ANT_MASTER_KEY_FILE", ""),
		JWTSecret:        getenv("JWT_SECRET", ""),

		Port:        getenv("PORT", "8080"),
		SpillDir:    getenv("SPILL_DIR", "/var/lib/ant/spill"),
		GeoIPDBPath: getenv("GEOIP_DB_PATH", "/var/lib/ant/geoip/GeoLite2-Country.mmdb"),

		RequireKYC:           getenvBool("REQUIRE_KYC", false),
		RequireDisclaimer:    getenvBool("REQUIRE_DISCLAIMER", false),
		RequireQuestionnaire: getenvBool("REQUIRE_QUESTIONNAIRE", false),

		MtapiToken: getenv("MTAPI_TOKEN", ""),

		MtapiMT4Host: getenv("MTAPI_MT4_HOST", ""),
		MtapiMT5Host: getenv("MTAPI_MT5_HOST", ""),

		AIDailyMaxSessions:  getenvInt("AI_DAILY_MAX_SESSIONS", 5),
		AIDailyMaxTokens:    getenvInt("AI_DAILY_MAX_TOKENS", 200_000),
		AIDailyCostLimitUSD: getenv("AI_DAILY_COST_LIMIT_USD", "50"),
		AIMinBalance:        getenv("AI_MIN_BALANCE", "1.0"),

		SMTPHost:     getenv("SMTP_HOST", ""),
		SMTPPort:     getenv("SMTP_PORT", "587"),
		SMTPUser:     getenv("SMTP_USER", ""),
		SMTPPassword: getenv("SMTP_PASSWORD", ""),
		SMTPFrom:     getenv("SMTP_FROM", "ant@localhost"),
		SMTPTo:       getenv("SMTP_TO", ""),

		AppURL: getenv("APP_URL", "https://alfq.org"),

		RequireEmailVerification: getenvBool("REQUIRE_EMAIL_VERIFICATION", false),

		TrongridAPIKey:       getenv("TRONGRID_API_KEY", ""),
		TronscanAPIKey:       getenv("TRONSCAN_API_KEY", ""),
		TronGridGRPCEndpoint: getenv("TRONGRID_GRPC_ENDPOINT", "grpc.trongrid.io:50051"),
		ChainMonitorEnabled:  getenvBool("CHAIN_MONITOR_ENABLED", true),

		DepositXpub:            getenv("DEPOSIT_XPUB", ""),
		DepositXpubFingerprint: getenv("DEPOSIT_XPUB_FINGERPRINT", ""),

		WebAuthnRPID:     getenv("WEBAUTHN_RP_ID", "alfq.org"),
		WebAuthnRPOrigin: getenv("WEBAUTHN_RP_ORIGIN", "https://alfq.org"),

		CookieSecure: getenvBool("COOKIE_SECURE", true),

		RiskGateEnabled:          getenvBool("ALPHAFORGE_RISK_GATE_ENABLED", true),
		RiskGateKillSwitch:       getenvBool("ALPHAFORGE_RISK_GATE_KILLSWITCH_DEFAULT", false),
		RiskGateAutotradeEnabled: getenvBool("ALPHAFORGE_RISK_GATE_AUTOTRADE_DEFAULT", true),

		RateLimitLoginPerMinute: getenvInt("RATE_LIMIT_LOGIN_PER_MINUTE", 10),
		RateLimitEnabled:        getenvBool("RATE_LIMIT_ENABLED", true),
	}
}

// Validate checks that required configuration fields are present.
func (c *Config) Validate() error {
	if c.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}
	return nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getenvBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v == "true" || v == "1" || v == "yes"
}

func getenvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
