// Auto-generated from proto/ant/v1/i18n/accounts_en.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const Accounts = {
  "accounts": {
    "analytics": {
      "monthlyAnalysis": {
        "bonus": {
          "chartHoldingTitle": "{{month}}'s average holding time.",
          "chartPopularTitle": "{{month}}'s currency popularity.",
          "chartRiskTitle": "Bonus: The risk ratio is {{month}}.",
          "emptyCharts": "No trades in this month",
          "legendBulls": "Bulls",
          "legendShortTerm": "Short-term",
          "popularityShare": "Lot volume share",
          "sliceOther": "Other"
        },
        "metrics": {
          "change": "Change",
          "lots": "Lots",
          "pips": "Pips",
          "profit": "Profit"
        },
        "chartMainTitle": "Monthly returns ({{metric}})",
        "focusedValue": "{{period}} · {{metric}}: {{value}}",
        "title": "Monthly analysis"
      },
      "monthlyDetail": {
        "fields": {
          "averageHours": "Avg",
          "bestTrade": "Best Trade",
          "maxHours": "Max",
          "medianHours": "Median",
          "minHours": "Min",
          "netReturn": "Net Return",
          "profitFactor": "Profit Factor",
          "totalTrades": "Total Trades",
          "winRate": "Win Rate",
          "worstTrade": "Worst Trade"
        },
        "holdingTitle": "Holding Time",
        "long": "Long",
        "metricsTitle": "Monthly Metrics",
        "popularityTitle": "Currency Popularity",
        "riskRewardTitle": "Reward:Risk Ratio",
        "short": "Short",
        "symbolPnLTitle": "Symbol P&L"
      },
      "advancedTabs": {
        "daily": "Daily",
        "hourly": "Hourly"
      },
      "chartPeriod": {
        "all": "All",
        "day": "Today",
        "month": "This month",
        "week": "This week",
        "year": "This year"
      },
      "chartSeries": {
        "balance": "Balance",
        "equity": "Equity",
        "profit": "Profit",
        "tradeCount": "Trades"
      },
      "chartType": {
        "balance": "Balance",
        "equity": "Equity",
        "profit": "Profit"
      },
      "empty": {
        "dailyPnL": "No daily P/L data",
        "equityCurve": "No equity curve data",
        "hourly": "No time-of-day analysis data",
        "monthlyProfit": "No monthly profit data",
        "symbolDistribution": "No symbol distribution data"
      },
      "stats": {
        "avgDailyReturn": "Avg daily return",
        "avgHolding": "Average holding",
        "avgLoss": "Average loss",
        "avgProfit": "Average profit",
        "calmar": "Calmar ratio",
        "consecutiveWinsLosses": "Consecutive wins/losses",
        "largestLoss": "Largest loss",
        "largestWin": "Largest win",
        "maxDrawdown": "Max drawdown",
        "netDeposit": "Net deposit",
        "netProfit": "Net profit",
        "profitFactor": "Profit factor",
        "sharpe": "Sharpe ratio",
        "sortino": "Sortino ratio",
        "totalDeposit": "Total deposit",
        "totalTrades": "Total trades",
        "totalWithdrawal": "Total withdrawal",
        "volatility": "Volatility",
        "winRate": "Win rate"
      },
      "timeDetail": {
        "balance": "Balance",
        "lots": "Lots",
        "maxFloatingLossAmount": "Max floating loss amount",
        "maxFloatingLossRatio": "Max floating loss ratio",
        "maxFloatingProfitAmount": "Max floating profit amount",
        "maxFloatingProfitRatio": "Max floating profit ratio",
        "profitAmount": "Profit amount",
        "profitFactor": "Profit factor",
        "trades": "Trades"
      },
      "advancedStatsTitle": "Advanced Statistics",
      "dailyPnLTitle": "Daily P/L",
      "hourlyTitle": "Time-of-day Analysis",
      "monthlyProfitTitle": "Monthly Profit",
      "symbolDistributionTitle": "Symbol Distribution"
    },
    "bind": {
      "actions": {
        "confirmBind": "Confirm bind",
        "retryVerify": "Retry",
        "search": "Search",
        "verifyAccount": "Verify account"
      },
      "errorModal": {
        "title": "Binding failed"
      },
      "errors": {
        "brokerUnavailable": "Server error or incorrect password",
        "connectionFailed": "Unable to connect to broker server, please check your network",
        "invalidCredentials": "Account not found or invalid password",
        "timeout": "Connection timed out, please try again later"
      },
      "fields": {
        "brokerName": "Broker name",
        "company": "Company",
        "password": "Password",
        "platform": "Platform",
        "server": "Server",
        "tradingAccount": "Trading account"
      },
      "labels": {
        "serverCount": "{{count}} servers"
      },
      "messages": {
        "bindFailed": "Failed to bind account",
        "bindSuccess": "Account bound successfully",
        "enterBrokerName": "Please enter broker name",
        "enterPassword": "Please enter password",
        "enterTradingAccount": "Please enter trading account",
        "foundBrokers": "Found {{count}} brokers",
        "loginDigitsOnly": "Trading account must contain only digits",
        "noAccessHosts": "No server addresses available for the selected broker",
        "noBrokersFound": "No matching brokers found. Please check the name.",
        "searchFailed": "Search failed. Please try again later.",
        "selectServer": "Please select a server",
        "verifyFailed": "Account verification failed"
      },
      "placeholders": {
        "brokerName": "Enter broker name, e.g. XM, IC Markets",
        "company": "Select company",
        "password": "Enter password",
        "server": "Select server",
        "tradingAccount": "Enter trading account"
      },
      "step1": {
        "subtitle": "Select your trading platform and search for your broker",
        "title": "Choose platform and broker"
      },
      "step2": {
        "subtitle": "Enter your trading account and password",
        "title": "Enter account information"
      },
      "step3": {
        "subtitle": "Verify credentials and confirm to complete",
        "title": "Verify & confirm"
      },
      "summary": {
        "balance": "Balance",
        "broker": "Broker",
        "currency": "Currency",
        "equity": "Equity",
        "freeMargin": "Free margin",
        "leverage": "Leverage",
        "margin": "Margin",
        "password": "Password",
        "platform": "Platform",
        "server": "Server",
        "tradingAccount": "Trading account",
        "verified": "Account verified"
      },
      "passwordHint": "Password is transmitted via HTTPS and stored as an Argon2id hash (non-reversible) on the backend",
      "title": "Bind MT Account"
    },
    "card": {
      "actions": {
        "details": "Details",
        "orders": "Orders",
        "positions": "Positions"
      },
      "deleteConfirm": {
        "content": "This action cannot be undone",
        "title": "Delete this account?"
      },
      "fields": {
        "balance": "Balance",
        "broker": "Broker",
        "equity": "Equity",
        "server": "Server"
      },
      "status": {
        "connected": "Connected",
        "connecting": "Connecting",
        "disabled": "Disabled",
        "disconnected": "Disconnected",
        "error": "Error"
      }
    },
    "detail": {
      "accountType": {
        "demo": "Demo",
        "real": "Real"
      },
      "actions": {
        "deleteAccount": "Delete account",
        "deleteConfirm": "Verify & Delete",
        "deletePasswordHint": "Enter the MT trading password or read-only password to verify:",
        "deletePasswordPlaceholder": "MT trading / read-only password",
        "deleteWarning": "This action is irreversible. All account data (trade records, analytics, etc.) will be permanently deleted.",
        "disableAccount": "Disable account",
        "enableAccount": "Enable account",
        "syncHistory": "Sync history"
      },
      "balanceRecord": {
        "deposit": "Deposit",
        "depositIconText": "Deposit",
        "withdraw": "Withdraw",
        "withdrawIconText": "Withdraw"
      },
      "cards": {
        "balance": "Balance",
        "credit": "Credit",
        "equity": "Equity",
        "floatingProfit": "Floating P/L",
        "marginFree": "Free margin",
        "marginLevel": "Margin level",
        "marginUsed": "Margin used"
      },
      "messages": {
        "fetchAccountFailed": "Failed to load account information. Please try again later.",
        "syncHistoryFailed": "Failed to sync order history. Please ensure the account is connected to the MT server.",
        "syncHistorySuccess": "Order history synced successfully"
      },
      "mode": {
        "investor": "Investor mode",
        "trader": "Trader mode"
      },
      "orderTypes": {
        "buyLimit": "Buy limit",
        "buyStop": "Buy stop",
        "sellLimit": "Sell limit",
        "sellStop": "Sell stop"
      },
      "status": {
        "connected": "Connected",
        "connecting": "Connecting",
        "disabled": "Disabled",
        "disconnected": "Disconnected",
        "error": "Error"
      },
      "syncHistory": {
        "content": "Sync the last year of order history from the MT server? This may take some time.",
        "ok": "Sync",
        "title": "Sync Order History"
      },
      "connected": "Connected",
      "lastConnected": "{{time}}",
      "leverage": "Leverage {{leverage}}x"
    },
    "disabled": {
      "confirmDelete": {
        "content": "This action cannot be undone",
        "title": "Delete this account?"
      },
      "mobile": {
        "balanceLabel": "Balance: ",
        "equityLabel": "Equity: "
      },
      "table": {
        "account": "Account",
        "actions": "Actions",
        "balance": "Balance",
        "broker": "Broker",
        "equity": "Equity",
        "type": "Type"
      },
      "title": "Disabled Accounts"
    },
    "edit": {
      "fields": {
        "oldPassword": "Current password",
        "password": "New password",
        "server": "Server",
        "tradingAccount": "Trading account"
      },
      "messages": {
        "enterOldPassword": "Please enter current password",
        "enterPassword": "Please enter new password",
        "passwordSaved": "Password saved",
        "passwordVerifyFailed": "Password change failed"
      },
      "placeholders": {
        "newPassword": "Enter new password",
        "oldPassword": "Enter current password"
      },
      "title": "Edit Account"
    },
    "report": {
      "periods": {
        "month": "This Month",
        "quarter": "This Quarter",
        "week": "This Week",
        "year": "This Year"
      },
      "sections": {
        "findings": "Key Findings",
        "recommendations": "Recommendations",
        "summary": "Summary"
      },
      "aiAnalysis": "AI Analysis",
      "direction": "Direction Breakdown",
      "directionLong": "Long",
      "directionShort": "Short",
      "drawdownEvents": "Drawdown Events",
      "drawdownOverlay": "Equity Curve + Drawdown",
      "generate": "Generate Report",
      "goToAISettings": "Go to AI Settings →",
      "recovered": "Recovered",
      "symbolPnL": "P&L by Symbol",
      "title": "Trading Report",
      "titleShort": "Report",
      "tradeDistribution": "Trade Profit Distribution",
      "winRateTrend": "Monthly Win Rate Trend"
    },
    "tradeTabs": {
      "pagination": {
        "total": "{{total}} total"
      },
      "table": {
        "closePrice": "Close price",
        "closeTime": "Close time",
        "currentPrice": "Current price",
        "openPrice": "Open price",
        "openTime": "Open time",
        "orderId": "Order ID",
        "pendingPrice": "Pending price",
        "pendingTime": "Pending time",
        "profit": "P/L",
        "side": "Side",
        "symbol": "Symbol",
        "type": "Type",
        "volume": "Volume"
      },
      "emptyHistory": "No order history",
      "emptyPositions": "No open positions",
      "historyWithCount": "History ({{count}})",
      "pendingWithCount": "Pending ({{count}})",
      "positionsWithCount": "Positions ({{count}})",
      "syncHistory": "Sync History"
    },
    "empty": {
      "subtitle": "Click the button below to bind your MT4/MT5 trading account",
      "title": "No bound accounts"
    },
    "legend": {
      "connected": "Connected",
      "connecting": "Connecting",
      "disabled": "Disabled",
      "disconnectedOrError": "Disconnected/Error",
      "title": "Legend:"
    },
    "messages": {
      "connectFailed": "Connection failed",
      "connectSuccess": "Connected successfully",
      "connectingMtServer": "Connecting to MT server",
      "createFailed": "Failed to create account",
      "createdSuccess": "Account created successfully",
      "deleteFailed": "Delete failed",
      "deleted": "Account deleted",
      "disableFailed": "Failed to disable account",
      "disabledSuccess": "Account disabled successfully",
      "disconnectFailed": "Failed to disconnect",
      "enableFailed": "Failed to enable account",
      "enabledSuccess": "Account enabled successfully",
      "fetchAccountFailed": "Failed to load account information",
      "fetchListFailed": "Failed to load account list"
    },
    "bindNew": "Bind New Account",
    "subtitle": "Manage your MT4/MT5 trading accounts",
    "title": "My Accounts"
  }
} as const;
export default Accounts;
