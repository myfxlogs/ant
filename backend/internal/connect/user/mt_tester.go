// Package user provides MT connection validation for account binding.
package user

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"alphaforge/internal/mdgateway"
	"alphaforge/internal/mdgateway/adapter/mdtick"
	"alphaforge/internal/mdgateway/adapter/mt4"
	"alphaforge/internal/mdgateway/adapter/mt5"
)

// mtConnectionTester implements MTConnectionTester using mt4/mt5 gateway adapters.
type mtConnectionTester struct {
	token        string
	rediscoverer *mdgateway.HostRediscoverer
	log          *zap.Logger
}

// NewMTConnectionTester creates a connection tester with an optional mtapi token.
// rediscoverer is optional (nil → no broker host rediscovery on connection failure).
func NewMTConnectionTester(token string, rediscoverer *mdgateway.HostRediscoverer, log *zap.Logger) MTConnectionTester {
	return &mtConnectionTester{token: token, rediscoverer: rediscoverer, log: log}
}

func (t *mtConnectionTester) Test(ctx context.Context, platform, brokerHost, login, password string) (*mdtick.MTAccountInfo, error) {
	cfg := mdtick.AccountConfig{
		Platform:   platform,
		Login:      login,
		Password:   password,
		BrokerHost: brokerHost,
		MtapiToken: t.token,
	}

	switch strings.ToLower(platform) {
	case "mt4":
		return t.testMT4(ctx, cfg)
	case "mt5":
		return t.testMT5(ctx, cfg)
	default:
		return nil, fmt.Errorf("unsupported platform: %s", platform)
	}
}

// VerifyPassword connects to the broker to verify credentials.
// It does not call AccountSummary, so it works for investor/read-only accounts too.
// Handles comma-separated broker_host by trying each one. On host errors
// (connection refused / DNS failure) on the last host, delegates to HostRediscoverer
// which searches for the broker company and tries each host. On success, updates mt_accounts.broker_host.
func (t *mtConnectionTester) VerifyPassword(ctx context.Context, platform, brokerHost, login, password, brokerCompany, accountID string) error {
	cfg := mdtick.AccountConfig{
		Platform:   platform,
		Login:      login,
		Password:   password,
		BrokerHost: brokerHost,
		MtapiToken: t.token,
	}

	var newGW func(mdtick.AccountConfig, *zap.Logger) gatewayConn
	switch strings.ToLower(platform) {
	case "mt4":
		newGW = func(c mdtick.AccountConfig, l *zap.Logger) gatewayConn { return mt4.New(c, l) }
	case "mt5":
		newGW = func(c mdtick.AccountConfig, l *zap.Logger) gatewayConn { return mt5.New(c, l) }
	default:
		return fmt.Errorf("unsupported platform: %s", platform)
	}

	hosts := strings.Split(brokerHost, ",")
	var lastErr error
	for i, host := range hosts {
		host = strings.TrimSpace(host)
		if host == "" {
			continue
		}
		testCfg := cfg
		testCfg.BrokerHost = host
		gw := newGW(testCfg, t.log)
		if err := gw.Connect(ctx); err != nil {
			lastErr = err
			// Only attempt rediscovery on the last host — if there are more hosts to try,
			// the next iteration may succeed without rediscovery.
			if i == len(hosts)-1 && t.rediscoverer != nil {
				_, rerr := t.rediscoverer.MaybeRediscover(ctx, err, brokerCompany, platform, accountID,
					func(newHost string) error {
						rc := cfg
						rc.BrokerHost = newHost
						rgw := newGW(rc, t.log)
						if cerr := rgw.Connect(ctx); cerr != nil {
							return cerr
						}
						rgw.Disconnect(ctx)
						return nil
					})
				return rerr
			}
			continue
		}
		gw.Disconnect(ctx)
		return nil
	}
	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("no valid hosts to try")
}

// gatewayConn is the subset of MT4/MT5 gateway methods needed for connection testing.
type gatewayConn interface {
	Connect(ctx context.Context) error
	FetchAccountInfo(ctx context.Context) (*mdtick.MTAccountInfo, error)
	Disconnect(ctx context.Context) error
}

// testMT iterates through comma-separated broker hosts, trying each one.
// Returns account info from the first host that connects successfully.
func (t *mtConnectionTester) testMT(ctx context.Context, cfg mdtick.AccountConfig, newGW func(mdtick.AccountConfig, *zap.Logger) gatewayConn) (*mdtick.MTAccountInfo, error) {
	hosts := strings.Split(cfg.BrokerHost, ",")
	var lastErr error
	for _, host := range hosts {
		host = strings.TrimSpace(host)
		if host == "" {
			continue
		}
		cfg.BrokerHost = host
		gw := newGW(cfg, t.log)
		if err := gw.Connect(ctx); err != nil {
			lastErr = err
			continue
		}
		info, err := gw.FetchAccountInfo(ctx)
		gw.Disconnect(ctx)
		if err != nil {
			lastErr = err
			continue
		}
		info.BrokerHost = host
		return info, nil
	}
	if lastErr != nil {
		return nil, fmt.Errorf("all %d hosts failed, last error: %w", len(hosts), lastErr)
	}
	return nil, fmt.Errorf("no valid hosts to try")
}

func (t *mtConnectionTester) testMT4(ctx context.Context, cfg mdtick.AccountConfig) (*mdtick.MTAccountInfo, error) {
	return t.testMT(ctx, cfg, func(c mdtick.AccountConfig, l *zap.Logger) gatewayConn { return mt4.New(c, l) })
}

func (t *mtConnectionTester) testMT5(ctx context.Context, cfg mdtick.AccountConfig) (*mdtick.MTAccountInfo, error) {
	return t.testMT(ctx, cfg, func(c mdtick.AccountConfig, l *zap.Logger) gatewayConn { return mt5.New(c, l) })
}
