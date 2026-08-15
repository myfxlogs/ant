// Auto-generated from proto/ant/v1/i18n/accounts_zh-cn.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const Accounts = {
  "accounts": {
    "analytics": {
      "monthlyAnalysis": {
        "bonus": {
          "chartHoldingTitle": "{{month}} 平均持仓时间",
          "chartPopularTitle": "{{month}} 货币热度",
          "chartRiskTitle": "Bonus：{{month}} 各品种风险回报比（盈利因子）。",
          "emptyCharts": "该月无成交",
          "legendBulls": "买入侧",
          "legendShortTerm": "卖出侧",
          "popularityShare": "手数份额",
          "sliceOther": "其他"
        },
        "metrics": {
          "change": "变化",
          "lots": "手数",
          "pips": "点",
          "profit": "盈亏"
        },
        "chartMainTitle": "每月收益（{{metric}}）",
        "focusedValue": "{{period}} · {{metric}}：{{value}}",
        "title": "月度分析"
      },
      "monthlyDetail": {
        "fields": {
          "averageHours": "平均",
          "bestTrade": "最优单笔",
          "maxHours": "最长",
          "medianHours": "中位",
          "minHours": "最短",
          "netReturn": "净收益",
          "profitFactor": "盈亏比",
          "totalTrades": "总笔数",
          "winRate": "胜率",
          "worstTrade": "最差单笔"
        },
        "holdingTitle": "持仓时长",
        "long": "做多",
        "metricsTitle": "月度指标",
        "popularityTitle": "货币流行度",
        "riskRewardTitle": "奖励:风险比率",
        "short": "做空",
        "symbolPnLTitle": "品种盈亏"
      },
      "advancedTabs": {
        "daily": "日",
        "hourly": "按小时"
      },
      "chartPeriod": {
        "all": "全部",
        "day": "今日",
        "month": "本月",
        "week": "本周",
        "year": "本年"
      },
      "chartSeries": {
        "balance": "余额",
        "equity": "净值",
        "profit": "盈亏",
        "tradeCount": "次数"
      },
      "chartType": {
        "balance": "余额",
        "equity": "净值",
        "profit": "盈亏"
      },
      "empty": {
        "dailyPnL": "暂无每日盈亏数据",
        "equityCurve": "暂无净值曲线数据",
        "hourly": "暂无时段分析数据",
        "monthlyProfit": "暂无月度盈亏数据",
        "symbolDistribution": "暂无品种数据"
      },
      "stats": {
        "avgDailyReturn": "日均收益",
        "avgHolding": "平均持仓",
        "avgLoss": "平均亏损",
        "avgProfit": "平均盈利",
        "calmar": "卡尔马",
        "consecutiveWinsLosses": "连胜/连败",
        "largestLoss": "最大亏损",
        "largestWin": "最大盈利",
        "maxDrawdown": "最大回撤",
        "netDeposit": "净入金",
        "netProfit": "净利润",
        "profitFactor": "盈亏比",
        "sharpe": "夏普比率",
        "sortino": "索提诺",
        "totalDeposit": "入金",
        "totalTrades": "总交易数",
        "totalWithdrawal": "出金",
        "volatility": "波动率",
        "winRate": "胜率"
      },
      "timeDetail": {
        "balance": "余额",
        "lots": "手数",
        "maxFloatingLossAmount": "最大浮亏金额",
        "maxFloatingLossRatio": "最大浮亏比例",
        "maxFloatingProfitAmount": "最大浮盈金额",
        "maxFloatingProfitRatio": "最大浮动盈利比",
        "profitAmount": "盈亏金额",
        "profitFactor": "盈亏比",
        "trades": "次数"
      },
      "advancedStatsTitle": "高级统计",
      "dailyPnLTitle": "📅 每日盈亏",
      "hourlyTitle": "⏰ 时段分析",
      "monthlyProfitTitle": "月度盈亏",
      "symbolDistributionTitle": "品种分布"
    },
    "bind": {
      "actions": {
        "confirmBind": "确认绑定",
        "retryVerify": "重试",
        "search": "搜索",
        "verifyAccount": "验证账户"
      },
      "errorModal": {
        "title": "绑定失败"
      },
      "errors": {
        "brokerUnavailable": "连接服务器错误或者密码不正确",
        "connectionFailed": "无法连接到经纪商服务器，请检查网络",
        "invalidCredentials": "账号或密码错误，未找到该交易账户",
        "timeout": "连接超时，请稍后重试"
      },
      "fields": {
        "brokerName": "经纪商名称",
        "company": "选择公司",
        "password": "密码",
        "platform": "交易平台",
        "server": "服务器",
        "tradingAccount": "交易账号"
      },
      "labels": {
        "serverCount": "{{count}} 台服务器"
      },
      "messages": {
        "bindFailed": "账户绑定失败",
        "bindSuccess": "账户绑定成功",
        "enterBrokerName": "请输入经纪商名称",
        "enterPassword": "请输入密码",
        "enterTradingAccount": "请输入交易账号",
        "foundBrokers": "找到 {{count}} 个经纪商",
        "loginDigitsOnly": "交易账户只能包含数字",
        "noAccessHosts": "无可用服务器地址",
        "noBrokersFound": "未找到匹配的经纪商，请检查名称",
        "searchFailed": "搜索失败，请稍后重试",
        "selectServer": "请选择服务器",
        "verifyFailed": "账户验证失败"
      },
      "placeholders": {
        "brokerName": "输入经纪商名称，如：XMGlobal、ICMarkets",
        "company": "请选择经纪商公司",
        "password": "输入密码",
        "server": "请选择服务器",
        "tradingAccount": "输入交易账号"
      },
      "step1": {
        "subtitle": "选择您的交易平台并搜索经纪商",
        "title": "选择平台和经纪商"
      },
      "step2": {
        "subtitle": "输入您的交易账户和密码",
        "title": "输入账户信息"
      },
      "step3": {
        "subtitle": "验证凭据并确认完成",
        "title": "验证并确认"
      },
      "summary": {
        "balance": "余额",
        "broker": "经纪商",
        "currency": "货币",
        "equity": "净值",
        "freeMargin": "可用保证金",
        "leverage": "杠杆",
        "margin": "已用保证金",
        "password": "密码",
        "platform": "交易平台",
        "server": "服务器",
        "tradingAccount": "交易账号",
        "verified": "账户验证通过"
      },
      "passwordHint": "密码将通过 HTTPS 加密传输，后端使用 Argon2id 哈希存储不可回逆",
      "title": "绑定 MT 账户"
    },
    "card": {
      "actions": {
        "details": "详情",
        "orders": "订单",
        "positions": "持仓"
      },
      "deleteConfirm": {
        "content": "此操作不可撤销",
        "title": "确定删除此账户？"
      },
      "fields": {
        "balance": "余额",
        "broker": "经纪商",
        "equity": "净值",
        "server": "服务器"
      },
      "status": {
        "connected": "已连接",
        "connecting": "连接中",
        "disabled": "已停用",
        "disconnected": "已断开",
        "error": "错误"
      }
    },
    "detail": {
      "accountType": {
        "demo": "模拟",
        "real": "真实"
      },
      "actions": {
        "deleteAccount": "删除账户",
        "deleteConfirm": "验证并删除",
        "deletePasswordHint": "请输入该账户的 MT 交易密码或只读密码进行验证：",
        "deletePasswordPlaceholder": "MT 交易密码 / 只读密码",
        "deletePasswordWrong": "交易密码/只读密码错误，请输入正确的 MT 密码。",
        "deleteWarning": "此操作不可撤销。账户所有数据（交易记录、分析数据等）将被永久删除。",
        "disableAccount": "停用账户",
        "enableAccount": "启用账户",
        "syncHistory": "同步历史"
      },
      "balanceRecord": {
        "deposit": "💰 入金",
        "depositIconText": "💰 入金",
        "withdraw": "💸 出金",
        "withdrawIconText": "💸 出金"
      },
      "cards": {
        "balance": "余额",
        "credit": "授信",
        "equity": "净值",
        "floatingProfit": "浮动盈亏",
        "marginFree": "可用保证金",
        "marginLevel": "保证金比例",
        "marginUsed": "已用保证金"
      },
      "messages": {
        "fetchAccountFailed": "获取账户信息失败，请稍后重试",
        "syncHistoryFailed": "同步订单历史失败，请确保账户已连接到 MT 服务器。",
        "syncHistorySuccess": "同步历史订单成功"
      },
      "mode": {
        "investor": "投资者模式",
        "trader": "交易员模式"
      },
      "orderTypes": {
        "buyLimit": "买入限价",
        "buyStop": "买入止损",
        "sellLimit": "卖出限价",
        "sellStop": "卖出止损"
      },
      "status": {
        "connected": "已连接",
        "connecting": "连接中",
        "disabled": "已停用",
        "disconnected": "已断开",
        "error": "错误"
      },
      "syncHistory": {
        "content": "确定要从MT服务器同步过去一年的历史订单吗？这可能需要一些时间。",
        "ok": "同步",
        "title": "同步历史订单"
      },
      "connected": "已连接",
      "lastConnected": "{{time}}",
      "leverage": "杠杆 {{leverage}}x"
    },
    "disabled": {
      "confirmDelete": {
        "content": "此操作不可撤销",
        "title": "确定删除此账户？"
      },
      "mobile": {
        "balanceLabel": "余额: ",
        "equityLabel": "净值: "
      },
      "table": {
        "account": "账号",
        "actions": "操作",
        "balance": "余额",
        "broker": "经纪商",
        "equity": "净值",
        "type": "类型"
      },
      "title": "已停用的账户"
    },
    "edit": {
      "fields": {
        "oldPassword": "当前密码",
        "password": "新密码",
        "server": "服务器",
        "tradingAccount": "交易账号"
      },
      "messages": {
        "enterOldPassword": "请输入当前密码",
        "enterPassword": "请输入新密码",
        "passwordSaved": "密码已保存",
        "passwordVerifyFailed": "密码修改失败"
      },
      "placeholders": {
        "newPassword": "输入新密码",
        "oldPassword": "输入当前密码"
      },
      "title": "编辑账户"
    },
    "report": {
      "periods": {
        "month": "本月",
        "quarter": "本季度",
        "week": "本周",
        "year": "今年"
      },
      "sections": {
        "findings": "关键发现",
        "recommendations": "改进建议",
        "summary": "总体评价"
      },
      "aiAnalysis": "AI 分析",
      "direction": "多空分析",
      "directionLong": "做多",
      "directionShort": "做空",
      "drawdownEvents": "回撤事件",
      "drawdownOverlay": "权益曲线 + 回撤",
      "generate": "生成报告",
      "goToAISettings": "前往 AI 设置 →",
      "recovered": "已恢复",
      "symbolPnL": "品种盈亏",
      "title": "交易报告",
      "titleShort": "报告",
      "tradeDistribution": "盈亏分布",
      "winRateTrend": "月度胜率趋势"
    },
    "tradeTabs": {
      "pagination": {
        "total": "共 {{total}} 条"
      },
      "table": {
        "closePrice": "平仓价",
        "closeTime": "平仓时间",
        "magic": "Magic",
        "currentPrice": "当前价",
        "openPrice": "开仓价",
        "openTime": "开仓时间",
        "orderId": "订单号",
        "pendingPrice": "挂单价格",
        "pendingTime": "挂单时间",
        "profit": "盈亏",
        "side": "方向",
        "symbol": "品种",
        "type": "类型",
        "volume": "手数"
      },
      "emptyHistory": "暂无历史订单",
      "emptyPositions": "暂无持仓",
      "historyWithCount": "历史订单 ({{count}})",
      "pendingWithCount": "挂单 ({{count}})",
      "positionsWithCount": "持仓订单 ({{count}})",
      "syncHistory": "同步历史"
    },
    "empty": {
      "subtitle": "点击下方按钮绑定您的 MT4/MT5 交易账户",
      "title": "暂无绑定账户"
    },
    "legend": {
      "connected": "已连接",
      "connecting": "连接中",
      "disabled": "已停用",
      "disconnectedOrError": "已断开/错误",
      "title": "图例:"
    },
    "messages": {
      "connectFailed": "连接失败",
      "connectSuccess": "连接成功",
      "connectingMtServer": "正在连接 MT 服务器",
      "createFailed": "创建账户失败",
      "createdSuccess": "账户创建成功",
      "deleteFailed": "删除失败",
      "deleted": "账户已删除",
      "disableFailed": "停用账户失败",
      "disabledSuccess": "账户停用成功",
      "disconnectFailed": "断开连接失败",
      "enableFailed": "启用账户失败",
      "enabledSuccess": "账户启用成功",
      "fetchAccountFailed": "获取账户信息失败",
      "fetchListFailed": "获取账户列表失败"
    },
    "bindNew": "绑定新账户",
    "subtitle": "管理您的 MT4/MT5 交易账户",
    "title": "我的账户"
  }
} as const;
export default Accounts;
