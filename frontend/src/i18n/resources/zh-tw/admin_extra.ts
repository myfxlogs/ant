// Auto-generated supplementary keys for admin
const AdminExtra = {
  "admin": {
    "aiGateway": {
      "errors": {
        "loadProviders": "載入供應商失敗",
        "toggleFailed": "切換失敗",
        "loadModels": "載入模型失敗"
      },
      "addProviderPending": "新增供應商功能待後端支援",
      "title": "AI 閘道管理",
      "description": "管理 AI 供應商、模型和定價。使用者從可用模型中選擇，按 token 從錢包計費。",
      "addProvider": "新增供應商",
      "columns": {
        "baseUrl": "基礎 URL",
        "apiKey": "API 金鑰"
      },
      "configured": "未設定",
      "editProvider": "新增供應商",
      "providerId": "請輸入供應商 ID",
      "providerIdPlaceholder": "deepseek / openai / qwen ...",
      "displayName": "顯示名稱",
      "displayNamePlaceholder": "DeepSeek",
      "baseUrl": "請輸入基礎 URL",
      "apiKeyLabel": "API 金鑰，加密儲存",
      "apiKeyEditPlaceholder": "留空則保持不變",
      "editModel": "新增模型",
      "modelName": "模型名稱",
      "priceInput": "輸入價格（$/1M）",
      "priceOutput": "輸出價格（$/1M）",
      "addModel": "新增模型",
      "confirmDeleteModel": "刪除此模型？",
      "noModels": "暫無模型"
    },
    "account": {
      "errors": {
        "loadFailed": "載入帳號失敗",
        "freezeFailed": "凍結失敗",
        "unfreezeFailed": "解凍失敗"
      },
      "frozen": "帳號已凍結",
      "unfrozen": "帳號已解凍",
      "columns": {
        "createdAt": "建立時間"
      },
      "confirmFreeze": "凍結此帳號？",
      "title": "帳號管理",
      "searchPlaceholder": "搜尋帳號",
      "detail": "帳號詳情",
      "auditLogs": "稽核日誌"
    },
    "settings": {
      "saveSuccess": "儲存成功",
      "saveFailed": "儲存失敗",
      "deleteFailed": "刪除失敗",
      "actionFailed": "操作失敗",
      "columns": {
        "key": "設定鍵"
      },
      "confirmDelete": "確認刪除？",
      "title": "Agent 管理設定",
      "addSetting": "新增設定",
      "permissionRules": "許可權規則 (permission.rule.N)",
      "permissionFormat": "格式：",
      "permissionExample": "範例：",
      "permissionAddRule": "新增規則：建立設定鍵",
      "addManagedSetting": "新增託管設定",
      "settingKey": "設定鍵",
      "keyPlaceholder": "例如：allowed_models, disable_live_trading, permission.rule.1",
      "valuePlaceholder": "例如：claude-sonnet-5,deepseek-v4"
    },
    "autogen": {
      "approved": "任務已批准並發布",
      "rejected": "任務已拒絕",
      "enqueued": "{{count}} 個任務已排隊",
      "confirmApprove": "批准並發布？",
      "confirmReject": "拒絕此任務？",
      "title": "AI 策略生成任務",
      "allStatus": "全部狀態",
      "triggerBatch": "觸發批次生成",
      "symbols": "品種（逗號分隔）",
      "timeframes": "時間週期（逗號分隔）",
      "strategyTypes": "策略型別（逗號分隔）"
    },
    "billing": {
      "columns": {
        "autoRenew": "自動續費",
        "periodStart": "週期開始",
        "periodEnd": "週期結束",
        "createdAt": "建立時間",
        "balanceBefore": "交易前餘額",
        "balanceAfter": "交易後餘額"
      },
      "title": "帳單管理",
      "monthlyRevenue": "月收入",
      "totalRevenue": "總收入",
      "activeSubs": "活躍訂閱",
      "planRevenue": "方案收入明細",
      "filterByPlan": "按方案篩選",
      "filterByStatus": "按狀態篩選",
      "walletTransactions": "錢包交易",
      "filterByType": "按型別篩選",
      "txPlatformFee": "平臺手續費"
    },
    "coupon": {
      "loadFailed": "載入優惠券失敗",
      "fillRequired": "請填寫必填欄位",
      "created": "優惠券已建立",
      "createFailed": "建立優惠券失敗",
      "disabled": "優惠券已停用",
      "disableFailed": "停用優惠券失敗",
      "colMinPurchase": "最低消費",
      "create": "建立優惠券",
      "createTitle": "建立優惠券",
      "codePlaceholder": "優惠券碼（如 SUMMER20）",
      "valuePlaceholder": "折扣值（如 20 表示 20% 或 50 表示 ¥50）",
      "minPurchasePlaceholder": "最低消費金額（0 = 無限制）",
      "maxUsesPlaceholder": "最大使用次數（0 = 無限）",
      "expiresPlaceholder": "過期時間（ISO 8601，空 = 永不過期）"
    },
    "dashboard": {
      "errors": {
        "loadFailed": "載入儀錶板資料失敗"
      },
      "title": "管理儀錶板",
      "totalUsers": "總使用者數",
      "activeUsers": "活躍使用者",
      "verifiedUsers": "已驗證使用者",
      "mtAccounts": "MT 帳號",
      "onlineAccounts": "線上帳號",
      "todayTrades": "今日交易",
      "todayProfit": "今日盈虧",
      "activeSubs": "活躍訂閱",
      "monthlyRevenue": "月收入",
      "totalRevenue": "總收入",
      "marketStrategies": "市場策略",
      "marketSales": "市場銷售",
      "marketRevenue": "市場收入",
      "recentLogs": "最近日誌"
    },
    "logs": {
      "modules": {
        "userManagement": "使用者管理",
        "accountManagement": "帳號管理",
        "systemConfig": "系統設定"
      },
      "columns": {
        "actionType": "操作型別",
        "ip": "IP 位址"
      },
      "errors": {
        "loadFailed": "載入日誌失敗"
      },
      "title": "操作日誌",
      "filterModule": "按模組篩選",
      "filterAction": "按操作篩選"
    },
    "depositAddresses": {
      "importFailed": "匯入失敗",
      "user": "使用者 ID",
      "received": "已收到 USDT",
      "assignedAt": "分配時間",
      "importHint": "在離線機器上使用 hdgen 工具生成 deposit_addresses.bin，然後在此上傳。",
      "all": "全部狀態",
      "import": "匯入地址",
      "availablePool": "池中可用",
      "total": "總地址數"
    },
    "deposit": {
      "table": {
        "user": "使用者 ID",
        "amount": "USDT 金額",
        "txHash": "交易雜湊"
      },
      "title": "充值管理"
    },
    "analytics": {
      "platformRev": "平臺收入",
      "providerRev": "供應商收入",
      "activeBuyers": "活躍買家",
      "refundRate": "退款率",
      "newSubs": "新訂閱者",
      "totalStrategies": "總策略數",
      "newStrategies": "新增策略",
      "topByRevenue": "收入最高策略",
      "topBySubs": "訂閱最多策略",
      "topProvidersRev": "收入最高供應商",
      "topProvidersStrat": "策略最多供應商"
    },
    "marketplace": {
      "loadFailed": "載入策略失敗",
      "featureSuccess": "策略已設為推薦",
      "featureFailed": "設定推薦失敗",
      "unfeatureSuccess": "已取消推薦",
      "unfeatureFailed": "取消推薦失敗",
      "unfeature": "取消推薦",
      "filterStatus": "全部狀態",
      "searchPlaceholder": "按標題搜尋...",
      "featureTitle": "推薦策略",
      "featureDesc": "設定推薦展示優先順序。數值越高越顯著。"
    },
    "refund": {
      "loadFailed": "載入退款請求失敗",
      "approved": "退款已批准並執行",
      "rejected": "退款請求已拒絕",
      "processFailed": "處理退款失敗",
      "approve": "批准並執行",
      "filterStatus": "全部狀態",
      "approveTitle": "批准退款",
      "rejectTitle": "拒絕退款",
      "reviewNotePlaceholder": "審核備註（拒絕時可選，批准時建議填寫）..."
    },
    "sidebar": {
      "shareManagement": "分享分析"
    },
    "walletCalculator": {
      "title": "Token ↔ USD 計算器",
      "selectModel": "選擇模型（計價基準）",
      "usdAmount": "USD 金額",
      "fillResult": "填入結果"
    },
    "wallet": {
      "errors": {
        "noUserSelected": "未選擇使用者"
      },
      "messages": {
        "adjustSuccess": "餘額調整成功",
        "adjustFailed": "調整失敗"
      },
      "columns": {
        "walletNumber": "錢包編號",
        "balanceAfter": "交易後餘額"
      },
      "title": "錢包管理",
      "tabWallets": "使用者錢包",
      "userList": "使用者列表",
      "searchPlaceholder": "搜尋錢包/電子郵件/暱稱",
      "noMatch": "無使用者",
      "walletDetail": "錢包詳情",
      "adjustBalance": "調整餘額",
      "tabDepositAddresses": "充值地址"
    },
    "config": {
      "apiKey": "API 金鑰"
    },
    "userManagement": {
      "form": {
        "accountNumber": "帳號編號",
        "accountNumberInvalid": "5-6 位數字，不以 0 開頭，不含 4 或 7"
      },
      "messages": {
        "loadUsersFailed": "載入使用者失敗"
      }
    }
  }
} as const;
export default AdminExtra;
