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
  'Gateway': '网关',
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

function main() {
  const enFile = path.join(PROTO_DIR, 'base_en.textproto');
  const zhCnFile = path.join(PROTO_DIR, 'base_zh-cn.textproto');

  const enEntries = parseTextproto(fs.readFileSync(enFile, 'utf-8'));
  const zhCnContent = fs.readFileSync(zhCnFile, 'utf-8');
  const zhCnEntries = parseTextproto(zhCnContent);

  // Build a map of zh-cn entries by key
  const zhCnMap = new Map<string, string>();
  for (const e of zhCnEntries) {
    zhCnMap.set(e.key, e.value);
  }

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
  console.log(`Translated ${translated} entries in base_zh-cn.textproto`);
}

main();
