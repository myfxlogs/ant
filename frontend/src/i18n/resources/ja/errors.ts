const errors = {
  errors: {
    // General 0-9
    success: '操作が成功しました',
    unknown: '不明なエラーが発生しました',
    invalid_parameter: '無効なパラメータ',
    unauthorized: 'ログインしてください',
    forbidden: 'アクセスが拒否されました',
    not_found: 'リソースが見つかりません',
    internal: 'サーバー内部エラー',
    rate_limited: 'リクエストが多すぎます。しばらくしてから再試行してください。',
    service_unavailable: 'サービスは一時的に利用できません',
    request_timeout: 'リクエストがタイムアウトしました',

    // User 1001-1010
    user_not_found: 'ユーザーが見つかりません',
    user_already_exists: 'このメールアドレスは既に登録されています',
    invalid_password: 'メールアドレスまたはパスワードが無効です',
    token_expired: 'セッションの有効期限が切れました。再度ログインしてください。',
    token_invalid: '無効なセッショントークン',
    token_missing: '認証が必要です',
    user_disabled: 'アカウントが無効化されています',
    email_not_verified: 'メールアドレスが確認されていません',
    password_too_weak: 'パスワードは8文字以上必要です',
    old_password_incorrect: '現在のパスワードが正しくありません',

    // Account 2001-2010
    account_not_found: '取引口座が見つかりません',
    account_already_bound: 'この取引口座は既に連携されています',
    account_connection_failed: '取引サーバーに接続できませんでした',
    account_disconnected: '取引口座が切断されました',
    account_auth_failed: '取引口座の認証に失敗しました',
    account_timeout: '取引サーバーへの接続がタイムアウトしました',
    account_limit_exceeded: '取引口座の最大数に達しました',
    invalid_account_type: '未対応の口座タイプです',
    account_not_connected: '取引口座が接続されていません',
    platform_not_supported: 'この取引プラットフォームはサポートされていません',

    // Order 3001-3015
    order_not_found: '注文が見つかりません',
    order_rejected: '注文が拒否されました',
    insufficient_margin: '証拠金が不足しています',
    market_closed: '市場は現在閉まっています',
    invalid_order_type: '無効な注文タイプ',
    invalid_volume: '無効な取引量',
    invalid_price: '無効な注文価格',
    order_timeout: '注文の実行がタイムアウトしました',
    position_not_found: 'ポジションが見つかりません',
    cannot_close_position: 'このポジションを決済できません',
    cannot_modify_order: 'この注文を変更できません',
    order_already_filled: '注文は既に約定済みです',
    order_already_cancelled: '注文は既にキャンセル済みです',
    slippage_exceeded: '価格スリッページが許容範囲を超えました',
    symbol_not_subscribed: '銘柄が購読されていません',

    // Market Data 4001-4008
    symbol_not_found: '銘柄が見つかりません',
    no_market_data: '市場データがありません',
    subscription_failed: '市場データの購読に失敗しました',
    unsubscription_failed: '市場データの購読解除に失敗しました',
    quote_not_available: 'クォートは利用できません',
    history_not_available: '履歴データは利用できません',
    invalid_timeframe: '無効な時間枠',
    invalid_time_range: '無効な期間',

    // Analytics 5001-5004
    analytics_not_available: '分析データは利用できません',
    report_generation_failed: 'レポートの生成に失敗しました',
    invalid_date_range: '無効な日付範囲',
    insufficient_data: '分析に十分なデータがありません',

    // Admin 6001-6003
    admin_access_denied: '管理者権限が必要です',
    operation_forbidden: 'この操作は許可されていません',
    audit_log_not_found: '監査ログが見つかりません',

    // Broker 7001-7003
    broker_search_failed: 'ブローカーの検索に失敗しました',
    broker_not_found: 'ブローカーが見つかりません',
    broker_server_unavailable: 'ブローカーサーバーは現在利用できません',
  },
} as const;

export default errors;
