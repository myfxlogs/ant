// Auto-generated from proto/ant/v1/i18n/base_zh-cn.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const Base = {
  "admin": {
    "config": {
      "aiProviderCatalog": "AI提供商目录",
      "baseUrlLabel": "Base URL",
      "configItem": "配置项",
      "description": "描述",
      "econAIConfig": "经济日历AI配置",
      "editConfig": "编辑配置: {{key}}",
      "enableToggle": "启用",
      "fillTemplate": "填充模板",
      "formatJson": "格式化JSON",
      "maxAccountsPerUser": "每用户最大账户数",
      "messages": {
        "disabled": "已禁用",
        "enabled": "已启用",
        "loadFailed": "加载配置失败",
        "operationFailed": "操作失败",
        "updateFailed": "更新配置失败",
        "updated": "配置已更新"
      },
      "modelName": "模型名称",
      "off": "关",
      "on": "On",
      "placeholders": {
        "apiKey": "输入API Key",
        "baseUrl": "输入Base URL",
        "configValue": "输入配置值",
        "description": "Enter description",
        "json": "输入JSON",
        "model": "输入模型名称"
      },
      "provider": "提供商",
      "providerOptions": {
        "custom": "Custom / OpenAI Compatible",
        "deepseek": "DeepSeek",
        "zhipu": "智谱AI"
      },
      "status": "状态",
      "strategyHealthConfig": "策略健康度配置",
      "thresholdDesc": "阈值描述",
      "thresholdInfo": "阈值说明",
      "title": "系统配置",
      "toggle": "切换",
      "updatedAt": "更新时间",
      "validation": {
        "apiKeyRequired": "API Key不能为空",
        "greenMaxFailedRunsNonNegative": "绿色最大失败次数需≥0",
        "greenSuccessRateRange": "绿色成功率需在0-100之间",
        "jsonEmpty": "JSON不能为空",
        "jsonInvalid": "JSON格式无效",
        "minSampleSizeNonNegative": "最小样本量需≥0",
        "modelRequired": "Model name cannot be empty",
        "yellowNotGreaterThanGreen": "黄色阈值不能超过绿色阈值",
        "yellowSuccessRateRange": "黄色成功率需在0-100之间"
      },
      "value": "值"
    },
    "dashboard": {
      "activeUsers": "活跃用户",
      "loadFailed": "加载仪表盘数据失败",
      "logs": {
        "actionType": "操作",
        "failed": "失败",
        "module": "模块",
        "moduleMap": {
          "accountManagement": "账户管理",
          "systemConfig": "系统配置",
          "trading": "交易",
          "userManagement": "用户管理"
        },
        "status": "状态",
        "success": "成功",
        "target": "目标",
        "time": "时间"
      },
      "mtAccounts": "MT账户数",
      "onlineAccounts": "在线账户",
      "recentLogs": "最近日志",
      "riskMetrics": {
        "orderCloseFailed": "Order Closed Failed",
        "orderCloseSuccess": "平仓成功",
        "orderSendFailed": "下单失败",
        "orderSendSuccess": "下单成功",
        "riskValidateError": "错误",
        "riskValidatePass": "通过",
        "riskValidateReject": "拒绝",
        "riskValidateTotal": "总验证数",
        "title": "风控指标"
      },
      "riskWindow": {
        "noData": "No window metrics data",
        "noRejectData": "本时段无拒绝记录",
        "orderCloseFailed": "平仓失败",
        "orderCloseSuccess": "平仓成功",
        "orderSendFailed": "下单失败",
        "orderSendSuccess": "下单成功",
        "rejectCount": "拒绝次数",
        "rejectRiskCodesHeader": "风控代码",
        "title": "风控窗口",
        "validateError": "错误",
        "validatePass": "通过",
        "validateReject": "拒绝",
        "validateTotal": "总计"
      },
      "title": "管理仪表盘",
      "todayProfit": "今日盈亏",
      "todayTrades": "今日交易",
      "totalUsers": "总用户数"
    },
    "header": {
      "admin": "Admin",
      "adminMode": "管理员模式",
      "adminPanel": "管理后台",
      "backToUser": "返回用户端",
      "logout": "退出登录"
    },
    "jurisdiction": {
      "actions": "操作",
      "addCountry": "添加国家",
      "addSanctionedCountry": "添加制裁国家",
      "addedBy": "添加人",
      "confirmGrantOverride": "确认授予该用户豁免权限？",
      "confirmRevokeOverride": "确认撤销该用户的豁免权限？",
      "country": "国家",
      "countryCode": "国家代码",
      "countryLabel": "国家",
      "disclaimer": "免责声明",
      "emptyKYC": "暂无KYC记录",
      "emptySanctions": "暂无制裁国家",
      "filterByKYCStatus": "按KYC状态筛选",
      "grantOverride": "授予豁免",
      "kycStatus": "KYC状态",
      "kycStatusTab": "用户KYC状态",
      "messages": {
        "countryAddFailed": "添加国家失败",
        "countryAdded": "国家已添加",
        "countryRemoveFailed": "移除国家失败",
        "countryRemoved": "国家已移除",
        "kycUpdateFailed": "更新KYC状态失败",
        "kycUpdated": "KYC状态已更新",
        "overrideUpdateFailed": "Failed to update sanctioned override",
        "overrideUpdated": "豁免状态已更新"
      },
      "override": "豁免",
      "overrideWarning": "This user is from a sanctioned country. Granting override will allow trading.",
      "pending": "待审核",
      "questionnaire": "问卷",
      "rejected": "已拒绝",
      "revokeOverride": "撤销豁免",
      "sanctioned": "已制裁",
      "sanctionedCountries": "制裁国家",
      "sanctionedCountriesTab": "制裁国家",
      "setKYC": "设置KYC",
      "setKYCStatus": "设置KYC状态",
      "title": "管辖权管理",
      "unverified": "未验证",
      "userEmail": "邮箱",
      "userKYCStatus": "用户KYC状态",
      "verified": "已验证"
    },
    "sidebar": {
      "accountManagement": "账户管理",
      "dashboard": "仪表盘",
      "jurisdiction": "管辖权管理",
      "operationLogs": "操作日志",
      "shareManagement": "Share Analytics",
      "systemConfig": "系统配置",
      "tradingMonitor": "交易监控",
      "userManagement": "用户管理",
      "walletManagement": "钱包管理"
    },
    "trading": {
      "accounts": "账户",
      "activeUsers": "活跃用户",
      "byPlatform": "按平台",
      "closedOrders": "已平仓",
      "connectedAccounts": "已连接",
      "loadFailed": "加载交易统计失败",
      "netProfit": "净利润",
      "orders": "订单",
      "pendingOrders": "Pending Orders",
      "platform": "平台",
      "profitStats": "盈亏统计",
      "title": "交易监控",
      "totalAccounts": "总账户数",
      "totalLoss": "总亏损",
      "totalOrders": "总订单",
      "totalProfit": "总盈利",
      "totalUsers": "总用户数",
      "totalVolume": "总交易量",
      "volume": "数量"
    },
    "userManagement": {
      "actions": {
        "changePassword": "Change Password",
        "details": "详情",
        "disable": "禁用",
        "enable": "启用"
      },
      "addUser": "新建用户",
      "deleteConfirm": {
        "batchDeleteConfirm": "确认删除 {{count}} 个用户？此操作不可撤销。",
        "batchDeletePartial": "已删除 {{deleted}} 个，{{failed}} 个失败",
        "batchDeleteSuccess": "已删除 {{count}} 个用户",
        "title": "确认删除此用户？此操作不可撤销。"
      },
      "drawer": {
        "labels": {
          "createdAt": "创建时间",
          "email": "邮箱",
          "id": "ID",
          "lastLogin": "最后登录",
          "mtAccountCount": "MT账户数",
          "nickname": "昵称",
          "role": "角色",
          "status": "状态"
        },
        "title": "用户详情"
      },
      "filters": {
        "rolePlaceholder": "按角色筛选",
        "searchPlaceholder": "搜索邮箱或昵称",
        "statusPlaceholder": "Filter by status"
      },
      "form": {
        "accountNumber": "钱包号",
        "accountNumberInvalid": "5-6位数字，无前导零，不含4和7",
        "email": "邮箱",
        "nickname": "昵称",
        "password": "密码",
        "placeholders": {
          "email": "输入邮箱",
          "nickname": "输入昵称",
          "password": "Enter password"
        },
        "role": "角色",
        "status": "状态"
      },
      "messages": {
        "newPasswordIs": "New password is: {{password}}",
        "passwordUpdateFailed": "密码更新失败",
        "passwordUpdatedSuccess": "密码更新成功",
        "userCreateFailed": "创建用户失败",
        "userCreatedSuccess": "用户创建成功",
        "userDeleteFailed": "删除用户失败",
        "userDeletedSuccess": "用户已删除",
        "userDisabled": "用户已禁用",
        "userEnabled": "用户已启用",
        "userUpdateFailed": "更新用户失败",
        "userUpdatedSuccess": "用户更新成功"
      },
      "modals": {
        "createTitle": "新建用户",
        "editTitle": "编辑用户",
        "passwordTitle": "Change Password"
      },
      "pagination": {
        "total": "Total {{total}} users"
      },
      "passwordForm": {
        "confirmPassword": "确认密码",
        "newPassword": "新密码",
        "placeholders": {
          "confirmPassword": "Re-enter new password",
          "newPassword": "输入新密码"
        },
        "submit": "更新密码",
        "validation": {
          "confirmPasswordRequired": "请确认新密码",
          "newPasswordRequired": "请输入新密码",
          "passwordMin8": "密码至少8位",
          "passwordMismatch": "两次密码不一致",
          "passwordMustContainLettersAndNumbers": "Password must contain both letters and numbers"
        }
      },
      "roles": {
        "audit": "Audit",
        "customerService": "客服",
        "operation": "运营",
        "superAdmin": "超级管理员",
        "user": "普通用户"
      },
      "status": {
        "active": "正常",
        "suspended": "Suspended"
      },
      "table": {
        "actions": "操作",
        "createdAt": "创建时间",
        "email": "邮箱",
        "id": "ID",
        "mtAccountCount": "MT账户数",
        "nickname": "昵称",
        "role": "角色",
        "status": "状态"
      },
      "title": "用户管理"
    },
    "wallet": {
      "accountNumber": "钱包号",
      "add": "增加",
      "adjustBalance": "调整余额",
      "adjustFailed": "调整失败",
      "adjustSuccess": "余额已调整",
      "deduct": "扣除",
      "noUsers": "未找到用户",
      "reason": "调整原因...",
      "searchPlaceholder": "搜索邮箱或钱包号...",
      "title": "钱包管理",
      "walletFor": "钱包 -"
    }
  },
  "app": {
    "name": "AntTrader"
  },
  "auth": {
    "fields": {
      "confirmPassword": "Confirm password",
      "email": "邮箱",
      "password": "密码"
    },
    "forgotPassword": {
      "backToLogin": "Back to Login",
      "hint": "请联系管理员或支持人员重置密码。",
      "title": "重置密码"
    },
    "login": {
      "forgotPassword": "忘记密码？",
      "login": "立即登录",
      "noAccount": "Don't have an account?",
      "registerNow": "立即注册",
      "rememberMe": "记住我",
      "signingIn": "登录中...",
      "subtitle": "这是一个测试不具备责任能力"
    },
    "messages": {
      "fetchMeFailed": "Failed to load user profile",
      "loginFailed": "登录失败，请检查邮箱和密码",
      "loginSuccess": "登录成功",
      "logoutSuccess": "已退出登录",
      "registerFailed": "注册失败，请稍后重试",
      "registerSuccess": "注册成功，请登录"
    },
    "register": {
      "haveAccount": "已有账号？",
      "loginNow": "立即登录",
      "register": "注册",
      "signingUp": "注册中...",
      "subtitle": "创建新账号"
    },
    "validation": {
      "confirmPasswordRequired": "请确认密码",
      "emailInvalid": "请输入有效的邮箱地址",
      "emailRequired": "请输入邮箱",
      "passwordMin8": "密码至少8位",
      "passwordMismatch": "两次密码不一致",
      "passwordRequired": "请输入密码"
    }
  },
  "autoTrading": {
    "logs": {
      "columns": {
        "action": "操作",
        "price": "价格",
        "profit": "盈亏",
        "symbol": "品种",
        "ticket": "Ticket",
        "time": "时间",
        "volume": "数量"
      },
      "empty": "暂无交易日志",
      "title": "最近交易日志"
    },
    "messages": {
      "loadFailed": "加载自动交易数据失败",
      "toggleFailed": "Failed to toggle auto trading"
    },
    "settings": {
      "maxDailyLoss": "每日最大亏损",
      "maxDailyLossHint": "日亏损超过此值时自动停止交易",
      "maxDrawdownPercent": "最大回撤%",
      "maxDrawdownPercentHint": "回撤超过此值时自动停止交易",
      "maxLotSize": "最大手数",
      "maxLotSizeHint": "每笔交易最大交易量(手)",
      "maxPositions": "最大持仓数",
      "maxPositionsHint": "同时持有的最大仓位数量",
      "maxRiskPercent": "最大风险%",
      "maxRiskPercentHint": "每笔交易风险占余额百分比",
      "saveFailed": "Failed to save settings",
      "saveSuccess": "设置已保存",
      "title": "全局风控设置"
    },
    "status": {
      "activeStrategies": "活跃策略",
      "disabled": "自动交易已关闭",
      "enabled": "自动交易已开启",
      "todayExecutions": "Today's Executions",
      "todayProfit": "Today's Profit"
    },
    "title": "自动交易"
  },
  "common": {
    "active": "正常",
    "back": "返回",
    "cancel": "取消",
    "clear": "清除",
    "close": "平仓时间",
    "comingSoon": "即将上线",
    "confirm": "确定",
    "copied": "已复制",
    "copy": "复制",
    "copyFailed": "复制失败",
    "create": "新增",
    "created": "创建时间",
    "currentPosition": "📊 当前持仓",
    "delete": "删除",
    "deleteFailed": "删除失败",
    "deleteSelected": "删除选中 ({{count}})",
    "deleted": "已删除",
    "disable": "禁用",
    "disabled": "已禁用",
    "edit": "编辑",
    "enable": "启用",
    "enabled": "已启用",
    "error": "错误",
    "gotIt": "我知道了",
    "hideDetails": "收起详情",
    "inactive": "停用",
    "indicatorSettings": "{{name}} 设置",
    "lineColor": "线颜色",
    "loading": "加载中...",
    "loadingFailed": "加载失败",
    "months": {
      "jan": "1月",
      "jul": "7月"
    },
    "next": "下一步",
    "no": "No",
    "noData": "暂无数据",
    "noOpenPositionsForSymbol": "{{symbol}} 暂无持仓",
    "none": "无",
    "ok": "OK",
    "operationFailed": "操作失败",
    "pageError": "页面错误",
    "pageUnderDevelopment": "此页面开发中",
    "pleaseWait": "请稍候...",
    "previous": "上一步",
    "refresh": "刷新",
    "remove": "移除",
    "required": "必填",
    "retry": "重试",
    "save": "保存",
    "saveFailed": "保存失败",
    "saveSuccess": "保存成功",
    "searching": "搜索中...",
    "selectSymbolToViewChart": "选择品种查看图表",
    "send": "发送",
    "showDetails": "查看详情",
    "time": {
      "day": "{{n}}天",
      "hour": "{{n}}时",
      "lessThanMinute": "<1m",
      "minute": "{{n}}分"
    },
    "totalItems": "共 {{count}} 项",
    "translate": "翻译",
    "unexpectedError": "发生了意外错误",
    "unknown": "未知",
    "updated": "已更新",
    "viewOriginal": "查看原文",
    "viewTranslation": "查看译文",
    "yes": "是",
    "you": "你"
  },
  "errors": {
    "access_denied": "无权限访问",
    "account_connected": "连接成功",
    "account_connection_failed": "无法连接到交易服务器",
    "account_not_found": "账户不存在",
    "ai": {
      "api_key_required": "API Key 不能为空",
      "base_url_required": "Base URL 不能为空",
      "base_url_scheme_invalid": "Base URL 必须以 http:// 或 https:// 开头",
      "base_url_should_not_end_with_chat_completions": "Base URL 不应以 /chat/completions 结尾",
      "config_service_not_initialized": "AI 配置服务未初始化",
      "config_valid": "AI 配置有效",
      "failed_to_create_request": "创建请求失败",
      "forbidden_quota": "Quota exceeded",
      "free_tier_exhausted": "AI 模型免费额度已耗尽：请在模型供应商管理后台关闭“use free tier only”或更换付费 Key。",
      "invalid_base_url": "Base URL 无效",
      "invalid_provider": "服务商无效",
      "no_trade_data_available": "暂无可用交易数据",
      "not_configured": "AI 未配置：请先到 AI 设置中启用并配置。",
      "probe_ok": "OK",
      "probe_ok_no_models": "正常（未返回 models）",
      "provider_required": "请先选择服务商",
      "provider_returned_empty_message": "AI 服务返回空消息",
      "rate_limited": "AI 服务触发限流/额度不足（429/资源耗尽）。请稍后重试或更换可用的 API Key/模型配置。",
      "request_failed": "API 请求失败"
    },
    "auto_trading_disabled": "自动交易已关闭",
    "auto_trading_enabled": "自动交易已开启",
    "connection_failed": {
      "content": "Unable to connect to the server. Please check your network and try again.",
      "title": "连接失败"
    },
    "email_already_registered": "邮箱已注册",
    "invalid_credentials": "账号或密码错误",
    "not_authenticated": "未登录",
    "schedule_service_not_available": "调度服务不可用",
    "translate_failed": "翻译失败",
    "user_not_found": "用户不存在"
  },
  "language": {
    "english": "English",
    "japanese": "日本語",
    "simplifiedChinese": "简体中文",
    "traditionalChinese": "繁體中文",
    "vietnamese": "Tiếng Việt"
  },
  "market": {
    "allSymbols": "全部品种",
    "ask": "卖价",
    "bid": "买价",
    "common": "常用",
    "emptyWatchlist": "暂无自选",
    "loadingSymbols": "加载中...",
    "mid": "中间价",
    "noSymbolSelected": "选择一个品种以查看行情数据",
    "noSymbolsFound": "未找到品种",
    "popularSymbols": "热门品种",
    "searchPlaceholder": "搜索品种（如 EURUSD, XAUUSD）",
    "searchSymbol": "Search symbol...",
    "selectAccount": "选择交易账户",
    "selectSymbol": "选择品种",
    "spread": "点差",
    "watchlist": "自选"
  },
  "marketplace": {
    "author": {
      "avgRating": "平均评分",
      "empty": "暂无已发布策略。前往策略库发布一个。",
      "published": "已发布"
    },
    "card": {
      "by": "by",
      "free": "免费",
      "owned": "购买日期",
      "subscribers": "订阅者",
      "winRate": "胜率"
    },
    "detail": {
      "assetClass": "资产类别",
      "author": "作者",
      "commentPlaceholder": "写评论...",
      "comments": "评论",
      "description": "描述",
      "getFree": "免费获取",
      "rentPrice": "¥{{amount}} / 月",
      "subscribers": "订阅者",
      "yourRating": "我的评分"
    },
    "empty": "暂无已发布策略",
    "filterByClass": "按资产类别筛选",
    "messages": {
      "commentFailed": "评论失败",
      "commentPosted": "评论已发布",
      "loginFirst": "请先登录",
      "paymentComingSoon": "支付功能即将上线",
      "rateFailed": "评分失败",
      "rated": "评分已提交",
      "subscribeFailed": "失败",
      "subscribed": "已添加到您的购买"
    },
    "noSubscriptions": "暂无订阅",
    "payment": {
      "alreadyPurchased": "您已拥有此策略。",
      "balanceAfter": "购买后余额",
      "cancel": "取消",
      "confirm": "确认购买",
      "depositPrompt": "请先充值后再继续。",
      "goToDeposit": "充值",
      "insufficientBalance": "余额不足",
      "oneTimePurchase": "¥{{amount}} 一次性买断",
      "price": "价格",
      "purchaseFailed": "购买失败，请重试。",
      "purchaseSuccess": "购买成功！策略已添加到您的库中。",
      "purchasing": "处理中...",
      "strategyName": "策略",
      "title": "确认购买",
      "walletBalance": "我的余额"
    },
    "publish": "发布策略",
    "purchases": {
      "empty": "暂无购买记录。前往市场发现策略。",
      "status": "状态",
      "strategy": "策略"
    },
    "searchPlaceholder": "搜索策略...",
    "sort": {
      "newest": "最新",
      "performance": "最佳表现",
      "popular": "最热门",
      "priceAsc": "价格：从低到高",
      "priceDesc": "价格：从高到低",
      "rating": "最高评分",
      "score": "综合评分"
    },
    "subtitle": "发现、购买和使用社区策略",
    "tabs": {
      "author": "作者中心",
      "marketplace": "策略市场",
      "purchases": "我的购买",
      "subscriptions": "我的订阅"
    },
    "title": "策略市场"
  },
  "menu": {
    "accounts": "账户",
    "aiAssistant": "AI助手",
    "algoDashboard": "算法看板",
    "analytics": "分析",
    "assetAnalysis": "AI分析",
    "assets": "策略资产",
    "autoTrading": "自动交易",
    "dashboard": "仪表盘",
    "devGroup": "策略开发",
    "experiments": "策略实验",
    "indicatorCatalog": "指标目录",
    "logs": "系统日志",
    "market": "策略市场",
    "marketRegime": "市场状态",
    "marketTools": "市场分析工具",
    "marketplace": "市场",
    "opsGroup": "策略运营",
    "schedules": "策略调度",
    "strategies": "策略管理",
    "strategy": "策略",
    "strategyLibrary": "策略库",
    "strategyWorkspace": "策略工作台",
    "trading": "交易",
    "wallet": "钱包"
  },
  "notifications": {
    "actions": {
      "clearAll": "清空全部",
      "clearAllConfirm": "确定清空所有通知？",
      "markAllAsRead": "全部已读"
    },
    "all": "全部",
    "clearAll": "清空全部",
    "confirmClearAll": "确定清空所有通知？",
    "empty": "暂无通知",
    "markAllRead": "全部已读",
    "stream": {
      "autoTrading": {
        "fallback": "Auto trading event triggered",
        "title": "自动交易"
      },
      "riskAlert": {
        "fallback": "Alert type: {{alertType}}",
        "title": "风控告警"
      },
      "strategyExecution": {
        "completed": "{{symbol}} {{action}} 已完成",
        "failed": "Execution failed: {{error}}",
        "title": "策略执行"
      },
      "strategySignal": {
        "message": "{{symbol}} triggered {{signalType}}",
        "title": "策略信号"
      }
    },
    "tabs": {
      "all": "全部 ({{count}})",
      "unread": "Unread ({{count}})"
    },
    "title": "通知中心",
    "types": {
      "risk_alert": "风控告警",
      "signal": "信号",
      "strategy_execution": "策略执行",
      "system": "System",
      "trade": "交易"
    },
    "unread": "未读"
  },
  "profile": {
    "lastLogin": "最后登录",
    "nickname": "昵称",
    "registered": "Registered",
    "role": "角色",
    "status": "状态",
    "title": "个人信息"
  },
  "share": {
    "actions": "操作",
    "createNew": "创建新分享链接",
    "createdAt": "创建时间",
    "deleteConfirm": "Delete this share link?",
    "empty": "暂无分享链接",
    "expires": "过期时间",
    "positions": "持仓",
    "showPositions": "显示持仓",
    "title": "分享管理",
    "token": "分享链接",
    "userId": "普通用户",
    "views": "浏览量"
  },
  "sharePage": {
    "avgHolding": "平均持仓时长",
    "avgLoss": "平均亏损",
    "avgWin": "平均盈利",
    "bestTrade": "最佳交易",
    "bySymbol": "品种业绩",
    "closeTime": "平仓时间",
    "count": "笔数",
    "disclaimer": "过往业绩不代表未来表现。",
    "equityCurve": "净值曲线",
    "expired": "该分享链接已过期",
    "footer": "由 AntTrader 生成",
    "language": "语言",
    "loadFailed": "加载分享数据失败",
    "losingTrades": "亏损笔数",
    "maxDrawdown": "最大回撤",
    "netProfit": "净盈亏",
    "noPositions": "暂无持仓",
    "noTrades": "暂无交易记录",
    "notFound": "未找到",
    "openPrice": "开仓价",
    "positions": "当前持仓",
    "positionsLocked": "创建者未开放持仓查看",
    "profit": "盈亏",
    "profitFactor": "盈利因子",
    "sharpeRatio": "夏普比率",
    "side": "方向",
    "subtitle": "真实交易成绩",
    "symbol": "品种",
    "title": "交易业绩",
    "totalReturn": "净盈亏",
    "totalTrades": "总交易数",
    "totalVolume": "总交易量",
    "tradeRecords": "交易记录",
    "volume": "数量",
    "winRate": "胜率",
    "winningTrades": "盈利笔数",
    "worstTrade": "最差交易"
  },
  "symbolDetection": {
    "label": "识别到的交易品种",
    "loading": "正在解析…",
    "noSymbols": "未识别到交易品种，请尝试包含具体的品种名称（如\"比特币\"、\"EURUSD\"、\"黄金\"）",
    "resolvedTooltip": "broker: {{broker}} | 模式: {{mode}}",
    "tradeMode": {
      "disabled": "已禁用",
      "longOnly": "仅做多",
      "longShort": "多空双向",
      "shortOnly": "仅做空",
      "unknown": "未知"
    },
    "unresolvedTooltip": "尚未绑定交易账户，无法解析"
  },
  "topbar": {
    "logout": "退出登录",
    "profile": "个人信息",
    "settings": "设置",
    "switchToAdmin": "切换到管理",
    "systemOk": "系统正常运行",
    "user": "普通用户"
  },
  "wallet": {
    "accountNumber": "钱包号",
    "balance": "余额",
    "currency": "币种",
    "deposit": "充值",
    "frozen": "冻结",
    "frozenBalance": "冻结",
    "history": "历史记录",
    "table": {
      "amount": "金额",
      "balanceAfter": "调整后余额",
      "description": "描述",
      "time": "时间",
      "type": "类型"
    },
    "title": "我的钱包",
    "transactions": "交易记录",
    "txType": {
      "adjustment": "余额调整",
      "deposit": "充值",
      "fee": "手续费",
      "reversal": "冲正",
      "withdrawal": "提取"
    },
    "withdraw": "提取"
  }
} as const;
export default Base;
