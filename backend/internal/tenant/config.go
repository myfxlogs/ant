package tenant

// Config holds per-tenant configuration loaded from tenant_config PG table.
type Config struct {
	TenantID     string
	DisplayName  string
	MaxAccounts  int
	MaxStrategies int
	Tier         int // 0=free, 1=pro, 2=enterprise, 3=institutional

	// Rate limits (per second)
	SignalMaxPerSec  int64
	OrderMaxPerSec   int64
	CHWriteBytesPerSec int64
}

// DefaultConfig returns a default tenant configuration.
func DefaultConfig(tenantID string) Config {
	return Config{
		TenantID:           tenantID,
		DisplayName:        tenantID,
		MaxAccounts:        5,
		MaxStrategies:      10,
		Tier:               1,
		SignalMaxPerSec:    500,
		OrderMaxPerSec:     50,
		CHWriteBytesPerSec: 50_000_000,
	}
}

// IsValid checks required fields are populated.
func (c Config) IsValid() bool { return c.TenantID != "" }
