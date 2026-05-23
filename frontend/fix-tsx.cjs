const fs = require('fs');
const path = require('path');

const filesToFix = [
  'src/pages/accounts/AccountDetail.tsx',
  'src/pages/accounts/AccountList.tsx',
  'src/pages/market/Quotes.tsx',
  'src/pages/strategy/StrategySchedulePage.tsx',
  'src/pages/strategy/StrategyTemplatePage.tsx',
  'src/pages/trading/Trading.tsx',
  'src/components/strategy/CodeEditor.tsx',
];

const replacements = [
  // Position properties (variable name can be any)
  [/\b(\w+)\.open_price\b/g, '$1.openPrice'],
  [/\b(\w+)\.current_price\b/g, '$1.currentPrice'],
  [/\b(\w+)\.open_time\b/g, '$1.openTime'],
  [/\b(\w+)\.close_price\b/g, '$1.closePrice'],
  [/\b(\w+)\.stop_loss\b/g, '$1.stopLoss'],
  [/\b(\w+)\.take_profit\b/g, '$1.takeProfit'],
  [/\b(\w+)\.close_time\b/g, '$1.closeTime'],
  [/\b(\w+)\.swap\b/g, '$1.swap'],
  [/\b(\w+)\.commission\b/g, '$1.commission'],
  
  // Account properties (variable name can be any)
  [/\b(\w+)\.is_disabled\b/g, '$1.isDisabled'],
  [/\b(\w+)\.stream_status\b/g, '$1.streamStatus'],
  [/\b(\w+)\.account_status\b/g, '$1.accountStatus'],
  [/\b(\w+)\.mt_type\b/g, '$1.mtType'],
  [/\b(\w+)\.free_margin\b/g, '$1.freeMargin'],
  [/\b(\w+)\.margin_level\b/g, '$1.marginLevel'],
  [/\b(\w+)\.last_connected_at\b/g, '$1.lastConnectedAt'],
  [/\b(\w+)\.last_checked_at\b/g, '$1.lastCheckedAt'],
  
  // AccountInfo properties
  [/\baccountInfo\.free_margin\b/g, 'accountInfo.freeMargin'],
  [/\baccountInfo\.margin_level\b/g, 'accountInfo.marginLevel'],
  [/\baccountInfo\.balance\b/g, 'accountInfo.balance'],
  [/\baccountInfo\.equity\b/g, 'accountInfo.equity'],
  [/\baccountInfo\.margin\b/g, 'accountInfo.margin'],
  [/\baccountInfo\.profit\b/g, 'accountInfo.profit'],
  
  // StrategyTemplate properties
  [/\bis_public\b/g, 'isPublic'],
  [/\buse_count\b/g, 'useCount'],
  [/\bcreated_at\b/g, 'createdAt'],
  [/\bupdated_at\b/g, 'updatedAt'],
  [/\bma_fast\b/g, 'maFast'],
  [/\bma_slow\b/g, 'maSlow'],
  [/\bavg_gain\b/g, 'avgGain'],
  [/\bavg_loss\b/g, 'avgLoss'],
  [/\bmax_drawdown_percent\b/g, 'maxDrawdownPercent'],
  [/\bsharpe_ratio\b/g, 'sharpeRatio'],
  [/\bsortino_ratio\b/g, 'sortinoRatio'],
  [/\bcalmar_ratio\b/g, 'calmarRatio'],
  [/\bvolatility\b/g, 'volatility'],
  [/\baverage_daily_return\b/g, 'averageDailyReturn'],
  [/\btotal_trades\b/g, 'totalTrades'],
  [/\bwin_rate\b/g, 'winRate'],
  [/\bprofit_factor\b/g, 'profitFactor'],
  [/\baverage_profit\b/g, 'averageProfit'],
  [/\baverage_loss\b/g, 'averageLoss'],
  [/\blargest_win\b/g, 'largestWin'],
  [/\blargest_loss\b/g, 'largestLoss'],
  [/\bmax_consecutive_wins\b/g, 'maxConsecutiveWins'],
  [/\bmax_consecutive_losses\b/g, 'maxConsecutiveLosses'],
  [/\baverage_holding_time\b/g, 'averageHoldingTime'],
  [/\bnet_profit\b/g, 'netProfit'],
  [/\btotal_deposit\b/g, 'totalDeposit'],
  [/\btotal_withdrawal\b/g, 'totalWithdrawal'],
  [/\bnet_deposit\b/g, 'netDeposit'],
  [/\btotal_profit\b/g, 'totalProfit'],
  [/\btotal_loss\b/g, 'totalLoss'],
  [/\bprofit_loss_ratio\b/g, 'profitLossRatio'],
  [/\bexpected_value\b/g, 'expectedValue'],
  [/\brecovery_factor\b/g, 'recoveryFactor'],
  [/\brisk_reward_ratio\b/g, 'riskRewardRatio'],
  [/\bsharpe_ratio\b/g, 'sharpeRatio'],
  [/\bsortino_ratio\b/g, 'sortinoRatio'],
  [/\bcalmar_ratio\b/g, 'calmarRatio'],
  [/\bmax_drawdown_percent\b/g, 'maxDrawdownPercent'],
  [/\bvolatility\b/g, 'volatility'],
  [/\baverage_daily_return\b/g, 'averageDailyReturn'],
  [/\btotal_trades\b/g, 'totalTrades'],
  [/\bwin_rate\b/g, 'winRate'],
  [/\bprofit_factor\b/g, 'profitFactor'],
  [/\baverage_profit\b/g, 'averageProfit'],
  [/\baverage_loss\b/g, 'averageLoss'],
  [/\blargest_win\b/g, 'largestWin'],
  [/\blargest_loss\b/g, 'largestLoss'],
  [/\bmax_consecutive_wins\b/g, 'maxConsecutiveWins'],
  [/\bmax_consecutive_losses\b/g, 'maxConsecutiveLosses'],
  [/\baverage_holding_time\b/g, 'averageHoldingTime'],
  [/\bnet_profit\b/g, 'netProfit'],
  [/\btotal_deposit\b/g, 'totalDeposit'],
  [/\btotal_withdrawal\b/g, 'totalWithdrawal'],
  [/\bnet_deposit\b/g, 'netDeposit'],
  
  // dataIndex properties (string literals)
  [/'open_price'/g, "'openPrice'"],
  [/'current_price'/g, "'currentPrice'"],
  [/'open_time'/g, "'openTime'"],
  [/'close_price'/g, "'closePrice'"],
  [/'stop_loss'/g, "'stopLoss'"],
  [/'take_profit'/g, "'takeProfit'"],
  [/'close_time'/g, "'closeTime'"],
  [/'free_margin'/g, "'freeMargin'"],
  [/'margin_level'/g, "'marginLevel'"],
  [/'mt_type'/g, "'mtType'"],
  [/'is_disabled'/g, "'isDisabled'"],
  [/'stream_status'/g, "'streamStatus'"],
  [/'account_status'/g, "'accountStatus'"],
];

function fixFile(filePath) {
  const fullPath = path.join(__dirname, filePath);
  if (!fs.existsSync(fullPath)) {
    console.log(`❌ File not found: ${filePath}`);
    return;
  }
  
  let content = fs.readFileSync(fullPath, 'utf-8');
  let modified = false;
  
  for (const [pattern, replacement] of replacements) {
    if (pattern.test(content)) {
      content = content.replace(pattern, replacement);
      modified = true;
    }
  }
  
  if (modified) {
    fs.writeFileSync(fullPath, content, 'utf-8');
    console.log(`✅ Fixed: ${filePath}`);
  } else {
    console.log(`⏭️  Skipped: ${filePath} (no changes needed)`);
  }
}

console.log('🔧 Fixing TypeScript files...\n');
filesToFix.forEach(file => fixFile(file));
console.log('\n✅ Done!');
