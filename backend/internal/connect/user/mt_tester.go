// Package user provides MT connection validation for account binding.
package user

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"anttrader/internal/mdgateway/adapter/mdtick"
	"anttrader/internal/mdgateway/adapter/mt4"
	"anttrader/internal/mdgateway/adapter/mt5"
)

// mtConnectionTester implements MTConnectionTester using mt4/mt5 gateway adapters.
type mtConnectionTester struct {
	token  string
	log    *zap.Logger
}

// NewMTConnectionTester creates a connection tester with an optional mtapi token.
func NewMTConnectionTester(token string, log *zap.Logger) MTConnectionTester {
	return &mtConnectionTester{token: token, log: log}
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

// VerifyPassword only connects to the broker to verify credentials.
// It does not call AccountSummary, so it works for investor/read-only accounts too.
func (t *mtConnectionTester) VerifyPassword(ctx context.Context, platform, brokerHost, login, password string) error {
	cfg := mdtick.AccountConfig{
		Platform:   platform,
		Login:      login,
		Password:   password,
		BrokerHost: brokerHost,
		MtapiToken: t.token,
	}

	switch strings.ToLower(platform) {
	case "mt4":
		gw := mt4.New(cfg, t.log)
		if err := gw.Connect(ctx); err != nil {
			return fmt.Errorf("connection failed: %w", err)
		}
		gw.Disconnect(ctx)
		return nil
	case "mt5":
		gw := mt5.New(cfg, t.log)
		if err := gw.Connect(ctx); err != nil {
			return fmt.Errorf("connection failed: %w", err)
		}
		gw.Disconnect(ctx)
		return nil
	default:
		return fmt.Errorf("unsupported platform: %s", platform)
	}
}

func (t *mtConnectionTester) testMT4(ctx context.Context, cfg mdtick.AccountConfig) (*mdtick.MTAccountInfo, error) {
	gw := mt4.New(cfg, t.log)
	if err := gw.Connect(ctx); err != nil {
		return nil, fmt.Errorf("connection failed: %w", err)
	}
	defer gw.Disconnect(ctx)

	info, err := gw.FetchAccountInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("account info: %w", err)
	}
	return info, nil
}

func (t *mtConnectionTester) testMT5(ctx context.Context, cfg mdtick.AccountConfig) (*mdtick.MTAccountInfo, error) {
	gw := mt5.New(cfg, t.log)
	if err := gw.Connect(ctx); err != nil {
		return nil, fmt.Errorf("connection failed: %w", err)
	}
	defer gw.Disconnect(ctx)

	info, err := gw.FetchAccountInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("account info: %w", err)
	}
	return info, nil
}
