package redis

import (
	"testing"
)

func TestConfig_Defaults(t *testing.T) {
	t.Parallel()
	cfg := Config{
		Host: "localhost",
		Port: 6379,
		DB:   0,
	}
	if cfg.Host == "" {
		t.Fatal("expected non-empty host")
	}
	if cfg.Port == 0 {
		t.Fatal("expected non-zero port")
	}
}
