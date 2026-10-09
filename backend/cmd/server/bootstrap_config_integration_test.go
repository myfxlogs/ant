//go:build integration

package main

// bootstrap_config_integration_test.go —— ENV-TO-PG-1：seed-once + DB-wins overlay
// 端到端（真实 PG，DSN 门控跳过）。封闭性：独立 schema 内 LIKE 复制 system_config /
// platform_secrets，search_path 指向之——零接触真实行。

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"alphaforge/internal/config"
	"alphaforge/internal/repository"
	"alphaforge/internal/secrets"
)

const bootstrapTestSchema = "bootstrap_it_schema"

func getBootstrapTestPool(t *testing.T) (*pgxpool.Pool, secrets.Client) {
	t.Helper()
	dsn := osGetenvDefault("TEST_PG_DSN", "postgres://ant:ant@localhost:5432/ant?sslmode=disable")
	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Skipf("skipping: bad dsn: %v", err)
	}
	poolCfg.ConnConfig.RuntimeParams["search_path"] = bootstrapTestSchema + ",public"
	pool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		t.Skipf("skipping integration test: pg connect: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("skipping integration test: pg ping: %v", err)
	}
	t.Cleanup(pool.Close)

	// 独立 schema：system_config LIKE 复制真实表；platform_secrets 若尚未随 migration 282
	// 部署（勿部署纪律——测试先行）则按 282 DDL 原样建。测试零接触 public 真实行。
	_, err = pool.Exec(ctx, `DROP SCHEMA IF EXISTS `+bootstrapTestSchema+` CASCADE`)
	if err != nil {
		t.Fatalf("drop schema: %v", err)
	}
	if _, err := pool.Exec(ctx, `CREATE SCHEMA `+bootstrapTestSchema); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	for _, ddl := range []string{
		`CREATE TABLE ` + bootstrapTestSchema + `.system_config (LIKE public.system_config INCLUDING ALL)`,
		`CREATE TABLE IF NOT EXISTS ` + bootstrapTestSchema + `.platform_secrets (
			key VARCHAR(100) PRIMARY KEY,
			value_enc BYTEA NOT NULL,
			note TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
	} {
		if _, err := pool.Exec(ctx, ddl); err != nil {
			t.Fatalf("bootstrap test table: %v (%v)", err, ddl)
		}
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DROP SCHEMA IF EXISTS `+bootstrapTestSchema+` CASCADE`)
	})

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		t.Fatalf("rand: %v", err)
	}
	client, err := secrets.New(base64.StdEncoding.EncodeToString(raw), 1)
	if err != nil {
		t.Fatalf("secrets client: %v", err)
	}
	return pool, client
}

func osGetenvDefault(key, def string) string {
	if v := testEnvValue(key); v != "" {
		return v
	}
	return def
}

// testEnvValue 测试内 env 读（os.Getenv 直呼会触发本包门禁白名单心智——收口一词）。
func testEnvValue(key string) string { return envValue(key) }

func systemConfigValue(t *testing.T, pool *pgxpool.Pool, key string) (string, bool) {
	t.Helper()
	var v string
	err := pool.QueryRow(context.Background(),
		`SELECT value FROM system_config WHERE key = $1`, key).Scan(&v)
	if err != nil {
		return "", false
	}
	return v, true
}

func platformSecretRow(t *testing.T, pool *pgxpool.Pool, key string) ([]byte, bool) {
	t.Helper()
	var enc []byte
	err := pool.QueryRow(context.Background(),
		`SELECT value_enc FROM platform_secrets WHERE key = $1`, key).Scan(&enc)
	if err != nil {
		return nil, false
	}
	return enc, true
}

// M1·seed-once：env 非空→种入；二次 boot 改 env 不得覆写（幂等）；overlay DB-wins。
func TestSeedOnceIdempotentAndOverlayDBWins(t *testing.T) {
	pool, sec := getBootstrapTestPool(t)
	ctx := context.Background()

	t.Setenv("CHAIN_MONITOR_ENABLED", "false")
	cfg := config.Load()
	if err := seedAndOverlayConfig(ctx, pool, sec, cfg); err != nil {
		t.Fatalf("first boot: %v", err)
	}
	if v, ok := systemConfigValue(t, pool, "CHAIN_MONITOR_ENABLED"); !ok || v != "false" {
		t.Fatalf("seed must insert env value, got row=%q present=%v", v, ok)
	}
	if cfg.ChainMonitorEnabled {
		t.Fatal("overlay must apply DB row (false) onto cfg")
	}

	// 二次 boot：env 改值 + 行已存在 → seed 不得覆写；overlay 仍 DB 值。
	t.Setenv("CHAIN_MONITOR_ENABLED", "true")
	cfg2 := config.Load()
	if err := seedAndOverlayConfig(ctx, pool, sec, cfg2); err != nil {
		t.Fatalf("second boot: %v", err)
	}
	if v, _ := systemConfigValue(t, pool, "CHAIN_MONITOR_ENABLED"); v != "false" {
		t.Fatalf("second-boot seed must not overwrite, row=%q", v)
	}
	if cfg2.ChainMonitorEnabled {
		t.Fatal("overlay must keep DB value false after second boot")
	}
}

// env 缺 → 不种入（DB 无行 → cfg 走 env/default）。
func TestSeedSkipsAbsentEnv(t *testing.T) {
	pool, sec := getBootstrapTestPool(t)
	ctx := context.Background()

	t.Setenv("REQUIRE_KYC", "") // 空=缺
	cfg := config.Load()
	if err := seedAndOverlayConfig(ctx, pool, sec, cfg); err != nil {
		t.Fatalf("boot: %v", err)
	}
	if _, ok := systemConfigValue(t, pool, "REQUIRE_KYC"); ok {
		t.Fatal("absent env must not seed a row")
	}
	if cfg.RequireKYC {
		t.Fatal("default REQUIRE_KYC must be false")
	}
}

// D 档：seed=密文落库（不含明文）→ overlay 解密回填 cfg → PG 唯一源（env 改值失效）。
func TestSecretSeedRoundtripAndDBWins(t *testing.T) {
	pool, sec := getBootstrapTestPool(t)
	ctx := context.Background()

	t.Setenv("MTAPI_TOKEN", "tok-secret-e2e-123")
	cfg := config.Load()
	if err := seedAndOverlayConfig(ctx, pool, sec, cfg); err != nil {
		t.Fatalf("first boot: %v", err)
	}
	enc, ok := platformSecretRow(t, pool, "MTAPI_TOKEN")
	if !ok {
		t.Fatal("secret row must be seeded")
	}
	if strings.Contains(string(enc), "tok-secret-e2e-123") {
		t.Fatal("ciphertext must not contain plaintext")
	}
	if cfg.MtapiToken != "tok-secret-e2e-123" {
		t.Fatal("overlay must decrypt row into cfg")
	}

	// 二次 boot：env 改值不生效（seed IF ABSENT + overlay DB-wins）。
	t.Setenv("MTAPI_TOKEN", "rotated-env-should-lose")
	cfg2 := config.Load()
	if err := seedAndOverlayConfig(ctx, pool, sec, cfg2); err != nil {
		t.Fatalf("second boot: %v", err)
	}
	if cfg2.MtapiToken != "tok-secret-e2e-123" {
		t.Fatalf("PG must be single source of truth, cfg=%q", cfg2.MtapiToken)
	}
}

// 错主钥/篡改 → 解密失败 → error（main 层 fatal，fail-closed）。
func TestWrongMasterKeyFailsClosed(t *testing.T) {
	pool, sec := getBootstrapTestPool(t)
	ctx := context.Background()

	t.Setenv("SMTP_PASSWORD", "pw-123")
	cfg := config.Load()
	if err := seedAndOverlayConfig(ctx, pool, sec, cfg); err != nil {
		t.Fatalf("seed with correct key: %v", err)
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		t.Fatalf("rand: %v", err)
	}
	wrong, err := secrets.New(base64.StdEncoding.EncodeToString(raw), 1)
	if err != nil {
		t.Fatalf("wrong client: %v", err)
	}
	cfg2 := config.Load()
	if err := seedAndOverlayConfig(ctx, pool, wrong, cfg2); err == nil {
		t.Fatal("wrong master key must fail closed (decrypt error)")
	}
}

// 仓储层直接锚：InsertPlatformSecretIfAbsent 幂等 + LoadPlatformSecrets 全表读。
func TestPlatformSecretRepositoryRoundtrip(t *testing.T) {
	pool, sec := getBootstrapTestPool(t)
	ctx := context.Background()
	repo := repository.NewAdminRepository(pool)

	enc, err := sec.Encrypt(ctx, secrets.PurposePlatformSecret, []byte("v1"))
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if err := repo.InsertPlatformSecretIfAbsent(ctx, "TRONGRID_API_KEY", enc); err != nil {
		t.Fatalf("insert: %v", err)
	}
	enc2, err := sec.Encrypt(ctx, secrets.PurposePlatformSecret, []byte("v2-different"))
	if err != nil {
		t.Fatalf("encrypt2: %v", err)
	}
	if err := repo.InsertPlatformSecretIfAbsent(ctx, "TRONGRID_API_KEY", enc2); err != nil {
		t.Fatalf("insert2: %v", err)
	}
	rows, err := repo.LoadPlatformSecrets(ctx)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	got, err := sec.Decrypt(ctx, secrets.PurposePlatformSecret, rows["TRONGRID_API_KEY"])
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if string(got) != "v1" {
		t.Fatalf("IF ABSENT violated: got %q want v1", string(got))
	}
}
