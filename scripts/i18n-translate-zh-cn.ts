#!/usr/bin/env tsx
/**
 * Translates English placeholder entries in zh-cn textproto by comparing
 * with en textproto. Only translates entries that are identical (still English).
 * Uses a dictionary of known UI translations.
 *
 * Usage: npx tsx scripts/i18n-translate-zh-cn.ts
 */
import * as fs from 'fs';
import * as path from 'path';

const PROTO_DIR = '/opt/ant/proto/ant/v1/i18n';

// Parse textproto into ordered list of {key, value} pairs
function parseTextproto(content: string): { key: string; value: string; line: string }[] {
  const entries: { key: string; value: string; line: string }[] = [];
  const lines = content.split('\n');
  for (const line of lines) {
    const m = line.match(/^(\w+):\s*'(.*)'\s*$/);
    if (m) {
      entries.push({ key: m[1], value: m[2], line });
    }
  }
  return entries;
}

// Translation dictionary for common UI terms
const DICT: Record<string, string> = {
  // Dashboard
  'Monthly Revenue': '月度收入',
  'Total Revenue': '总收入',
  'Active Subscriptions': '活跃订阅',
  'Active Subs': '活跃订阅',
  'Transactions': '交易记录',
  'Marketplace Revenue': '市场收入',
  'Subscription Revenue': '订阅收入',
  'Total Users': '总用户数',
  'New Users': '新用户',
  'Active Users': '活跃用户',
  'Total Accounts': '总账户数',
  'Connected Accounts': '已连接账户',
  'Total Strategies': '总策略数',
  'Active Strategies': '活跃策略',
  'Total Deposits': '总充值',
  'Pending Deposits': '待处理充值',
  'Total Withdrawals': '总提现',
  // Trading
  'Order Send Success': '下单成功',
  'Order Send Failed': '下单失败',
  'Order Close Success': '平仓成功',
  'Order Close Failed': '平仓失败',
  'Validate Reject': '验证拒绝',
  'Validate Error': '验证错误',
  'Validate Success': '验证成功',
  // Table headers
  'User': '用户',
  'Plan': '方案',
  'Status': '状态',
  'Cycle': '周期',
  'Price': '价格',
  'Auto Renew': '自动续费',
  'Period Start': '周期开始',
  'Period End': '周期结束',
  'Amount': '金额',
  'USDT Amount': 'USDT 金额',
  'USD Credit': 'USD 到账',
  'Tx Hash': '交易哈希',
  'Review Note': '审核备注',
  'Time': '时间',
  'Action Type': '操作类型',
  'Target': '目标',
  'Module': '模块',
  'Wallet No.': '钱包号',
  'Email': '邮箱',
  'Nickname': '昵称',
  'Balance': '余额',
  'Frozen': '冻结',
  'Currency': '币种',
  'Account': '账户',
  'Symbol': '品种',
  'TF': '周期',
  'Mode': '模式',
  'Signals': '信号',
  'Errors': '错误',
  'Strategy ID': '策略 ID',
  'Version Tag': '版本标签',
  'Canary Accounts': '金丝雀账户',
  'Start At': '开始时间',
  'Days': '天数',
  'State': '状态',
  'Total P&L': '总盈亏',
  'Loss %': '亏损率',
  'Trades': '交易数',
  'Run ID': '运行 ID',
  'Provider': '提供商',
  'Display Name': '显示名称',
  'Base URL': '基础 URL',
  'API Key': 'API 密钥',
  'Models': '模型',
  'Configured': '已配置',
  'Not Configured': '未配置',
  // Card titles
  'User List': '用户列表',
  'Wallet Management': '钱包管理',
  'Billing Management': '计费管理',
  'Deposit Management': '充值管理',
  'Operation Logs': '操作日志',
  'System Monitoring': '系统监控',
  'AI Gateway Management': 'AI 网关管理',
  'Account Management': '账户管理',
  'Trading Monitor': '交易监控',
  'Strategy Management': '策略管理',
  'Share Management': '分享管理',
  'System Config': '系统配置',
  'Jurisdiction Gate': '管辖权管理',
  'Plan Revenue Details': '方案收入明细',
  'DB Connection Pool': '数据库连接池',
  'Total': '总计',
  'Idle': '空闲',
  'Acquired': '已获取',
  'Disk Usage': '磁盘使用',
  'SSE Connections': 'SSE 连接数',
  'Service Health': '服务健康',
  'MD Gateway': '行情网关',
  'Spill Files': '溢出文件',
  'Dropped Bars': '丢弃 K 线',
  'Dropped Signals': '丢弃信号',
  'Consumer Lag': '消费者延迟',
  'Stale Accounts': '过期账户',
  'Dead Accounts': '死账户',
  'Parse Errors': '解析错误',
  // Buttons / actions
  'Add Provider': '添加提供商',
  'Add Model': '添加模型',
  'Edit Provider': '编辑提供商',
  'Edit Model': '编辑模型',
  'New Deposit': '新建充值',
  'New Canary': '新建金丝雀',
  'Engage': '启用',
  'Disarm': '解除',
  'Refresh': '刷新',
  'Filter by plan': '按方案筛选',
  'Filter by status': '按状态筛选',
  'Filter by module': '按模块筛选',
  'All Statuses': '全部状态',
  // Tabs
  'Active Runs': '活跃运行',
  'Run History': '运行历史',
  'Schedules': '调度',
  // Plan names
  'Free': '免费',
  'Pro': '专业版',
  'Enterprise': '企业版',
  // Wallet
  'USDT Deposit': 'USDT 充值',
  'Network': '网络',
  'Receiving Address': '收款地址',
  'Exchange Rate': '汇率',
  'Copy': '复制',
  'Address copied to clipboard': '地址已复制到剪贴板',
  'Deposit': '充值',
  'Withdraw': '提现',
  'Wallet Transactions': '钱包交易',
  'Subscriptions': '订阅',
  // Live Strategy
  'Live Strategy Monitor': '实盘策略监控',
  'Active': '活跃',
  'History': '历史',
  'Monitor': '监控',
  'Strategy': '策略',
  // SRE
  'Kill Switch': '熔断开关',
  'Breakers': '断路器',
  'Canary': '金丝雀',
  'Engage Kill Switch': '启用熔断开关',
  'No canary configs': '无金丝雀配置',
  'Strategy Breakers': '策略断路器',
  'Strategy breakers monitor per-strategy drawdown': '策略断路器监控每策略回撤',
  // Misc
  'Disconnected': '已断开',
  'SSE Connected': 'SSE 已连接',
  'Manage AI providers': '管理 AI 提供商',
  'Manage AI providers, models, and pricing. Users select from available models, billed by token from wallet.': '管理 AI 提供商、模型和定价。用户从可用模型中选择，按 token 从钱包扣费。',
  'Agent Settings': 'Agent 设置',
  'Monitoring & Alerts': '监控与告警',
  'AI Gateway': 'AI 网关',
  'SRE Controls': 'SRE 控制',
  'Billing': '计费',
  'Deposits': '充值',
  'Strategies': '策略',
  'Share Analytics': '分享分析',
  // Common
  'No data': '暂无数据',
  'No results': '无结果',
  'Loading...': '加载中...',
  'Success': '成功',
  'Failed': '失败',
  'Error': '错误',
  'Warning': '警告',
  'Pending': '待处理',
  'Save': '保存',
  'Cancel': '取消',
  'Delete': '删除',
  'Edit': '编辑',
  'Add': '添加',
  'Remove': '移除',
  'Close': '关闭',
  'Submit': '提交',
  'Confirm': '确认',
  'Search': '搜索',
  'Filter': '筛选',
  'Enable': '启用',
  'Disable': '禁用',
  'Online': '在线',
  'Offline': '离线',
  'Settings': '设置',
  'Management': '管理',
  'Logs': '日志',
  'Config': '配置',
  'System': '系统',
  'Offline': '离线',
  'Settings': '设置',
  'Management': '管理',
  'Logs': '日志',
  'Config': '配置',
  'System': '系统',
  'Gateway': '网关',
  // ── I18N-MIXED-1 附录 A：zh-cn 未译整句/词条（2026-09-19 设计实查）──
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
};

function translate(value: string): string | null {
  // Exact match
  if (DICT[value]) return DICT[value];

  // Try partial replacements for composite strings
  let result = value;
  let changed = false;
  for (const [en, zh] of Object.entries(DICT)) {
    if (en.length < 3) continue; // Skip short words to avoid false replacements
    const re = new RegExp('\\b' + en.replace(/[.*+?^${}()|[\]\\]/g, '\\$&') + '\\b', 'g');
    if (re.test(result)) {
      result = result.replace(re, zh);
      changed = true;
    }
  }

  return changed ? result : null;
}

// processPair translates English-placeholder entries (zh value identical to
// en) in one zh-cn textproto, using its same-prefix en file as reference.
// I18N-MIXED-1 S1: generalized from base-only to all *_zh-cn.textproto files.
function processPair(enFile: string, zhCnFile: string): number {
  const enEntries = parseTextproto(fs.readFileSync(enFile, 'utf-8'));
  const zhCnContent = fs.readFileSync(zhCnFile, 'utf-8');

  // Find entries where zh-cn value is identical to en value (still English placeholder)
  let translated = 0;
  const newLines: string[] = [];

  for (const line of zhCnContent.split('\n')) {
    const m = line.match(/^(\w+):\s*'(.*)'\s*$/);
    if (m) {
      const key = m[1];
      const zhValue = m[2];
      const enEntry = enEntries.find(e => e.key === key);

      if (enEntry && zhValue === enEntry.value) {
        // This is still an English placeholder
        const translatedValue = translate(zhValue);
        if (translatedValue) {
          newLines.push(`${key}: '${translatedValue.replace(/'/g, "\\'")}'`);
          translated++;
        } else {
          newLines.push(line);
        }
      } else {
        newLines.push(line);
      }
    } else {
      newLines.push(line);
    }
  }

  fs.writeFileSync(zhCnFile, newLines.join('\n'));
  return translated;
}

function main() {
  // I18N-MIXED-1 S1: iterate all *_zh-cn.textproto files, pairing each with
  // its same-prefix *_en.textproto.
  const zhCnFiles = fs.readdirSync(PROTO_DIR)
    .filter(f => f.endsWith('_zh-cn.textproto'))
    .sort();

  let total = 0;
  for (const zhCnFile of zhCnFiles) {
    const prefix = zhCnFile.replace(/_zh-cn\.textproto$/, '');
    const enFile = path.join(PROTO_DIR, `${prefix}_en.textproto`);
    if (!fs.existsSync(enFile)) {
      console.log(`SKIP ${zhCnFile}: no matching ${prefix}_en.textproto`);
      continue;
    }
    const zhCnPath = path.join(PROTO_DIR, zhCnFile);
    const translated = processPair(enFile, zhCnPath);
    total += translated;
    console.log(`Translated ${translated} entries in ${zhCnFile}`);
  }
  console.log(`Total translated: ${total}`);
}

main();
