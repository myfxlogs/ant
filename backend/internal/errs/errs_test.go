package errs

import (
	"errors"
	"testing"
)

func TestNew(t *testing.T) {
	e := New(CodeInvalidParam)
	if e.Code != CodeInvalidParam {
		t.Errorf("expected code %d, got %d", CodeInvalidParam, e.Code)
	}
	if e.Message != "请求参数无效" {
		t.Errorf("expected Chinese message, got %q", e.Message)
	}
	if e.Severity != "warn" {
		t.Errorf("expected severity warn, got %q", e.Severity)
	}
}

func TestNewf(t *testing.T) {
	e := Newf(CodeOrderNotFound, "order_id=%s", "abc123")
	if e.Detail != "order_id=abc123" {
		t.Errorf("expected detail 'order_id=abc123', got %q", e.Detail)
	}
}

func TestWrap(t *testing.T) {
	inner := errors.New("connection refused")
	e := Wrap(CodeDBConnectionFailed, inner)
	if e.Code != CodeDBConnectionFailed {
		t.Errorf("expected code %d, got %d", CodeDBConnectionFailed, e.Code)
	}
	if !errors.Is(e, inner) {
		t.Error("expected Unwrap to return inner error")
	}
	if e.Severity != "fatal" {
		t.Errorf("expected severity fatal for 9xxx code, got %q", e.Severity)
	}
}

func TestWrapf(t *testing.T) {
	inner := errors.New("timeout")
	e := Wrapf(CodeAccountConnFailed, inner, "mt5.gateway:443")
	if e.Detail != "mt5.gateway:443" {
		t.Errorf("expected detail 'mt5.gateway:443', got %q", e.Detail)
	}
}

func TestMessageCN(t *testing.T) {
	tests := []struct {
		code int
		want string
	}{
		{CodeOK, "操作成功"},
		{CodeUserNotFound, "用户不存在"},
		{CodeAccountNotFound, "交易账户不存在"},
		{CodeOrderRejected, "订单被拒绝"},
		{CodeSymbolNotFound, "交易品种不存在"},
		{CodeStrategyNotFound, "策略不存在"},
		{CodeDBConnectionFailed, "数据库连接失败"},
		{CodeUnknown, "未知错误"},
		{99999, "未知错误"}, // unknown code
	}
	for _, tt := range tests {
		got := MessageCN(tt.code)
		if got != tt.want {
			t.Errorf("MessageCN(%d) = %q, want %q", tt.code, got, tt.want)
		}
	}
}

func TestMessageEN(t *testing.T) {
	tests := []struct {
		code int
		want string
	}{
		{CodeOK, "Success"},
		{CodeUnauthorized, "Unauthorized"},
		{CodeOrderNotFound, "Order not found"},
		{CodeAIGenerationFailed, "AI generation failed"},
		{CodeDBConnectionFailed, "Database connection failed"},
	}
	for _, tt := range tests {
		got := MessageEN(tt.code)
		if got != tt.want {
			t.Errorf("MessageEN(%d) = %q, want %q", tt.code, got, tt.want)
		}
	}
}

func TestError_Error(t *testing.T) {
	e := New(CodeInvalidParam)
	errStr := e.Error()
	if errStr != "[2] 请求参数无效" {
		t.Errorf("Error() = %q, want '[2] 请求参数无效'", errStr)
	}

	e2 := Wrap(CodeInternal, errors.New("boom"))
	errStr2 := e2.Error()
	if errStr2 != "[6] 服务器内部错误，请稍后重试: boom" {
		t.Errorf("Error() = %q", errStr2)
	}
}

func TestSeverity(t *testing.T) {
	tests := []struct {
		code int
		want string
	}{
		{CodeOK, "warn"},
		{CodeUnauthorized, "error"},
		{CodeForbidden, "error"},
		{CodeOrderRejected, "error"}, // 3xxx = error
		{CodeAccountNotFound, "warn"},
		{CodeDBConnectionFailed, "fatal"}, // 9xxx = fatal
	}
	for _, tt := range tests {
		e := New(tt.code)
		if e.Severity != tt.want {
			t.Errorf("code %d: severity = %q, want %q", tt.code, e.Severity, tt.want)
		}
	}
}

func TestTop50Coverage(t *testing.T) {
	// Verify at least 50 error codes have Chinese messages
	codes := []int{
		CodeOK, CodeUnknown, CodeInvalidParam, CodeUnauthorized, CodeForbidden,
		CodeNotFound, CodeInternal, CodeRateLimited, CodeServiceUnavail, CodeTimeout,
		CodeUserNotFound, CodeUserAlreadyExists, CodeInvalidPassword, CodeTokenExpired,
		CodeTokenInvalid, CodeTokenMissing, CodeUserDisabled, CodeEmailNotVerified,
		CodePasswordTooWeak, CodeOldPasswordIncorrect,
		CodeAccountNotFound, CodeAccountAlreadyBound, CodeAccountConnFailed,
		CodeAccountDisconnected, CodeAccountAuthFailed, CodeAccountTimeout,
		CodeAccountLimitExceeded, CodeInvalidAccountType, CodeAccountNotConnected,
		CodePlatformNotSupported,
		CodeOrderNotFound, CodeOrderRejected, CodeInsufficientMargin, CodeMarketClosed,
		CodeInvalidOrderType, CodeInvalidVolume, CodeInvalidPrice, CodeOrderTimeout,
		CodePositionNotFound,
		CodeSymbolNotFound, CodeNoMarketData, CodeSubFailed, CodeUnsubFailed,
		CodeQuoteNotAvailable,
		CodeAnalyticsNotAvail, CodeReportGenFailed, CodeInsufficientData,
		CodeAdminAccessDenied, CodeOperationForbidden,
		CodeBrokerSearchFailed, CodeBrokerNotFound, CodeBrokerServerUnavailable,
		CodeStrategyNotFound, CodeStrategyDisabled, CodeSandboxTimeout,
		CodeAIModelUnavailable, CodeAIGenerationFailed,
		CodeDBConnectionFailed, CodeRedisUnavailable, CodeMigrationFailed,
		CodeConfigInvalid,
	}
	count := 0
	for _, code := range codes {
		msg := MessageCN(code)
		if msg != "" && msg != "未知错误" {
			count++
		}
	}
	if count < 50 {
		t.Errorf("only %d of %d codes have Chinese messages, want ≥50", count, len(codes))
	}
	t.Logf("Chinese messages: %d/%d codes covered", count, len(codes))
}
