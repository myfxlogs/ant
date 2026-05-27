const errors = {
  errors: {
    // General 0-9
    success: 'Success',
    unknown: 'An unknown error occurred',
    invalid_parameter: 'Invalid parameter',
    unauthorized: 'Please sign in to continue',
    forbidden: 'Access denied',
    not_found: 'Resource not found',
    internal: 'Internal server error',
    rate_limited: 'Too many requests. Please try again later.',
    service_unavailable: 'Service temporarily unavailable',
    request_timeout: 'Request timed out',

    // User 1001-1010
    user_not_found: 'User not found',
    user_already_exists: 'An account with this email already exists',
    invalid_password: 'Invalid email or password',
    token_expired: 'Session expired. Please sign in again.',
    token_invalid: 'Invalid session token',
    token_missing: 'Authentication required',
    user_disabled: 'Account has been disabled',
    email_not_verified: 'Email not verified',
    password_too_weak: 'Password must be at least 8 characters',
    old_password_incorrect: 'Current password is incorrect',

    // Account 2001-2010
    account_not_found: 'Trading account not found',
    account_already_bound: 'This trading account is already linked',
    account_connection_failed: 'Failed to connect to trading server',
    account_disconnected: 'Trading account disconnected',
    account_auth_failed: 'Trading account authentication failed',
    account_timeout: 'Connection to trading server timed out',
    account_limit_exceeded: 'Maximum number of trading accounts reached',
    invalid_account_type: 'Unsupported account type',
    account_not_connected: 'Trading account is not connected',
    platform_not_supported: 'This trading platform is not supported',

    // Order 3001-3015
    order_not_found: 'Order not found',
    order_rejected: 'Order was rejected',
    insufficient_margin: 'Insufficient margin',
    market_closed: 'Market is currently closed',
    invalid_order_type: 'Invalid order type',
    invalid_volume: 'Invalid trade volume',
    invalid_price: 'Invalid order price',
    order_timeout: 'Order execution timed out',
    position_not_found: 'Position not found',
    cannot_close_position: 'Cannot close this position',
    cannot_modify_order: 'Cannot modify this order',
    order_already_filled: 'Order has already been filled',
    order_already_cancelled: 'Order has already been cancelled',
    slippage_exceeded: 'Price slippage exceeded tolerance',
    symbol_not_subscribed: 'Symbol is not subscribed',

    // Market Data 4001-4008
    symbol_not_found: 'Trading symbol not found',
    no_market_data: 'No market data available',
    subscription_failed: 'Failed to subscribe to market data',
    unsubscription_failed: 'Failed to unsubscribe from market data',
    quote_not_available: 'Quote is not available',
    history_not_available: 'Historical data is not available',
    invalid_timeframe: 'Invalid timeframe',
    invalid_time_range: 'Invalid time range',

    // Analytics 5001-5004
    analytics_not_available: 'Analytics data is not available',
    report_generation_failed: 'Failed to generate report',
    invalid_date_range: 'Invalid date range',
    insufficient_data: 'Insufficient data for analysis',

    // Admin 6001-6003
    admin_access_denied: 'Administrator access required',
    operation_forbidden: 'This operation is not allowed',
    audit_log_not_found: 'Audit log entry not found',

    // Broker 7001-7003
    broker_search_failed: 'Failed to search for brokers',
    broker_not_found: 'Broker not found',
    broker_server_unavailable: 'Broker server is currently unavailable',
  },
} as const;

export default errors;
