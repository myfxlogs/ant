// Auto-generated from proto/ant/v1/i18n/base_zh-cn.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const Base = {
  "admin": {
    "dashboard": {
      "logs": {
        "moduleMap": {
          "accountManagement": "賬戶管理",
          "systemConfig": "系統配置",
          "trading": "交易",
          "userManagement": "使用者管理"
        },
        "actionType": "操作",
        "failed": "失敗",
        "module": "模組",
        "status": "狀態",
        "success": "成功",
        "target": "目標",
        "time": "時間"
      },
      "riskMetrics": {
        "orderCloseFailed": "平倉失敗",
        "orderCloseSuccess": "平倉成功",
        "orderSendFailed": "下單失敗",
        "orderSendSuccess": "下單成功",
        "riskValidateError": "錯誤",
        "riskValidatePass": "透過",
        "riskValidateReject": "拒絕",
        "riskValidateTotal": "總驗證數",
        "title": "風控指標"
      },
      "riskWindow": {
        "noData": "暫無視窗指標資料",
        "noRejectData": "本時段無拒絕記錄",
        "orderCloseFailed": "平倉失敗",
        "orderCloseSuccess": "平倉成功",
        "orderSendFailed": "下單失敗",
        "orderSendSuccess": "下單成功",
        "rejectCount": "拒絕次數",
        "rejectRiskCodesHeader": "風控程式碼",
        "title": "風控視窗",
        "validateError": "錯誤",
        "validatePass": "透過",
        "validateReject": "拒絕",
        "validateTotal": "總計"
      },
      "errors": {
        "loadFailed": "失敗 to load dashboard data"
      },
      "activeUsers": "活躍使用者",
      "loadFailed": "載入儀表盤資料失敗",
      "mtAccounts": "MT賬戶數",
      "onlineAccounts": "線上賬戶",
      "recentLogs": "最近日誌",
      "title": "管理儀表盤",
      "todayProfit": "今日盈虧",
      "todayTrades": "今日交易",
      "totalUsers": "總使用者數",
      "verifiedUsers": "已驗證使用者",
      "activeSubs": "活躍訂閱",
      "monthlyRevenue": "月度收入",
      "totalRevenue": "總收入",
      "marketStrategies": "Market 策略",
      "marketSales": "市場銷售額",
      "marketRevenue": "市場收入",
      "validateTotal": "Validate 總計",
      "validatePass": "驗證透過",
      "validateReject": "驗證拒絕",
      "validateError": "驗證錯誤",
      "orderSendSuccess": "下單成功",
      "orderSendFailed": "下單失敗",
      "orderCloseSuccess": "平倉成功",
      "orderCloseFailed": "平倉失敗",
      "rejectCount": "拒絕次數"
    },
    "userManagement": {
      "drawer": {
        "labels": {
          "createdAt": "建立時間",
          "email": "郵箱",
          "id": "ID",
          "lastLogin": "最後登入",
          "mtAccountCount": "MT賬戶數",
          "nickname": "暱稱",
          "role": "角色",
          "status": "狀態"
        },
        "title": "使用者詳情"
      },
      "form": {
        "placeholders": {
          "email": "輸入郵箱",
          "nickname": "輸入暱稱",
          "password": "輸入密碼"
        },
        "accountNumber": "錢包號",
        "accountNumberInvalid": "5-6位數字，無前導零，不含4和7",
        "email": "郵箱",
        "nickname": "暱稱",
        "password": "密碼",
        "role": "角色",
        "status": "狀態"
      },
      "passwordForm": {
        "placeholders": {
          "confirmPassword": "再次輸入新密碼",
          "newPassword": "輸入新密碼"
        },
        "validation": {
          "confirmPasswordRequired": "請確認新密碼",
          "newPasswordRequired": "請輸入新密碼",
          "passwordMin8": "密碼至少8位",
          "passwordMismatch": "兩次密碼不一致",
          "passwordMustContainLettersAndNumbers": "密碼必須包含字母和數字"
        },
        "confirmPassword": "確認密碼",
        "newPassword": "新密碼",
        "submit": "更新密碼"
      },
      "actions": {
        "changePassword": "修改密碼",
        "details": "詳情",
        "disable": "禁用",
        "enable": "啟用"
      },
      "deleteConfirm": {
        "batchDeleteConfirm": "確認刪除 {{count}} 個使用者？此操作不可撤銷。",
        "batchDeletePartial": "已刪除 {{deleted}} 個，{{failed}} 個失敗",
        "batchDeleteSuccess": "已刪除 {{count}} 個使用者",
        "title": "確認刪除此使用者？此操作不可撤銷。"
      },
      "filters": {
        "rolePlaceholder": "按角色篩選",
        "searchPlaceholder": "搜尋郵箱或暱稱",
        "statusPlaceholder": "按狀態篩選"
      },
      "messages": {
        "newPasswordIs": "新密碼為: {{password}}",
        "passwordUpdateFailed": "密碼更新失敗",
        "passwordUpdatedSuccess": "密碼更新成功",
        "userCreateFailed": "建立使用者失敗",
        "userCreatedSuccess": "使用者建立成功",
        "userDeleteFailed": "刪除使用者失敗",
        "userDeletedSuccess": "使用者已刪除",
        "userDisabled": "使用者已禁用",
        "userEnabled": "使用者已啟用",
        "userUpdateFailed": "更新使用者失敗",
        "userUpdatedSuccess": "使用者更新成功",
        "loadUsersFailed": "失敗 to load users"
      },
      "modals": {
        "createTitle": "新建使用者",
        "editTitle": "編輯使用者",
        "passwordTitle": "修改密碼"
      },
      "pagination": {
        "total": "共 {{total}} 位使用者"
      },
      "roles": {
        "audit": "審計",
        "customerService": "客服",
        "operation": "運營",
        "superAdmin": "超級管理員",
        "user": "普通使用者"
      },
      "status": {
        "active": "正常",
        "suspended": "已停用"
      },
      "table": {
        "actions": "操作",
        "createdAt": "建立時間",
        "email": "郵箱",
        "id": "ID",
        "mtAccountCount": "MT賬戶數",
        "nickname": "暱稱",
        "role": "角色",
        "status": "狀態"
      },
      "addUser": "新建使用者",
      "title": "使用者管理"
    },
    "config": {
      "messages": {
        "disabled": "已禁用",
        "enabled": "已啟用",
        "loadFailed": "載入配置失敗",
        "operationFailed": "操作失敗",
        "updateFailed": "更新配置失敗",
        "updated": "配置已更新"
      },
      "placeholders": {
        "apiKey": "輸入API Key",
        "baseUrl": "輸入Base URL",
        "configValue": "輸入配置值",
        "description": "輸入描述",
        "json": "輸入JSON",
        "model": "輸入模型名稱"
      },
      "providerOptions": {
        "custom": "自定義 / OpenAI 相容",
        "deepseek": "DeepSeek",
        "zhipu": "智譜AI"
      },
      "validation": {
        "apiKeyRequired": "API Key不能為空",
        "greenMaxFailedRunsNonNegative": "綠色最大失敗次數需≥0",
        "greenSuccessRateRange": "綠色成功率需在0-100之間",
        "jsonEmpty": "JSON不能為空",
        "jsonInvalid": "JSON格式無效",
        "minSampleSizeNonNegative": "最小樣本量需≥0",
        "modelRequired": "模型名稱不能為空",
        "yellowNotGreaterThanGreen": "黃色閾值不能超過綠色閾值",
        "yellowSuccessRateRange": "黃色成功率需在0-100之間"
      },
      "aiProviderCatalog": "AI提供商目錄",
      "baseUrlLabel": "基礎 URL",
      "configItem": "配置項",
      "description": "描述",
      "econAIConfig": "經濟日曆AI配置",
      "editConfig": "編輯配置: {{key}}",
      "enableToggle": "啟用",
      "fillTemplate": "填充模板",
      "formatJson": "格式化JSON",
      "maxAccountsPerUser": "每使用者最大賬戶數",
      "modelName": "模型名稱",
      "off": "關",
      "on": "開",
      "provider": "提供商",
      "status": "狀態",
      "strategyHealthConfig": "策略健康度配置",
      "thresholdDesc": "閾值描述",
      "thresholdInfo": "閾值說明",
      "title": "系統配置",
      "toggle": "切換",
      "updatedAt": "更新時間",
      "value": "值",
      "apiKey": "API 金鑰"
    },
    "jurisdiction": {
      "messages": {
        "countryAddFailed": "新增國家失敗",
        "countryAdded": "國家已新增",
        "countryRemoveFailed": "移除國家失敗",
        "countryRemoved": "國家已移除",
        "kycUpdateFailed": "更新KYC狀態失敗",
        "kycUpdated": "KYC狀態已更新",
        "overrideUpdateFailed": "更新制裁豁免失敗",
        "overrideUpdated": "豁免狀態已更新"
      },
      "actions": "操作",
      "addCountry": "新增國家",
      "addSanctionedCountry": "新增制裁國家",
      "addedBy": "新增人",
      "confirmGrantOverride": "確認授予該使用者豁免許可權？",
      "confirmRevokeOverride": "確認撤銷該使用者的豁免許可權？",
      "country": "國家",
      "countryCode": "國家程式碼",
      "countryLabel": "國家",
      "disclaimer": "免責宣告",
      "emptyKYC": "暫無KYC記錄",
      "emptySanctions": "暫無制裁國家",
      "filterByKYCStatus": "按KYC狀態篩選",
      "grantOverride": "授予豁免",
      "kycStatus": "KYC狀態",
      "kycStatusTab": "使用者KYC狀態",
      "override": "豁免",
      "overrideWarning": "此使用者來自受制裁國家，授予豁免將允許交易。",
      "pending": "待稽核",
      "questionnaire": "問卷",
      "rejected": "已拒絕",
      "revokeOverride": "撤銷豁免",
      "sanctioned": "已制裁",
      "sanctionedCountries": "制裁國家",
      "sanctionedCountriesTab": "制裁國家",
      "setKYC": "設定KYC",
      "setKYCStatus": "設定KYC狀態",
      "title": "管轄權管理",
      "unverified": "未驗證",
      "userEmail": "郵箱",
      "userKYCStatus": "使用者KYC狀態",
      "verified": "已驗證"
    },
    "aiGateway": {
      "errors": {
        "loadProviders": "失敗 to load providers",
        "toggleFailed": "切換失敗",
        "loadModels": "失敗 to load models"
      },
      "columns": {
        "baseUrl": "基礎 URL",
        "apiKey": "API 金鑰"
      },
      "addProviderPending": "新增 provider feature pending backend support",
      "title": "AI 閘道器管理",
      "description": "管理 AI 提供商、模型和定價。使用者從可用模型中選擇，按 token 從錢包扣費。",
      "addProvider": "新增提供商",
      "provider": "提供商",
      "configured": "已配置",
      "notConfigured": "未配置",
      "models": "模型",
      "editProvider": "編輯提供商",
      "providerId": "提供商 ID",
      "providerIdRequired": "請輸入提供商ID",
      "displayName": "顯示名稱",
      "displayNameRequired": "請輸入顯示名稱",
      "baseUrl": "基礎 URL",
      "baseUrlRequired": "Please enter 基礎 URL",
      "apiKeyLabel": "API 金鑰",
      "apiKeyEditHint": "留空則保留現有金鑰",
      "apiKeyHint": "API金鑰，靜態加密儲存",
      "apiKeyEditPlaceholder": "留空則保留",
      "editModel": "編輯模型",
      "addModel": "新增模型",
      "modelName": "模型名稱",
      "modelNameRequired": "請輸入模型名稱",
      "priceInput": "Input 價格 ($/1M)",
      "priceOutput": "Output 價格 ($/1M)",
      "confirmDeleteModel": "刪除 this model?",
      "noModels": "無模型"
    },
    "account": {
      "errors": {
        "loadFailed": "失敗 to load accounts",
        "freezeFailed": "凍結失敗",
        "unfreezeFailed": "解凍失敗"
      },
      "columns": {
        "id": "ID",
        "user": "使用者",
        "login": "登入",
        "type": "型別",
        "broker": "經紀商",
        "status": "狀態",
        "balance": "餘額",
        "createdAt": "建立時間",
        "action": "操作",
        "server": "伺服器",
        "equity": "淨值",
        "margin": "保證金",
        "time": "時間",
        "detail": "詳情"
      },
      "frozen": "賬戶 frozen",
      "unfrozen": "賬戶 unfrozen",
      "detail": "詳情",
      "unfreeze": "解凍",
      "confirmFreeze": "凍結此賬戶？",
      "freeze": "凍結",
      "title": "賬戶管理",
      "searchPlaceholder": "搜尋 accounts",
      "status": "狀態",
      "online": "線上",
      "offline": "離線",
      "auditLogs": "Audit 日誌"
    },
    "settings": {
      "columns": {
        "key": "設定鍵",
        "value": "值",
        "action": "操作"
      },
      "saveSuccess": "儲存成功",
      "saveFailed": "儲存 failed",
      "deleted": "已刪除",
      "deleteFailed": "刪除 failed",
      "actionFailed": "操作失敗",
      "confirmDelete": "確認 delete?",
      "title": "Agent 管理 設定",
      "addSetting": "新增 Setting",
      "permissionRules": "許可權規則 (permission.rule.N)",
      "permissionFormat": "格式：",
      "permissionExample": "示例：",
      "permissionAddRule": "新增 rule: create setting with key ",
      "addManagedSetting": "新增 Managed Setting",
      "settingKey": "設定鍵",
      "keyPlaceholder": "例如：allowed_models, disable_live_trading, permission.rule.1",
      "valuePlaceholder": "例如：claude-sonnet-5,deepseek-v4"
    },
    "billing": {
      "columns": {
        "user": "使用者",
        "plan": "方案",
        "status": "狀態",
        "cycle": "週期",
        "price": "價格",
        "autoRenew": "自動續費",
        "periodStart": "週期開始",
        "periodEnd": "週期結束",
        "createdAt": "建立時間",
        "type": "型別",
        "amount": "金額",
        "balanceBefore": "餘額 Before",
        "balanceAfter": "餘額 After",
        "description": "描述",
        "time": "時間"
      },
      "title": "計費管理",
      "monthlyRevenue": "月度收入",
      "totalRevenue": "總收入",
      "activeSubs": "活躍訂閱",
      "txRecords": "交易記錄",
      "planRevenue": "方案收入明細",
      "activeCount": "活躍",
      "subscriptions": "訂閱",
      "filterByPlan": "按方案篩選",
      "planFree": "免費",
      "planPro": "專業版",
      "planEnterprise": "企業版",
      "filterByStatus": "按狀態篩選",
      "statusActive": "活躍",
      "statusCancelled": "已取消",
      "statusExpired": "已過期",
      "walletTransactions": "錢包交易",
      "filterByType": "篩選 by type",
      "txPurchase": "購買",
      "txSale": "銷售",
      "txPlatformFee": "平臺費用",
      "txDeposit": "充值",
      "txWithdrawal": "提現"
    },
    "logs": {
      "columns": {
        "time": "時間",
        "module": "模組",
        "actionType": "操作型別",
        "target": "目標",
        "status": "狀態",
        "ip": "IP地址",
        "action": "操作",
        "details": "詳情"
      },
      "modules": {
        "userManagement": "使用者 管理",
        "accountManagement": "賬戶管理",
        "trading": "交易",
        "systemConfig": "系統配置"
      },
      "errors": {
        "loadFailed": "失敗 to load logs"
      },
      "actions": {
        "create": "建立",
        "update": "更新",
        "delete": "刪除",
        "disable": "禁用",
        "enable": "啟用",
        "freeze": "凍結",
        "unfreeze": "解凍"
      },
      "title": "操作日誌",
      "filterModule": "按模組篩選",
      "filterAction": "篩選 by action"
    },
    "deposit": {
      "table": {
        "user": "使用者",
        "amount": "USDT 金額",
        "amountUsd": "USD 到賬",
        "txHash": "交易雜湊",
        "status": "狀態",
        "reviewNote": "稽核備註",
        "time": "時間",
        "action": "操作"
      },
      "approved": "充值 approved and wallet credited.",
      "approveFailed": "失敗 to approve deposit.",
      "rejected": "充值 rejected.",
      "rejectFailed": "失敗 to reject deposit.",
      "approve": "透過",
      "reject": "拒絕",
      "title": "充值管理",
      "allStatuses": "全部狀態",
      "statusPending": "待處理",
      "statusApproved": "已透過",
      "statusRejected": "已拒絕",
      "approveTitle": "Approve 充值",
      "rejectTitle": "Reject 充值",
      "reviewNoteLabel": "稽核備註 (optional)",
      "reviewNotePlaceholder": "新增 a note for this review...",
      "approveWarning": "透過後使用者錢包將立即到賬。"
    },
    "wallet": {
      "errors": {
        "noUserSelected": "未選擇使用者"
      },
      "messages": {
        "adjustSuccess": "餘額 adjusted successfully",
        "adjustFailed": "調整失敗"
      },
      "columns": {
        "walletNumber": "錢包號",
        "email": "郵箱",
        "nickname": "暱稱",
        "type": "型別",
        "amount": "金額",
        "balanceAfter": "餘額 After",
        "description": "描述",
        "time": "時間",
        "balance": "餘額",
        "frozen": "凍結",
        "currency": "幣種"
      },
      "accountNumber": "錢包號",
      "add": "增加",
      "adjustBalance": "調整餘額",
      "adjustFailed": "調整失敗",
      "adjustSuccess": "餘額已調整",
      "deduct": "扣除",
      "noUsers": "未找到使用者",
      "reason": "調整原因...",
      "searchPlaceholder": "搜尋郵箱或錢包號...",
      "title": "錢包管理",
      "walletFor": "錢包 -",
      "unassigned": "未分配",
      "userList": "使用者列表",
      "noMatch": "無匹配使用者",
      "walletDetail": "錢包詳情",
      "transactions": "交易記錄",
      "adjustReason": "原因"
    },
    "header": {
      "admin": "管理",
      "adminMode": "管理員模式",
      "adminPanel": "管理後臺",
      "backToUser": "返回使用者端",
      "logout": "退出登入"
    },
    "sidebar": {
      "accountManagement": "賬戶管理",
      "agentSettings": "Agent 設定",
      "aiGateway": "AI 閘道器",
      "billing": "計費管理",
      "dashboard": "儀表盤",
      "deposits": "充值管理",
      "jurisdiction": "管轄權管理",
      "monitoring": "監控與告警",
      "operationLogs": "操作日誌",
      "shareManagement": "分享分析",
      "sre": "SRE 控制",
      "strategies": "策略管理",
      "systemConfig": "系統配置",
      "tradingMonitor": "交易監控",
      "userManagement": "使用者管理",
      "walletManagement": "錢包管理",
      "sweep": "歸集管理",
      "autogenTasks": "AI 生成任務",
      "marketplace": "市場管理",
      "refunds": "退款管理",
      "analytics": "資料分析",
      "coupons": "優惠券管理"
    },
    "trading": {
      "accounts": "賬戶",
      "activeUsers": "活躍使用者",
      "byPlatform": "按平臺",
      "closedOrders": "已平倉",
      "connectedAccounts": "已連線",
      "loadFailed": "載入交易統計失敗",
      "netProfit": "淨利潤",
      "orders": "訂單",
      "pendingOrders": "掛單",
      "platform": "平臺",
      "profitStats": "盈虧統計",
      "title": "交易監控",
      "totalAccounts": "總賬戶數",
      "totalLoss": "總虧損",
      "totalOrders": "總訂單",
      "totalProfit": "總盈利",
      "totalUsers": "總使用者數",
      "totalVolume": "總交易量",
      "volume": "數量"
    },
    "walletCalculator": {
      "title": "Token ↔ USD計算器",
      "selectModel": "選擇模型（定價基準）",
      "usdAmount": "USD 金額",
      "tokenAmount": "Token 金額",
      "fillResult": "填入結果"
    }
  },
  "autoTrading": {
    "logs": {
      "columns": {
        "action": "操作",
        "price": "價格",
        "profit": "盈虧",
        "symbol": "品種",
        "ticket": "單號",
        "time": "時間",
        "volume": "數量"
      },
      "empty": "暫無交易日誌",
      "title": "最近交易日誌"
    },
    "messages": {
      "loadFailed": "載入自動交易資料失敗",
      "toggleFailed": "切換自動交易失敗"
    },
    "settings": {
      "maxDailyLoss": "每日最大虧損",
      "maxDailyLossHint": "日虧損超過此值時自動停止交易",
      "maxDrawdownPercent": "最大回撤%",
      "maxDrawdownPercentHint": "回撤超過此值時自動停止交易",
      "maxLotSize": "最大手數",
      "maxLotSizeHint": "每筆交易最大交易量(手)",
      "maxPositions": "最大持倉數",
      "maxPositionsHint": "同時持有的最大倉位數量",
      "maxRiskPercent": "最大風險%",
      "maxRiskPercentHint": "每筆交易風險佔餘額百分比",
      "saveFailed": "儲存設定失敗",
      "saveSuccess": "設定已儲存",
      "title": "全域性風控設定"
    },
    "status": {
      "activeStrategies": "活躍策略",
      "disabled": "自動交易已關閉",
      "enabled": "自動交易已開啟",
      "todayExecutions": "今日成交",
      "todayProfit": "今日盈虧"
    },
    "title": "自動交易"
  },
  "notifications": {
    "stream": {
      "autoTrading": {
        "fallback": "自動交易事件觸發",
        "title": "自動交易"
      },
      "riskAlert": {
        "fallback": "警報型別: {{alertType}}",
        "title": "風控告警"
      },
      "strategyExecution": {
        "completed": "{{symbol}} {{action}} 已完成",
        "failed": "執行失敗: {{error}}",
        "title": "策略執行"
      },
      "strategySignal": {
        "message": "{{symbol}} 觸發 {{signalType}}",
        "title": "策略訊號"
      }
    },
    "actions": {
      "clearAll": "清空全部",
      "clearAllConfirm": "確定清空所有通知？",
      "markAllAsRead": "全部已讀"
    },
    "tabs": {
      "all": "全部 ({{count}})",
      "unread": "未讀 ({{count}})"
    },
    "types": {
      "risk_alert": "風控告警",
      "signal": "訊號",
      "strategy_execution": "策略執行",
      "system": "系統",
      "trade": "交易"
    },
    "all": "全部",
    "clearAll": "清空全部",
    "confirmClearAll": "確定清空所有通知？",
    "empty": "暫無通知",
    "markAllRead": "全部已讀",
    "title": "通知中心",
    "unread": "未讀"
  },
  "wallet": {
    "deposit": {
      "table": {
        "amount": "USDT 金額",
        "amountUsd": "USD 到賬",
        "status": "狀態",
        "time": "時間",
        "txHash": "交易雜湊",
        "confirmations": "確認數"
      },
      "address": "收款地址",
      "addressCopied": "地址已複製到剪貼簿",
      "amountLabel": "USDT 金額",
      "button": "新建充值",
      "copy": "複製",
      "exchangeRate": "匯率",
      "failed": "提交充值請求失敗。",
      "history": "充值記錄",
      "modalTitle": "提交充值請求",
      "network": "網路",
      "notConfigured": "USDT 充值尚未配置，請聯絡客服。",
      "notice": "請僅透過指定網路傳送 USDT。傳送其他代幣或使用不同網路可能導致永久丟失。",
      "submit": "提交",
      "success": "充值請求已提交，您的充值將自動確認。",
      "title": "充值",
      "txHashLabel": "交易雜湊（可選）",
      "willCredit": "預計到賬"
    },
    "table": {
      "amount": "金額",
      "balanceAfter": "調整後餘額",
      "description": "描述",
      "time": "時間",
      "type": "型別"
    },
    "txType": {
      "adjustment": "餘額調整",
      "deposit": "充值",
      "fee": "手續費",
      "reversal": "衝正",
      "withdrawal": "提取"
    },
    "passkey": {
      "title": "通行金鑰管理",
      "add": "新增通行金鑰",
      "name": "名稱",
      "credentialId": "憑證 ID",
      "signCount": "簽名次數",
      "createdAt": "建立時間",
      "confirmRemove": "確認刪除此通行金鑰？",
      "register": "註冊",
      "registered": "通行金鑰註冊成功",
      "registerFailed": "註冊失敗",
      "registerHint": "為此通行金鑰輸入名稱，然後點選註冊開始 WebAuthn 流程。",
      "namePlaceholder": "例如：我的 YubiKey",
      "removed": "通行金鑰已刪除"
    },
    "withdraw": {
      "title": "提取",
      "new": "新建提取",
      "submit": "提交",
      "available": "可用餘額",
      "amount": "金額",
      "amountLabel": "提取金額 (USDT)",
      "amountRequired": "請輸入金額",
      "destAddress": "目標地址",
      "destLabel": "目標 TRC20 地址",
      "destRequired": "請輸入目標地址",
      "whitelist": "白名單（點選填充）",
      "status": "狀態",
      "txHash": "交易雜湊",
      "time": "時間",
      "cancelled": "提取已取消",
      "confirmCancel": "確認取消此提取？",
      "success": "提取提交成功",
      "failed": "提取失敗",
      "noBalance": "無可用餘額可提取",
      "warning": "提取需要通行金鑰驗證。請確保目標地址正確 — 區塊鏈交易不可逆。"
    },
    "whitelist": {
      "title": "白名單管理",
      "add": "新增地址",
      "added": "白名單地址已新增",
      "removed": "白名單地址已刪除",
      "label": "標籤",
      "address": "地址",
      "status": "狀態",
      "confirmedAt": "確認時間",
      "confirmRemove": "確認刪除此白名單地址？",
      "addressLabel": "TRC20 地址",
      "addressRequired": "請輸入地址",
      "labelLabel": "標籤（可選）",
      "labelPlaceholder": "例如：我的幣安錢包"
    },
    "accountNumber": "錢包號",
    "balance": "餘額",
    "currency": "幣種",
    "frozen": "凍結",
    "frozenBalance": "凍結",
    "history": "歷史記錄",
    "title": "我的錢包",
    "transactions": "交易記錄"
  },
  "strategy": {
    "workspace": {
      "chartIndicators": {
        "overlay": "主圖疊加",
        "subPane": "副圖指標"
      }
    },
    "tuning": {
      "searchMethod": {
        "grid": "網格",
        "random": "隨機"
      }
    },
    "backtest": {
      "canceled": "回測已取消",
      "lotSize": "手數",
      "strategyParameters": "策略 Parameters"
    },
    "chat": {
      "executionPlan": "Execution 方案",
      "codeGenerated": "程式碼已生成，使用下方按鈕進行策略審查和回測。"
    },
    "aiChat": {
      "historyTab": "歷史",
      "strategiesTab": "策略"
    },
    "templates": {
      "title": "策略 Templates",
      "saveCurrent": "儲存 Current 策略",
      "lines": "條數",
      "chatEdit": "Chat 編輯",
      "source": "來源",
      "rename": "重新命名",
      "confirmDelete": "刪除 this strategy?",
      "noTemplates": "無已儲存策略模板",
      "sourceCode": "策略 Source",
      "copyAll": "複製 All"
    },
    "live": {
      "stopSuccess": "策略 stopped",
      "stopFailed": "失敗 to stop",
      "runId": "執行 ID",
      "account": "賬戶",
      "symbol": "品種",
      "timeframe": "週期",
      "mode": "模式",
      "signals": "訊號",
      "errors": "錯誤",
      "startedAt": "已啟動",
      "watchSignals": "Watch 訊號",
      "confirmStop": "確定停止此策略？",
      "status": "狀態",
      "totalSignals": "總計 訊號",
      "stoppedAt": "已停止",
      "error": "錯誤",
      "title": "實盤策略監控",
      "activeTab": "活躍執行",
      "noActive": "無活躍策略",
      "historyTab": "執行歷史",
      "noRuns": "無策略執行記錄",
      "schedulesTab": "排程",
      "time": "時間",
      "signalType": "型別",
      "volume": "交易量",
      "price": "價格",
      "sl": "SL",
      "tp": "TP",
      "reason": "原因",
      "signalLog": "訊號日誌",
      "waitingSignals": "等待訊號..."
    },
    "schedule": {
      "maxPositionsPlaceholder": "不限"
    },
    "ai": {
      "reviseHint": "先編寫程式碼，然後讓AI最佳化。",
      "explainHint": "編寫程式碼以檢視AI解釋。",
      "settingsHint": "配置 AI 提供商和模型"
    },
    "validate": {
      "running": "校驗執行中...",
      "errors": "錯誤",
      "warnings": "警告",
      "fixWithAI": "提交錯誤至 AI 修正",
      "parameters": "引數",
      "hints": "建議",
      "allClear": "所有檢查透過 — 未發現問題。",
      "passed": "Validation passed — 儲存 is now unlocked."
    },
    "importEA": {
      "writeTab": "策略 Code",
      "importTab": "匯入EA",
      "codeTooShort": "請貼上完整的EA/指標原始碼。",
      "pastePlaceholder": "貼上MQL4/MQL5 EA程式碼...",
      "migration": "策略匯入",
      "aiTranslate": "AI 翻譯",
      "bridge": "盲區橋接",
      "analyze": "分析策略結構",
      "confirmImport": "確認匯入",
      "tryAI": "AI 翻譯補充",
      "apply": "應用到編輯器",
      "importSuccess": "MQL 原始碼已匯入，點選「Apply to Editor」寫入編輯器",
      "hint": "貼上MQL4/MQL5程式碼並點選分析",
      "translate": "翻譯為Go",
      "translating": "AI翻譯中...",
      "bridgeBtn": "盲區橋接翻譯",
      "bridgeSuccess": "橋接成功",
      "bridgeFailedTag": "橋接失敗",
      "bridging": "AI 正在橋接盲區…",
      "bridgeFailedMsg": "Agent 無法自動橋接所有盲區",
      "noBridgeNeeded": "覆蓋率 100%，無需橋接",
      "bridgeHint": "貼上 MQL4/MQL5 EA 程式碼，AI 將自動翻譯盲區為 Python 子集"
    },
    "version": {
      "loadFailed": "失敗 to load versions",
      "rollbackFailed": "回滾失敗",
      "loadVersionFailed": "失敗 to load version",
      "loadDiffFailed": "失敗 to load diff",
      "colVersion": "版本",
      "colSummary": "變更摘要",
      "colLang": "語言",
      "colHash": "雜湊",
      "colDate": "日期",
      "colActions": "操作",
      "title": "Version 歷史",
      "diff": "差異",
      "empty": "暫無版本歷史",
      "history": "Version 歷史"
    }
  },
  "accounts": {
    "bind": {
      "fields": {
        "alias": "賬戶 Alias"
      },
      "placeholders": {
        "alias": "可選自定義名稱"
      },
      "messages": {
        "changeCredentials": "修改憑證"
      }
    },
    "messages": {
      "shareLinkCopied": "分享連結已複製到剪貼簿",
      "shareLinkFailed": "失敗 to create share link"
    }
  },
  "sre": {
    "breakers": {
      "columns": {
        "strategyId": "策略 ID",
        "state": "狀態",
        "totalPnl": "總盈虧",
        "lossPercent": "虧損率",
        "tradeCount": "交易數",
        "trippedAt": "熔斷時間",
        "tripReason": "熔斷原因"
      },
      "title": "策略斷路器",
      "stateClosed": "正常",
      "stateOpen": "已熔斷",
      "stateHalfOpen": "半開（探測中）",
      "confirmReset": "重置此斷路器？",
      "description": "策略 breaker status overview — auto-detects abnormal losses and trips",
      "noBreakers": "無已註冊斷路器"
    },
    "canary": {
      "columns": {
        "strategyId": "策略 ID",
        "versionTag": "版本標籤",
        "accounts": "金絲雀賬戶",
        "startAt": "開始時間",
        "days": "天數",
        "status": "狀態"
      },
      "promoted": "已晉升",
      "canarying": "金絲雀",
      "confirmDelete": "刪除 this canary config?",
      "title": "金絲雀 Configuration",
      "description": "新策略版本先在少量賬戶上執行N天，再晉升至全部",
      "newCanary": "新建金絲雀",
      "noCanaries": "無金絲雀配置",
      "newCanaryTitle": "新建金絲雀",
      "accountIdsLabel": "金絲雀 賬戶 IDs (comma or newline separated)",
      "durationDays": "金絲雀 天數"
    },
    "killSwitch": {
      "description": "一鍵停止所有交易 — 需要輸入 KILL 確認；5 分鐘內可撤銷",
      "engaged": "熔斷開關 engaged — all trading stopped",
      "disarmed": "熔斷開關 disarmed — trading normal",
      "status": "狀態",
      "reason": "原因",
      "operator": "操作人",
      "engagedAt": "啟用時間",
      "undo": "Undo 熔斷開關",
      "disengage": "Disengage 熔斷開關",
      "engage": "啟用熔斷開關",
      "confirmTitle": "啟用 熔斷開關 — Confirmation",
      "confirmEngage": "確認 啟用",
      "confirmWarning": "此操作將立即停止所有賬戶的所有交易活動，包括掛單和已提交訂單。輸入原因並鍵入 KILL 確認。",
      "reasonLabel": "原因（必填）",
      "reasonPlaceholder": "例如：檢測到市場異常波動，緊急停止所有交易",
      "typeKill": "鍵入 KILL 確認",
      "typeKillPlaceholder": "鍵入 KILL（大寫）",
      "undoWindow": "撤銷視窗: {{minutes}}分 {{seconds}}秒 剩餘",
      "title": "熔斷開關"
    }
  },
  "marketplace": {
    "publish": {
      "priceModel": {
        "free": "免費",
        "monthly": "按月訂閱",
        "once": "One-時間 Purchase",
        "label": "定價方式"
      },
      "assetClass": {
        "label": "資產類別"
      },
      "riskLevel": {
        "label": "風險等級"
      },
      "return": "收益率",
      "winRate": "勝率",
      "trades": "交易數",
      "title": "釋出到市場",
      "titleLabel": "標題",
      "titlePlaceholder": "e.g. Golden Cross 策略",
      "descriptionLabel": "描述",
      "descriptionPlaceholder": "描述策略邏輯、開平倉規則...",
      "priceAmount": "金額",
      "tags": "標籤",
      "tagsPlaceholder": "輸���後按回車新增標籤",
      "codeSnippet": "策略 Preview (public)",
      "codeSnippetPlaceholder": "可選：分享策略程式碼片段或思路（所有人可見）",
      "includeBacktestSnapshot": "包含最新回測結果"
    },
    "author": {
      "avgRating": "平均評分",
      "empty": "暫無已釋出策略。前往策略庫釋出一個。",
      "published": "已釋出",
      "myStrategies": "My Published 策略",
      "publishNew": "Publish New 策略",
      "monthlyRevenue": "月度收入",
      "totalRevenue": "總收入",
      "goToLibrary": "Go to 策略 Library"
    },
    "card": {
      "by": "由",
      "free": "免費",
      "owned": "購買日期",
      "subscribers": "訂閱者",
      "winRate": "勝率",
      "yourStrategy": "Your 策略"
    },
    "detail": {
      "assetClass": "資產類別",
      "author": "作者",
      "commentPlaceholder": "寫評論...",
      "comments": "評論",
      "description": "描述",
      "getFree": "免費獲取",
      "rentPrice": "¥{{amount}} / 月",
      "subscribers": "訂閱者",
      "yourRating": "我的評分",
      "runBacktest": "執行回測"
    },
    "messages": {
      "commentFailed": "評論失敗",
      "commentPosted": "評論已釋出",
      "loginFirst": "請先登入",
      "paymentComingSoon": "支付功能即將上線",
      "rateFailed": "評分失敗",
      "rated": "評分已提交",
      "subscribeFailed": "失敗",
      "subscribed": "已新增到您的購買",
      "published": "策略 published to marketplace!",
      "publishFailed": "失敗 to publish strategy"
    },
    "payment": {
      "alreadyPurchased": "您已擁有此策略。",
      "balanceAfter": "購買後餘額",
      "cancel": "取消",
      "confirm": "確認購買",
      "depositPrompt": "請先充值後再繼續。",
      "goToDeposit": "充值",
      "insufficientBalance": "餘額不足",
      "oneTimePurchase": "¥{{amount}} 一次性買斷",
      "price": "價格",
      "purchaseFailed": "購買失敗，請重試。",
      "purchaseSuccess": "購買成功！策略已新增到您的庫中。",
      "purchasing": "處理中...",
      "strategyName": "策略",
      "title": "確認購買",
      "walletBalance": "我的餘額"
    },
    "purchases": {
      "empty": "暫無購買記錄。前往市場發現策略。",
      "status": "狀態",
      "strategy": "策略",
      "runBacktest": "執行回測"
    },
    "sort": {
      "newest": "最新",
      "performance": "最佳表現",
      "popular": "最熱門",
      "priceAsc": "價格：從低到高",
      "priceDesc": "價格：從高到低",
      "rating": "最高評分",
      "score": "綜合評分"
    },
    "tabs": {
      "author": "作者中心",
      "marketplace": "策略市場",
      "purchases": "我的購買",
      "subscriptions": "我的訂閱"
    },
    "backtest": {
      "title": "策略 Backtest",
      "capital": "資金",
      "commission": "佣金",
      "leverage": "槓桿",
      "completed": "已完成",
      "totalReturn": "總計 Return",
      "maxDrawdown": "最大回撤",
      "sharpe": "夏普比率",
      "winRate": "勝率",
      "totalTrades": "總計 交易數",
      "equityCurve": "權益曲線",
      "protected": "策略 code is protected. Backtest runs on our servers.",
      "run": "執行回測",
      "idle": "設定引數並執行回測"
    },
    "empty": "暫無已釋出策略",
    "filterByClass": "按資產類別篩選",
    "noSubscriptions": "暫無訂閱",
    "searchPlaceholder": "搜尋策略...",
    "subtitle": "發現、購買和使用社群策略",
    "title": "策略市場"
  },
  "onboarding": {
    "step1": {
      "title": "連線您的賬戶",
      "desc": "繫結您的 MT4/MT5 交易賬戶以開始。",
      "action": "Bind 賬戶"
    },
    "step2": {
      "title": "建立您的第一個策略",
      "desc": "使用 AI 從自然語言生成交易策略。",
      "action": "開啟工作區"
    },
    "step3": {
      "title": "升級您的計劃",
      "desc": "解鎖更多 AI 代幣、策略和實盤交易功能。",
      "action": "檢視方案"
    },
    "subtitle": "3 個簡單步驟即可開始",
    "dismiss": "知道了，忽略"
  },
  "auth": {
    "fields": {
      "confirmPassword": "確認密碼",
      "email": "郵箱",
      "password": "密碼",
      "login": "郵箱/賬號"
    },
    "forgotPassword": {
      "backToLogin": "返回登入",
      "hint": "請聯絡管理員或支援人員重置密碼。",
      "title": "重置密碼"
    },
    "login": {
      "forgotPassword": "忘記密碼？",
      "login": "立即登入",
      "noAccount": "沒有賬戶？",
      "registerNow": "立即註冊",
      "rememberMe": "記住我",
      "signingIn": "登入中...",
      "subtitle": "這是一個測試不具備責任能力"
    },
    "messages": {
      "fetchMeFailed": "載入使用者資訊失敗",
      "loginFailed": "登入失敗，請檢查郵箱和密碼",
      "loginSuccess": "登入成功",
      "logoutSuccess": "已退出登入",
      "registerFailed": "註冊失敗，請稍後重試",
      "registerSuccess": "註冊成功，請登入"
    },
    "register": {
      "haveAccount": "已有賬號？",
      "loginNow": "立即登入",
      "register": "註冊",
      "signingUp": "註冊中...",
      "subtitle": "建立新賬號"
    },
    "validation": {
      "confirmPasswordRequired": "請確認密碼",
      "emailInvalid": "請輸入有效的郵箱地址",
      "emailRequired": "請輸入郵箱",
      "passwordMin8": "密碼至少8位",
      "passwordMismatch": "兩次密碼不一致",
      "passwordRequired": "請輸入密碼",
      "loginRequired": "請輸入郵箱或賬號"
    }
  },
  "common": {
    "months": {
      "jan": "1月",
      "jul": "7月"
    },
    "time": {
      "day": "{{n}}天",
      "hour": "{{n}}時",
      "lessThanMinute": "<1分鐘",
      "minute": "{{n}}分"
    },
    "active": "正常",
    "back": "返回",
    "cancel": "取消",
    "clear": "清除",
    "close": "平倉時間",
    "comingSoon": "即將上線",
    "confirm": "確定",
    "copied": "已複製",
    "copy": "複製",
    "copyFailed": "複製失敗",
    "create": "新增",
    "created": "建立時間",
    "currentPosition": "📊 當前持倉",
    "delete": "刪除",
    "deleteFailed": "刪除失敗",
    "deleteSelected": "刪除選中 ({{count}})",
    "deleted": "已刪除",
    "disable": "禁用",
    "disabled": "已禁用",
    "edit": "編輯",
    "enable": "啟用",
    "enabled": "已啟用",
    "error": "錯誤",
    "gotIt": "我知道了",
    "hideDetails": "收起詳情",
    "inactive": "停用",
    "indicatorSettings": "{{name}} 設定",
    "lineColor": "線顏色",
    "loading": "載入中...",
    "loadingFailed": "載入失敗",
    "next": "下一步",
    "no": "否",
    "noData": "暫無資料",
    "noOpenPositionsForSymbol": "{{symbol}} 暫無持倉",
    "none": "無",
    "ok": "確定",
    "operationFailed": "操作失敗",
    "pageError": "頁面錯誤",
    "pageUnderDevelopment": "此頁面開發中",
    "pleaseWait": "請稍候...",
    "previous": "上一步",
    "refresh": "重新整理",
    "remove": "移除",
    "required": "必填",
    "retry": "重試",
    "save": "儲存",
    "saveFailed": "儲存失敗",
    "saveSuccess": "儲存成功",
    "searching": "搜尋中...",
    "selectSymbolToViewChart": "選擇品種檢視圖表",
    "send": "傳送",
    "showDetails": "檢視詳情",
    "totalItems": "共 {{count}} 項",
    "translate": "翻譯",
    "unexpectedError": "發生了意外錯誤",
    "unknown": "未知",
    "updated": "已更新",
    "viewOriginal": "檢視原文",
    "viewTranslation": "檢視譯文",
    "yes": "是",
    "you": "你",
    "unsaved": "未儲存",
    "saved": "已儲存",
    "unknownError": "未知錯誤",
    "duplicateName": "名稱已存在",
    "step1Label": "經紀商",
    "step2Label": "憑證",
    "step3Label": "確認",
    "unit": "單位",
    "action": "操作",
    "on": "開",
    "off": "關",
    "true": "是",
    "false": "否",
    "success": "成功",
    "failed": "失敗",
    "reset": "重置",
    "saving": "儲存中…",
    "total": "共 {{total}}"
  },
  "errors": {
    "ai": {
      "api_key_required": "API Key 不能為空",
      "base_url_required": "Base URL 不能為空",
      "base_url_scheme_invalid": "Base URL 必須以 http:// 或 https:// 開頭",
      "base_url_should_not_end_with_chat_completions": "Base URL 不應以 /chat/completions 結尾",
      "config_service_not_initialized": "AI 配置服務未初始化",
      "config_valid": "AI 配置有效",
      "failed_to_create_request": "建立請求失敗",
      "forbidden_quota": "配額超限",
      "free_tier_exhausted": "AI 模型免費額度已耗盡：請在模型供應商管理後臺關閉“use free tier only”或更換付費 Key。",
      "invalid_base_url": "Base URL 無效",
      "invalid_provider": "服務商無效",
      "no_trade_data_available": "暫無可用交易資料",
      "not_configured": "AI 未配置：請先到 AI 設定中啟用並配置。",
      "probe_ok": "正常",
      "probe_ok_no_models": "正常（未返回 models）",
      "provider_required": "請先選擇服務商",
      "provider_returned_empty_message": "AI 服務返回空訊息",
      "rate_limited": "AI 服務觸發限流/額度不足（429/資源耗盡）。請稍後重試或更換可用的 API Key/模型配置。",
      "request_failed": "API 請求失敗",
      "insufficient_balance_title": "Insufficient 餘額",
      "insufficient_balance": "AI錢包餘額不足，請充值後繼續。"
    },
    "connection_failed": {
      "content": "無法連線到伺服器，請檢查網路後重試。",
      "title": "連線失敗"
    },
    "access_denied": "無許可權訪問",
    "account_connected": "連線成功",
    "account_connection_failed": "無法連線到交易伺服器",
    "account_not_found": "賬戶不存在",
    "auto_trading_disabled": "自動交易已關閉",
    "auto_trading_enabled": "自動交易已開啟",
    "email_already_registered": "郵箱已註冊",
    "invalid_credentials": "賬號或密碼錯誤",
    "not_authenticated": "未登入",
    "schedule_service_not_available": "排程服務不可用",
    "translate_failed": "翻譯失敗",
    "user_not_found": "使用者不存在"
  },
  "symbolDetection": {
    "tradeMode": {
      "disabled": "已禁用",
      "longOnly": "僅做多",
      "longShort": "多空雙向",
      "shortOnly": "僅做空",
      "unknown": "未知"
    },
    "label": "識別到的交易品種",
    "loading": "正在解析…",
    "noSymbols": "未識別到交易品種，請嘗試包含具體的品種名稱（如\"比特幣\"、\"EURUSD\"、\"黃金\"）",
    "resolvedTooltip": "broker: {{broker}} | 模式: {{mode}}",
    "unresolvedTooltip": "尚未繫結交易賬戶，無法解析"
  },
  "subscription": {
    "feature": {
      "aiTokens": "{{count}} AI Token/月",
      "strategies": "{{count}} 個策略",
      "backtests": "{{count}} 次回測/天",
      "liveStrategies": "{{count}} 個實盤策略",
      "symbols": "{{count}} 個品種/策略"
    },
    "title": "訂閱方案",
    "subscribeSuccess": "訂閱啟用成功！",
    "charged": "已扣費: {{amount}}, 餘額: {{balance}}",
    "insufficientBalance": "錢包餘額不足，請先充值。",
    "subscribeFailed": "訂閱失敗，請重試。",
    "cancelSuccess": "自動續訂已取消。您的訂閱在當前週期結束前仍然有效。",
    "cancelFailed": "取消失敗，請重試。",
    "changeSuccess": "方案切換成功！",
    "changeFailed": "方案切換失敗，請重試。",
    "billingCycle": "計費",
    "autoRenew": "自動續訂",
    "period": "當前週期",
    "cancelAutoRenew": "取消自動續訂",
    "usageTitle": "本月使用量",
    "aiTokens": "AI 代幣",
    "activeStrategies": "活躍策略",
    "runtimeMinutes": "執行時長（分鐘）",
    "walletBalance": "錢包餘額",
    "month": "月",
    "year": "年",
    "freeForever": "永久免費",
    "currentPlan": "當前方案",
    "choosePlan": "選擇方案",
    "noPlans": "暫無可用方案",
    "changePlanTitle": "切換方案",
    "subscribeTitle": "訂閱方案",
    "selectBillingCycle": "計費週期",
    "monthly": "月付",
    "yearly": "年付",
    "chargeNotice": "付費方案將從錢包扣款。免費方案不扣費。"
  },
  "agent": {
    "analysis": {
      "title": "回測分析",
      "sharpe": "夏普",
      "drawdown": "最大回撤",
      "winrate": "勝率",
      "consistency": "一致性",
      "risk_adj": "風險調整收益",
      "overfitting": "過擬合風險",
      "observations": "關鍵觀察",
      "suggestions": "改進建議",
      "detailed": "詳細分析"
    },
    "semantic_diff": {
      "title": "策略 Changes",
      "effect": "影響"
    },
    "profile": {
      "title": "策略 Profile",
      "timeframe": "時間週期",
      "regime": "市場狀態",
      "indicators": "指標",
      "entry": "入場",
      "exit": "出場",
      "risk": "Risk 管理",
      "coverage": "覆蓋範圍",
      "strengths": "優勢",
      "weaknesses": "劣勢",
      "blind_spots": "盲點"
    }
  },
  "importAnalysis": {
    "execution": {
      "onBar": "K線收盤驅動",
      "onTick": "逐筆驅動",
      "onInitGrid": "初始化網格"
    },
    "sizing": {
      "fixed": "固定手數",
      "martingale": "馬丁格爾",
      "percentBalance": "餘額百分比"
    },
    "analyzing": "正在分析策略結構...",
    "tradeLogicComplete": "交易邏輯已全部識別",
    "guiNoiseDesc": "以下盲區屬於圖表顯示/按鈕功能，服務端執行時跳過，不影響交易結果。可以安全匯入。",
    "cannotImport": "無法自動匯入",
    "incompleteCoverage": "交易邏輯覆蓋不完整",
    "goodCoverage": "匯入覆蓋率良好",
    "goodCoverageDesc": "策略 main logic recognized. Safe to import. Check parameter list before use.",
    "coverageTitle": "匯入覆蓋率",
    "location": "位置",
    "handling": "處理方式",
    "userActionRequired": "需要您操作",
    "noBlindSpots": "無需確認邏輯",
    "noBlindSpotsDesc": "所有策略邏輯已自動識別，可以安全匯入。"
  },
  "dashboard": {
    "quickActions": {
      "aiStrategy": "AI 策略"
    }
  },
  "logs": {
    "triggerSource": {
      "manual": "手動",
      "strategy": "策略",
      "recovery": "恢復"
    },
    "result": {
      "pass": "PASS",
      "reject": "REJECT"
    }
  },
  "app": {
    "name": "AlphaForge"
  },
  "language": {
    "english": "English",
    "japanese": "日本語",
    "simplifiedChinese": "簡體中文",
    "traditionalChinese": "繁體中文",
    "vietnamese": "Tiếng Việt"
  },
  "market": {
    "allSymbols": "全部品種",
    "ask": "賣價",
    "bid": "買價",
    "common": "常用",
    "emptyWatchlist": "暫無自選",
    "loadingSymbols": "載入中...",
    "mid": "中間價",
    "mtSessionLost": "⚠ MT 會話丟失 — 正在重連…",
    "noSymbolSelected": "選擇一個品種以檢視行情資料",
    "noSymbolsFound": "未找到品種",
    "popularSymbols": "熱門品種",
    "searchPlaceholder": "搜尋品種（如 EURUSD, XAUUSD）",
    "searchSymbol": "搜尋品種...",
    "selectAccount": "選擇交易賬戶",
    "selectSymbol": "選擇品種",
    "spread": "點差",
    "watchlist": "自選"
  },
  "menu": {
    "accounts": "賬戶",
    "aiAssistant": "AI助手",
    "algoDashboard": "演算法看板",
    "analytics": "分析",
    "assetAnalysis": "AI分析",
    "assets": "策略資產",
    "autoTrading": "自動交易",
    "dashboard": "儀表盤",
    "devGroup": "策略開發",
    "experiments": "策略實驗",
    "indicatorCatalog": "指標目錄",
    "logs": "系統日誌",
    "market": "策略市場",
    "marketRegime": "市場狀態",
    "marketTools": "市場分析工具",
    "marketplace": "市場",
    "opsGroup": "策略運營",
    "schedules": "策略排程",
    "strategies": "策略管理",
    "strategy": "策略",
    "strategyLibrary": "策略庫",
    "strategyLive": "實盤監控",
    "strategyWorkspace": "策略工作臺",
    "subscription": "訂閱",
    "trading": "交易",
    "wallet": "錢包"
  },
  "profile": {
    "lastLogin": "最後登入",
    "nickname": "暱稱",
    "registered": "已註冊",
    "role": "角色",
    "status": "狀態",
    "title": "個人資訊"
  },
  "share": {
    "actions": "操作",
    "createNew": "建立新分享連結",
    "createdAt": "建立時間",
    "deleteConfirm": "刪除此分享連結？",
    "empty": "暫無分享連結",
    "expires": "過期時間",
    "positions": "持倉",
    "showPositions": "顯示持倉",
    "title": "分享管理",
    "token": "分享連結",
    "userId": "普通使用者",
    "views": "瀏覽量"
  },
  "sharePage": {
    "avgHolding": "平均持倉時長",
    "avgLoss": "平均虧損",
    "avgWin": "平均盈利",
    "bestTrade": "最佳交易",
    "bySymbol": "品種業績",
    "closeTime": "平倉時間",
    "count": "筆數",
    "disclaimer": "過往業績不代表未來表現。",
    "equityCurve": "淨值曲線",
    "expired": "該分享連結已過期",
    "footer": "由 AlphaForge 生成",
    "language": "語言",
    "loadFailed": "載入分享資料失敗",
    "losingTrades": "虧損筆數",
    "maxDrawdown": "最大回撤",
    "netProfit": "淨盈虧",
    "noPositions": "暫無持倉",
    "noTrades": "暫無交易記錄",
    "notFound": "未找到",
    "openPrice": "開倉價",
    "positions": "當前持倉",
    "positionsLocked": "建立者未開放持倉檢視",
    "profit": "盈虧",
    "profitFactor": "盈利因子",
    "sharpeRatio": "夏普比率",
    "side": "方向",
    "subtitle": "真實交易成績",
    "symbol": "品種",
    "title": "交易業績",
    "totalReturn": "淨盈虧",
    "totalTrades": "總交易數",
    "totalVolume": "總交易量",
    "tradeRecords": "交易記錄",
    "volume": "數量",
    "winRate": "勝率",
    "winningTrades": "盈利筆數",
    "worstTrade": "最差交易",
    "countUnit": "筆"
  },
  "topbar": {
    "logout": "退出登入",
    "profile": "個人資訊",
    "settings": "設定",
    "switchToAdmin": "切換到管理",
    "systemOk": "系統正常執行",
    "user": "普通使用者"
  },
  "theme": {
    "switchToDark": "切換到深色模式",
    "switchToLight": "切換到淺色模式"
  },
  "monitoring": {
    "unknown": "未知",
    "healthy": "正常",
    "title": "系統監控",
    "sseConnected": "SSE 已連線",
    "disconnected": "已斷開",
    "streamError": "Stream 錯誤",
    "waitingData": "等待資料...",
    "serviceHealth": "服務健康",
    "uptime": "執行時長",
    "database": "資料庫",
    "diskUsage": "磁碟使用",
    "goRuntime": "Go執行時",
    "goroutines": "協程數",
    "gcCount": "GC次數",
    "gcPauseAvg": "GC平均暫停",
    "stackUsage": "棧使用",
    "heapMemory": "堆記憶體",
    "dbPool": "資料庫連線池",
    "totalConns": "總計",
    "idle": "空閒",
    "acquired": "已獲取",
    "mdGateway": "行情閘道器",
    "spillFiles": "溢位檔案",
    "droppedBars": "丟棄 K 線",
    "droppedSignals": "丟棄訊號",
    "consumerLag": "消費者延遲",
    "staleAccounts": "過期賬戶",
    "deadAccounts": "死賬戶",
    "avgGapSec": "平均間隔 (秒)",
    "maxGapSec": "最大間隔 (秒)",
    "dlq": "死信佇列 (DLQ)",
    "parseErrors": "解析錯誤",
    "bidGtAsk": "買價>賣價",
    "nonPositive": "非正數",
    "pushInterval": "推送間隔：5秒",
    "lastUpdate": "最後更新",
    "uptimeDays": "{{d}}天 {{h}}小時",
    "uptimeHours": "{{h}}小時 {{m}}分",
    "uptimeMinutes": "{{m}}分 {{s}}秒",
    "uptimeSeconds": "{{s}}秒"
  }
} as const;
export default Base;
