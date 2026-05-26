package config

import (
	"fmt"
	"os"

	"github.com/nats-io/nats.go"
)

// Config holds all application configuration sourced from environment variables.
type Config struct {
	// Database (PostgreSQL)
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	// ClickHouse
	CHHost     string
	CHPort     string
	CHUser     string
	CHPassword string
	CHDatabase string

	// NATS
	NATSURL string

	// Redis
	RedisHost     string
	RedisPort     string
	RedisPassword string

	// Secrets / crypto
	AntMasterKey string

	// JWT
	JWTSecret string

	// HTTP server
	Port string

	// Data pipeline
	SpillDir    string
	GeoIPDBPath string

	// Jurisdictional gate flags
	RequireKYC           bool
	RequireDisclaimer    bool
	RequireQuestionnaire bool
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

		CHHost:     getenv("CH_HOST", "clickhouse"),
		CHPort:     getenv("CH_PORT", "9000"),
		CHUser:     getenv("CH_USER", "default"),
		CHPassword: getenv("CH_PASSWORD", ""),
		CHDatabase: getenv("CH_DATABASE", "ant"),

		NATSURL: getenv("NATS_URL", nats.DefaultURL),

		RedisHost:     getenv("REDIS_HOST", "redis"),
		RedisPort:     getenv("REDIS_PORT", ""),
		RedisPassword: getenv("REDIS_PASSWORD", ""),

		AntMasterKey: os.Getenv("ANT_MASTER_KEY"),
		JWTSecret:    os.Getenv("JWT_SECRET"),

		Port:        getenv("PORT", "8080"),
		SpillDir:    getenv("SPILL_DIR", "/var/lib/ant/spill"),
		GeoIPDBPath: getenv("GEOIP_DB_PATH", "/var/lib/ant/geoip/GeoLite2-Country.mmdb"),

		RequireKYC:           getenvBool("REQUIRE_KYC", false),
		RequireDisclaimer:    getenvBool("REQUIRE_DISCLAIMER", false),
		RequireQuestionnaire: getenvBool("REQUIRE_QUESTIONNAIRE", false),
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
