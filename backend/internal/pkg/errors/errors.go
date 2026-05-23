package errors

import "fmt"

const (
	Success            = 0
	UnknownError       = 1
	InvalidParameter   = 2
	Unauthorized       = 3
	Forbidden          = 4
	NotFound           = 5
	InternalError      = 6
	RateLimited        = 7
	ServiceUnavailable = 8
	RequestTimeout     = 9

	UserNotFound         = 1001
	UserAlreadyExists    = 1002
	InvalidPassword      = 1003
	TokenExpired         = 1004
	TokenInvalid         = 1005
	TokenMissing         = 1006
	UserDisabled         = 1007
	EmailNotVerified     = 1008
	PasswordTooWeak      = 1009
	OldPasswordIncorrect = 1010

	AccountNotFound         = 2001
	AccountAlreadyBound     = 2002
	AccountConnectionFailed = 2003
	AccountDisconnected     = 2004
	AccountAuthFailed       = 2005
	AccountTimeout          = 2006
	AccountLimitExceeded    = 2007
	InvalidAccountType      = 2008
	AccountNotConnected     = 2009
	PlatformNotSupported    = 2010

	OrderNotFound         = 3001
	OrderRejected         = 3002
	InsufficientMargin    = 3003
	MarketClosed          = 3004
	InvalidOrderType      = 3005
	InvalidVolume         = 3006
	InvalidPrice          = 3007
	OrderTimeout          = 3008
	PositionNotFound      = 3009
	CannotClosePosition   = 3010
	CannotModifyOrder     = 3011
	OrderAlreadyFilled    = 3012
	OrderAlreadyCancelled = 3013
	SlippageExceeded      = 3014
	SymbolNotSubscribed   = 3015

	SymbolNotFound       = 4001
	NoMarketData         = 4002
	SubscriptionFailed   = 4003
	UnsubscriptionFailed = 4004
	QuoteNotAvailable    = 4005
	HistoryNotAvailable  = 4006
	InvalidTimeframe     = 4007
	InvalidTimeRange     = 4008

	AnalyticsNotAvailable  = 5001
	ReportGenerationFailed = 5002
	InvalidDateRange       = 5003
	InsufficientData       = 5004

	BrokerSearchFailed      = 7001
	BrokerNotFound          = 7002
	BrokerServerUnavailable = 7003

	AdminAccessDenied  = 6001
	OperationForbidden = 6002
	AuditLogNotFound   = 6003
)

var errorMessages = map[int]string{
	Success:            "errors.success",
	UnknownError:       "errors.unknown",
	InvalidParameter:   "errors.invalid_parameter",
	Unauthorized:       "errors.unauthorized",
	Forbidden:          "errors.forbidden",
	NotFound:           "errors.not_found",
	InternalError:      "errors.internal",
	RateLimited:        "errors.rate_limited",
	ServiceUnavailable: "errors.service_unavailable",
	RequestTimeout:     "errors.request_timeout",

	UserNotFound:         "errors.user_not_found",
	UserAlreadyExists:    "errors.user_already_exists",
	InvalidPassword:      "errors.invalid_password",
	TokenExpired:         "errors.token_expired",
	TokenInvalid:         "errors.token_invalid",
	TokenMissing:         "errors.token_missing",
	UserDisabled:         "errors.user_disabled",
	EmailNotVerified:     "errors.email_not_verified",
	PasswordTooWeak:      "errors.password_too_weak",
	OldPasswordIncorrect: "errors.old_password_incorrect",

	AccountNotFound:         "errors.account_not_found",
	AccountAlreadyBound:     "errors.account_already_bound",
	AccountConnectionFailed: "errors.account_connection_failed",
	AccountDisconnected:     "errors.account_disconnected",
	AccountAuthFailed:       "errors.account_auth_failed",
	AccountTimeout:          "errors.account_timeout",
	AccountLimitExceeded:    "errors.account_limit_exceeded",
	InvalidAccountType:      "errors.invalid_account_type",
	AccountNotConnected:     "errors.account_not_connected",
	PlatformNotSupported:    "errors.platform_not_supported",

	OrderNotFound:         "errors.order_not_found",
	OrderRejected:         "errors.order_rejected",
	InsufficientMargin:    "errors.insufficient_margin",
	MarketClosed:          "errors.market_closed",
	InvalidOrderType:      "errors.invalid_order_type",
	InvalidVolume:         "errors.invalid_volume",
	InvalidPrice:          "errors.invalid_price",
	OrderTimeout:          "errors.order_timeout",
	PositionNotFound:      "errors.position_not_found",
	CannotClosePosition:   "errors.cannot_close_position",
	CannotModifyOrder:     "errors.cannot_modify_order",
	OrderAlreadyFilled:    "errors.order_already_filled",
	OrderAlreadyCancelled: "errors.order_already_cancelled",
	SlippageExceeded:      "errors.slippage_exceeded",
	SymbolNotSubscribed:   "errors.symbol_not_subscribed",

	SymbolNotFound:       "errors.symbol_not_found",
	NoMarketData:         "errors.no_market_data",
	SubscriptionFailed:   "errors.subscription_failed",
	UnsubscriptionFailed: "errors.unsubscription_failed",
	QuoteNotAvailable:    "errors.quote_not_available",
	HistoryNotAvailable:  "errors.history_not_available",
	InvalidTimeframe:     "errors.invalid_timeframe",
	InvalidTimeRange:     "errors.invalid_time_range",

	AnalyticsNotAvailable:  "errors.analytics_not_available",
	ReportGenerationFailed: "errors.report_generation_failed",
	InvalidDateRange:       "errors.invalid_date_range",
	InsufficientData:       "errors.insufficient_data",

	BrokerSearchFailed:      "errors.broker_search_failed",
	BrokerNotFound:          "errors.broker_not_found",
	BrokerServerUnavailable: "errors.broker_server_unavailable",

	AdminAccessDenied:  "errors.admin_access_denied",
	OperationForbidden: "errors.operation_forbidden",
	AuditLogNotFound:   "errors.audit_log_not_found",
}

type AppError struct {
	Code    int
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

func New(code int, err error) *AppError {
	msg, ok := errorMessages[code]
	if !ok {
		msg = errorMessages[UnknownError]
	}
	return &AppError{
		Code:    code,
		Message: msg,
		Err:     err,
	}
}

func needsDetail(code int) bool {
	noDetailCodes := map[int]bool{
		AccountAlreadyBound:  true,
		UserNotFound:         true,
		UserAlreadyExists:    true,
		InvalidPassword:      true,
		AccountNotFound:      true,
		AccountLimitExceeded: true,
	}
	return !noDetailCodes[code]
}

func NewWithMessage(code int, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

func GetMessage(code int) string {
	if msg, ok := errorMessages[code]; ok {
		return msg
	}
	return errorMessages[UnknownError]
}
