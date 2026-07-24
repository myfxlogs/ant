// Auto-generated from proto/ant/v1/i18n/base_zh-tw.textproto
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
        "noRejectData": "此時段無拒絕記錄",
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
        "loadFailed": "載入儀錶板資料失敗"
      },
      "activeUsers": "活躍使用者",
      "loadFailed": "載入儀錶板資料失敗",
      "mtAccounts": "MT賬戶數",
      "onlineAccounts": "線上賬戶",
      "recentLogs": "最近日誌",
      "title": "管理儀錶板",
      "todayProfit": "今日盈虧",
      "todayTrades": "今日交易",
      "totalUsers": "總使用者數",
      "verifiedUsers": "已驗證使用者",
      "activeSubs": "有效訂閱",
      "monthlyRevenue": "月營收",
      "totalRevenue": "總營收",
      "marketStrategies": "市場策略",
      "marketSales": "市場銷售",
      "marketRevenue": "市場營收",
      "validateTotal": "驗證總數",
      "validatePass": "驗證透過",
      "validateReject": "驗證拒絕",
      "validateError": "驗證錯誤",
      "orderSendSuccess": "訂單傳送成功",
      "orderSendFailed": "訂單傳送失敗",
      "orderCloseSuccess": "訂單關閉成功",
      "orderCloseFailed": "訂單關閉失敗",
      "rejectCount": "拒絕計數"
    },
    "userManagement": {
      "drawer": {
        "labels": {
          "createdAt": "建立時間",
          "email": "電子郵件",
          "id": "ID",
          "lastLogin": "最後登入",
          "mtAccountCount": "MT賬戶數",
          "nickname": "暱稱",
          "role": "角色",
          "status": "狀態"
        },
        "title": "使用者詳細"
      },
      "form": {
        "placeholders": {
          "email": "輸入電子郵件",
          "nickname": "輸入暱稱",
          "password": "輸入密碼"
        },
        "accountNumber": "錢包號",
        "accountNumberInvalid": "5-6位數字，無前導零，不含4和7",
        "email": "電子郵件",
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
          "passwordMin8": "密碼至少8位元",
          "passwordMismatch": "兩次密碼不一致",
          "passwordMustContainLettersAndNumbers": "密碼必須包含字母和數字"
        },
        "confirmPassword": "確認密碼",
        "newPassword": "新密碼",
        "submit": "更新密碼"
      },
      "actions": {
        "changePassword": "修改密碼",
        "details": "詳細",
        "disable": "停用",
        "enable": "啟用"
      },
      "deleteConfirm": {
        "batchDeleteConfirm": "確認刪除 {{count}} 個使用者？此操作不可復原。",
        "batchDeletePartial": "已刪除 {{deleted}} 個，{{failed}} 個失敗",
        "batchDeleteSuccess": "已刪除 {{count}} 個使用者",
        "title": "確認刪除此使用者？此操作不可復原。"
      },
      "filters": {
        "rolePlaceholder": "按角色篩選",
        "searchPlaceholder": "搜尋電子郵件或暱稱",
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
        "userDisabled": "使用者已停用",
        "userEnabled": "使用者已啟用",
        "userUpdateFailed": "更新使用者失敗",
        "userUpdatedSuccess": "使用者更新成功",
        "loadUsersFailed": "載入使用者失敗"
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
        "operation": "營運",
        "superAdmin": "超級管理員",
        "user": "一般使用者"
      },
      "status": {
        "active": "正常",
        "suspended": "已停用"
      },
      "table": {
        "actions": "操作",
        "createdAt": "建立時間",
        "email": "電子郵件",
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
        "disabled": "已停用",
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
        "apiKeyRequired": "API Key不能為空白",
        "greenMaxFailedRunsNonNegative": "綠色最大失敗次數需≥0",
        "greenSuccessRateRange": "綠色成功率需在0-100之間",
        "jsonEmpty": "JSON不能為空白",
        "jsonInvalid": "JSON格式無效",
        "minSampleSizeNonNegative": "最小樣本量需≥0",
        "modelRequired": "模型名稱不能為空",
        "yellowNotGreaterThanGreen": "黃色閾值不能超過綠色閾值",
        "yellowSuccessRateRange": "黃色成功率需在0-100之間"
      },
      "aiProviderCatalog": "AI提供者目錄",
      "baseUrlLabel": "基礎 URL",
      "configItem": "配置專案",
      "description": "描述",
      "econAIConfig": "經濟日曆AI配置",
      "editConfig": "編輯配置: {{key}}",
      "enableToggle": "啟用",
      "fillTemplate": "填入範本",
      "formatJson": "格式化JSON",
      "maxAccountsPerUser": "每使用者最大賬戶數",
      "modelName": "模型名稱",
      "off": "關",
      "on": "開",
      "provider": "提供者",
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
      "pending": "待審核",
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
      "userEmail": "電子郵件",
      "userKYCStatus": "使用者KYC狀態",
      "verified": "已驗證"
    },
    "aiGateway": {
      "errors": {
        "loadProviders": "載入提供者失敗",
        "toggleFailed": "切換失敗",
        "loadModels": "載入模型失敗"
      },
      "columns": {
        "baseUrl": "基礎 URL",
        "apiKey": "API 金鑰"
      },
      "addProviderPending": "新增提供者功能待後端支援",
      "title": "AI 閘道管理",
      "description": "管理 AI 提供者、模型與定價。使用者從可用模型中選擇，按代幣從錢包計費。",
      "addProvider": "新增提供者",
      "provider": "提供者",
      "configured": "已設定",
      "notConfigured": "未設定",
      "models": "模型",
      "editProvider": "編輯提供者",
      "providerId": "提供者 ID",
      "providerIdRequired": "請輸入提供者 ID",
      "displayName": "顯示名稱",
      "displayNameRequired": "請輸入顯示名稱",
      "baseUrl": "基礎 URL",
      "baseUrlRequired": "請輸入基礎 URL",
      "apiKeyLabel": "API 金鑰",
      "apiKeyEditHint": "留空以保留現有金鑰",
      "apiKeyHint": "API 金鑰，儲存時加密",
      "apiKeyEditPlaceholder": "留空以保留",
      "editModel": "編輯模型",
      "addModel": "新增模型",
      "modelName": "模型名稱",
      "modelNameRequired": "請輸入模型名稱",
      "priceInput": "輸入價格 ($/1M)",
      "priceOutput": "輸出價格 ($/1M)",
      "confirmDeleteModel": "刪除此模型？",
      "noModels": "無模型"
    },
    "account": {
      "errors": {
        "loadFailed": "載入帳戶失敗",
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
      "frozen": "帳戶已凍結",
      "unfrozen": "帳戶已解凍",
      "detail": "詳情",
      "unfreeze": "解凍",
      "confirmFreeze": "凍結此帳戶？",
      "freeze": "凍結",
      "title": "帳戶管理",
      "searchPlaceholder": "搜尋帳戶",
      "status": "狀態",
      "online": "線上",
      "offline": "離線",
      "auditLogs": "稽核記錄"
    },
    "settings": {
      "columns": {
        "key": "設定鍵",
        "value": "值",
        "action": "操作"
      },
      "saveSuccess": "儲存成功",
      "saveFailed": "儲存失敗",
      "deleted": "已刪除",
      "deleteFailed": "刪除失敗",
      "actionFailed": "操作失敗",
      "confirmDelete": "確認刪除？",
      "title": "代理管理設定",
      "addSetting": "新增設定",
      "permissionRules": "許可權規則 (permission.rule.N)",
      "permissionFormat": "格式：",
      "permissionExample": "範例：",
      "permissionAddRule": "新增規則：使用鍵建立設定",
      "addManagedSetting": "新增管理設定",
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
        "autoRenew": "自動續訂",
        "periodStart": "週期開始",
        "periodEnd": "週期結束",
        "createdAt": "建立時間",
        "type": "型別",
        "amount": "金額",
        "balanceBefore": "交易前餘額",
        "balanceAfter": "交易後餘額",
        "description": "描述",
        "time": "時間"
      },
      "title": "帳單管理",
      "monthlyRevenue": "月營收",
      "totalRevenue": "總營收",
      "activeSubs": "有效訂閱",
      "txRecords": "交易記錄",
      "planRevenue": "方案營收明細",
      "activeCount": "有效數量",
      "subscriptions": "訂閱",
      "filterByPlan": "依方案篩選",
      "planFree": "免費",
      "planPro": "Pro",
      "planEnterprise": "企業版",
      "filterByStatus": "按狀態篩選",
      "statusActive": "啟用",
      "statusCancelled": "已取消",
      "statusExpired": "已過期",
      "walletTransactions": "錢包交易",
      "filterByType": "按型別篩選",
      "txPurchase": "購買",
      "txSale": "出售",
      "txPlatformFee": "平臺費用",
      "txDeposit": "存款",
      "txWithdrawal": "提款"
    },
    "logs": {
      "columns": {
        "time": "時間",
        "module": "模組",
        "actionType": "動作型別",
        "target": "目標",
        "status": "狀態",
        "ip": "IP 位址",
        "action": "操作",
        "details": "詳細資訊"
      },
      "modules": {
        "userManagement": "使用者管理",
        "accountManagement": "帳戶管理",
        "trading": "交易",
        "systemConfig": "系統設定"
      },
      "errors": {
        "loadFailed": "載入日誌失敗"
      },
      "actions": {
        "create": "建立",
        "update": "更新",
        "delete": "刪除",
        "disable": "停用",
        "enable": "啟用",
        "freeze": "凍結",
        "unfreeze": "解凍"
      },
      "title": "操作日誌",
      "filterModule": "依模組篩選",
      "filterAction": "依操作篩選"
    },
    "deposit": {
      "table": {
        "user": "使用者",
        "amount": "USDT 金額",
        "amountUsd": "美元信用額",
        "txHash": "交易雜湊",
        "status": "狀態",
        "reviewNote": "審核備註",
        "time": "時間",
        "action": "操作"
      },
      "approved": "存款已批准，錢包已入帳。",
      "approveFailed": "批准存款失敗。",
      "rejected": "存款已拒絕。",
      "rejectFailed": "拒絕存款失敗。",
      "approve": "核准",
      "reject": "拒絕",
      "title": "存款管理",
      "allStatuses": "所有狀態",
      "statusPending": "待處理",
      "statusApproved": "已核准",
      "statusRejected": "已拒��",
      "approveTitle": "核准存款",
      "rejectTitle": "拒絕存款",
      "reviewNoteLabel": "審核備註（選填）",
      "reviewNotePlaceholder": "為此審核新增備註...",
      "approveWarning": "核准將會立即將款項存入使用者錢包。"
    },
    "wallet": {
      "errors": {
        "noUserSelected": "尚未選取使用者"
      },
      "messages": {
        "adjustSuccess": "餘額調整成功",
        "adjustFailed": "調整失敗"
      },
      "columns": {
        "walletNumber": "錢包編號",
        "email": "電子郵件",
        "nickname": "暱稱",
        "type": "型別",
        "amount": "金額",
        "balanceAfter": "調整後餘額",
        "description": "說明",
        "time": "時間",
        "balance": "餘額",
        "frozen": "凍結",
        "currency": "貨幣"
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
      "unassigned": "未指派",
      "userList": "使用者清單",
      "noMatch": "無相符的使用者",
      "walletDetail": "錢包明細",
      "transactions": "交易明細",
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
      "aiGateway": "AI 閘道",
      "billing": "計費管理",
      "dashboard": "儀錶板",
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
      "title": "Token ↔ USD 計算機",
      "selectModel": "選擇模型（定價基礎）",
      "usdAmount": "USD 金額",
      "tokenAmount": "Token 數量",
      "fillResult": "填入結果"
    }
  },
  "autoTrading": {
    "logs": {
      "columns": {
        "action": "操作",
        "price": "價格",
        "profit": "盈虧",
        "symbol": "商品",
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
      "title": "全域風控設定"
    },
    "status": {
      "activeStrategies": "活躍策略",
      "disabled": "自動交易已關閉",
      "enabled": "自動交易已開啟",
      "todayExecutions": "今日執行",
      "todayProfit": "今日損益"
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
        "title": "風險警示"
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
      "clearAll": "清空",
      "clearAllConfirm": "確定清空所有通知？",
      "markAllAsRead": "全部已讀"
    },
    "tabs": {
      "all": "全部 ({{count}})",
      "unread": "未讀 ({{count}})"
    },
    "types": {
      "risk_alert": "風險警示",
      "signal": "訊號",
      "strategy_execution": "策略",
      "system": "系統",
      "trade": "交易"
    },
    "all": "全部",
    "clearAll": "清空",
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
        "amountUsd": "USD 到帳",
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
      "notConfigured": "USDT 充值尚未設定，請聯絡客服。",
      "notice": "請僅透過指定網路傳送 USDT。傳送其他代幣或使用不同網路可能導致永久遺失。",
      "submit": "提交",
      "success": "充值請求已提交，您的充值將自動確認。",
      "title": "充值",
      "txHashLabel": "交易雜湊（選填）",
      "willCredit": "預計到帳"
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
      "reversal": "沖正",
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
        "overlay": "主圖疊加指標",
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
      "strategyParameters": "策略引數"
    },
    "chat": {
      "executionPlan": "執行計畫",
      "codeGenerated": "程式碼已生成，請使用下方按鈕執行策略審查與回測。"
    },
    "aiChat": {
      "historyTab": "歷史",
      "strategiesTab": "策略"
    },
    "templates": {
      "title": "策略範本",
      "saveCurrent": "儲存目前策略",
      "lines": "行數",
      "chatEdit": "對話編輯",
      "source": "來源",
      "rename": "重新命名",
      "confirmDelete": "刪除此策略？",
      "noTemplates": "沒有儲存的策略範本",
      "sourceCode": "策略原始碼",
      "copyAll": "全部複製"
    },
    "live": {
      "stopSuccess": "策略已停止",
      "stopFailed": "停止失敗",
      "runId": "執行 ID",
      "account": "帳戶",
      "symbol": "商品程式碼",
      "timeframe": "TF",
      "mode": "模式",
      "signals": "訊號",
      "errors": "錯誤",
      "startedAt": "啟動時間",
      "watchSignals": "檢視訊號",
      "confirmStop": "停止此策略？",
      "status": "狀態",
      "totalSignals": "總訊號數",
      "stoppedAt": "停止時間",
      "error": "錯誤",
      "title": "即時策略監控",
      "activeTab": "執行中",
      "noActive": "尚無執行中的策略",
      "historyTab": "執行歷史",
      "noRuns": "尚無策略執行紀錄",
      "schedulesTab": "排程",
      "time": "時間",
      "signalType": "型別",
      "volume": "交易量",
      "price": "價格",
      "sl": "SL",
      "tp": "TP",
      "reason": "原因",
      "signalLog": "訊號紀錄",
      "waitingSignals": "等待訊號中..."
    },
    "schedule": {
      "maxPositionsPlaceholder": "無限制"
    },
    "ai": {
      "reviseHint": "先撰寫程式碼，再請 AI 改進。",
      "explainHint": "撰寫程式碼以檢視 AI 解釋。",
      "settingsHint": "設定 AI 提供者和模型"
    },
    "validate": {
      "running": "正在執行驗證...",
      "errors": "錯誤",
      "warnings": "警告",
      "fixWithAI": "將錯誤傳送給 AI 修訂",
      "parameters": "引數",
      "hints": "建議",
      "allClear": "所有檢查透過 — 沒有發現問題。",
      "passed": "驗證透過 — 儲存已解鎖。"
    },
    "importEA": {
      "writeTab": "策略程式碼",
      "importTab": "匯入 EA",
      "codeTooShort": "請貼上完整的 EA/指標原始碼。",
      "pastePlaceholder": "貼上 MQL4/MQL5 EA 程式碼...",
      "migration": "策略匯入",
      "aiTranslate": "AI 翻譯",
      "bridge": "盲區橋接",
      "analyze": "分析策略結構",
      "confirmImport": "確認匯入",
      "tryAI": "AI 翻譯補充",
      "apply": "套用至編輯器",
      "importSuccess": "MQL 原始碼已匯入，點選「Apply to Editor」寫入編輯器",
      "hint": "貼上 MQL4/MQL5 程式碼並點選分析",
      "translate": "翻譯為 Go",
      "translating": "AI 翻譯中...",
      "bridgeBtn": "盲區橋接翻譯",
      "bridgeSuccess": "橋接成功",
      "bridgeFailedTag": "橋接失敗",
      "bridging": "AI 橋接盲區中...",
      "bridgeFailedMsg": "Agent 無法自動橋接所有盲區",
      "noBridgeNeeded": "覆蓋率 100%，無需橋接",
      "bridgeHint": "貼上 MQL4/MQL5 EA 程式碼，AI 將自動翻譯盲區為 Python 子集"
    },
    "version": {
      "loadFailed": "載入版本失敗",
      "rollbackFailed": "還原失敗",
      "loadVersionFailed": "載入版本失敗",
      "loadDiffFailed": "載入差異失敗",
      "colVersion": "版本",
      "colSummary": "變更摘要",
      "colLang": "語言",
      "colHash": "雜湊",
      "colDate": "日期",
      "colActions": "操作",
      "title": "版本記錄",
      "diff": "差異",
      "empty": "尚無版本記錄",
      "history": "版本記錄"
    }
  },
  "accounts": {
    "bind": {
      "fields": {
        "alias": "帳戶別名"
      },
      "placeholders": {
        "alias": "選填自訂名稱"
      },
      "messages": {
        "changeCredentials": "變更憑證"
      }
    },
    "messages": {
      "shareLinkCopied": "分享連結已複製到剪貼簿",
      "shareLinkFailed": "建立分享連結失敗"
    }
  },
  "sre": {
    "breakers": {
      "columns": {
        "strategyId": "策略 ID",
        "state": "狀態",
        "totalPnl": "總盈虧",
        "lossPercent": "損失百分比",
        "tradeCount": "交易次數",
        "trippedAt": "觸發時間",
        "tripReason": "觸發原因"
      },
      "title": "策略斷路器",
      "stateClosed": "正常",
      "stateOpen": "已觸發",
      "stateHalfOpen": "半開（測試中）",
      "confirmReset": "重設此斷路器？",
      "description": "策略斷路器狀態總覽 — 自動偵測異常虧損並觸發斷路",
      "noBreakers": "無已註冊的斷路器"
    },
    "canary": {
      "columns": {
        "strategyId": "策略 ID",
        "versionTag": "版本標籤",
        "accounts": "金絲雀帳戶",
        "startAt": "開始時間",
        "days": "天數",
        "status": "狀態"
      },
      "promoted": "已推廣",
      "canarying": "金絲雀測試中",
      "confirmDelete": "刪除此金絲雀設定？",
      "title": "金絲雀設定",
      "description": "新策略版本先在少數帳戶執行 N 天，再推廣至所有帳戶",
      "newCanary": "新增金絲雀",
      "noCanaries": "無金絲雀設定",
      "newCanaryTitle": "新增金絲雀",
      "accountIdsLabel": "金絲雀帳戶 ID (以逗號或換行分隔)",
      "durationDays": "金絲雀天數"
    },
    "killSwitch": {
      "description": "一鍵停止所有交易 — 需輸入 KILL 確認；5 分鐘內可復原",
      "engaged": "緊急停止已觸發 — 所有交易已停止",
      "disarmed": "緊急停止已解除 — 交易正常",
      "status": "狀態",
      "reason": "原因",
      "operator": "操作者",
      "engagedAt": "觸發時間",
      "undo": "復原緊急停止",
      "disengage": "解除緊急停止",
      "engage": "觸發緊急停止",
      "confirmTitle": "觸發緊急停止 — 確認",
      "confirmEngage": "確認觸發",
      "confirmWarning": "此操作將立即停止所有帳戶的所有交易活動，包括掛單和已送出訂單。請輸入原因並輸入 KILL 確認。",
      "reasonLabel": "原因（必填）",
      "reasonPlaceholder": "例如：偵測到市場異常波動，緊急停止所有交易",
      "typeKill": "請輸入 KILL 以確認",
      "typeKillPlaceholder": "輸入 KILL（大寫）",
      "undoWindow": "撤銷視窗: {{minutes}}分 {{seconds}}秒 剩餘",
      "title": "熔斷開關"
    }
  },
  "marketplace": {
    "publish": {
      "priceModel": {
        "free": "免費",
        "monthly": "每月訂閱",
        "once": "一次性購買",
        "label": "定價"
      },
      "assetClass": {
        "label": "資產類別"
      },
      "riskLevel": {
        "label": "風險等級"
      },
      "return": "回報率",
      "winRate": "勝率",
      "trades": "交易筆數",
      "title": "釋出到市集",
      "titleLabel": "標題",
      "titlePlaceholder": "例如：黃金交叉策略",
      "descriptionLabel": "描述",
      "descriptionPlaceholder": "請描述您的策略邏輯、進出場規則...",
      "priceAmount": "金額",
      "tags": "標籤",
      "tagsPlaceholder": "輸入後按 Enter 新增標籤",
      "codeSnippet": "策略預覽（公開）",
      "codeSnippetPlaceholder": "可選：分享您的策略片段或高階構想（對所有人可見）",
      "includeBacktestSnapshot": "包含最新的回測結果"
    },
    "author": {
      "avgRating": "平均評分",
      "empty": "尚無已發布策略。前往策略庫發布一個。",
      "published": "已發布",
      "myStrategies": "我發布的策略",
      "publishNew": "發布新策略",
      "monthlyRevenue": "月營收",
      "totalRevenue": "總收益",
      "goToLibrary": "前往策略庫"
    },
    "card": {
      "by": "由",
      "free": "免費",
      "owned": "購買日期",
      "subscribers": "訂閱者",
      "winRate": "勝率",
      "yourStrategy": "您的策略"
    },
    "detail": {
      "assetClass": "資產類別",
      "author": "作者",
      "commentPlaceholder": "撰寫評論...",
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
      "commentPosted": "評論已發布",
      "loginFirst": "請先登入",
      "paymentComingSoon": "支付功能即將上線",
      "rateFailed": "評分失敗",
      "rated": "評分已提交",
      "subscribeFailed": "失敗",
      "subscribed": "已加入您的購買",
      "published": "策略已釋出到市集！",
      "publishFailed": "釋出策略失敗"
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
      "purchaseSuccess": "購買成功！策略已加入您的庫中。",
      "purchasing": "處理中...",
      "strategyName": "策略",
      "title": "確認購買",
      "walletBalance": "我的餘額"
    },
    "purchases": {
      "empty": "尚無購買記錄。前往市場發現策略。",
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
      "title": "策略回測",
      "capital": "資金",
      "commission": "手續費",
      "leverage": "槓桿",
      "completed": "已完成",
      "totalReturn": "總回報",
      "maxDrawdown": "最大回撤",
      "sharpe": "夏普比率",
      "winRate": "勝率",
      "totalTrades": "總交易次數",
      "equityCurve": "權益曲線",
      "protected": "策略程式碼受保護，回測在我們的伺服器上執行。",
      "run": "執行回測",
      "idle": "設定引數並執行回測"
    },
    "empty": "尚無已發布策略",
    "filterByClass": "依資產類別篩選",
    "noSubscriptions": "尚無訂閱",
    "searchPlaceholder": "搜尋策略...",
    "subtitle": "發現、購買和使用社群策略",
    "title": "策略市場"
  },
  "onboarding": {
    "step1": {
      "title": "連線您的帳戶",
      "desc": "繫結您的 MT4/MT5 交易帳戶以開始。",
      "action": "繫結帳戶"
    },
    "step2": {
      "title": "建立您的第一個策略",
      "desc": "使用 AI 從自然語言生成交易策略。",
      "action": "開啟工作區"
    },
    "step3": {
      "title": "升級您的方案",
      "desc": "解鎖更多 AI 代幣、策略和實盤交易功能。",
      "action": "檢視方案"
    },
    "subtitle": "只需簡單三步，立即開始",
    "dismiss": "知道了，關閉"
  },
  "auth": {
    "fields": {
      "confirmPassword": "確認密碼",
      "email": "電子郵件",
      "password": "密碼",
      "login": "電子郵件/帳號"
    },
    "forgotPassword": {
      "backToLogin": "返回登入",
      "hint": "請聯絡管理員或支援人員重設密碼。",
      "title": "重設密碼"
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
      "loginFailed": "登入失敗，請檢查信箱與密碼",
      "loginSuccess": "登入成功",
      "logoutSuccess": "已登出",
      "registerFailed": "註冊失敗，請稍後重試",
      "registerSuccess": "註冊成功，請登入"
    },
    "register": {
      "haveAccount": "已有帳號？",
      "loginNow": "立即登入",
      "register": "註冊",
      "signingUp": "註冊中...",
      "subtitle": "建立新帳號"
    },
    "validation": {
      "confirmPasswordRequired": "請確認密碼",
      "emailInvalid": "請輸入有效的信箱",
      "emailRequired": "請輸入信箱",
      "passwordMin8": "密碼至少8位元",
      "passwordMismatch": "兩次密碼不一致",
      "passwordRequired": "請輸入密碼",
      "loginRequired": "請輸入您的電子郵件或帳號"
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
    "created": "已建立",
    "currentPosition": "📊 當前持倉",
    "delete": "刪除",
    "deleteFailed": "刪除失敗",
    "deleteSelected": "刪除選中 ({{count}})",
    "deleted": "已刪除",
    "disable": "停用",
    "disabled": "已停用",
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
    "noData": "尚無資料",
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
    "unexpectedError": "發生意外錯誤",
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
    "step2Label": "登入憑證",
    "step3Label": "確認",
    "unit": "單位",
    "action": "操作",
    "on": "開啟",
    "off": "關閉",
    "true": "是",
    "false": "否",
    "success": "成功",
    "failed": "失敗",
    "reset": "重設",
    "saving": "儲存中..."
  },
  "errors": {
    "ai": {
      "api_key_required": "API Key 不能為空",
      "base_url_required": "Base URL 不能為空",
      "base_url_scheme_invalid": "Base URL 必須以 http:// 或 https:// 開頭",
      "base_url_should_not_end_with_chat_completions": "Base URL 不應以 /chat/completions 結尾",
      "config_service_not_initialized": "AI 設定服務未初始化",
      "config_valid": "AI 設定有效",
      "failed_to_create_request": "建立請求失敗",
      "forbidden_quota": "配額超限",
      "free_tier_exhausted": "AI 模型免費額度已耗盡：請在模型供應商管理後臺關閉「use free tier only」或更換付費 Key。",
      "invalid_base_url": "Base URL 無效",
      "invalid_provider": "服務商無效",
      "no_trade_data_available": "暫無可用交易資料",
      "not_configured": "AI 未配置：請先到 AI 設定中啟用並配置。",
      "probe_ok": "正常",
      "probe_ok_no_models": "正常（未回傳 models）",
      "provider_required": "請先選擇服務商",
      "provider_returned_empty_message": "AI 服務回傳空訊息",
      "rate_limited": "AI 服務觸發限流/額度不足（429/資源耗盡）。請稍後重試或更換可用的 API Key/模型配置。",
      "request_failed": "API 請求失敗",
      "insufficient_balance_title": "餘額不足",
      "insufficient_balance": "您的 AI 錢包餘額不足，請先充值後再繼續。"
    },
    "connection_failed": {
      "content": "無法連線到伺服器，請檢查網路後重試。",
      "title": "連線失敗"
    },
    "access_denied": "無許可權存取",
    "account_connected": "連線成功",
    "account_connection_failed": "無法連線到交易伺服器",
    "account_not_found": "帳戶不存在",
    "auto_trading_disabled": "自動交易已關閉",
    "auto_trading_enabled": "自動交易已開啟",
    "email_already_registered": "信箱已註冊",
    "invalid_credentials": "帳號或密碼錯誤",
    "not_authenticated": "未登入",
    "schedule_service_not_available": "排程服務不可用",
    "translate_failed": "翻譯失敗",
    "user_not_found": "使用者不存在"
  },
  "symbolDetection": {
    "tradeMode": {
      "disabled": "已停用",
      "longOnly": "僅做多",
      "longShort": "多空雙向",
      "shortOnly": "僅做空",
      "unknown": "未知"
    },
    "label": "偵測到的交易品種",
    "loading": "解析中…",
    "noSymbols": "未偵測到交易品種。請嘗試包含具體品種名稱（如「比特幣」、「EURUSD」、「黃金」）。",
    "resolvedTooltip": "券商：{{broker}} | 模式：{{mode}}",
    "unresolvedTooltip": "尚未繫結交易帳戶，無法解析"
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
    "usageTitle": "本月使用量",
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
      "title": "回測分析",
      "drawdown": "DD",
      "winrate": "勝率",
      "consistency": "一致性",
      "risk_adj": "風險調整回報",
      "overfitting": "過度擬合風險",
      "observations": "重點觀察",
      "suggestions": "改善建議",
      "detailed": "詳細分析"
    },
    "semantic_diff": {
      "title": "策略變更",
      "effect": "影響"
    },
    "profile": {
      "title": "策略概況",
      "timeframe": "時間框架",
      "regime": "市場狀態",
      "indicators": "指標",
      "entry": "進場",
      "exit": "出場",
      "risk": "風險管理",
      "coverage": "涵蓋範圍",
      "strengths": "優勢",
      "weaknesses": "劣勢",
      "blind_spots": "盲點"
    }
  },
  "importAnalysis": {
    "execution": {
      "onBar": "K線收盤事件驅動",
      "onTick": "Tick 驅動",
      "onInitGrid": "初始網格"
    },
    "sizing": {
      "fixed": "固定手數",
      "martingale": "馬丁格爾",
      "percentBalance": "餘額百分比"
    },
    "analyzing": "正在分析策略���構...",
    "tradeLogicComplete": "交易邏輯已完整識別",
    "guiNoiseDesc": "以下盲點為圖表顯示/按鈕功能，伺服器端執行時會略過，不影響交易結果，可安全匯入。",
    "cannotImport": "無法自動匯入",
    "incompleteCoverage": "交易邏輯涵蓋不完整",
    "goodCoverage": "匯入涵蓋良好",
    "goodCoverageDesc": "策略主要邏輯已識別，可安全匯入。使用前請檢查引數清單。",
    "coverageTitle": "匯入涵蓋範圍",
    "location": "位置",
    "handling": "處理方式",
    "userActionRequired": "需要您的操作",
    "noBlindSpots": "無需確認",
    "noBlindSpotsDesc": "所有策略邏輯已自動識別，可安全匯入。"
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
      "recovery": "復原"
    },
    "result": {
      "pass": "透過",
      "reject": "拒絕"
    }
  },
  "app": {
  "app": {
  },
  "language": {
  "language": {
    "japanese": "日本語",
    "simplifiedChinese": "簡體中文",
    "traditionalChinese": "繁體中文",
    "traditionalChinese": "繁體中文",
  },
  "market": {
    "allSymbols": "全部品種",
    "ask": "賣價",
    "bid": "買價",
    "common": "常用",
    "emptyWatchlist": "暫無自選",
    "loadingSymbols": "載入中...",
    "mid": "中間價",
    "mtSessionLost": "⚠ MT 工作階段遺失 — 正在重連…",
    "noSymbolSelected": "選擇一個品種以檢視行情資料",
    "noSymbolsFound": "找不到品種",
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
    "algoDashboard": "演演算法看板",
    "analytics": "分析",
    "assetAnalysis": "AI 分析",
    "assets": "策略資產",
    "autoTrading": "自動交易",
    "dashboard": "儀錶板",
    "devGroup": "策略開發",
    "experiments": "策略實驗",
    "indicatorCatalog": "指標目錄",
    "logs": "系統日誌",
    "market": "策略市場",
    "marketRegime": "市場狀態",
    "marketTools": "市場分析工具",
    "marketplace": "市場",
    "opsGroup": "策略營運",
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
    "createdAt": "已建立",
    "deleteConfirm": "刪除此分享連結？",
    "empty": "尚無分享連結",
    "expires": "過期時間",
    "positions": "持倉",
    "showPositions": "顯示持倉",
    "title": "分享管理",
    "token": "分享連結",
    "userId": "一般使用者",
    "views": "瀏覽量"
  },
  "sharePage": {
    "avgHolding": "平均持倉時長",
    "avgLoss": "平均虧損",
    "avgWin": "平均獲利",
    "bestTrade": "最佳交易",
    "bySymbol": "商品績效",
    "closeTime": "平倉時間",
    "count": "筆數",
    "disclaimer": "過往績效不代表未來表現。",
    "equityCurve": "淨值曲線",
    "expired": "此分享連結已過期",
    "footer": "由 AlphaForge 產生",
    "language": "語言",
    "loadFailed": "載入分享資料失敗",
    "losingTrades": "虧損筆數",
    "maxDrawdown": "最大回撤",
    "netProfit": "淨損益",
    "noPositions": "暫無持倉",
    "noTrades": "暫無交易紀錄",
    "notFound": "找不到",
    "openPrice": "開倉價",
    "positions": "當前持倉",
    "positionsLocked": "建立者未開放持倉檢視",
    "profit": "損益",
    "profitFactor": "獲利因子",
    "sharpeRatio": "夏普比率",
    "side": "方向",
    "subtitle": "真實交易成績",
    "symbol": "商品",
    "title": "交易績效",
    "totalReturn": "淨損益",
    "totalTrades": "總交易數",
    "totalVolume": "總交易量",
    "tradeRecords": "交易紀錄",
    "volume": "數量",
    "winRate": "勝率",
    "winningTrades": "獲利筆數",
    "worstTrade": "最差交易",
    "countUnit": "筆"
  },
  "topbar": {
    "logout": "退出登入",
    "profile": "個人資訊",
    "settings": "設定",
    "switchToAdmin": "切換到管理",
    "systemOk": "系統正常執行",
    "user": "一般使用者"
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
    "disconnected": "已斷線",
    "streamError": "串流錯誤",
    "waitingData": "等待資料中...",
    "serviceHealth": "服務健康狀態",
    "uptime": "執行時間",
    "database": "資料庫",
    "diskUsage": "磁碟使用率",
    "goRuntime": "Go 執行環境",
    "goRuntime": "Go執行時",
    "gcCount": "GC 次數",
    "gcPauseAvg": "GC 平均暫停",
    "stackUsage": "堆疊使用量",
    "heapMemory": "堆積記憶體",
    "dbPool": "資料庫連線池",
    "totalConns": "總計",
    "idle": "閒置",
    "acquired": "已取得",
    "mdGateway": "MD 閘道",
    "spillFiles": "溢寫檔案",
    "droppedBars": "遺失K線",
    "droppedSignals": "遺失訊號",
    "consumerLag": "消費者延遲",
    "staleAccounts": "過時帳戶",
    "deadAccounts": "失效帳戶",
    "avgGapSec": "平均間隔（秒）",
    "maxGapSec": "最大間隔（秒）",
    "dlq": "無效信件佇列（DLQ）",
    "parseErrors": "解析錯誤",
    "bidGtAsk": "買價>賣價",
    "nonPositive": "非正值",
    "pushInterval": "推送間隔：5秒",
    "lastUpdate": "最後更新"
  }
} as const;
export default Base;
