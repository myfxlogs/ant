package errs

// msgCN returns the Chinese (user-facing) message for the given error code.
func msgCN(code int) string {
	m, ok := cnMessages[code]
	if ok {
		return m
	}
	return cnMessages[CodeUnknown]
}

// msgEN returns the English message for the given error code.
func msgEN(code int) string {
	m, ok := enMessages[code]
	if ok {
		return m
	}
	return enMessages[CodeUnknown]
}

var cnMessages = map[int]string{
	// 0–99: generic
	CodeOK:             "操作成功",
	CodeUnknown:        "未知错误",
	CodeInvalidParam:   "请求参数无效",
	CodeUnauthorized:   "未登录或登录已过期",
	CodeForbidden:      "无权限访问",
	CodeNotFound:       "资源不存在",
	CodeInternal:       "服务器内部错误，请稍后重试",
	CodeRateLimited:    "请求过于频繁，请稍后重试",
	CodeServiceUnavail: "服务暂时不可用，请稍后重试",
	CodeTimeout:        "请求超时，请检查网络后重试",

	// 1000–1999: auth / user
	CodeUserNotFound:         "用户不存在",
	CodeUserAlreadyExists:    "该用户名或邮箱已被注册",
	CodeInvalidPassword:      "密码错误",
	CodeTokenExpired:         "登录已过期，请重新登录",
	CodeTokenInvalid:         "无效的登录凭证",
	CodeTokenMissing:         "未提供登录凭证",
	CodeUserDisabled:         "该账号已被禁用",
	CodeEmailNotVerified:     "邮箱未验证，请先完成邮箱验证",
	CodePasswordTooWeak:      "密码强度不足，需包含字母和数字且长度≥8位",
	CodeOldPasswordIncorrect: "原密码错误",

	// 2000–2999: account
	CodeAccountNotFound:      "交易账户不存在",
	CodeAccountAlreadyBound:  "该交易账户已被其他用户绑定",
	CodeAccountConnFailed:    "交易账户连接失败，请检查账户信息",
	CodeAccountDisconnected:  "交易账户已断开连接",
	CodeAccountAuthFailed:    "交易账户认证失败，请检查账号和密码",
	CodeAccountTimeout:       "交易账户连接超时",
	CodeAccountLimitExceeded: "绑定交易账户数已达上限",
	CodeInvalidAccountType:   "不支持的账户类型",
	CodeAccountNotConnected:  "交易账户未连接",
	CodePlatformNotSupported: "不支持该交易平台",

	// 3000–3999: order / trading
	CodeOrderNotFound:         "订单不存在",
	CodeOrderRejected:         "订单被拒绝",
	CodeInsufficientMargin:    "保证金不足",
	CodeMarketClosed:          "当前市场已休市",
	CodeInvalidOrderType:      "无效的订单类型",
	CodeInvalidVolume:         "无效的交易手数",
	CodeInvalidPrice:          "无效的价格",
	CodeOrderTimeout:          "订单超时未成交",
	CodePositionNotFound:      "持仓不存在",
	CodeCannotClosePosition:   "无法平仓",
	CodeCannotModifyOrder:     "无法修改订单",
	CodeOrderAlreadyFilled:    "订单已成交，无法修改",
	CodeOrderAlreadyCancelled: "订单已撤销",
	CodeSlippageExceeded:      "滑点超过允许范围",
	CodeSymbolNotSubscribed:   "未订阅该交易品种行情",

	// 4000–4999: market / symbol
	CodeSymbolNotFound:      "交易品种不存在",
	CodeNoMarketData:        "暂无行情数据",
	CodeSubFailed:           "行情订阅失败",
	CodeUnsubFailed:         "行情取消订阅失败",
	CodeQuoteNotAvailable:   "当前无可用报价",
	CodeHistoryNotAvailable: "历史数据暂不可用",
	CodeInvalidTimeframe:    "无效的K线周期",
	CodeInvalidTimeRange:    "无效的时间范围",

	// 5000–5999: analytics
	CodeAnalyticsNotAvail: "分析服务暂不可用",
	CodeReportGenFailed:   "报告生成失败",
	CodeInvalidDateRange:  "无效的日期范围",
	CodeInsufficientData:  "数据不足，无法完成分析",

	// 6000–6999: admin
	CodeAdminAccessDenied:  "管理员权限不足",
	CodeOperationForbidden: "操作被禁止",
	CodeAuditLogNotFound:   "审计日志不存在",

	// 7000–7999: broker
	CodeBrokerSearchFailed:      "券商搜索失败",
	CodeBrokerNotFound:          "券商不存在",
	CodeBrokerServerUnavailable: "券商服务器不可用",

	// 8000–8999: strategy / AI
	CodeStrategyNotFound:      "策略不存在",
	CodeStrategyDisabled:      "策略已被禁用",
	CodeSandboxTimeout:        "策略执行超时",
	CodeSandboxMemoryExceeded: "策略执行内存超限",
	CodeAIModelUnavailable:    "AI 模型暂时不可用",
	CodeAIGenerationFailed:    "AI 策略生成失败",
	CodeInvalidStrategyCode:   "策略代码无效",
	CodeBacktestFailed:        "回测执行失败",
	CodeScheduleNotActive:     "策略调度未激活",
	CodeSymbolNotInWhitelist:  "交易品种不在白名单中",

	// 9000–9999: system
	CodeDBConnectionFailed:  "数据库连接失败",
	CodeRedisUnavailable:    "缓存服务不可用",
	CodeMigrationFailed:     "数据库迁移失败",
	CodeConfigInvalid:       "系统配置错误",
	CodeGracefulShutdownErr: "服务关闭异常",
}

var enMessages = map[int]string{
	// 0–99: generic
	CodeOK:             "Success",
	CodeUnknown:        "Unknown error",
	CodeInvalidParam:   "Invalid parameter",
	CodeUnauthorized:   "Unauthorized",
	CodeForbidden:      "Forbidden",
	CodeNotFound:       "Not found",
	CodeInternal:       "Internal server error",
	CodeRateLimited:    "Too many requests",
	CodeServiceUnavail: "Service unavailable",
	CodeTimeout:        "Request timeout",

	// 1000–1999
	CodeUserNotFound:         "User not found",
	CodeUserAlreadyExists:    "User already exists",
	CodeInvalidPassword:      "Invalid password",
	CodeTokenExpired:         "Token expired",
	CodeTokenInvalid:         "Invalid token",
	CodeTokenMissing:         "Token missing",
	CodeUserDisabled:         "User disabled",
	CodeEmailNotVerified:     "Email not verified",
	CodePasswordTooWeak:      "Password too weak",
	CodeOldPasswordIncorrect: "Old password incorrect",

	// 2000–2999
	CodeAccountNotFound:      "Account not found",
	CodeAccountAlreadyBound:  "Account already bound",
	CodeAccountConnFailed:    "Account connection failed",
	CodeAccountDisconnected:  "Account disconnected",
	CodeAccountAuthFailed:    "Account authentication failed",
	CodeAccountTimeout:       "Account connection timeout",
	CodeAccountLimitExceeded: "Account limit exceeded",
	CodeInvalidAccountType:   "Invalid account type",
	CodeAccountNotConnected:  "Account not connected",
	CodePlatformNotSupported: "Platform not supported",

	// 3000–3999
	CodeOrderNotFound:         "Order not found",
	CodeOrderRejected:         "Order rejected",
	CodeInsufficientMargin:    "Insufficient margin",
	CodeMarketClosed:          "Market closed",
	CodeInvalidOrderType:      "Invalid order type",
	CodeInvalidVolume:         "Invalid volume",
	CodeInvalidPrice:          "Invalid price",
	CodeOrderTimeout:          "Order timeout",
	CodePositionNotFound:      "Position not found",
	CodeCannotClosePosition:   "Cannot close position",
	CodeCannotModifyOrder:     "Cannot modify order",
	CodeOrderAlreadyFilled:    "Order already filled",
	CodeOrderAlreadyCancelled: "Order already cancelled",
	CodeSlippageExceeded:      "Slippage exceeded",
	CodeSymbolNotSubscribed:   "Symbol not subscribed",

	// 4000–4999
	CodeSymbolNotFound:      "Symbol not found",
	CodeNoMarketData:        "No market data",
	CodeSubFailed:           "Subscription failed",
	CodeUnsubFailed:         "Unsubscription failed",
	CodeQuoteNotAvailable:   "Quote not available",
	CodeHistoryNotAvailable: "History not available",
	CodeInvalidTimeframe:    "Invalid timeframe",
	CodeInvalidTimeRange:    "Invalid time range",

	// 5000–5999
	CodeAnalyticsNotAvail: "Analytics not available",
	CodeReportGenFailed:   "Report generation failed",
	CodeInvalidDateRange:  "Invalid date range",
	CodeInsufficientData:  "Insufficient data",

	// 6000–6999
	CodeAdminAccessDenied:  "Admin access denied",
	CodeOperationForbidden: "Operation forbidden",
	CodeAuditLogNotFound:   "Audit log not found",

	// 7000–7999
	CodeBrokerSearchFailed:      "Broker search failed",
	CodeBrokerNotFound:          "Broker not found",
	CodeBrokerServerUnavailable: "Broker server unavailable",

	// 8000–8999
	CodeStrategyNotFound:      "Strategy not found",
	CodeStrategyDisabled:      "Strategy disabled",
	CodeSandboxTimeout:        "Sandbox execution timeout",
	CodeSandboxMemoryExceeded: "Sandbox memory exceeded",
	CodeAIModelUnavailable:    "AI model unavailable",
	CodeAIGenerationFailed:    "AI generation failed",
	CodeInvalidStrategyCode:   "Invalid strategy code",
	CodeBacktestFailed:        "Backtest failed",
	CodeScheduleNotActive:     "Schedule not active",
	CodeSymbolNotInWhitelist:  "Symbol not in whitelist",

	// 9000–9999
	CodeDBConnectionFailed:  "Database connection failed",
	CodeRedisUnavailable:    "Redis unavailable",
	CodeMigrationFailed:     "Migration failed",
	CodeConfigInvalid:       "Config invalid",
	CodeGracefulShutdownErr: "Graceful shutdown error",
}
