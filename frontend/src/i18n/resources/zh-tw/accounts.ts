// Auto-generated from proto/ant/v1/i18n/accounts_zh-tw.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const Accounts = {
  "accounts": {
    "analytics": {
      "monthlyAnalysis": {
        "bonus": {
          "chartHoldingTitle": "{{month}} 平均持仓时间",
          "chartPopularTitle": "{{month}} 货币热度",
          "chartRiskTitle": "Bonus：{{month}} 各品種風險報酬比（盈利因子）。",
          "emptyCharts": "該月無成交",
          "legendBulls": "買入側",
          "legendShortTerm": "賣出側",
          "popularityShare": "手數份額",
          "sliceOther": "其他"
        },
        "metrics": {
          "change": "變化",
          "lots": "手數",
          "pips": "点",
          "profit": "盈虧"
        },
        "chartMainTitle": "每月收益（{{metric}}）",
        "focusedValue": "{{period}} · {{metric}}：{{value}}",
        "title": "月度分析"
      },
      "monthlyDetail": {
        "fields": {
          "averageHours": "平均",
          "bestTrade": "最優單筆",
          "maxHours": "最長",
          "medianHours": "中位",
          "minHours": "最短",
          "netReturn": "淨收益",
          "profitFactor": "盈虧比",
          "totalTrades": "總筆數",
          "winRate": "勝率",
          "worstTrade": "最差單筆"
        },
        "holdingTitle": "持倉時長",
        "long": "做多",
        "metricsTitle": "月度指標",
        "popularityTitle": "貨幣流行度",
        "riskRewardTitle": "獎勵:風險比率",
        "short": "做空",
        "symbolPnLTitle": "品種盈虧"
      },
      "advancedTabs": {
        "daily": "日",
        "hourly": "每小時"
      },
      "chartPeriod": {
        "all": "全部",
        "day": "今日",
        "month": "本月",
        "week": "本週",
        "year": "今年"
      },
      "chartSeries": {
        "balance": "餘額",
        "equity": "淨值",
        "profit": "盈虧",
        "tradeCount": "交易次數"
      },
      "chartType": {
        "balance": "餘額",
        "equity": "淨值",
        "profit": "盈虧"
      },
      "empty": {
        "dailyPnL": "暫無每日盈虧資料",
        "equityCurve": "暫無淨值曲線資料",
        "hourly": "暫無時段分析資料",
        "monthlyProfit": "暫無月度盈虧資料",
        "symbolDistribution": "暫無品種資料"
      },
      "stats": {
        "avgDailyReturn": "日均收益",
        "avgHolding": "平均持倉",
        "avgLoss": "平均虧損",
        "avgProfit": "平均盈利",
        "calmar": "卡爾馬",
        "consecutiveWinsLosses": "連勝/連敗",
        "largestLoss": "最大虧損",
        "largestWin": "最大盈利",
        "maxDrawdown": "最大回撤",
        "netDeposit": "淨入金",
        "netProfit": "淨利潤",
        "profitFactor": "盈虧比",
        "sharpe": "夏普比率",
        "sortino": "索提諾",
        "totalDeposit": "入金",
        "totalTrades": "總交易數",
        "totalWithdrawal": "出金",
        "volatility": "波動率",
        "winRate": "勝率"
      },
      "timeDetail": {
        "balance": "餘額",
        "lots": "手數",
        "maxFloatingLossAmount": "最大浮動虧損金額",
        "maxFloatingLossRatio": "最大浮動虧損比例",
        "maxFloatingProfitAmount": "最大浮動盈利金額",
        "maxFloatingProfitRatio": "最大浮動盈利比",
        "profitAmount": "利潤金額",
        "profitFactor": "盈虧比",
        "trades": "交易次數"
      },
      "advancedStatsTitle": "進階統計",
      "dailyPnLTitle": "📅 每日盈虧",
      "hourlyTitle": "⏰ 時段分析",
      "monthlyProfitTitle": "月度盈虧",
      "symbolDistributionTitle": "品種分布"
    },
    "bind": {
      "actions": {
        "confirmBind": "確認綁定",
        "retryVerify": "重试",
        "search": "搜尋",
        "verifyAccount": "驗證帳戶"
      },
      "errorModal": {
        "title": "绑定失敗"
      },
      "errors": {
        "brokerUnavailable": "連線伺服器錯誤或密碼不正確",
        "connectionFailed": "無法連線到經紀商伺服器，請檢查網路",
        "invalidCredentials": "帳號或密碼錯誤，未找到該交易帳戶",
        "timeout": "連接逾時，请稍后重试"
      },
      "fields": {
        "brokerName": "經紀商名稱",
        "company": "選擇公司",
        "password": "密碼",
        "platform": "交易平台",
        "server": "伺服器",
        "tradingAccount": "交易帳號"
      },
      "labels": {
        "serverCount": "{{count}} 台伺服器"
      },
      "messages": {
        "bindFailed": "帳戶綁定失敗",
        "bindSuccess": "帳戶綁定成功",
        "changeCredentials": "修改憑證",
        "enterBrokerName": "請輸入經紀商名稱",
        "enterPassword": "請輸入密碼",
        "enterTradingAccount": "請輸入交易帳號",
        "foundBrokers": "找到 {{count}} 個經紀商",
        "loginDigitsOnly": "交易帳戶只能包含數字",
        "noAccessHosts": "無可用伺服器位址",
        "noBrokersFound": "未找到匹配的經紀商，請檢查名稱",
        "searchFailed": "搜尋失敗，請稍後重試",
        "selectServer": "請選擇伺服器",
        "verifyFailed": "帳戶驗證失敗"
      },
      "placeholders": {
        "brokerName": "輸入經紀商名稱，如：XMGlobal、ICMarkets",
        "company": "請選擇經紀商公司",
        "password": "輸入密碼",
        "server": "請選擇伺服器",
        "tradingAccount": "輸入交易帳號",
        "alias": "可選自訂名稱"
      },
      "step1": {
        "subtitle": "選擇您的交易平台并搜尋经纪商",
        "title": "選擇平台與經紀商"
      },
      "step2": {
        "subtitle": "輸入您的交易帳戶和密碼",
        "title": "輸入帳戶資訊"
      },
      "step3": {
        "subtitle": "驗證凭据并確認完成",
        "title": "確認綁定"
      },
      "summary": {
        "balance": "餘額",
        "broker": "經紀商",
        "currency": "货币",
        "equity": "淨值",
        "freeMargin": "可用保證金",
        "leverage": "槓桿",
        "margin": "保證金",
        "password": "密碼",
        "platform": "交易平台",
        "server": "伺服器",
        "tradingAccount": "交易帳號",
        "verified": "帳戶已驗證"
      },
      "passwordHint": "密碼將透過 HTTPS 加密傳輸，後端使用 Argon2id 雜湊儲存不可回逆",
      "title": "綁定 MT 帳戶"
    },
    "card": {
      "actions": {
        "details": "詳情",
        "orders": "訂單",
        "positions": "持倉"
      },
      "deleteConfirm": {
        "content": "此操作不可撤銷",
        "title": "確定刪除此帳戶？"
      },
      "fields": {
        "balance": "餘額",
        "broker": "經紀商",
        "equity": "淨值",
        "server": "伺服器"
      },
      "status": {
        "connected": "已連接",
        "connecting": "連線中",
        "disabled": "已停用",
        "disconnected": "已斷線",
        "error": "錯誤"
      }
    },
    "detail": {
      "accountType": {
        "demo": "模擬",
        "real": "真實"
      },
      "actions": {
        "deleteAccount": "刪除帳戶",
        "deleteConfirm": "驗證並刪除",
        "deletePasswordHint": "請輸入MT交易密碼或唯讀密碼以驗證：",
        "deletePasswordPlaceholder": "MT交易密碼/唯讀密碼",
        "deletePasswordWrong": "交易密碼/唯讀密碼錯誤，請輸入正確的 MT 密碼。",
        "deleteWarning": "此操作不可逆轉。所有帳戶資料（交易記錄、分析等）將被永久刪除。",
        "disableAccount": "停用帳戶",
        "enableAccount": "啟用帳戶",
        "syncHistory": "同步歷史"
      },
      "balanceRecord": {
        "deposit": "💰 入金",
        "depositIconText": "💰 入金",
        "withdraw": "💸 出金",
        "withdrawIconText": "💸 出金"
      },
      "cards": {
        "balance": "餘額",
        "credit": "授信",
        "equity": "淨值",
        "floatingProfit": "浮動盈虧",
        "marginFree": "可用保證金",
        "marginLevel": "保證金比例",
        "marginUsed": "已用保證金"
      },
      "messages": {
        "fetchAccountFailed": "獲取帳戶資訊失敗，請稍後重試",
        "syncHistoryFailed": "同步訂單歷史失敗，请確保帳戶已連接到 MT 伺服器。",
        "syncHistorySuccess": "同步歷史訂單成功"
      },
      "mode": {
        "investor": "投資者模式",
        "trader": "交易员模式"
      },
      "orderTypes": {
        "buyLimit": "買入限價",
        "buyStop": "買入止損",
        "sellLimit": "賣出限價",
        "sellStop": "卖出止損"
      },
      "status": {
        "connected": "已連接",
        "connecting": "連線中",
        "disabled": "已停用",
        "disconnected": "已斷線",
        "error": "錯誤"
      },
      "syncHistory": {
        "content": "確定要從MT伺服器同步過去一年的歷史訂單嗎？這可能需要一些時間。",
        "ok": "同步",
        "title": "同步歷史訂單"
      },
      "connected": "已連接",
      "lastConnected": "{{time}}",
      "leverage": "槓桿 {{leverage}}x"
    },
    "disabled": {
      "confirmDelete": {
        "content": "此操作不可撤銷",
        "title": "確定刪除此帳戶？"
      },
      "mobile": {
        "balanceLabel": "餘額: ",
        "equityLabel": "净值: "
      },
      "table": {
        "account": "帳號",
        "actions": "操作",
        "balance": "餘額",
        "broker": "經紀商",
        "equity": "淨值",
        "type": "類型"
      },
      "title": "已停用的帳戶"
    },
    "edit": {
      "fields": {
        "oldPassword": "當前密碼",
        "password": "新密碼",
        "server": "伺服器",
        "tradingAccount": "交易帳號"
      },
      "messages": {
        "enterOldPassword": "請輸入當前密碼",
        "enterPassword": "請輸入新密碼",
        "passwordSaved": "密碼已儲存",
        "passwordVerifyFailed": "密碼修改失敗"
      },
      "placeholders": {
        "newPassword": "輸入新密碼",
        "oldPassword": "輸入當前密碼"
      },
      "title": "編輯帳戶"
    },
    "report": {
      "periods": {
        "month": "本月",
        "quarter": "本季度",
        "week": "本週",
        "year": "今年"
      },
      "sections": {
        "findings": "關鍵發現",
        "recommendations": "改進建議",
        "summary": "總體評價"
      },
      "aiAnalysis": "AI 分析",
      "direction": "多空分析",
      "directionLong": "做多",
      "directionShort": "做空",
      "drawdownEvents": "回撤事件",
      "drawdownOverlay": "權益曲線 + 回撤",
      "generate": "生成報告",
      "goToAISettings": "前往 AI 設定 →",
      "recovered": "已恢復",
      "symbolPnL": "品種盈虧",
      "title": "交易報告",
      "titleShort": "報告",
      "tradeDistribution": "盈虧分佈",
      "winRateTrend": "月度勝率趨勢"
    },
    "tradeTabs": {
      "pagination": {
        "total": "共 {{total}} 条"
      },
      "table": {
        "closePrice": "平倉價",
        "closeTime": "平倉时间",
        "currentPrice": "當前價",
        "openPrice": "開倉價",
        "openTime": "開倉時間",
        "orderId": "訂單號",
        "pendingPrice": "掛單價格",
        "pendingTime": "掛單時間",
        "profit": "盈虧",
        "side": "方向",
        "symbol": "品種",
        "type": "類型",
        "volume": "手數"
      },
      "emptyHistory": "暫無歷史訂單",
      "emptyPositions": "暫無持倉",
      "historyWithCount": "歷史訂單 ({{count}})",
      "pendingWithCount": "掛單 ({{count}})",
      "positionsWithCount": "持倉訂單 ({{count}})",
      "syncHistory": "同步历史"
    },
    "empty": {
      "subtitle": "點選下方按钮绑定您的 MT4/MT5 交易帳戶",
      "title": "暫無綁定帳戶"
    },
    "legend": {
      "connected": "已連接",
      "connecting": "連線中",
      "disabled": "已停用",
      "disconnectedOrError": "已斷線/錯誤",
      "title": "圖例:"
    },
    "messages": {
      "connectFailed": "連線失敗",
      "connectSuccess": "連線成功",
      "connectingMtServer": "正在連線 MT 伺服器",
      "createFailed": "建立帳戶失敗",
      "createdSuccess": "帳戶建立成功",
      "deleteFailed": "刪除失敗",
      "deleted": "帳戶已刪除",
      "disableFailed": "停用帳戶失敗",
      "disabledSuccess": "帳戶停用成功",
      "disconnectFailed": "斷開連線失敗",
      "enableFailed": "启用帳戶失敗",
      "enabledSuccess": "帳戶啟用成功",
      "fetchAccountFailed": "取得帳戶資訊失敗",
      "fetchListFailed": "取得帳戶列表失敗"
    },
    "bindNew": "綁定新帳戶",
    "subtitle": "管理您的 MT4/MT5 交易帳戶",
    "title": "我的帳戶"
  }
} as const;
export default Accounts;
