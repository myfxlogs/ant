# I18N-MIXED-2 派工单（相位 1：zh-tw）——2026-09-19 Devin CLI 设计实查

> 承接 I18N-MIXED-1（zh-cn ✅done）。**本相位范围 = zh-tw 全清**；
> ja/vi 同债同构（各 124 Missing + 300/318 Untranslated），机制完全相同，
> zh-tw 验收后按本文档模式续相位（DICT 换 ja/vi 译文即可）。
> 业主指令"所有信息以中文显示"——zh-tw 繁中属指令范围。

## 设计实查（独立实证）

- `i18n-check --strict --locale zh-tw`：**424 errors**（300 Untranslated + 124 Missing）+5 warnings
- Missing 124 分布：admin×116 + base×5 + strategy_gen×3——en 新功能键从未传播到 zh-tw
  （zh-cn 已含全部 124——传播机制曾覆盖 zh-cn 未覆盖其他 locale）
- Untranslated 300：202 多词句 + 93 单词 + 5 模板串——全部需译（zh-cn 先例全译：
  Magic→魔术号/Pro→专业版/Discover→发现/Symbol→品种，非白名单）
- zh-tw 混杂候选仅 1 条（Class B 基本是 zh-cn 特有）——顺带核查处理
- 机制瓶颈：`i18n-translate-zh-cn.ts` DICT 内嵌 zh-cn 译文——不可复用于 zh-tw

## S1 — 翻译脚本泛化（共享引擎 + per-locale DICT）

`scripts/i18n-translate-zh-cn.ts` 当前结构：DICT 内嵌 + main() 遍历 `*_zh-cn.textproto` 文件对。
泛化为共享引擎，最小改动方案：

- 新建 `scripts/i18n-dict/zh-tw.ts`：`export const DICT: Record<string,string> = {...}`
- `i18n-translate-zh-cn.ts` 重构为 `scripts/i18n-translate.ts --locale <loc>`：
  `import(`./i18n-dict/${locale}.ts`)` 取 DICT，文件对后缀 `_${locale}.textproto`；
  zh-cn 的现有 DICT 原样迁到 `scripts/i18n-dict/zh-cn.ts`（纯搬迁零改写）
- 兼容：保留 `i18n-translate-zh-cn.ts` 为薄壳转发 `--locale zh-cn`（Makefile/文档引用不断链）

## S2 — zh-tw DICT ~300 条（附录 A）

`scripts/i18n-dict/zh-tw.ts` 收录附录 A 全部 en 值的繁中译文。要求：
- 繁体中文（非简体照搬）：已发布到市场→已發佈到市場、订单→訂單、执行→執行、账户→帳戶
- zh-cn DICT 可作初稿加速（简→繁机械转换后逐条审校一对多字：发→發/髮、台→台/臺、
  面→面/麵、复→復/複/覆；金融业惯例：平台/帳戶/訂單/執行/持倉）
- 品牌/语言名/技术词保留（白名单同 zh-cn 附录 C：AlphaForge/MT4/MT5/DeepSeek/API Key 等）
- {{var}} 占位符原样保留；AI prompt 内部串保留英文（同 zh-cn 处理）

## S3 — zh-tw Missing 124 字段补齐（附录 B）

对应 section 的 `*_zh-tw.textproto` 追加附录 B 全部 124 key+繁中译文
（admin×116/base×5/strategy_gen×3）。map.json 无需动（en 侧映射已存在）。
`exec_feedback_placeholder` 译文对齐 zh-cn 同键语义：
'試試說：\n"把止損收緊到 1%"\n"為什麼夏普比率這麼低？幫我改進"\n"改成只做多，不做空"'

## S4 — 门禁扩展

zh-tw strict 归零后，pre-commit i18n 段从 `--locale zh-cn` 扩为双 locale：
`npx tsx scripts/i18n-check.ts --strict --locale zh-cn && npx tsx scripts/i18n-check.ts --strict --locale zh-tw`
（ja/vi 相位完成后去 --locale 改全量）。提示语同步更新。

## S5 — 守卫测试扩展

`i18n-key-presence.test.ts` 增补 zh-tw 断言：
- 43 路径 zh-tw 非空（已有循环覆盖，确认 LOCALE_BASES 含 zh-tw ✓ 已含）
- zh-tw≠en 样本断言 2 条（orderTruth='訂單真相' 已验证模式）
- exec_feedback_placeholder zh-tw 存在（S3 相位本债补）

## 对抗证明

- M1：删 zh-tw DICT 一条目 → strict Untranslated error RED → restore GREEN
- M2：删 zh-tw textproto 补入的 missing 字段 → strict Missing RED → restore GREEN
- M3：DICT 值改回英文 → zh-tw≠en 断言/strict RED

## 门禁

```
npx tsx scripts/i18n-translate.ts --locale zh-tw
npx tsx scripts/i18n-build.ts
npx tsx scripts/i18n-check.ts --strict --locale zh-tw   # 0 errors 0 warnings
npx tsx scripts/i18n-check.ts --strict --locale zh-cn   # 仍 0/0（防回归）
cd frontend && npx tsc --noEmit && npx vitest run
git diff --check
```

边界：ja/vi textproto 不动（后续相位）；zh-cn 零改动（防回归）；
不删既有 zh-tw 译文；勿部署，完成报证据等复审。

## 附录 A — zh-tw Untranslated 300 条（en 值全清单）

```json
{
 "orders_table_magic": "Magic",
 "trade_tabs_table_magic": "Magic",
 "admin_billing_plan_pro": "Pro",
 "admin_billing_plan_enterprise": "Enterprise",
 "strategy_backtest_auto_gate": "Auto Gate Evaluation",
 "strategy_backtest_publishable": "Publishable",
 "strategy_backtest_not_publishable": "Not Publishable",
 "strategy_backtest_diagnostic_invariant": "Invariant Violation",
 "strategy_backtest_diagnostic_defense_a": "Structural Validation",
 "strategy_backtest_diagnostic_lookahead": "Lookahead Bias",
 "strategy_backtest_diagnostic_statistical": "Statistical Hint",
 "strategy_backtest_diagnostic_unknown": "Diagnostic",
 "strategy_backtest_diagnostic_coverage": "Coverage",
 "strategy_backtest_diagnostic_compatible": "compatible",
 "strategy_backtest_diagnostic_unsupported": "unsupported",
 "strategy_backtest_diagnostic_fatal": "Critical Issues",
 "strategy_backtest_diagnostic_warning": "Risk Warnings",
 "strategy_backtest_diagnostic_all_silenced": "All warnings acknowledged as intentional",
 "strategy_backtest_diagnostic_info": "Quality Hints",
 "strategy_backtest_diagnostic_ai_fix": "AI Fix",
 "strategy_backtest_diagnostic_no_code": "No strategy code to fix",
 "strategy_backtest_diagnostic_ai_no_result": "AI returned no code",
 "strategy_backtest_diagnostic_ai_failed": "AI fix failed",
 "strategy_backtest_diagnostic_fix_applied_compile_warn": "Fix applied but compile has warnings",
 "strategy_backtest_diagnostic_apply_failed": "Failed to apply fix",
 "strategy_backtest_diagnostic_save_first": "Please save the strategy first to apply AI fixes",
 "strategy_backtest_diagnostic_diff_preview": "AI Fix Preview",
 "strategy_backtest_diagnostic_apply": "Apply & Re-run",
 "strategy_backtest_diagnostic_diff_hint": "Review the AI-generated code below. Apply to create a new ve",
 "strategy_backtest_cancel_failed": "Cancel failed",
 "strategy_templates_gallery_title": "Strategies",
 "notifications_prefs_save_failed": "Failed to save preferences",
 "notifications_prefs_new_strategy": "New strategy published",
 "notifications_prefs_price_change": "Strategy price changed",
 "notifications_prefs_sub_expiring": "Subscription expiring soon",
 "notifications_prefs_performance": "Strategy performance anomaly",
 "notifications_prefs_new_rating": "New rating or comment received",
 "notifications_prefs_title": "Notification Preferences",
 "strategy_ai_chat_code_loaded": "Strategy code in context",
 "strategy_chat_entry": "Entry:",
 "strategy_chat_exit": "Exit:",
 "strategy_chat_risk": "Risk:",
 "strategy_chat_indicators": "Indicators:",
 "import_analysis_empty_analysis_desc": "No strategy logic was recognized. The source code may be inc",
 "accounts_status_circuit_open": "Circuit Open",
 "accounts_status_circuit_half_open": "Circuit Testing",
 "analytics_pnl": "P&L:",
 "admin_ai_gateway_no_models_discovered": "No models discovered. Check API key and base URL.",
 "admin_ai_gateway_discover_failed": "Failed to discover models",
 "admin_ai_gateway_discover": "Discover",
 "admin_autogen_load_failed": "Failed to load tasks",
 "admin_autogen_approved": "Task approved and published",
 "admin_autogen_approve_failed": "Approve failed",
 "admin_autogen_rejected": "Task rejected",
 "admin_autogen_reject_failed": "Reject failed",
 "admin_autogen_trigger_failed": "Trigger failed",
 "admin_autogen_symbol": "Symbol",
 "admin_autogen_strategy_type": "Type",
 "admin_autogen_status": "Status",
 "admin_autogen_quality": "Quality",
 "admin_autogen_error": "Error",
 "admin_autogen_actions": "Actions",
 "admin_autogen_confirm_approve": "Approve and publish?",
 "admin_autogen_approve": "Approve",
 "admin_autogen_confirm_reject": "Reject this task?",
 "admin_autogen_reject": "Reject",
 "admin_autogen_title": "AI Strategy Generation Tasks",
 "admin_autogen_all_status": "All Status",
 "admin_autogen_refresh": "Refresh",
 "admin_autogen_trigger_batch": "Trigger Batch",
 "admin_autogen_enqueue": "Enqueue",
 "admin_autogen_symbols": "Symbols (comma-separated)",
 "admin_autogen_timeframes": "Timeframes (comma-separated)",
 "admin_autogen_strategy_types": "Strategy Types (comma-separated)",
 "admin_coupon_load_failed": "Failed to load coupons",
 "admin_coupon_fill_required": "Please fill required fields",
 "admin_coupon_created": "Coupon created",
 "admin_coupon_create_failed": "Failed to create coupon",
 "admin_coupon_disabled": "Coupon disabled",
 "admin_coupon_disable_failed": "Failed to disable coupon",
 "admin_coupon_col_code": "Code",
 "admin_coupon_col_type": "Type",
 "admin_coupon_col_value": "Value",
 "admin_coupon_col_min_purchase": "Min Purchase",
 "admin_coupon_col_usage": "Usage",
 "admin_coupon_col_expires": "Expires",
 "admin_coupon_col_status": "Status",
 "admin_coupon_col_actions": "Actions",
 "admin_coupon_disable": "Disable",
 "admin_coupon_create": "Create Coupon",
 "admin_coupon_create_title": "Create Coupon",
 "admin_coupon_code_placeholder": "Coupon code (e.g. SUMMER20)",
 "admin_coupon_value_placeholder": "Discount value (e.g. 20 for 20% or 50 for $50)",
 "admin_coupon_min_purchase_placeholder": "Minimum purchase amount (0 = none)",
 "admin_coupon_max_uses_placeholder": "Max uses (0 = unlimited)",
 "admin_coupon_expires_placeholder": "Expires at (ISO 8601, empty = never)",
 "admin_deposit_addresses_import_failed": "Import failed",
 "admin_deposit_addresses_address": "Address",
 "admin_deposit_addresses_user": "User ID",
 "admin_deposit_addresses_index": "Index",
 "admin_deposit_addresses_status": "Status",
 "admin_deposit_addresses_received": "Received USDT",
 "admin_deposit_addresses_network": "Network",
 "admin_deposit_addresses_assigned_at": "Assigned At",
 "admin_deposit_addresses_import_hint": "Use hdgen tool on an offline machine to generate deposit_add",
 "admin_deposit_addresses_all": "All Status",
 "admin_deposit_addresses_import": "Import Addresses",
 "admin_deposit_addresses_available_pool": "Available in Pool",
 "admin_deposit_addresses_total": "Total Addresses",
 "admin_deposit_table_block": "Block",
 "admin_deposit_table_confirmations": "Confirmations",
 "admin_analytics_name": "Name",
 "admin_analytics_value": "Value",
 "admin_analytics_platform_rev": "Platform Rev",
 "admin_analytics_provider_rev": "Provider Rev",
 "admin_analytics_active_buyers": "Active Buyers",
 "admin_analytics_refund_rate": "Refund Rate",
 "admin_analytics_total_tx": "Transactions",
 "admin_analytics_new_subs": "New Subscribers",
 "admin_analytics_total_strategies": "Total Strategies",
 "admin_analytics_new_strategies": "New Strategies",
 "admin_analytics_top_by_revenue": "Top Strategies by Revenue",
 "admin_analytics_top_by_subs": "Top Strategies by Subscribers",
 "admin_analytics_top_providers_rev": "Top Providers by Revenue",
 "admin_analytics_top_providers_strat": "Top Providers by Strategies",
 "admin_marketplace_load_failed": "Failed to load strategies",
 "admin_marketplace_feature_success": "Strategy featured",
 "admin_marketplace_feature_failed": "Failed to feature strategy",
 "admin_marketplace_unfeature_success": "Removed featured",
 "admin_marketplace_unfeature_failed": "Failed to unfeature",
 "admin_marketplace_col_title": "Title",
 "admin_marketplace_col_publisher": "Publisher",
 "admin_marketplace_col_status": "Status",
 "admin_marketplace_col_price": "Price",
 "admin_marketplace_col_sales": "Sales",
 "admin_marketplace_col_revenue": "Revenue",
 "admin_marketplace_col_featured": "Featured",
 "admin_marketplace_col_actions": "Actions",
 "admin_marketplace_feature": "Feature",
 "admin_marketplace_unfeature": "Remove featured",
 "admin_marketplace_filter_status": "All statuses",
 "admin_marketplace_search_placeholder": "Search by title...",
 "admin_marketplace_feature_title": "Feature Strategy",
 "admin_marketplace_feature_desc": "Set priority for featured placement. Higher = more prominent",
 "admin_refund_load_failed": "Failed to load refund requests",
 "admin_refund_approved": "Refund approved and executed",
 "admin_refund_rejected": "Refund request rejected",
 "admin_refund_process_failed": "Failed to process refund",
 "admin_refund_col_user": "User",
 "admin_refund_col_strategy": "Strategy",
 "admin_refund_col_amount": "Amount",
 "admin_refund_col_reason": "Reason",
 "admin_refund_col_status": "Status",
 "admin_refund_col_date": "Date",
 "admin_refund_col_actions": "Actions",
 "admin_refund_approve": "Approve & Execute",
 "admin_refund_reject": "Reject",
 "admin_refund_filter_status": "All statuses",
 "admin_refund_approve_title": "Approve Refund",
 "admin_refund_reject_title": "Reject Refund",
 "admin_refund_review_note_placeholder": "Review note (optional for reject, recommended for approve)..",
 "admin_wallet_tab_wallets": "User Wallets",
 "admin_wallet_tab_deposit_addresses": "Deposit Addresses",
 "admin_user_management_form_account_number_placeholder": "e.g. 123568",
 "sre_kill_switch_title": "Kill Switch",
 "auth_forgot_password_email_sent": "If the email exists, a reset link has been sent.",
 "auth_forgot_password_mt_verified": "Identity verified. Redirecting to password reset.",
 "auth_forgot_password_mt_failed": "MT credential verification failed.",
 "auth_forgot_password_email_tab": "Email",
 "auth_forgot_password_send_reset_link": "Send Reset Link",
 "auth_forgot_password_mt_tab": "MT Verify",
 "auth_forgot_password_mt_login": "MT Account Number",
 "auth_forgot_password_mt_login_placeholder": "e.g. 12345678",
 "auth_forgot_password_mt_password": "MT Password",
 "auth_forgot_password_mt_password_placeholder": "MT trading password",
 "auth_forgot_password_mt_hint": "Enter your bound MT account credentials to verify your ident",
 "auth_forgot_password_verify_and_reset": "Verify & Reset Password",
 "auth_forgot_password_admin_tab": "Admin",
 "auth_forgot_password_admin_hint": "Please contact your administrator or support to reset your p",
 "auth_reset_password_mismatch": "Passwords do not match.",
 "auth_reset_password_invalid_token": "Invalid or missing reset token.",
 "auth_reset_password_success": "Password has been reset. Please log in with your new passwor",
 "auth_reset_password_failed": "Failed to reset password.",
 "auth_reset_password_title": "Set New Password",
 "auth_reset_password_new_password": "New Password",
 "auth_reset_password_confirm_required": "Please confirm your password",
 "auth_reset_password_confirm_password": "Confirm Password",
 "auth_reset_password_submit": "Reset Password",
 "dashboard_no_accounts_desc": "Bind your first MT4/MT5 account to start monitoring and trad",
 "landing_brokers_title": "Compatible with 30+ MT4/MT5 Brokers",
 "landing_brokers_desc": "IC Markets, Pepperstone, XM, Exness, OANDA, FXTM, FBS, OctaF",
 "landing_brokers_link": "View all supported brokers",
 "marketplace_live_load_error": "Failed to load live performance data",
 "marketplace_optimization_decay_score": "Decay Score",
 "marketplace_optimization_trigger": "Trigger",
 "marketplace_optimization_sharpe_decline": "Sharpe Decline",
 "marketplace_optimization_win_rate_decline": "Win Rate Decline",
 "marketplace_optimization_return_delta": "Return Delta",
 "strategy_templates_actions_deploy": "Deploy",
 "marketplace_payment_deploy_guide": "Your strategy is ready to deploy.",
 "marketplace_payment_go_deploy": "Deploy Now",
 "strategy_templates_schedule_launch_metrics_win_rate": "Win Rate",
 "strategy_templates_detail_profit_factor": "Profit Factor",
 "strategy_templates_schedule_launch_metrics_max_drawdown": "Max Drawdown",
 "strategy_templates_schedule_launch_metrics_sharpe": "Sharpe Ratio",
 "strategy_templates_detail_not_found": "Strategy not found",
 "strategy_templates_gallery_system": "System",
 "strategy_templates_gallery_fork_edit": "Fork & Edit",
 "strategy_templates_detail_open_in_workspace": "Open in Workspace",
 "strategy_templates_detail_overview": "Overview",
 "strategy_templates_detail_no_description": "No description",
 "strategy_templates_detail_equity_curve": "Equity Curve",
 "strategy_templates_detail_trade_stats": "Trade Statistics",
 "strategy_templates_table_use_count": "Use Count",
 "strategy_templates_table_created_at": "Created",
 "strategy_templates_table_visibility": "Visibility",
 "strategy_templates_visibility_public": "Public",
 "strategy_templates_visibility_private": "Private",
 "strategy_templates_table_status": "Status",
 "strategy_templates_detail_parameters": "Parameters",
 "strategy_templates_code_modal_title": "Code",
 "strategy_templates_actions_create": "New Strategy",
 "strategy_templates_gallery_ai_generate": "AI Generate",
 "strategy_templates_gallery_search_placeholder": "Search strategies...",
 "strategy_templates_gallery_filter_all": "All",
 "strategy_templates_gallery_filter_mine": "Mine",
 "strategy_templates_gallery_filter_system": "System",
 "strategy_templates_gallery_sort_recent": "Recent",
 "strategy_templates_gallery_sort_return": "Return",
 "strategy_templates_gallery_sort_risk": "Risk",
 "strategy_templates_gallery_sort_usage": "Usage",
 "strategy_templates_messages_fetch_template_list_failed": "Failed to load strategies",
 "strategy_templates_gallery_empty": "No strategies found",
 "schedule_launch_no_account_bind_button": "Bind MT Account",
 "marketplace_publish_trial_days_7": "7 days",
 "marketplace_publish_trial_days_14": "14 days",
 "marketplace_publish_trial_days_30": "30 days",
 "marketplace_publish_trial_days_label": "Trial Period",
 "marketplace_publish_trial_days_placeholder": "Select or enter custom days",
 "marketplace_publish_trial_days_custom": "Custom days",
 "strategy_templates_gallery_fork_success": "Forked to new strategy",
 "strategy_templates_gallery_fork_failed": "Fork failed",
 "strategy_templates_messages_publish_failed": "Publish failed",
 "strategy_templates_gallery_unpublish_success": "Unpublished",
 "strategy_templates_gallery_unpublish_failed": "Unpublish failed",
 "strategy_templates_messages_template_deleted": "Deleted",
 "strategy_templates_gallery_delete_failed": "Delete failed",
 "strategy_templates_gallery_deploy": "Deploy",
 "strategy_templates_gallery_publish": "Publish",
 "strategy_templates_gallery_unpublish": "Unpublish",
 "strategy_templates_delete_confirm": "Delete this strategy?",
 "strategy_templates_actions_delete": "Delete",
 "strategy_tuning_strategy_name": "Strategy",
 "strategy_tuning_total_trades": "Trades",
 "strategy_workspace_sidebar_no_runs": "No backtest runs yet",
 "common_selected": "selected",
 "strategy_workspace_sidebar_batch_delete_runs_confirm": "Delete selected runs?",
 "strategy_workspace_sidebar_trades": "trades",
 "strategy_workspace_sidebar_delete_run_confirm": "Delete this backtest run?",
 "strategy_workspace_sidebar_view_all": "View all",
 "strategy_workspace_sidebar_no_strategies": "No strategies yet",
 "strategy_workspace_sidebar_batch_delete_confirm": "Delete selected strategies?",
 "strategy_workspace_sidebar_delete_strategy_confirm": "Delete this strategy?",
 "strategy_tuning_no_params_title": "No tunable parameters detected",
 "strategy_tuning_no_params_desc": "Add @param annotations to your strategy code to enable Smart",
 "strategy_tuning_disabled_hint": "Need strategy code and symbol. Select a strategy from the si",
 "strategy_tuning_no_dims_hint": "Enable at least one parameter dimension below.",
 "strategy_workspace_sidebar_title": "Workspace",
 "strategy_workspace_sidebar_my_strategies": "My Strategies",
 "strategy_workspace_sidebar_backtest_history": "Backtest History",
 "strategy_workspace_sidebar_new_strategy": "New Strategy",
 "strategy_workspace_import_mql": "Import MQL",
 "strategy_workspace_tour_ai": "AI Assistant",
 "strategy_workspace_tour_ai_desc": "Ask AI to generate, optimize, or debug your strategy. Applie",
 "strategy_workspace_tour_code": "Code Editor",
 "strategy_workspace_tour_code_desc": "Write or paste your MQL strategy code here. You can also imp",
 "strategy_workspace_tour_backtest": "Backtest",
 "strategy_workspace_tour_backtest_desc": "Run backtests with configurable parameters. View equity curv",
 "strategy_workspace_tour_save": "Save & Publish",
 "strategy_workspace_tour_save_desc": "Save your strategy as a template, publish to marketplace, or",
 "strategy_validate_auto_fix_failed": "Auto-fix failed",
 "strategy_validate_failed": "Validation failed",
 "strategy_templates_load_failed": "Failed to load templates",
 "strategy_templates_load_one_failed": "Failed to load template",
 "strategy_tuning_failed": "Tuning failed",
 "subscription_unbind_success": "Account unbound successfully.",
 "subscription_unbind_failed": "Failed to unbind account.",
 "subscription_account_login": "Login",
 "subscription_account_broker": "Broker",
 "subscription_account_server": "Server",
 "subscription_account_type": "Type",
 "subscription_account_status": "Status",
 "subscription_bound_at": "Bound At",
 "subscription_unbind_confirm": "Unbind this account? Active schedules on it will be stopped.",
 "subscription_unbind": "Unbind",
 "subscription_bound_accounts_count": "Bound Accounts",
 "subscription_no_bound_accounts": "No bound accounts yet. Schedule a strategy to auto-bind an a",
 "subscription_feature_unlimited_accounts": "Unlimited MT accounts",
 "subscription_ai_tokens_remaining": "AI Tokens Remaining",
 "subscription_bound_accounts_title": "Bound MT Accounts"
}
```

## 附录 B — zh-tw Missing 124 条（en 值按 section）

```json
{
 "admin": [
  [
   "aiGateway_modelList",
   "Models ({{count}})"
  ],
  [
   "analytics_activeBuyers",
   "Active Buyers"
  ],
  [
   "analytics_name",
   "Name"
  ],
  [
   "analytics_newStrategies",
   "New Strategies"
  ],
  [
   "analytics_newSubs",
   "New Subscribers"
  ],
  [
   "analytics_platformRev",
   "Platform Revenue"
  ],
  [
   "analytics_providerRev",
   "Provider Revenue"
  ],
  [
   "analytics_refundRate",
   "Refund Rate"
  ],
  [
   "analytics_topByRevenue",
   "Top Strategies by Revenue"
  ],
  [
   "analytics_topBySubs",
   "Top Strategies by Subscribers"
  ],
  [
   "analytics_topProvidersRev",
   "Top Providers by Revenue"
  ],
  [
   "analytics_topProvidersStrat",
   "Top Providers by Strategies"
  ],
  [
   "analytics_totalStrategies",
   "Total Strategies"
  ],
  [
   "analytics_totalTx",
   "Transactions"
  ],
  [
   "analytics_value",
   "Value"
  ],
  [
   "autogen_actions",
   "Actions"
  ],
  [
   "autogen_allStatus",
   "All Status"
  ],
  [
   "autogen_approve",
   "Approve"
  ],
  [
   "autogen_approved",
   "Task approved and published"
  ],
  [
   "autogen_confirmApprove",
   "Approve and publish?"
  ],
  [
   "autogen_confirmReject",
   "Reject this task?"
  ],
  [
   "autogen_enqueue",
   "Enqueue"
  ],
  [
   "autogen_enqueued",
   "{{count}} tasks enqueued"
  ],
  [
   "autogen_error",
   "Error"
  ],
  [
   "autogen_quality",
   "Quality"
  ],
  [
   "autogen_refresh",
   "Refresh"
  ],
  [
   "autogen_reject",
   "Reject"
  ],
  [
   "autogen_rejected",
   "Task rejected"
  ],
  [
   "autogen_status",
   "Status"
  ],
  [
   "autogen_strategyType",
   "Type"
  ],
  [
   "autogen_strategyTypes",
   "Strategy Types (comma-separated)"
  ],
  [
   "autogen_symbol",
   "Symbol"
  ],
  [
   "autogen_symbols",
   "Symbols (comma-separated)"
  ],
  [
   "autogen_timeframe",
   "TF"
  ],
  [
   "autogen_timeframes",
   "Timeframes (comma-separated)"
  ],
  [
   "autogen_title",
   "AI Strategy Generation Tasks"
  ],
  [
   "autogen_triggerBatch",
   "Trigger Batch Generation"
  ],
  [
   "coupon_codePlaceholder",
   "Coupon code (e.g. SUMMER20)"
  ],
  [
   "coupon_colActions",
   "Actions"
  ],
  [
   "coupon_colCode",
   "Code"
  ],
  [
   "coupon_colExpires",
   "Expires"
  ],
  [
   "coupon_colMinPurchase",
   "Min Purchase"
  ],
  [
   "coupon_colStatus",
   "Status"
  ],
  [
   "coupon_colType",
   "Type"
  ],
  [
   "coupon_colUsage",
   "Usage"
  ],
  [
   "coupon_colValue",
   "Value"
  ],
  [
   "coupon_create",
   "Create Coupon"
  ],
  [
   "coupon_createFailed",
   "Failed to create coupon"
  ],
  [
   "coupon_createTitle",
   "Create Coupon"
  ],
  [
   "coupon_created",
   "Coupon created"
  ],
  [
   "coupon_disable",
   "Disable"
  ],
  [
   "coupon_disableFailed",
   "Failed to disable coupon"
  ],
  [
   "coupon_disabled",
   "Coupon disabled"
  ],
  [
   "coupon_expiresPlaceholder",
   "Expires at (ISO 8601, empty = never)"
  ],
  [
   "coupon_fillRequired",
   "Please fill required fields"
  ],
  [
   "coupon_loadFailed",
   "Failed to load coupons"
  ],
  [
   "coupon_maxUsesPlaceholder",
   "Max uses (0 = unlimited)"
  ],
  [
   "coupon_minPurchasePlaceholder",
   "Minimum purchase amount (0 = none)"
  ],
  [
   "coupon_valuePlaceholder",
   "Discount value (e.g. 20 for 20% or 50 for ¥50)"
  ],
  [
   "deposit_table_block",
   "Block"
  ],
  [
   "deposit_table_confirmations",
   "Confirmations"
  ],
  [
   "depositAddresses_address",
   "Address"
  ],
  [
   "depositAddresses_all",
   "All Status"
  ],
  [
   "depositAddresses_assignedAt",
   "Assigned At"
  ],
  [
   "depositAddresses_availablePool",
   "Available in Pool"
  ],
  [
   "depositAddresses_import",
   "Import Addresses"
  ],
  [
   "depositAddresses_importFailed",
   "Import failed"
  ],
  [
   "depositAddresses_importHint",
   "Use hdgen tool on an offline machine to generate deposit_addresses.bin, then upload it here."
  ],
  [
   "depositAddresses_importSuccess",
   "Imported {{imported}} addresses"
  ],
  [
   "depositAddresses_index",
   "Index"
  ],
  [
   "depositAddresses_network",
   "Network"
  ],
  [
   "depositAddresses_received",
   "Received USDT"
  ],
  [
   "depositAddresses_status",
   "Status"
  ],
  [
   "depositAddresses_total",
   "Total Addresses"
  ],
  [
   "depositAddresses_totalItems",
   "{{total}} addresses"
  ],
  [
   "depositAddresses_user",
   "User ID"
  ],
  [
   "marketplace_colActions",
   "Actions"
  ],
  [
   "marketplace_colFeatured",
   "Featured"
  ],
  [
   "marketplace_colPrice",
   "Price"
  ],
  [
   "marketplace_colPublisher",
   "Publisher"
  ],
  [
   "marketplace_colRevenue",
   "Revenue"
  ],
  [
   "marketplace_colSales",
   "Sales"
  ],
  [
   "marketplace_colStatus",
   "Status"
  ],
  [
   "marketplace_colTitle",
   "Title"
  ],
  [
   "marketplace_feature",
   "Feature"
  ],
  [
   "marketplace_featureDesc",
   "Set priority for featured placement. Higher = more prominent."
  ],
  [
   "marketplace_featureFailed",
   "Failed to feature strategy"
  ],
  [
   "marketplace_featureSuccess",
   "Strategy featured"
  ],
  [
   "marketplace_featureTitle",
   "Feature Strategy"
  ],
  [
   "marketplace_filterStatus",
   "All statuses"
  ],
  [
   "marketplace_loadFailed",
   "Failed to load strategies"
  ],
  [
   "marketplace_searchPlaceholder",
   "Search by title..."
  ],
  [
   "marketplace_unfeature",
   "Remove featured"
  ],
  [
   "marketplace_unfeatureFailed",
   "Failed to unfeature"
  ],
  [
   "marketplace_unfeatureSuccess",
   "Removed featured"
  ],
  [
   "refund_approve",
   "Approve & Execute"
  ],
  [
   "refund_approveTitle",
   "Approve Refund"
  ],
  [
   "refund_approved",
   "Refund approved and executed"
  ],
  [
   "refund_colActions",
   "Actions"
  ],
  [
   "refund_colAmount",
   "Amount"
  ],
  [
   "refund_colDate",
   "Date"
  ],
  [
   "refund_colReason",
   "Reason"
  ],
  [
   "refund_colStatus",
   "Status"
  ],
  [
   "refund_colStrategy",
   "Strategy"
  ],
  [
   "refund_colUser",
   "User"
  ],
  [
   "refund_filterStatus",
   "All statuses"
  ],
  [
   "refund_loadFailed",
   "Failed to load refund requests"
  ],
  [
   "refund_processFailed",
   "Failed to process refund"
  ],
  [
   "refund_reject",
   "Reject"
  ],
  [
   "refund_rejectTitle",
   "Reject Refund"
  ],
  [
   "refund_rejected",
   "Refund request rejected"
  ],
  [
   "refund_reviewNotePlaceholder",
   "Review note (optional for reject, recommended for approve)..."
  ],
  [
   "settings_editSetting",
   "Edit: {{setting}}"
  ],
  [
   "wallet_tabDepositAddresses",
   "Deposit Addresses"
  ],
  [
   "wallet_tabWallets",
   "User Wallets"
  ],
  [
   "wallet_totalUsers",
   "{{total}} users"
  ]
 ],
 "base": [
  [
   "common_total",
   "{{total}} total"
  ],
  [
   "monitoring_uptimeDays",
   "{{d}}d {{h}}h"
  ],
  [
   "monitoring_uptimeHours",
   "{{h}}h {{m}}m"
  ],
  [
   "monitoring_uptimeMinutes",
   "{{m}}m {{s}}s"
  ],
  [
   "monitoring_uptimeSeconds",
   "{{s}}s"
  ]
 ],
 "strategy_gen": [
  [
   "plan_hint",
   "You can discuss this plan, or say \"generate code\" to start implementing."
  ],
  [
   "plan_input_placeholder",
   "Share your thoughts..."
  ],
  [
   "exec_feedback_placeholder",
   "Try saying:\\n\"Tighten stop loss to 1%\"\\n\"Why is the Sharpe ratio so low? Help me improve it\"\\n\"Change to long-only, no short positions\""
  ]
 ]
}
```
