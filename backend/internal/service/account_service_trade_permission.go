package service

import (
	"context"
	"errors"
	"strconv"

	"github.com/google/uuid"

	"anttrader/internal/mt4client"
	"anttrader/internal/mt5client"
)

// ErrNoTradePermission 表示账户当前以投资者（只读）身份登录，不具备下单权限。
// 由 VerifyTradePermission / OrderSend 路径使用。
var ErrNoTradePermission = errors.New("account has no trade permission (investor mode)")

// TradePermissionResult 统一描述一次"交易权限探测"的结果。
//
//   - Verified = false 表示无法完成探测（连接失败、账户已禁用等），其余字段不应被信任；
//   - Verified = true  表示已经通过券商验证，此时 HasTradePermission / IsInvestor 才是权威的。
type TradePermissionResult struct {
	Verified           bool
	HasTradePermission bool
	IsInvestor         bool
	Message            string
}

// VerifyTradePermission 用当前数据库里存的密码做一次轻量 Connect + AccountSummary，
// 刷新 is_investor 字段并返回当前是否具备交易权限。
func (s *AccountService) VerifyTradePermission(ctx context.Context, userID, accountID uuid.UUID) (*TradePermissionResult, error) {
	account, err := s.GetAccount(ctx, userID, accountID)
	if err != nil {
		return nil, err
	}
	if account.IsDisabled {
		return &TradePermissionResult{Verified: false, Message: "account is disabled"}, nil
	}
	isInv, verified, msg := s.probeIsInvestor(ctx, account.MTType, account.Login, account.Password, account.BrokerHost)
	if verified {
		// 探测成功才落库，避免临时连通性问题把正确值误改成 false。
		_ = s.repo.UpdateIsInvestor(ctx, accountID, isInv)
	}
	return &TradePermissionResult{
		Verified:           verified,
		HasTradePermission: verified && !isInv,
		IsInvestor:         isInv,
		Message:            msg,
	}, nil
}

// UpdateTradingPassword 用新密码做一次 Connect + Summary：
//   - 连接失败 → 不改密码，返回 verified=false
//   - 连接成功 + 非投资者身份 → 覆盖密码并刷新 is_investor
//   - 连接成功 + 投资者身份  → 仍然覆盖密码（用户确实用的是这组凭证），同时 is_investor=true，
//     上层会告诉用户"这组密码是投资者只读密码，无法下单"。
func (s *AccountService) UpdateTradingPassword(ctx context.Context, userID, accountID uuid.UUID, newPassword string) (*TradePermissionResult, error) {
	if newPassword == "" {
		return nil, errors.New("new password cannot be empty")
	}
	account, err := s.GetAccount(ctx, userID, accountID)
	if err != nil {
		return nil, err
	}
	if account.IsDisabled {
		return &TradePermissionResult{Verified: false, Message: "account is disabled"}, nil
	}
	isInv, verified, msg := s.probeIsInvestor(ctx, account.MTType, account.Login, newPassword, account.BrokerHost)
	if !verified {
		// 新密码连不上，不改任何字段。
		return &TradePermissionResult{Verified: false, Message: msg}, nil
	}
	// 新密码能连上：覆盖密码 + 刷新 is_investor。
	if err := s.repo.UpdatePassword(ctx, accountID, newPassword); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateIsInvestor(ctx, accountID, isInv); err != nil {
		return nil, err
	}
	return &TradePermissionResult{
		Verified:           true,
		HasTradePermission: !isInv,
		IsInvestor:         isInv,
		Message:            msg,
	}, nil
}

// probeIsInvestor 用给定 login/password 做一次轻量 Connect + Summary，
// 完毕后立刻断开（不污染 ConnectionManager 的常驻连接）。
//
// 返回值：
//
//	isInvestor — 仅当 verified=true 时有意义；
//	verified   — 是否完成了 Connect + Summary；
//	message    — 给上层用户看的提示。
func (s *AccountService) probeIsInvestor(ctx context.Context, mtType, login, password, brokerHost string) (bool, bool, string) {
	host, port := s.parseHostPort(brokerHost)
	loginInt, err := strconv.ParseInt(login, 10, 64)
	if err != nil {
		return false, false, "invalid login id"
	}

	switch mtType {
	case "MT4":
		client := mt4client.NewMT4Client(s.mt4Config)
		conn, cerr := client.Connect(ctx, int32(loginInt), password, host, port)
		if cerr != nil {
			return false, false, "connect failed: " + cerr.Error()
		}
		defer client.Disconnect(ctx, conn.GetAccountID())
		if _, serr := conn.AccountSummary(ctx); serr != nil {
			return false, false, "account summary failed: " + serr.Error()
		}
		// MT4 协议不返回投资者标志；默认假定凭新密码登上即具备交易权限。
		return false, true, "ok"

	case "MT5":
		client := mt5client.NewMT5Client(s.mt5Config)
		conn, cerr := client.Connect(ctx, uint64(loginInt), password, host, port)
		if cerr != nil {
			return false, false, "connect failed: " + cerr.Error()
		}
		defer client.Disconnect(ctx, conn.GetAccountID())
		summary, serr := conn.AccountSummary(ctx)
		if serr != nil {
			return false, false, "account summary failed: " + serr.Error()
		}
		return summary.IsInvestor, true, "ok"

	default:
		return false, false, "unsupported mt_type: " + mtType
	}
}
