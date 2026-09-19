# 施工提示词 — I18N-MIXED-1（zh-cn locale 系统性未译/混杂修复）

## 立项背景

登记为 2 个中英混杂 key；2026-09-19 Devin CLI 设计实查发现真实债面：

- **Class A（i18n-check 可检）**：zh-cn 全 locale **216 条**值与 en 完全相同（未译）——`npx tsx scripts/i18n-check.ts --strict` 报 `[zh-cn] Untranslated (matches en)`。
- **Class B（i18n-check 不可检）**：`base_zh-cn.textproto` **~95 条**中英混杂（系统性模板：`失败 to X` / `删除 this Y` / `Top 策略 by Revenue` / `Please enter 基础 URL`）。
- **1 缺字段**：`strategy_gen_zh-cn` 缺 `exec_feedback_placeholder`。
- **门禁缺口**：`scripts/i18n-check.ts --strict` 存在但**未接入任何门禁**（pre-commit 无 i18n 步骤），strict 全量累计 1519 错误无人问津。
- zh-tw base 212 条未译 + ja/vi 同类缺口 → **另立 I18N-MIXED-2**（本任务范围外）。

## 设计 SSOT

- `scripts/i18n-translate-zh-cn.ts`：`translate()` 已有 exact-match 优先（`:226` `if (DICT[value]) return DICT[value]`）→ DICT 追加附录 A 整句条目即可，**勿改替换逻辑**。
- `main()` 当前硬编码 `base_en`/`base_zh-cn` → 泛化为遍历全部 `*_zh-cn.textproto`（按同名前缀配对 `*_en.textproto`）。
- Class B 混杂值**不等于 en 值**，脚本跳过 → 直接编辑 `base_zh-cn.textproto` 行值（附录 B 逐行替换）。
- 构建链：`npx tsx scripts/i18n-translate-zh-cn.ts`（改写 textproto）→ `npx tsx scripts/i18n-build.ts`（生成 `frontend/src/i18n/resources/zh-cn/*.ts`，git 跟踪产物须提交）。

## S1 — DICT 补全 + 脚本泛化

1. `scripts/i18n-translate-zh-cn.ts` DICT 末尾追加**附录 A 全部条目**（逐字，含占位符）。
2. `main()` 泛化：`fs.readdirSync(PROTO_DIR)` 找所有 `*_zh-cn.textproto`，对每个配对 `{prefix}_en.textproto`，现有比对逻辑套用到每对（抽 `processPair(enFile, zhCnFile)` helper）。
3. 附录 C 白名单值**不进** DICT（合法英文保留）。

## S2 — Class B 直接编辑

`proto/ant/v1/i18n/base_zh-cn.textproto` 按附录 B 逐行替换 value（保留 key 与行序）。

## S3 — 缺字段补齐

`strategy_gen_zh-cn.textproto` 在 en 同位置补：

```
exec_feedback_placeholder: '试试说：\n"把止损收紧到 1%"\n"为什么夏普比率这么低？帮我改进"\n"改成只做多，不做空"'
```

## S4 — 重生成 + 门禁接线

1. `npx tsx scripts/i18n-translate-zh-cn.ts` → 报告各文件翻译数。
2. `npx tsx scripts/i18n-build.ts` → 重生成资源。
3. `scripts/i18n-check.ts` 加 `--locale <loc>` 过滤旗标（只检指定 locale 的 Missing/Untranslated）。
4. `scripts/hooks/pre-commit` 末尾加 zh-cn strict 步骤：

```bash
if command -v npx >/dev/null 2>&1 && git diff --cached --name-only | grep -q 'i18n/'; then
  (cd "$(git rev-parse --show-toplevel)" && npx tsx scripts/i18n-check.ts --strict --locale zh-cn) || {
    echo "❌ i18n zh-cn strict 未过（未译/缺字段）。跑 npx tsx scripts/i18n-check.ts --strict --locale zh-cn 看明细。" >&2
    fail=1
  }
fi
```

## 验收（机检）

- `npx tsx scripts/i18n-check.ts --strict --locale zh-cn` → **0 errors**（216 untranslated + 1 missing 清零；残留白名单见附录 C——白名单条目会被报 Untranslated，故同时加 `--whitelist` 机制或将附录 C 值写入脚本内白名单数组，使 gate 对 zh-cn 干净归零）。
- 残留扫描：`for f in proto/ant/v1/i18n/*_zh-cn.textproto; do grep -hoE "^[a-z_0-9]+: '[^']* [^']*'$" $f | grep -vP '[\x{4e00}-\x{9fff}]'; done` → 仅剩附录 C 白名单。
- Class B 复查：`grep -P '失败 to |this strategy|this account|this model|by Revenue|by Subscribers|selected runs|Please enter|is now unlocked|pending backend'` base_zh-cn.textproto → 零命中。
- `cd frontend && npx tsc --noEmit` → 0 errors。
- **对抗证明**：M1 DICT 删任一条目 → strict 该 key RED；M2 附录 B 任一行回退混杂 → 残留扫描 RED；M3 删 `exec_feedback_placeholder` → strict Missing RED。restore → GREEN。

## 边界（不做）

- zh-tw/ja/vi 任何改动（I18N-MIXED-2 另债）。
- zh-cn 以外 locale 的 Missing field（vi/zh-tw 存量缺字段，另债）。
- prompt-internal 串（`{{text}}`/`{{intent}}`/`{{params}}`/`【...结论】`/FOMC-NFP Example/`Multiple experts`/`No code block found (```)`/`Current provider: {{provider}}...`）——AI prompt 模板非用户面，保留英文，列入白名单。
- 前端代码逻辑、key 名、map.json 不动。

## 附录 A — DICT 追加表（按 en 值排序，逐字进 DICT）

```
'10+ MT4/MT5 Brokers' 见 landing_brokers_title 整句
'14 days': '14 天',
'30 days': '30 天',
'7 days': '7 天',
'AI Assistant': 'AI 助手',
'AI Fix': 'AI 修复',
'AI Fix Preview': 'AI 修复预览',
'AI Generate': 'AI 生成',
'AI Tokens Remaining': 'AI Token 余额',
'AI fix failed': 'AI 修复失败',
'AI returned no code': 'AI 未返回代码',
'API Key': 'API 密钥',
'Actions': '操作',
'Address': '地址',
'Admin': '管理员',
'All': '全部',
'All statuses': '全部状态',
'All warnings acknowledged as intentional': '所有警告已确认为有意为之',
'Approve': '批准',
'Approve & Execute': '批准并执行',
'Approve Refund': '批准退款',
'Approve and publish?': '批准并发布？',
'Approve failed': '批准失败',
'Apply & Re-run': '应用并重跑',
'Assigned At': '分配时间',
'Auto Gate Evaluation': '自动 Gate 评估',
'Auto-fix failed': '自动修复失败',
'Available in Pool': '池中可用',
'Backtest': '回测',
'Bind your first MT4/MT5 account to start monitoring and trading.': '绑定您的第一个 MT4/MT5 账户，开始监控和交易。',
'Block': '区块',
'Bound Accounts': '已绑定账户',
'Bound At': '绑定时间',
'Bound MT Accounts': '已绑定 MT 账户',
'Broker': '经纪商',
'Circuit Open': '已熔断',
'Circuit Testing': '半开测试',
'Close Position': '平仓',
'Close your current position.': '平掉当前持仓。',
'Code': '代码',
'Code Editor': '代码编辑器',
'Compatible with 30+ MT4/MT5 Brokers': '兼容 30+ 家 MT4/MT5 经纪商',
'Confirmations': '确认数',
'Coupon code (e.g. SUMMER20)': '优惠券码（如 SUMMER20）',
'Coupon created': '优惠券已创建',
'Coupon disabled': '优惠券已禁用',
'Coverage': '覆盖率',
'Create Coupon': '创建优惠券',
'Created': '创建时间',
'Critical Issues': '严重问题',
'Custom days': '自定义天数',
'Date': '日期',
'Decay Score': '衰减评分',
'Deleted': '已删除',
'Deploy': '部署',
'Deploy Now': '立即部署',
'Diagnostic': '诊断',
'Discover': '发现',
'Discount value (e.g. 20 for 20% or 50 for $50)': '折扣值（如 20 表示 20% 或 50 表示 $50）',
'e.g. 12345678': '例：12345678',
'e.g. 123568': '例：123568',
'Enqueue': '入队',
'Enter your bound MT account credentials to verify your identity. Server and platform are detected automatically.': '输入已绑定的 MT 账户凭证以验证身份。服务器和平台将自动识别。',
'Entry:': '入场：',
'Equity Curve': '净值曲线',
'Exit:': '出场：',
'Expires': '过期时间',
'Expires at (ISO 8601, empty = never)': '过期时间（ISO 8601，留空=永不过期）',
'Feature': '精选',
'Featured': '精选',
'Fix applied — re-running backtest': '修复已应用——正在重跑回测',
'Fix applied but compile has warnings': '修复已应用但编译有警告',
'Fork': 'Fork',
'Fork failed': 'Fork 失败',
'Forked to new strategy': '已 Fork 为新策略',
'Gate': 'Gate',
'IC Markets, Pepperstone, XM, Exness, OANDA, FXTM, FBS, OctaFX, HotForex, Alpari, RoboForex and more. Connect your existing broker account in seconds.': 'IC Markets、Pepperstone、XM、Exness、OANDA、FXTM、FBS、OctaFX、HotForex、Alpari、RoboForex 等。数秒内即可连接您的现有经纪商账户。',
'Identity verified. Redirecting to password reset.': '身份已验证，正在跳转至密码重置。',
'If the email exists, a reset link has been sent.': '如果该邮箱存在，重置链接已发送。',
'Import Addresses': '导入地址',
'Import MQL': '导入 MQL',
'Import failed': '导入失败',
'Index': '序号',
'Indicators:': '指标：',
'Invalid or missing reset token.': '重置令牌无效或缺失。',
'Invariant Violation': '不变量违规',
'Login': '登录名',
'Lookahead Bias': '前视偏差',
'MT Password': 'MT 密码',
'MT Verify': 'MT 验证',
'MT credential verification failed.': 'MT 凭证验证失败。',
'MT trading password': 'MT 交易密码',
'Magic': '魔术号',
'Max Drawdown': '最大回撤',
'Max uses (0 = unlimited)': '最大使用次数（0 = 不限）',
'Min Purchase': '最低消费',
'Mine': '我的',
'Minimum purchase amount (0 = none)': '最低消费金额（0 = 无限制）',
'Name': '名称',
'Need strategy code and symbol. Select a strategy from the sidebar or run a backtest first.': '需要策略代码和品种。请从侧边栏选择策略或先运行回测。',
'New Password': '新密码',
'New Subscribers': '新增订阅者',
'New rating or comment received': '收到新评分或评论',
'New strategy published': '新策略已发布',
'No backtest runs yet': '暂无回测记录',
'No bound accounts yet. Schedule a strategy to auto-bind an account.': '暂无绑定账户。调度策略以自动绑定账户。',
'No description': '暂无描述',
'No models discovered. Check API key and base URL.': '未发现模型。请检查 API Key 和 Base URL。',
'No strategies found': '未找到策略',
'No strategies yet': '暂无策略',
'No strategy code to fix': '没有可修复的策略代码',
'No strategy loaded — describe what you want': '未加载策略——请描述您的需求',
'No strategy logic was recognized. The source code may be incomplete or use a different language.': '未能识别策略逻辑。源代码可能不完整或使用了其他语言。',
'No tunable parameters detected': '未检测到可调参数',
'Not Publishable': '不可发布',
'Not recommended for direct live: high risk or unreliable, optimize before trying.': '不建议直接实盘：风险高或不可靠，请先优化再尝试。',
'Notification Preferences': '通知偏好',
'Open in Workspace': '在工作区打开',
'Overview': '概览',
'P&L:': '盈亏：',
'Parameters': '参数',
'Password has been reset. Please log in with your new password.': '密码已重置，请使用新密码登录。',
'Passwords do not match.': '两次输入的密码不一致。',
'Platform Rev': '平台收入',
'Please confirm your password': '请确认密码',
'Please contact your administrator or support to reset your password.': '请联系管理员或客服重置密码。',
'Please fill required fields': '请填写必填项',
'Please save the strategy first to apply AI fixes': '请先保存策略以应用 AI 修复',
'Private': '私有',
'Profit Factor': '盈利因子',
'Public': '公开',
'Publish': '发布',
'Publish failed': '发布失败',
'Publishable': '可发布',
'Publisher': '发布者',
'Quality': '质量',
'Quality Hints': '质量提示',
'REJECT': '拒绝',
'Reason': '原因',
'Received USDT': '已收 USDT',
'Recent': '最新',
'Refund Rate': '退款率',
'Refund approved and executed': '退款已批准并执行',
'Refund request rejected': '退款请求已拒绝',
'Reject': '拒绝',
'Reject Refund': '拒绝退款',
'Reject failed': '拒绝失败',
'Reject this task?': '拒绝此任务？',
'Removed featured': '已移除精选',
'Reset Password': '重置密码',
'Return': '收益',
'Return Delta': '收益差值',
'Revenue': '收入',
'Review note (optional for reject, recommended for approve)...': '审核备注（拒绝时可选，批准时建议填写）...',
'Review the AI-generated code below. Apply to create a new version and re-run backtest.': '请检查下方 AI 生成的代码。应用后将创建新版本并重新运行回测。',
'Risk': '风险',
'Risk Warnings': '风险警告',
'Risk:': '风险：',
'Sales': '销量',
'Select or enter custom days': '选择或输入自定义天数',
'Send Reset Link': '发送重置链接',
'Server': '服务器',
'Set New Password': '设置新密码',
'Set priority for featured placement. Higher = more prominent.': '设置精选展示优先级，数值越大越显眼。',
'Sharpe Decline': '夏普衰减',
'Sharpe Ratio': '夏普比率',
'Statistical Hint': '统计提示',
'Structural Validation': '结构验证',
'Subscription expiring soon': '订阅即将到期',
'Symbols (comma-separated)': '品种（逗号分隔）',
'Task approved and published': '任务已批准并发布',
'Task rejected': '任务已拒绝',
'Timeframes (comma-separated)': '周期（逗号分隔）',
'Title': '标题',
'Top Providers by Revenue': '按收入排行提供商',
'Trade Statistics': '交易统计',
'Trial Period': '试用期',
'Trigger': '触发',
'Trigger Batch': '触发批次',
'Trigger failed': '触发失败',
'Tuning failed': '调优失败',
'Type': '类型',
'Unbind': '解绑',
'Unlimited MT accounts': '无限 MT 账户',
'Unpublish': '下架',
'Unpublish failed': '下架失败',
'Unpublished': '已下架',
'Usage': '使用情况',
'Use Count': '使用次数',
'Use hdgen tool on an offline machine to generate deposit_addresses.bin, then upload it here.': '请在离线机器上使用 hdgen 工具生成 deposit_addresses.bin，然后在此处上传。',
'Validation failed': '验证失败',
'Value': '数值',
'Verify & Reset Password': '验证并重置密码',
'View all': '查看全部',
'View all supported brokers': '查看全部支持的经纪商',
'Visibility': '可见性',
'Win Rate': '胜率',
'Win Rate Decline': '胜率衰减',
'Workspace': '工作区',
'Your strategy is ready to deploy.': '您的策略已就绪，可以部署。',
'compatible': '兼容',
'selected': '已选',
'trades': '笔',
'unsupported': '不支持',
```

## 附录 B — Class B 直接替换表（base_zh-cn.textproto 行号 → 新值）

```
404  errors_ai_probe_ok_no_models: '正常（未返回 models）' → '正常（未返回模型）'
705  errors_ai_insufficient_balance_title: 'Insufficient 余额' → '余额不足'
710  onboarding_step1_action: 'Bind 账户' → '绑定账户'
726  strategy_chat_execution_plan: 'Execution 方案' → '执行方案'
740  import_analysis_good_coverage_desc: '策略 main logic recognized. Safe to import. Check parameter list before use.' → '策略主要逻辑已识别，可安全导入。使用前请检查参数列表。'
752  strategy_templates_save_current: '保存 Current 策略' → '保存当前策略'
754  strategy_templates_chat_edit: 'Chat 编辑' → '对话编辑'
757  strategy_templates_confirm_delete: '删除 this strategy?' → '删除此策略？'
767  agent_profile_risk: 'Risk 管理' → '风险管理'
780  accounts_messages_share_link_failed: '失败 to create share link' → '创建分享链接失败'
781  admin_ai_gateway_errors_load_providers: '失败 to load providers' → '加载提供商失败'
782  admin_ai_gateway_add_provider_pending: '添加 provider feature pending backend support' → '添加提供商功能开发中（待后端支持）'
784  admin_ai_gateway_errors_load_models: '失败 to load models' → '加载模型失败'
802  admin_ai_gateway_base_url_required: 'Please enter 基础 URL' → '请输入 Base URL'
811  admin_ai_gateway_price_input: 'Input 价格 ($/1M)' → '输入价格（$/1M）'
812  admin_ai_gateway_price_output: 'Output 价格 ($/1M)' → '输出价格（$/1M）'
813  admin_ai_gateway_confirm_delete_model: '删除 this model?' → '删除此模型？'
815  admin_account_errors_load_failed: '失败 to load accounts' → '加载账户失败'
816  admin_account_frozen: '账户 frozen' → '账户已冻结'
818  admin_account_unfrozen: '账户 unfrozen' → '账户已解冻'
834  admin_account_search_placeholder: '搜索 accounts' → '搜索账户'
841  admin_account_audit_logs: 'Audit 日志' → '审计日志'
845  admin_settings_save_failed: '保存 failed' → '保存失败'
847  admin_settings_delete_failed: '删除 failed' → '删除失败'
854  admin_settings_confirm_delete: '确认 delete?' → '确认删除？'
855  admin_settings_title: 'Agent 管理 设置' → 'Agent 管理设置'
860  admin_settings_permission_add_rule: '添加 rule: create setting with key ' → '添加规则：创建设置，键为 '
899  admin_billing_filter_by_type: '筛选 by type' → '按类型筛选'
905  admin_dashboard_errors_load_failed: '失败 to load dashboard data' → '加载仪表盘数据失败'
910  admin_dashboard_market_strategies: 'Market 策略' → '市场策略'
925  admin_dashboard_validate_total: 'Validate 总计' → '验证总计'
934  admin_deposit_approved: '充值 approved and wallet credited.' → '充值已批准，钱包已入账。'
935  admin_deposit_approve_failed: '失败 to approve deposit.' → '批准充值失败。'
936  admin_deposit_rejected: '充值 rejected.' → '充值已拒绝。'
937  admin_deposit_reject_failed: '失败 to reject deposit.' → '拒绝充值失败。'
953  admin_deposit_approve_title: 'Approve 充值' → '批准充值'
954  admin_deposit_reject_title: 'Reject 充值' → '拒绝充值'
963  monitoring_stream_error: 'Stream 错误' → '数据流错误'
994  admin_logs_errors_load_failed: '失败 to load logs' → '加载日志失败'
1000 admin_logs_filter_action: '筛选 by action' → '按操作筛选'
1014 admin_wallet_messages_adjust_success: '余额 adjusted successfully' → '余额调整成功'
1046 sre_breakers_description: '策略 breaker status overview — auto-detects abnormal losses and trips' → '策略断路器状态总览——自动检测异常亏损并熔断'
1056 sre_canary_confirm_delete: '删除 this canary config?' → '删除此金丝雀配置？'
1065 sre_kill_switch_engaged: '熔断开关 engaged — all trading stopped' → '熔断开关已启用——所有交易已停止'
1066 sre_kill_switch_disarmed: '熔断开关 disarmed — trading normal' → '熔断开关已解除——交易正常'
1071 sre_kill_switch_undo: 'Undo 熔断开关' → '撤销熔断开关'
1072 sre_kill_switch_disengage: 'Disengage 熔断开关' → '解除熔断开关'
1081 admin_user_management_messages_load_users_failed: '失败 to load users' → '加载用户失败'
1091 marketplace_author_my_strategies: 'My Published 策略' → '我发布的策略'
1092 marketplace_author_publish_new: 'Publish New 策略' → '发布新策略'
1095 marketplace_author_go_to_library: 'Go to 策略 Library' → '前往策略库'
1106 marketplace_backtest_protected: '策略 code is protected. Backtest runs on our servers.' → '策略代码受保护，回测在我们的服务器上运行。'
1111 marketplace_card_your_strategy: 'Your 策略' → '你的策略'
1182 marketplace_messages_published: '策略 published to marketplace!' → '策略已发布到市场！'
1183 marketplace_messages_publish_failed: '失败 to publish strategy' → '发布策略失败'
1186 marketplace_publish_title_placeholder: 'e.g. Golden Cross 策略' → '例：黄金交叉策略'
1255 strategy_validate_passed: 'Validation passed — 保存 is now unlocked.' → '验证通过——保存已解锁。'
1342 strategy_backtest_diagnostic_apply_failed: '失败 to apply fix' → '应用修复失败'
1347 strategy_backtest_cancel_failed: '取消 failed' → '取消失败'
1349 notifications_prefs_save_failed: '失败 to save preferences' → '保存偏好失败'
1351 notifications_prefs_price_change: '策略 price changed' → '策略价格变更'
1353 notifications_prefs_performance: '策略 performance anomaly' → '策略绩效异常'
1356 strategy_ai_chat_code_loaded: '策略 code in context' → '策略代码已载入上下文'
1367 admin_ai_gateway_discover_failed: '失败 to discover models' → '发现模型失败'
1370 admin_autogen_load_failed: '失败 to load tasks' → '加载任务失败'
1388 admin_autogen_all_status: 'All 状态' → '全部状态'
1395 admin_coupon_load_failed: '失败 to load coupons' → '加载优惠券失败'
1398 admin_coupon_create_failed: '失败 to create coupon' → '创建优惠券失败'
1400 admin_coupon_disable_failed: '失败 to disable coupon' → '禁用优惠券失败'
1426 admin_deposit_addresses_all: 'All 状态' → '全部状态'
1441 admin_analytics_new_strategies: 'New 策略' → '新策略'
1442 admin_analytics_top_by_revenue: 'Top 策略 by Revenue' → '策略收入排行'
1443 admin_analytics_top_by_subs: 'Top 策略 by Subscribers' → '策略订阅排行'
1445 admin_analytics_top_providers_strat: 'Top Providers by 策略' → '策略提供商排行'
1446 admin_marketplace_load_failed: '失败 to load strategies' → '加载策略失败'
1447 admin_marketplace_feature_success: '策略 featured' → '策略已设为精选'
1448 admin_marketplace_feature_failed: '失败 to feature strategy' → '设为精选失败'
1450 admin_marketplace_unfeature_failed: '失败 to unfeature' → '取消精选失败'
1460 admin_marketplace_unfeature: '移除 featured' → '移除精选'
1462 admin_marketplace_search_placeholder: '搜索 by title...' → '按标题搜索...'
1463 admin_marketplace_feature_title: 'Feature 策略' → '精选策略'
1465 admin_refund_load_failed: '失败 to load refund requests' → '加载退款请求失败'
1468 admin_refund_process_failed: '失败 to process refund' → '处理退款失败'
1504 auth_reset_password_failed: '失败 to reset password.' → '重置密码失败。'
1514 marketplace_live_load_error: '失败 to load live performance data' → '加载实盘绩效数据失败'
1547 strategy_templates_detail_not_found: '策略 not found' → '未找到策略'
1564 strategy_templates_actions_create: 'New 策略' → '新建策略'
1566 strategy_templates_gallery_search_placeholder: '搜索 strategies...' → '搜索策略...'
1574 strategy_templates_messages_fetch_template_list_failed: '失败 to load strategies' → '加载策略列表失败'
1596 strategy_templates_gallery_delete_failed: '删除 failed' → '删除失败'
1601 strategy_templates_delete_confirm: '删除 this strategy?' → '删除此策略？'
1607 strategy_workspace_sidebar_batch_delete_runs_confirm: '删除 selected runs?' → '删除所选回测？'
1609 strategy_workspace_sidebar_delete_run_confirm: '删除 this backtest run?' → '删除此回测？'
1612 strategy_workspace_sidebar_batch_delete_confirm: '删除 selected strategies?' → '删除所选策略？'
1613 strategy_workspace_sidebar_delete_strategy_confirm: '删除 this strategy?' → '删除此策略？'
1617 strategy_tuning_no_dims_hint: '启用 at least one parameter dimension below.' → '请在下方启用至少一个参数维度。'
1621 strategy_workspace_sidebar_backtest_history: 'Backtest 历史' → '回测历史'
1622 strategy_workspace_sidebar_new_strategy: 'New 策略' → '新建策略'
1631 strategy_workspace_tour_save_desc: '保存 your strategy as a template, publish to marketplace, or deploy to a live schedule.' → '将策略保存为模板、发布到市场或部署到实盘调度。'
1639 strategy_templates_load_failed: '失败 to load templates' → '加载模板失败'
1640 strategy_templates_load_one_failed: '失败 to load template' → '加载模板失败'
1642 subscription_unbind_success: '账户 unbound successfully.' → '账户解绑成功。'
1643 subscription_unbind_failed: '失败 to unbind account.' → '解绑账户失败。'
1650 subscription_unbind_confirm: 'Unbind this account? 活跃 schedules on it will be stopped.' → '解绑此账户？其上的活跃调度将被停止。'
```

## 附录 C — 白名单（合法英文保留，进 i18n-check 白名单数组）

```
值级白名单（DICT 不覆盖、check 不报）：
  'Tiếng Việt'（语言名）/ 'Anthropic Claude'（品牌）/ 'DeepSeek Chat'（模型示例名）
  'ATR %' / 'VaR 95%'（指标符号）/ 'Goroutines'（Go 运行时术语）
  'account-1, account-2'（ID 格式示例）/ 'Base URL'（配置术语）
  '{symbol} · {timeframe}' / '{{period}} · {{metric}}：{{value}}' / '{{symbol}} {{timeframe}} {{name}}'（模板占位）
AI prompt 内部串（非用户面，模式匹配豁免——含 {{text}}/{{intent}}/{{params}} 占位符或【结论】模板头）：
  'Macro events (user-provided):\n{{text}}' / 'Parameters (defs+current values...):\n{{params}}'
  'User expectation (natural language):\n{{intent}}' / 'User strategy goal (natural language):\n{{intent}}'
  '【Market condition/style conclusion】\n{{text}}' / '【Risk control conclusion】\n{{text}}' / '【Signal design conclusion】\n{{text}}'
  'Example:\n2024-01-03 21:15 FOMC minutes\n2024-01-05 20:30 NFP' / 'Multiple experts\\' / 'No code block found (```...```)'
  'Current provider: {{provider}}. Go to the provider\\'（截断转义，prompt-adjacent 保留待业主确认）
单词技术术语（zh-cn 惯例保留）：'AI Token' / 'ID' / 'SL' / 'TP' / 'OK' / 'PASS' 若出现
```

## 门禁（完工前必跑）

```
npx tsx scripts/i18n-translate-zh-cn.ts        # S1 产出
npx tsx scripts/i18n-build.ts                 # S4 重生成
npx tsx scripts/i18n-check.ts --strict --locale zh-cn   # 0 errors
附录残留扫描两条                              # 零命中/仅白名单
cd frontend && npx tsc --noEmit               # 0 errors
git diff --check
```

勿部署，停手等 Devin CLI 复审。
