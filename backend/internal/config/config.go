package config

import (
	"fmt"
	"os"
	"regexp"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server     ServerConfig     `yaml:"server"`
	Database   DatabaseConfig   `yaml:"database"`
	Redis      RedisConfig      `yaml:"redis"`
	JWT        JWTConfig        `yaml:"jwt"`
	MT4        MT4Config        `yaml:"mt4"`
	MT5        MT5Config        `yaml:"mt5"`
	StrategyService StrategyServiceConfig `yaml:"strategy_service"`
	Log        LogConfig        `yaml:"log"`
	RateLimit  RateLimitConfig  `yaml:"rate_limit"`
	Business   BusinessConfig   `yaml:"business"`
	Monitoring MonitoringConfig `yaml:"monitoring"`
	Cache      CacheConfig      `yaml:"cache"`
	FMP        FMPConfig        `yaml:"fmp"`
}

type ServerConfig struct {
	HTTPPort        int           `yaml:"http_port"`
	GRPCPort        int           `yaml:"grpc_port"`
	Mode            string        `yaml:"mode"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
}

type DatabaseConfig struct {
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	Name            string        `yaml:"name"`
	User            string        `yaml:"user"`
	Password        string        `yaml:"password"`
	SSLMode         string        `yaml:"sslmode"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `yaml:"conn_max_idle_time"`
}

func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.Name, c.SSLMode)
}

type RedisConfig struct {
	Host               string        `yaml:"host"`
	Port               int           `yaml:"port"`
	Password           string        `yaml:"password"`
	DB                 int           `yaml:"db"`
	PoolSize           int           `yaml:"pool_size"`
	MinIdleConns       int           `yaml:"min_idle_conns"`
	MaxRetries         int           `yaml:"max_retries"`
	DialTimeout        time.Duration `yaml:"dial_timeout"`
	ReadTimeout        time.Duration `yaml:"read_timeout"`
	WriteTimeout       time.Duration `yaml:"write_timeout"`
	PoolTimeout        time.Duration `yaml:"pool_timeout"`
	IdleTimeout        time.Duration `yaml:"idle_timeout"`
	IdleCheckFrequency time.Duration `yaml:"idle_check_frequency"`
}

func (c *RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

type JWTConfig struct {
	Secret        string        `yaml:"secret"`
	AccessExpire  time.Duration `yaml:"access_expire"`
	RefreshExpire time.Duration `yaml:"refresh_expire"`
	Issuer        string        `yaml:"issuer"`
}

type MT4Config struct {
	GatewayHost   string        `yaml:"gateway_host"`
	GatewayPort   int           `yaml:"gateway_port"`
	Timeout       time.Duration `yaml:"timeout"`
	MaxRetries    int           `yaml:"max_retries"`
	RetryInterval time.Duration `yaml:"retry_interval"`
	UseTLS        bool          `yaml:"use_tls"`
}

func (c *MT4Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.GatewayHost, c.GatewayPort)
}

type MT5Config struct {
	GatewayHost   string        `yaml:"gateway_host"`
	GatewayPort   int           `yaml:"gateway_port"`
	Timeout       time.Duration `yaml:"timeout"`
	MaxRetries    int           `yaml:"max_retries"`
	RetryInterval time.Duration `yaml:"retry_interval"`
	UseTLS        bool          `yaml:"use_tls"`
}

func (c *MT5Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.GatewayHost, c.GatewayPort)
}

type LogConfig struct {
	Level          string `yaml:"level"`
	Format         string `yaml:"format"`
	Output         string `yaml:"output"`
	FileMaxSize    int    `yaml:"file_max_size"`
	FileMaxBackups int    `yaml:"file_max_backups"`
	FileMaxAge     int    `yaml:"file_max_age"`
	Compress       bool   `yaml:"compress"`
}

type RateLimitConfig struct {
	Enabled           bool    `yaml:"enabled"`
	RequestsPerSecond float64 `yaml:"requests_per_second"`
	Burst             int     `yaml:"burst"`
	ByIP              bool    `yaml:"by_ip"`
	ByUser            bool    `yaml:"by_user"`
}

type BusinessConfig struct {
	MaxAccountsPerUser     int     `yaml:"max_accounts_per_user"`
	MaxPositionsPerAccount int     `yaml:"max_positions_per_account"`
	DefaultLeverage        int     `yaml:"default_leverage"`
	MinLotSize             float64 `yaml:"min_lot_size"`
}

type MonitoringConfig struct {
	Enabled    bool          `yaml:"enabled"`
	MetricsURL string        `yaml:"metrics_url"`
	AlertsURL  string        `yaml:"alerts_url"`
	Interval   time.Duration `yaml:"interval"`
}

type CacheConfig struct {
	Enabled            bool          `yaml:"enabled"`
	DefaultTTL         time.Duration `yaml:"default_ttl"`
	CleanupInterval    time.Duration `yaml:"cleanup_interval"`
	MaxMemoryMB        int           `yaml:"max_memory_mb"`
	CompressionEnabled bool          `yaml:"compression_enabled"`
}

type StrategyServiceConfig struct {
	URL string `yaml:"url"`
}

type FMPConfig struct {
	APIKey  string `yaml:"api_key"`
	BaseURL string `yaml:"base_url"`
}

var globalConfig *Config

var envVarRegex = regexp.MustCompile(`\$\{([^}:]+)(?::([^}]*))?\}`)

func expandEnvWithDefault(s string) string {
	return envVarRegex.ReplaceAllStringFunc(s, func(match string) string {
		parts := envVarRegex.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}
		envVar := parts[1]
		defaultVal := ""
		if len(parts) > 2 {
			defaultVal = parts[2]
		}
		if val := os.Getenv(envVar); val != "" {
			return val
		}
		return defaultVal
	})
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	expanded := expandEnvWithDefault(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, err
	}

	globalConfig = &cfg
	return &cfg, nil
}

func Get() *Config {
	return globalConfig
}
