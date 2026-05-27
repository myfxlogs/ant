const errors = {
  errors: {
    // General 0-9
    success: '操作成功',
    unknown: '发生未知错误',
    invalid_parameter: '参数无效',
    unauthorized: '请先登录',
    forbidden: '访问被拒绝',
    not_found: '未找到资源',
    internal: '服务器内部错误',
    rate_limited: '请求过于频繁，请稍后再试',
    service_unavailable: '服务暂时不可用',
    request_timeout: '请求超时',

    // User 1001-1010
    user_not_found: '用户不存在',
    user_already_exists: '该邮箱已被注册',
    invalid_password: '邮箱或密码错误',
    token_expired: '会话已过期，请重新登录',
    token_invalid: '无效的会话令牌',
    token_missing: '需要认证',
    user_disabled: '账户已被禁用',
    email_not_verified: '邮箱未验证',
    password_too_weak: '密码长度不能少于8位',
    old_password_incorrect: '当前密码不正确',

    // Account 2001-2010
    account_not_found: '交易账户未找到',
    account_already_bound: '该交易账户已被绑定',
    account_connection_failed: '无法连接到交易服务器',
    account_disconnected: '交易账户已断开连接',
    account_auth_failed: '交易账户认证失败',
    account_timeout: '连接交易服务器超时',
    account_limit_exceeded: '已达到最大交易账户数量',
    invalid_account_type: '不支持的账户类型',
    account_not_connected: '交易账户未连接',
    platform_not_supported: '不支持该交易平台',

    // Order 3001-3015
    order_not_found: '订单未找到',
    order_rejected: '订单被拒绝',
    insufficient_margin: '保证金不足',
    market_closed: '市场已关闭',
    invalid_order_type: '无效的订单类型',
    invalid_volume: '无效的交易量',
    invalid_price: '无效的订单价格',
    order_timeout: '订单执行超时',
    position_not_found: '持仓未找到',
    cannot_close_position: '无法平仓',
    cannot_modify_order: '无法修改订单',
    order_already_filled: '订单已成交',
    order_already_cancelled: '订单已取消',
    slippage_exceeded: '价格滑点超出容忍范围',
    symbol_not_subscribed: '未订阅该交易品种',

    // Market Data 4001-4008
    symbol_not_found: '交易品种未找到',
    no_market_data: '无市场数据',
    subscription_failed: '订阅市场数据失败',
    unsubscription_failed: '取消订阅失败',
    quote_not_available: '报价不可用',
    history_not_available: '历史数据不可用',
    invalid_timeframe: '无效的时间周期',
    invalid_time_range: '无效的时间范围',

    // Analytics 5001-5004
    analytics_not_available: '分析数据不可用',
    report_generation_failed: '生成报告失败',
    invalid_date_range: '无效的日期范围',
    insufficient_data: '数据不足，无法分析',

    // Admin 6001-6003
    admin_access_denied: '需要管理员权限',
    operation_forbidden: '不允许执行此操作',
    audit_log_not_found: '审计日志未找到',

    // Broker 7001-7003
    broker_search_failed: '搜索券商失败',
    broker_not_found: '券商未找到',
    broker_server_unavailable: '券商服务器当前不可用',
  },
} as const;

export default errors;
