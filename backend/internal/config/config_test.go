package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadDefaults(t *testing.T) {
	cfg := Load()

	assert.Equal(t, "postgres", cfg.DBHost)
	assert.Equal(t, "5432", cfg.DBPort)
	assert.Equal(t, "ant", cfg.DBUser)
	assert.Equal(t, "ant", cfg.DBPassword)
	assert.Equal(t, "ant", cfg.DBName)
	assert.Equal(t, "disable", cfg.DBSSLMode)

	assert.Equal(t, "clickhouse", cfg.CHHost)
	assert.Equal(t, "9000", cfg.CHPort)
	assert.Equal(t, "default", cfg.CHUser)
	assert.Equal(t, "", cfg.CHPassword)
	assert.Equal(t, "ant", cfg.CHDatabase)

	assert.NotEmpty(t, cfg.NATSURL)

	assert.Equal(t, "redis", cfg.RedisHost)
	assert.Equal(t, "", cfg.RedisPort)
	assert.Equal(t, "", cfg.RedisPassword)

	assert.Equal(t, "", cfg.AntMasterKey)
	assert.Equal(t, "", cfg.JWTSecret)

	assert.Equal(t, "8080", cfg.Port)
	assert.Equal(t, "/var/lib/ant/spill", cfg.SpillDir)
	assert.Contains(t, cfg.GeoIPDBPath, "GeoLite2-Country.mmdb")

	assert.False(t, cfg.RequireKYC)
	assert.False(t, cfg.RequireDisclaimer)
	assert.False(t, cfg.RequireQuestionnaire)
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("DB_HOST", "custom-db")
	t.Setenv("DB_PORT", "9999")
	t.Setenv("DB_USER", "custom-user")
	t.Setenv("DB_PASSWORD", "secret")
	t.Setenv("DB_NAME", "customdb")
	t.Setenv("DB_SSLMODE", "require")

	t.Setenv("CH_HOST", "ch-custom")
	t.Setenv("CH_PORT", "8123")
	t.Setenv("CH_USER", "chuser")
	t.Setenv("CH_PASSWORD", "chpass")
	t.Setenv("CH_DATABASE", "analytics")

	t.Setenv("NATS_URL", "nats://nats:4222")
	t.Setenv("REDIS_HOST", "redis-custom")
	t.Setenv("REDIS_PORT", "6380")
	t.Setenv("REDIS_PASSWORD", "redispass")

	t.Setenv("ANT_MASTER_KEY", "master-key-123")
	t.Setenv("JWT_SECRET", "jwt-secret-456")

	t.Setenv("PORT", "9090")
	t.Setenv("SPILL_DIR", "/tmp/spill")
	t.Setenv("GEOIP_DB_PATH", "/opt/geoip/custom.mmdb")

	t.Setenv("REQUIRE_KYC", "true")
	t.Setenv("REQUIRE_DISCLAIMER", "1")
	t.Setenv("REQUIRE_QUESTIONNAIRE", "yes")

	cfg := Load()

	assert.Equal(t, "custom-db", cfg.DBHost)
	assert.Equal(t, "9999", cfg.DBPort)
	assert.Equal(t, "custom-user", cfg.DBUser)
	assert.Equal(t, "secret", cfg.DBPassword)
	assert.Equal(t, "customdb", cfg.DBName)
	assert.Equal(t, "require", cfg.DBSSLMode)

	assert.Equal(t, "ch-custom", cfg.CHHost)
	assert.Equal(t, "8123", cfg.CHPort)
	assert.Equal(t, "chuser", cfg.CHUser)
	assert.Equal(t, "chpass", cfg.CHPassword)
	assert.Equal(t, "analytics", cfg.CHDatabase)

	assert.Equal(t, "nats://nats:4222", cfg.NATSURL)
	assert.Equal(t, "redis-custom", cfg.RedisHost)
	assert.Equal(t, "6380", cfg.RedisPort)
	assert.Equal(t, "redispass", cfg.RedisPassword)

	assert.Equal(t, "master-key-123", cfg.AntMasterKey)
	assert.Equal(t, "jwt-secret-456", cfg.JWTSecret)

	assert.Equal(t, "9090", cfg.Port)
	assert.Equal(t, "/tmp/spill", cfg.SpillDir)
	assert.Equal(t, "/opt/geoip/custom.mmdb", cfg.GeoIPDBPath)

	assert.True(t, cfg.RequireKYC)
	assert.True(t, cfg.RequireDisclaimer)
	assert.True(t, cfg.RequireQuestionnaire)
}

func TestValidateMissingJWT(t *testing.T) {
	cfg := Load()
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_SECRET")
}

func TestValidateOK(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	cfg := Load()
	assert.NoError(t, cfg.Validate())
}

func TestGetenvBool(t *testing.T) {
	tests := []struct {
		val  string
		want bool
	}{
		{"true", true},
		{"1", true},
		{"yes", true},
		{"", false}, // empty returns fallback
		{"false", false},
		{"0", false},
		{"no", false},
		{"True", false}, // case-sensitive
		{"YES", false},
	}
	for _, tc := range tests {
		t.Setenv("TEST_BOOL", tc.val)
		assert.Equal(t, tc.want, getenvBool("TEST_BOOL", false), "val=%q", tc.val)
	}
}
