const errors = {
  errors: {
    // General 0-9
    success: '操作成功',
    unknown: '發生未知錯誤',
    invalid_parameter: '參數無效',
    unauthorized: '請先登入',
    forbidden: '存取被拒絕',
    not_found: '未找到資源',
    internal: '伺服器內部錯誤',
    rate_limited: '請求過於頻繁，請稍後再試',
    service_unavailable: '服務暫時不可用',
    request_timeout: '請求逾時',

    // User 1001-1010
    user_not_found: '使用者不存在',
    user_already_exists: '該信箱已被註冊',
    invalid_password: '信箱或密碼錯誤',
    token_expired: '工作階段已過期，請重新登入',
    token_invalid: '無效的工作階段權杖',
    token_missing: '需要驗證',
    user_disabled: '帳戶已被停用',
    email_not_verified: '信箱未驗證',
    password_too_weak: '密碼長度不能少於8位',
    old_password_incorrect: '目前密碼不正確',

    // Account 2001-2010
    account_not_found: '交易帳戶未找到',
    account_already_bound: '該交易帳戶已被綁定',
    account_connection_failed: '無法連線到交易伺服器',
    account_disconnected: '交易帳戶已中斷連線',
    account_auth_failed: '交易帳戶驗證失敗',
    account_timeout: '連線交易伺服器逾時',
    account_limit_exceeded: '已達到最大交易帳戶數量',
    invalid_account_type: '不支援的帳戶類型',
    account_not_connected: '交易帳戶未連線',
    platform_not_supported: '不支援該交易平台',

    // Order 3001-3015
    order_not_found: '訂單未找到',
    order_rejected: '訂單被拒絕',
    insufficient_margin: '保證金不足',
    market_closed: '市場已關閉',
    invalid_order_type: '無效的訂單類型',
    invalid_volume: '無效的交易量',
    invalid_price: '無效的訂單價格',
    order_timeout: '訂單執行逾時',
    position_not_found: '持倉未找到',
    cannot_close_position: '無法平倉',
    cannot_modify_order: '無法修改訂單',
    order_already_filled: '訂單已成交',
    order_already_cancelled: '訂單已取消',
    slippage_exceeded: '價格滑點超出容忍範圍',
    symbol_not_subscribed: '未訂閱該交易品種',

    // Market Data 4001-4008
    symbol_not_found: '交易品種未找到',
    no_market_data: '無市場資料',
    subscription_failed: '訂閱市場資料失敗',
    unsubscription_failed: '取消訂閱失敗',
    quote_not_available: '報價不可用',
    history_not_available: '歷史資料不可用',
    invalid_timeframe: '無效的時間週期',
    invalid_time_range: '無效的時間範圍',

    // Analytics 5001-5004
    analytics_not_available: '分析資料不可用',
    report_generation_failed: '產生報告失敗',
    invalid_date_range: '無效的日期範圍',
    insufficient_data: '資料不足，無法分析',

    // Admin 6001-6003
    admin_access_denied: '需要管理員權限',
    operation_forbidden: '不允許執行此操作',
    audit_log_not_found: '稽核記錄未找到',

    // Broker 7001-7003
    broker_search_failed: '搜尋券商失敗',
    broker_not_found: '券商未找到',
    broker_server_unavailable: '券商伺服器目前不可用',
  },
} as const;

export default errors;
