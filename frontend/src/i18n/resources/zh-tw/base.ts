// Auto-generated from proto/ant/v1/i18n/base_zh-tw.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const Base = {
  "admin": {
    "dashboard": {
      "logs": {
        "moduleMap": {
          "accountManagement": "账戶管理",
          "systemConfig": "系统配置",
          "trading": "交易",
          "userManagement": "用戶管理"
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
        "orderCloseFailed": "平仓失败",
        "orderCloseSuccess": "平倉成功",
        "orderSendFailed": "下單失敗",
        "orderSendSuccess": "下單成功",
        "riskValidateError": "錯誤",
        "riskValidatePass": "通過",
        "riskValidateReject": "拒絕",
        "riskValidateTotal": "總驗證數",
        "title": "風控指標"
      },
      "riskWindow": {
        "noData": "暂无窗口指标数据",
        "noRejectData": "此時段无拒絕記錄",
        "orderCloseFailed": "平倉失敗",
        "orderCloseSuccess": "平倉成功",
        "orderSendFailed": "下單失敗",
        "orderSendSuccess": "下單成功",
        "rejectCount": "拒絕次數",
        "rejectRiskCodesHeader": "風控代碼",
        "title": "風控窗口",
        "validateError": "錯誤",
        "validatePass": "通過",
        "validateReject": "拒絕",
        "validateTotal": "總计"
      },
      "activeUsers": "活躍用戶",
      "loadFailed": "加载儀表板數据失敗",
      "mtAccounts": "MT账戶數",
      "onlineAccounts": "在線账戶",
      "recentLogs": "最近日誌",
      "title": "管理儀表板",
      "todayProfit": "今日盈虧",
      "todayTrades": "今日交易",
      "totalUsers": "總用戶數"
    },
    "userManagement": {
      "drawer": {
        "labels": {
          "createdAt": "建立時間",
          "email": "電子郵件",
          "id": "ID",
          "lastLogin": "最后登入",
          "mtAccountCount": "MT账戶數",
          "nickname": "暱稱",
          "role": "角色",
          "status": "狀態"
        },
        "title": "用戶詳細"
      },
      "form": {
        "placeholders": {
          "email": "輸入電子郵件",
          "nickname": "輸入暱稱",
          "password": "输入密码"
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
          "confirmPassword": "再次输入新密码",
          "newPassword": "輸入新密碼"
        },
        "validation": {
          "confirmPasswordRequired": "请確認新密碼",
          "newPasswordRequired": "请輸入新密碼",
          "passwordMin8": "密碼至少8位元",
          "passwordMismatch": "两次密碼不一致",
          "passwordMustContainLettersAndNumbers": "密码必须包含字母和数字"
        },
        "confirmPassword": "確認密碼",
        "newPassword": "新密碼",
        "submit": "更新密碼"
      },
      "actions": {
        "changePassword": "修改密码",
        "details": "詳細",
        "disable": "停用",
        "enable": "啟用"
      },
      "deleteConfirm": {
        "batchDeleteConfirm": "確認刪除 {{count}} 個用戶？此操作不可復原。",
        "batchDeletePartial": "已刪除 {{deleted}} 個，{{failed}} 個失敗",
        "batchDeleteSuccess": "已刪除 {{count}} 個用戶",
        "title": "確認刪除此用戶？此操作不可復原。"
      },
      "filters": {
        "rolePlaceholder": "按角色篩選",
        "searchPlaceholder": "搜尋電子郵件或暱稱",
        "statusPlaceholder": "按状态筛选"
      },
      "messages": {
        "newPasswordIs": "新密码为: {{password}}",
        "passwordUpdateFailed": "密碼更新失敗",
        "passwordUpdatedSuccess": "密碼更新成功",
        "userCreateFailed": "建立用戶失敗",
        "userCreatedSuccess": "用戶建立成功",
        "userDeleteFailed": "刪除用戶失敗",
        "userDeletedSuccess": "用戶已刪除",
        "userDisabled": "用戶已停用",
        "userEnabled": "用戶已啟用",
        "userUpdateFailed": "更新用戶失敗",
        "userUpdatedSuccess": "用戶更新成功"
      },
      "modals": {
        "createTitle": "新建用戶",
        "editTitle": "編輯用戶",
        "passwordTitle": "修改密码"
      },
      "pagination": {
        "total": "共 {{total}} 位用户"
      },
      "roles": {
        "audit": "审计",
        "customerService": "客服",
        "operation": "營運",
        "superAdmin": "超級管理員",
        "user": "一般用戶"
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
        "mtAccountCount": "MT账戶數",
        "nickname": "暱稱",
        "role": "角色",
        "status": "狀態"
      },
      "addUser": "新建用戶",
      "title": "用戶管理"
    },
    "config": {
      "messages": {
        "disabled": "已停用",
        "enabled": "已啟用",
        "loadFailed": "加载配置失敗",
        "operationFailed": "操作失敗",
        "updateFailed": "更新配置失敗",
        "updated": "配置已更新"
      },
      "placeholders": {
        "apiKey": "輸入API Key",
        "baseUrl": "輸入Base URL",
        "configValue": "輸入配置值",
        "description": "输入描述",
        "json": "輸入JSON",
        "model": "輸入模型名稱"
      },
      "providerOptions": {
        "custom": "自定义 / OpenAI 兼容",
        "deepseek": "DeepSeek",
        "zhipu": "智谱AI"
      },
      "validation": {
        "apiKeyRequired": "API Key不能為空白",
        "greenMaxFailedRunsNonNegative": "绿色最大失敗次數需≥0",
        "greenSuccessRateRange": "绿色成功率需在0-100之間",
        "jsonEmpty": "JSON不能為空白",
        "jsonInvalid": "JSON格式無效",
        "minSampleSizeNonNegative": "最小样本量需≥0",
        "modelRequired": "模型名称不能为空",
        "yellowNotGreaterThanGreen": "黄色閾值不能超過绿色閾值",
        "yellowSuccessRateRange": "黄色成功率需在0-100之間"
      },
      "aiProviderCatalog": "AI提供者目錄",
      "baseUrlLabel": "Base URL",
      "configItem": "配置項目",
      "description": "描述",
      "econAIConfig": "經濟日曆AI配置",
      "editConfig": "编辑配置: {{key}}",
      "enableToggle": "啟用",
      "fillTemplate": "填入範本",
      "formatJson": "格式化JSON",
      "maxAccountsPerUser": "每用戶最大账戶數",
      "modelName": "模型名稱",
      "off": "關",
      "on": "开",
      "provider": "提供者",
      "status": "狀態",
      "strategyHealthConfig": "策略健康度配置",
      "thresholdDesc": "閾值描述",
      "thresholdInfo": "閾值說明",
      "title": "系统配置",
      "toggle": "切换",
      "updatedAt": "更新時間",
      "value": "值"
    },
    "jurisdiction": {
      "messages": {
        "countryAddFailed": "新增國家失敗",
        "countryAdded": "國家已新增",
        "countryRemoveFailed": "移除國家失敗",
        "countryRemoved": "國家已移除",
        "kycUpdateFailed": "更新KYC狀態失敗",
        "kycUpdated": "KYC狀態已更新",
        "overrideUpdateFailed": "更新制裁豁免失败",
        "overrideUpdated": "豁免狀態已更新"
      },
      "actions": "操作",
      "addCountry": "新增國家",
      "addSanctionedCountry": "新增制裁國家",
      "addedBy": "新增人",
      "confirmGrantOverride": "確認授予该用戶豁免權限？",
      "confirmRevokeOverride": "確認撤銷该用戶的豁免權限？",
      "country": "國家",
      "countryCode": "國家代碼",
      "countryLabel": "國家",
      "disclaimer": "免責聲明",
      "emptyKYC": "暫無KYC記錄",
      "emptySanctions": "暫無制裁國家",
      "filterByKYCStatus": "按KYC狀態篩選",
      "grantOverride": "授予豁免",
      "kycStatus": "KYC狀態",
      "kycStatusTab": "用戶KYC狀態",
      "override": "豁免",
      "overrideWarning": "此用户来自受制裁国家，授予豁免将允许交易。",
      "pending": "待審核",
      "questionnaire": "問卷",
      "rejected": "已拒絕",
      "revokeOverride": "撤銷豁免",
      "sanctioned": "已制裁",
      "sanctionedCountries": "制裁國家",
      "sanctionedCountriesTab": "制裁國家",
      "setKYC": "設定KYC",
      "setKYCStatus": "設定KYC狀態",
      "title": "管轄权管理",
      "unverified": "未驗證",
      "userEmail": "電子郵件",
      "userKYCStatus": "用戶KYC狀態",
      "verified": "已驗證"
    },
    "header": {
      "admin": "管理",
      "adminMode": "管理员模式",
      "adminPanel": "管理后台",
      "backToUser": "返回用户端",
      "logout": "退出登入"
    },
    "sidebar": {
      "accountManagement": "账戶管理",
      "dashboard": "儀表板",
      "jurisdiction": "管轄权管理",
      "operationLogs": "操作日志",
      "shareManagement": "分享分析",
      "systemConfig": "系统配置",
      "tradingMonitor": "交易監控",
      "userManagement": "用戶管理",
      "walletManagement": "钱包管理"
    },
    "trading": {
      "accounts": "账戶",
      "activeUsers": "活躍用戶",
      "byPlatform": "按平台",
      "closedOrders": "已平倉",
      "connectedAccounts": "已连接",
      "loadFailed": "加载交易统计失败",
      "netProfit": "淨利潤",
      "orders": "訂單",
      "pendingOrders": "挂单",
      "platform": "平台",
      "profitStats": "盈虧統計",
      "title": "交易監控",
      "totalAccounts": "總账戶數",
      "totalLoss": "總虧損",
      "totalOrders": "總訂單",
      "totalProfit": "總盈利",
      "totalUsers": "總用戶數",
      "totalVolume": "總交易量",
      "volume": "數量"
    },
    "wallet": {
      "accountNumber": "錢包號",
      "add": "增加",
      "adjustBalance": "調整餘額",
      "adjustFailed": "調整失敗",
      "adjustSuccess": "餘額已調整",
      "deduct": "扣除",
      "noUsers": "未找到用戶",
      "reason": "調整原因...",
      "searchPlaceholder": "搜尋郵箱或錢包號...",
      "title": "錢包管理",
      "walletFor": "錢包 -"
    }
  },
  "autoTrading": {
    "logs": {
      "columns": {
        "action": "操作",
        "price": "價格",
        "profit": "盈虧",
        "symbol": "商品",
        "ticket": "单号",
        "time": "時間",
        "volume": "數量"
      },
      "empty": "暫無交易日誌",
      "title": "最近交易日誌"
    },
    "messages": {
      "loadFailed": "載入自動交易資料失敗",
      "toggleFailed": "切换自动交易失败"
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
      "saveFailed": "保存设置失败",
      "saveSuccess": "設定已儲存",
      "title": "全域風控設定"
    },
    "status": {
      "activeStrategies": "活躍策略",
      "disabled": "自動交易已關閉",
      "enabled": "自動交易已開啟",
      "todayExecutions": "Today's Executions",
      "todayProfit": "Today's Profit"
    },
    "title": "自動交易"
  },
  "notifications": {
    "stream": {
      "autoTrading": {
        "fallback": "自动交易事件触发",
        "title": "自動交易"
      },
      "riskAlert": {
        "fallback": "警报类型: {{alertType}}",
        "title": "風險警示"
      },
      "strategyExecution": {
        "completed": "{{symbol}} {{action}} 已完成",
        "failed": "执行失败: {{error}}",
        "title": "策略執行"
      },
      "strategySignal": {
        "message": "{{symbol}} triggered {{signalType}}",
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
      "unread": "未读 ({{count}})"
    },
    "types": {
      "risk_alert": "風險警示",
      "signal": "訊號",
      "strategy_execution": "策略",
      "system": "系统",
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
  "auth": {
    "fields": {
      "confirmPassword": "确认密码",
      "email": "電子郵件",
      "password": "密碼"
    },
    "forgotPassword": {
      "backToLogin": "返回登录",
      "hint": "請聯繫管理員或支援人員重設密碼。",
      "title": "重設密碼"
    },
    "login": {
      "forgotPassword": "忘記密碼？",
      "login": "立即登入",
      "noAccount": "没有账户？",
      "registerNow": "立即註冊",
      "rememberMe": "記住我",
      "signingIn": "登入中...",
      "subtitle": "這是一個測試不具備責任能力"
    },
    "messages": {
      "fetchMeFailed": "加载用户信息失败",
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
      "passwordMismatch": "两次密碼不一致",
      "passwordRequired": "請輸入密碼"
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
      "lessThanMinute": "<1分钟",
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
    "ok": "OK",
    "operationFailed": "操作失敗",
    "pageError": "頁面錯誤",
    "pageUnderDevelopment": "此页面开发中",
    "pleaseWait": "請稍候...",
    "previous": "上一步",
    "refresh": "刷新",
    "remove": "移除",
    "required": "必填",
    "retry": "重試",
    "save": "保存",
    "saveFailed": "保存失敗",
    "saveSuccess": "儲存成功",
    "searching": "搜尋中...",
    "selectSymbolToViewChart": "選擇品種查看圖表",
    "send": "發送",
    "showDetails": "查看詳情",
    "totalItems": "共 {{count}} 項",
    "translate": "翻譯",
    "unexpectedError": "發生意外錯誤",
    "unknown": "未知",
    "updated": "已更新",
    "viewOriginal": "查看原文",
    "viewTranslation": "查看譯文",
    "yes": "是",
    "you": "你",
    "unsaved": "未儲存",
    "saved": "已儲存"
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
      "forbidden_quota": "配额超限",
      "free_tier_exhausted": "AI 模型免費額度已耗盡：請在模型供應商管理後台關閉「use free tier only」或更換付費 Key。",
      "invalid_base_url": "Base URL 無效",
      "invalid_provider": "服務商無效",
      "no_trade_data_available": "暫無可用交易資料",
      "not_configured": "AI 未配置：請先到 AI 設定中啟用並配置。",
      "probe_ok": "OK",
      "probe_ok_no_models": "正常（未回傳 models）",
      "provider_required": "請先選擇服務商",
      "provider_returned_empty_message": "AI 服務回傳空訊息",
      "rate_limited": "AI 服務觸發限流/額度不足（429/資源耗盡）。請稍後重試或更換可用的 API Key/模型配置。",
      "request_failed": "API 請求失敗"
    },
    "connection_failed": {
      "content": "无法连接到服务器，请检查网络后重试。",
      "title": "連線失敗"
    },
    "access_denied": "無權限存取",
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
  "marketplace": {
    "author": {
      "avgRating": "平均評分",
      "empty": "尚無已發布策略。前往策略庫發布一個。",
      "published": "已發布"
    },
    "card": {
      "by": "by",
      "free": "免費",
      "owned": "購買日期",
      "subscribers": "訂閱者",
      "winRate": "勝率"
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
      "yourRating": "我的評分"
    },
    "messages": {
      "commentFailed": "評論失敗",
      "commentPosted": "評論已發布",
      "loginFirst": "請先登入",
      "paymentComingSoon": "支付功能即將上線",
      "rateFailed": "評分失敗",
      "rated": "評分已提交",
      "subscribeFailed": "失敗",
      "subscribed": "已加入您的購買"
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
      "strategy": "策略"
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
    "empty": "尚無已發布策略",
    "filterByClass": "依資產類別篩選",
    "noSubscriptions": "尚無訂閱",
    "publish": "發布策略",
    "searchPlaceholder": "搜尋策略...",
    "subtitle": "發現、購買和使用社群策略",
    "title": "策略市場"
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
    "unresolvedTooltip": "尚未綁定交易帳戶，無法解析"
  },
  "wallet": {
    "table": {
      "amount": "金額",
      "balanceAfter": "調整後餘額",
      "description": "描述",
      "time": "時間",
      "type": "類型"
    },
    "txType": {
      "adjustment": "餘額調整",
      "deposit": "充值",
      "fee": "手續費",
      "reversal": "沖正",
      "withdrawal": "提取"
    },
    "accountNumber": "錢包號",
    "balance": "餘額",
    "currency": "幣種",
    "deposit": "充值",
    "frozen": "凍結",
    "frozenBalance": "凍結",
    "history": "歷史記錄",
    "title": "我的錢包",
    "transactions": "交易記錄",
    "withdraw": "提取"
  },
  "app": {
    "name": "AntTrader"
  },
  "language": {
    "english": "English",
    "japanese": "日本語",
    "simplifiedChinese": "简体中文",
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
    "noSymbolSelected": "選擇一個品種以檢視行情數據",
    "noSymbolsFound": "找不到品種",
    "popularSymbols": "熱門品種",
    "searchPlaceholder": "搜尋品種（如 EURUSD, XAUUSD）",
    "searchSymbol": "搜索品种...",
    "selectAccount": "選擇交易賬戶",
    "selectSymbol": "選擇品種",
    "spread": "點差",
    "watchlist": "自選"
  },
  "menu": {
    "accounts": "账戶",
    "aiAssistant": "AI助手",
    "algoDashboard": "演算法看板",
    "analytics": "分析",
    "assetAnalysis": "AI 分析",
    "assets": "策略資產",
    "autoTrading": "自動交易",
    "dashboard": "儀表板",
    "devGroup": "策略開發",
    "experiments": "策略實驗",
    "indicatorCatalog": "指標目錄",
    "logs": "系統日誌",
    "market": "策略市場",
    "marketRegime": "市場狀態",
    "marketTools": "市場分析工具",
    "marketplace": "市場",
    "opsGroup": "策略營運",
    "schedules": "策略調度",
    "strategies": "策略管理",
    "strategy": "策略",
    "strategyLibrary": "策略庫",
    "strategyWorkspace": "策略工作台",
    "trading": "交易",
    "wallet": "錢包"
  },
  "profile": {
    "lastLogin": "最后登入",
    "nickname": "暱稱",
    "registered": "已注册",
    "role": "角色",
    "status": "狀態",
    "title": "個人資訊"
  },
  "share": {
    "actions": "操作",
    "createNew": "創建新分享連結",
    "createdAt": "已建立",
    "deleteConfirm": "删除此分享链接？",
    "empty": "尚無分享連結",
    "expires": "過期時間",
    "positions": "持仓",
    "showPositions": "显示持仓",
    "title": "分享管理",
    "token": "分享連結",
    "userId": "一般用戶",
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
    "footer": "由 AntTrader 產生",
    "language": "語言",
    "loadFailed": "載入分享資料失敗",
    "losingTrades": "虧損筆數",
    "maxDrawdown": "最大回撤",
    "netProfit": "淨損益",
    "noPositions": "暂无持仓",
    "noTrades": "暫無交易紀錄",
    "notFound": "找不到",
    "openPrice": "开仓价",
    "positions": "当前持仓",
    "positionsLocked": "创建者未开放持仓查看",
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
    "worstTrade": "最差交易"
  },
  "topbar": {
    "logout": "退出登入",
    "profile": "個人資訊",
    "settings": "設定",
    "switchToAdmin": "切換到管理",
    "systemOk": "系統正常運行",
    "user": "一般用戶"
  }
} as const;
export default Base;
