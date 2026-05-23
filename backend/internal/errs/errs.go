// Package errs provides the centralized error-code system for ant.
// All business errors MUST use this package — no bare strings.
//
// Error code ranges (ADR-0010):
//
//	1000–1999  auth / user
//	2000–2999  account / broker connection
//	3000–3999  order / position / trading
//	4000–4999  market / symbol / quote
//	5000–5999  analytics / report
//	6000–6999  admin / audit
//	7000–7999  broker gateway
//	8000–8999  strategy / AI / sandbox
//	9000–9999  system / generic
//
// Message convention: Chinese message required; English optional (falls back to Chinese).
package errs

import "fmt"

// ── Error codes ──────────────────────────────────────────────────────

const (
	// 0–99: generic
	CodeOK               = 0
	CodeUnknown          = 1
	CodeInvalidParam     = 2
	CodeUnauthorized     = 3
	CodeForbidden        = 4
	CodeNotFound         = 5
	CodeInternal         = 6
	CodeRateLimited      = 7
	CodeServiceUnavail   = 8
	CodeTimeout          = 9

	// 1000–1999: auth / user
	CodeUserNotFound         = 1001
	CodeUserAlreadyExists    = 1002
	CodeInvalidPassword      = 1003
	CodeTokenExpired         = 1004
	CodeTokenInvalid         = 1005
	CodeTokenMissing         = 1006
	CodeUserDisabled         = 1007
	CodeEmailNotVerified     = 1008
	CodePasswordTooWeak      = 1009
	CodeOldPasswordIncorrect = 1010

	// 2000–2999: account
	CodeAccountNotFound         = 2001
	CodeAccountAlreadyBound     = 2002
	CodeAccountConnFailed       = 2003
	CodeAccountDisconnected     = 2004
	CodeAccountAuthFailed       = 2005
	CodeAccountTimeout          = 2006
	CodeAccountLimitExceeded    = 2007
	CodeInvalidAccountType      = 2008
	CodeAccountNotConnected     = 2009
	CodePlatformNotSupported    = 2010

	// 3000–3999: order / trading
	CodeOrderNotFound         = 3001
	CodeOrderRejected         = 3002
	CodeInsufficientMargin    = 3003
	CodeMarketClosed          = 3004
	CodeInvalidOrderType      = 3005
	CodeInvalidVolume         = 3006
	CodeInvalidPrice          = 3007
	CodeOrderTimeout          = 3008
	CodePositionNotFound      = 3009
	CodeCannotClosePosition   = 3010
	CodeCannotModifyOrder     = 3011
	CodeOrderAlreadyFilled    = 3012
	CodeOrderAlreadyCancelled = 3013
	CodeSlippageExceeded      = 3014
	CodeSymbolNotSubscribed   = 3015

	// 4000–4999: market / symbol
	CodeSymbolNotFound       = 4001
	CodeNoMarketData         = 4002
	CodeSubFailed            = 4003
	CodeUnsubFailed          = 4004
	CodeQuoteNotAvailable    = 4005
	CodeHistoryNotAvailable  = 4006
	CodeInvalidTimeframe     = 4007
	CodeInvalidTimeRange     = 4008

	// 5000–5999: analytics
	CodeAnalyticsNotAvail  = 5001
	CodeReportGenFailed    = 5002
	CodeInvalidDateRange   = 5003
	CodeInsufficientData   = 5004

	// 6000–6999: admin
	CodeAdminAccessDenied  = 6001
	CodeOperationForbidden = 6002
	CodeAuditLogNotFound   = 6003

	// 7000–7999: broker
	CodeBrokerSearchFailed      = 7001
	CodeBrokerNotFound          = 7002
	CodeBrokerServerUnavailable = 7003

	// 8000–8999: strategy / AI
	CodeStrategyNotFound        = 8001
	CodeStrategyDisabled        = 8002
	CodeSandboxTimeout          = 8003
	CodeSandboxMemoryExceeded   = 8004
	CodeAIModelUnavailable      = 8005
	CodeAIGenerationFailed      = 8006
	CodeInvalidStrategyCode     = 8007
	CodeBacktestFailed          = 8008
	CodeScheduleNotActive       = 8009
	CodeSymbolNotInWhitelist    = 8010

	// 9000–9999: system
	CodeDBConnectionFailed  = 9001
	CodeRedisUnavailable    = 9002
	CodeMigrationFailed     = 9003
	CodeConfigInvalid       = 9004
	CodeGracefulShutdownErr = 9005
)

// ── Type ──────────────────────────────────────────────────────────────

type Error struct {
	Code     int    `json:"code"`
	Message  string `json:"message"`            // localized (user-facing)
	Detail   string `json:"detail,omitempty"`   // developer-facing
	Severity string `json:"severity,omitempty"` // info | warn | error | fatal
	Err      error  `json:"-"`                  // wrapped error (not serialized)
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error { return e.Err }

// ── Constructors ─────────────────────────────────────────────────────

// New returns an Error with the Chinese message for code.
func New(code int) *Error {
	return &Error{
		Code:     code,
		Message:  msgCN(code),
		Severity: severity(code),
	}
}

// Newf is like New but formats detail with Sprintf.
func Newf(code int, format string, args ...any) *Error {
	return &Error{
		Code:     code,
		Message:  msgCN(code),
		Detail:   fmt.Sprintf(format, args...),
		Severity: severity(code),
	}
}

// Wrap wraps an underlying error with the Chinese message for code.
func Wrap(code int, err error) *Error {
	return &Error{
		Code:     code,
		Message:  msgCN(code),
		Severity: severity(code),
		Err:      err,
	}
}

// Wrapf is like Wrap but adds formatted detail.
func Wrapf(code int, err error, format string, args ...any) *Error {
	return &Error{
		Code:     code,
		Message:  msgCN(code),
		Detail:   fmt.Sprintf(format, args...),
		Severity: severity(code),
		Err:      err,
	}
}

// ── Message helpers ──────────────────────────────────────────────────

// MessageCN returns the Chinese (user-facing) message for code.
func MessageCN(code int) string { return msgCN(code) }

// MessageEN returns the English message for code.
func MessageEN(code int) string { return msgEN(code) }

// ── Severity ─────────────────────────────────────────────────────────

func severity(code int) string {
	switch {
	case code >= 9000:
		return "fatal"
	case code >= 3000 && code < 4000:
		return "error"
	case code == CodeUnauthorized || code == CodeForbidden:
		return "error"
	default:
		return "warn"
	}
}
