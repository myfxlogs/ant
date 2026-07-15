// Auto-generated from proto/ant/v1/i18n/base_zh-cn.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const Base = {
  "admin": {
    "dashboard": {
      "logs": {
        "moduleMap": {
          "accountManagement": "账户管理",
          "systemConfig": "系统配置",
          "trading": "交易",
          "userManagement": "用户管理"
        },
        "actionType": "操作",
        "failed": "失败",
        "module": "模块",
        "status": "状态",
        "success": "成功",
        "target": "目标",
        "time": "时间"
      },
      "riskMetrics": {
        "orderCloseFailed": "平仓失败",
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
        "noData": "暂无窗口指标数据",
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
      "errors": {
        "loadFailed": "失败 to load dashboard data"
      },
      "activeUsers": "活跃用户",
      "loadFailed": "加载仪表盘数据失败",
      "mtAccounts": "MT账户数",
      "onlineAccounts": "在线账户",
      "recentLogs": "最近日志",
      "title": "管理仪表盘",
      "todayProfit": "今日盈亏",
      "todayTrades": "今日交易",
      "totalUsers": "总用户数",
      "verifiedUsers": "已验证用户",
      "activeSubs": "活跃订阅",
      "monthlyRevenue": "月度收入",
      "totalRevenue": "总收入",
      "marketStrategies": "Market 策略",
      "marketSales": "市场销售额",
      "marketRevenue": "市场收入",
      "validateTotal": "Validate 总计",
      "validatePass": "验证通过",
      "validateReject": "验证拒绝",
      "validateError": "验证错误",
      "orderSendSuccess": "下单成功",
      "orderSendFailed": "下单失败",
      "orderCloseSuccess": "平仓成功",
      "orderCloseFailed": "平仓失败",
      "rejectCount": "拒绝次数"
    },
    "userManagement": {
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
      "form": {
        "placeholders": {
          "email": "输入邮箱",
          "nickname": "输入昵称",
          "password": "输入密码"
        },
        "accountNumber": "钱包号",
        "accountNumberInvalid": "5-6位数字，无前导零，不含4和7",
        "email": "邮箱",
        "nickname": "昵称",
        "password": "密码",
        "role": "角色",
        "status": "状态"
      },
      "passwordForm": {
        "placeholders": {
          "confirmPassword": "再次输入新密码",
          "newPassword": "输入新密码"
        },
        "validation": {
          "confirmPasswordRequired": "请确认新密码",
          "newPasswordRequired": "请输入新密码",
          "passwordMin8": "密码至少8位",
          "passwordMismatch": "两次密码不一致",
          "passwordMustContainLettersAndNumbers": "密码必须包含字母和数字"
        },
        "confirmPassword": "确认密码",
        "newPassword": "新密码",
        "submit": "更新密码"
      },
      "actions": {
        "changePassword": "修改密码",
        "details": "详情",
        "disable": "禁用",
        "enable": "启用"
      },
      "deleteConfirm": {
        "batchDeleteConfirm": "确认删除 {{count}} 个用户？此操作不可撤销。",
        "batchDeletePartial": "已删除 {{deleted}} 个，{{failed}} 个失败",
        "batchDeleteSuccess": "已删除 {{count}} 个用户",
        "title": "确认删除此用户？此操作不可撤销。"
      },
      "filters": {
        "rolePlaceholder": "按角色筛选",
        "searchPlaceholder": "搜索邮箱或昵称",
        "statusPlaceholder": "按状态筛选"
      },
      "messages": {
        "newPasswordIs": "新密码为: {{password}}",
        "passwordUpdateFailed": "密码更新失败",
        "passwordUpdatedSuccess": "密码更新成功",
        "userCreateFailed": "创建用户失败",
        "userCreatedSuccess": "用户创建成功",
        "userDeleteFailed": "删除用户失败",
        "userDeletedSuccess": "用户已删除",
        "userDisabled": "用户已禁用",
        "userEnabled": "用户已启用",
        "userUpdateFailed": "更新用户失败",
        "userUpdatedSuccess": "用户更新成功",
        "loadUsersFailed": "失败 to load users"
      },
      "modals": {
        "createTitle": "新建用户",
        "editTitle": "编辑用户",
        "passwordTitle": "修改密码"
      },
      "pagination": {
        "total": "共 {{total}} 位用户"
      },
      "roles": {
        "audit": "审计",
        "customerService": "客服",
        "operation": "运营",
        "superAdmin": "超级管理员",
        "user": "普通用户"
      },
      "status": {
        "active": "正常",
        "suspended": "已停用"
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
      "addUser": "新建用户",
      "title": "用户管理"
    },
    "config": {
      "messages": {
        "disabled": "已禁用",
        "enabled": "已启用",
        "loadFailed": "加载配置失败",
        "operationFailed": "操作失败",
        "updateFailed": "更新配置失败",
        "updated": "配置已更新"
      },
      "placeholders": {
        "apiKey": "输入API Key",
        "baseUrl": "输入Base URL",
        "configValue": "输入配置值",
        "description": "输入描述",
        "json": "输入JSON",
        "model": "输入模型名称"
      },
      "providerOptions": {
        "custom": "自定义 / OpenAI 兼容",
        "deepseek": "DeepSeek",
        "zhipu": "智谱AI"
      },
      "validation": {
        "apiKeyRequired": "API Key不能为空",
        "greenMaxFailedRunsNonNegative": "绿色最大失败次数需≥0",
        "greenSuccessRateRange": "绿色成功率需在0-100之间",
        "jsonEmpty": "JSON不能为空",
        "jsonInvalid": "JSON格式无效",
        "minSampleSizeNonNegative": "最小样本量需≥0",
        "modelRequired": "模型名称不能为空",
        "yellowNotGreaterThanGreen": "黄色阈值不能超过绿色阈值",
        "yellowSuccessRateRange": "黄色成功率需在0-100之间"
      },
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
      "modelName": "模型名称",
      "off": "关",
      "on": "开",
      "provider": "提供商",
      "status": "状态",
      "strategyHealthConfig": "策略健康度配置",
      "thresholdDesc": "阈值描述",
      "thresholdInfo": "阈值说明",
      "title": "系统配置",
      "toggle": "切换",
      "updatedAt": "更新时间",
      "value": "值",
      "apiKey": "API 密钥"
    },
    "jurisdiction": {
      "messages": {
        "countryAddFailed": "添加国家失败",
        "countryAdded": "国家已添加",
        "countryRemoveFailed": "移除国家失败",
        "countryRemoved": "国家已移除",
        "kycUpdateFailed": "更新KYC状态失败",
        "kycUpdated": "KYC状态已更新",
        "overrideUpdateFailed": "更新制裁豁免失败",
        "overrideUpdated": "豁免状态已更新"
      },
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
      "override": "豁免",
      "overrideWarning": "此用户来自受制裁国家，授予豁免将允许交易。",
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
    "aiGateway": {
      "errors": {
        "loadProviders": "失败 to load providers",
        "toggleFailed": "切换失败",
        "loadModels": "失败 to load models"
      },
      "columns": {
        "baseUrl": "基础 URL",
        "apiKey": "API 密钥"
      },
      "addProviderPending": "添加 provider feature pending backend support",
      "title": "AI 网关管理",
      "description": "管理 AI 提供商、模型和定价。用户从可用模型中选择，按 token 从钱包扣费。",
      "addProvider": "添加提供商",
      "provider": "提供商",
      "configured": "已配置",
      "notConfigured": "未配置",
      "models": "模型",
      "editProvider": "编辑提供商",
      "providerId": "提供商 ID",
      "providerIdRequired": "请输入提供商ID",
      "displayName": "显示名称",
      "displayNameRequired": "请输入显示名称",
      "baseUrl": "基础 URL",
      "baseUrlRequired": "Please enter 基础 URL",
      "apiKeyLabel": "API 密钥",
      "apiKeyEditHint": "留空则保留现有密钥",
      "apiKeyHint": "API密钥，静态加密存储",
      "apiKeyEditPlaceholder": "留空则保留",
      "editModel": "编辑模型",
      "addModel": "添加模型",
      "modelName": "模型名称",
      "modelNameRequired": "请输入模型名称",
      "priceInput": "Input 价格 ($/1M)",
      "priceOutput": "Output 价格 ($/1M)",
      "confirmDeleteModel": "删除 this model?",
      "noModels": "无模型"
    },
    "account": {
      "errors": {
        "loadFailed": "失败 to load accounts",
        "freezeFailed": "冻结失败",
        "unfreezeFailed": "解冻失败"
      },
      "columns": {
        "id": "ID",
        "user": "用户",
        "login": "登录",
        "type": "类型",
        "broker": "经纪商",
        "status": "状态",
        "balance": "余额",
        "createdAt": "创建时间",
        "action": "操作",
        "server": "服务器",
        "equity": "净值",
        "margin": "保证金",
        "time": "时间",
        "detail": "详情"
      },
      "frozen": "账户 frozen",
      "unfrozen": "账户 unfrozen",
      "detail": "详情",
      "unfreeze": "解冻",
      "confirmFreeze": "冻结此账户？",
      "freeze": "冻结",
      "title": "账户管理",
      "searchPlaceholder": "搜索 accounts",
      "status": "状态",
      "online": "在线",
      "offline": "离线",
      "auditLogs": "Audit 日志"
    },
    "settings": {
      "columns": {
        "key": "设置键",
        "value": "值",
        "action": "操作"
      },
      "saveSuccess": "保存成功",
      "saveFailed": "保存 failed",
      "deleted": "已删除",
      "deleteFailed": "删除 failed",
      "actionFailed": "操作失败",
      "confirmDelete": "确认 delete?",
      "title": "Agent 管理 设置",
      "addSetting": "添加 Setting",
      "permissionRules": "权限规则 (permission.rule.N)",
      "permissionFormat": "格式：",
      "permissionExample": "示例：",
      "permissionAddRule": "添加 rule: create setting with key ",
      "addManagedSetting": "添加 Managed Setting",
      "settingKey": "设置键",
      "keyPlaceholder": "例如：allowed_models, disable_live_trading, permission.rule.1",
      "valuePlaceholder": "例如：claude-sonnet-5,deepseek-v4"
    },
    "billing": {
      "columns": {
        "user": "用户",
        "plan": "方案",
        "status": "状态",
        "cycle": "周期",
        "price": "价格",
        "autoRenew": "自动续费",
        "periodStart": "周期开始",
        "periodEnd": "周期结束",
        "createdAt": "创建时间",
        "type": "类型",
        "amount": "金额",
        "balanceBefore": "余额 Before",
        "balanceAfter": "余额 After",
        "description": "描述",
        "time": "时间"
      },
      "title": "计费管理",
      "monthlyRevenue": "月度收入",
      "totalRevenue": "总收入",
      "activeSubs": "活跃订阅",
      "txRecords": "交易记录",
      "planRevenue": "方案收入明细",
      "activeCount": "活跃",
      "subscriptions": "订阅",
      "filterByPlan": "按方案筛选",
      "planFree": "免费",
      "planPro": "专业版",
      "planEnterprise": "企业版",
      "filterByStatus": "按状态筛选",
      "statusActive": "活跃",
      "statusCancelled": "已取消",
      "statusExpired": "已过期",
      "walletTransactions": "钱包交易",
      "filterByType": "筛选 by type",
      "txPurchase": "购买",
      "txSale": "销售",
      "txPlatformFee": "平台费用",
      "txDeposit": "充值",
      "txWithdrawal": "提现"
    },
    "logs": {
      "columns": {
        "time": "时间",
        "module": "模块",
        "actionType": "操作类型",
        "target": "目标",
        "status": "状态",
        "ip": "IP地址",
        "action": "操作",
        "details": "详情"
      },
      "modules": {
        "userManagement": "用户 管理",
        "accountManagement": "账户管理",
        "trading": "交易",
        "systemConfig": "系统配置"
      },
      "errors": {
        "loadFailed": "失败 to load logs"
      },
      "actions": {
        "create": "创建",
        "update": "更新",
        "delete": "删除",
        "disable": "禁用",
        "enable": "启用",
        "freeze": "冻结",
        "unfreeze": "解冻"
      },
      "title": "操作日志",
      "filterModule": "按模块筛选",
      "filterAction": "筛选 by action"
    },
    "deposit": {
      "table": {
        "user": "用户",
        "amount": "USDT 金额",
        "amountUsd": "USD 到账",
        "txHash": "交易哈希",
        "status": "状态",
        "reviewNote": "审核备注",
        "time": "时间",
        "action": "操作"
      },
      "approved": "充值 approved and wallet credited.",
      "approveFailed": "失败 to approve deposit.",
      "rejected": "充值 rejected.",
      "rejectFailed": "失败 to reject deposit.",
      "approve": "通过",
      "reject": "拒绝",
      "title": "充值管理",
      "allStatuses": "全部状态",
      "statusPending": "待处理",
      "statusApproved": "已通过",
      "statusRejected": "已拒绝",
      "approveTitle": "Approve 充值",
      "rejectTitle": "Reject 充值",
      "reviewNoteLabel": "审核备注 (optional)",
      "reviewNotePlaceholder": "添加 a note for this review...",
      "approveWarning": "通过后用户钱包将立即到账。"
    },
    "wallet": {
      "errors": {
        "noUserSelected": "未选择用户"
      },
      "messages": {
        "adjustSuccess": "余额 adjusted successfully",
        "adjustFailed": "调整失败"
      },
      "columns": {
        "walletNumber": "钱包号",
        "email": "邮箱",
        "nickname": "昵称",
        "type": "类型",
        "amount": "金额",
        "balanceAfter": "余额 After",
        "description": "描述",
        "time": "时间",
        "balance": "余额",
        "frozen": "冻结",
        "currency": "币种"
      },
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
      "walletFor": "钱包 -",
      "unassigned": "未分配",
      "userList": "用户列表",
      "noMatch": "无匹配用户",
      "walletDetail": "钱包详情",
      "transactions": "交易记录",
      "adjustReason": "原因"
    },
    "header": {
      "admin": "管理",
      "adminMode": "管理员模式",
      "adminPanel": "管理后台",
      "backToUser": "返回用户端",
      "logout": "退出登录"
    },
    "sidebar": {
      "accountManagement": "账户管理",
      "agentSettings": "Agent 设置",
      "aiGateway": "AI 网关",
      "billing": "计费管理",
      "dashboard": "仪表盘",
      "deposits": "充值管理",
      "jurisdiction": "管辖权管理",
      "monitoring": "监控与告警",
      "operationLogs": "操作日志",
      "shareManagement": "分享分析",
      "sre": "SRE 控制",
      "strategies": "策略管理",
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
      "pendingOrders": "挂单",
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
    "walletCalculator": {
      "title": "Token ↔ USD计算器",
      "selectModel": "选择模型（定价基准）",
      "usdAmount": "USD 金额",
      "tokenAmount": "Token 金额",
      "fillResult": "填入结果"
    }
  },
  "autoTrading": {
    "logs": {
      "columns": {
        "action": "操作",
        "price": "价格",
        "profit": "盈亏",
        "symbol": "品种",
        "ticket": "单号",
        "time": "时间",
        "volume": "数量"
      },
      "empty": "暂无交易日志",
      "title": "最近交易日志"
    },
    "messages": {
      "loadFailed": "加载自动交易数据失败",
      "toggleFailed": "切换自动交易失败"
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
      "saveFailed": "保存设置失败",
      "saveSuccess": "设置已保存",
      "title": "全局风控设置"
    },
    "status": {
      "activeStrategies": "活跃策略",
      "disabled": "自动交易已关闭",
      "enabled": "自动交易已开启",
      "todayExecutions": "今日成交",
      "todayProfit": "今日盈亏"
    },
    "title": "自动交易"
  },
  "notifications": {
    "stream": {
      "autoTrading": {
        "fallback": "自动交易事件触发",
        "title": "自动交易"
      },
      "riskAlert": {
        "fallback": "警报类型: {{alertType}}",
        "title": "风控告警"
      },
      "strategyExecution": {
        "completed": "{{symbol}} {{action}} 已完成",
        "failed": "执行失败: {{error}}",
        "title": "策略执行"
      },
      "strategySignal": {
        "message": "{{symbol}} 触发 {{signalType}}",
        "title": "策略信号"
      }
    },
    "actions": {
      "clearAll": "清空全部",
      "clearAllConfirm": "确定清空所有通知？",
      "markAllAsRead": "全部已读"
    },
    "tabs": {
      "all": "全部 ({{count}})",
      "unread": "未读 ({{count}})"
    },
    "types": {
      "risk_alert": "风控告警",
      "signal": "信号",
      "strategy_execution": "策略执行",
      "system": "系统",
      "trade": "交易"
    },
    "all": "全部",
    "clearAll": "清空全部",
    "confirmClearAll": "确定清空所有通知？",
    "empty": "暂无通知",
    "markAllRead": "全部已读",
    "title": "通知中心",
    "unread": "未读"
  },
  "wallet": {
    "deposit": {
      "table": {
        "amount": "USDT 金额",
        "amountUsd": "USD 到账",
        "status": "状态",
        "time": "时间",
        "txHash": "交易哈希"
      },
      "address": "收款地址",
      "addressCopied": "地址已复制到剪贴板",
      "amountLabel": "USDT 金额",
      "button": "新建充值",
      "copy": "复制",
      "exchangeRate": "汇率",
      "failed": "提交充值请求失败。",
      "history": "充值记录",
      "modalTitle": "提交充值请求",
      "network": "网络",
      "notConfigured": "USDT 充值尚未配置，请联系客服。",
      "notice": "请仅通过指定网络发送 USDT。发送其他代币或使用不同网络可能导致永久丢失。发送后，请提交充值请求并填写金额和可选的交易哈希，等待管理员审核。",
      "submit": "提交",
      "success": "充值请求已提交，请等待管理员审核。",
      "title": "充值",
      "txHashLabel": "交易哈希（可选）",
      "willCredit": "预计到账"
    },
    "table": {
      "amount": "金额",
      "balanceAfter": "调整后余额",
      "description": "描述",
      "time": "时间",
      "type": "类型"
    },
    "txType": {
      "adjustment": "余额调整",
      "deposit": "充值",
      "fee": "手续费",
      "reversal": "冲正",
      "withdrawal": "提取"
    },
    "accountNumber": "钱包号",
    "balance": "余额",
    "currency": "币种",
    "frozen": "冻结",
    "frozenBalance": "冻结",
    "history": "历史记录",
    "title": "我的钱包",
    "transactions": "交易记录",
    "withdraw": "提取"
  },
  "strategy": {
    "workspace": {
      "chartIndicators": {
        "overlay": "主图叠加",
        "subPane": "副图指标"
      }
    },
    "tuning": {
      "searchMethod": {
        "grid": "网格",
        "random": "随机"
      }
    },
    "backtest": {
      "canceled": "回测已取消",
      "lotSize": "手数",
      "strategyParameters": "策略 Parameters"
    },
    "chat": {
      "executionPlan": "Execution 方案",
      "codeGenerated": "代码已生成，使用下方按钮进行策略审查和回测。"
    },
    "aiChat": {
      "historyTab": "历史",
      "strategiesTab": "策略"
    },
    "templates": {
      "title": "策略 Templates",
      "saveCurrent": "保存 Current 策略",
      "lines": "条数",
      "chatEdit": "Chat 编辑",
      "source": "来源",
      "rename": "重命名",
      "confirmDelete": "删除 this strategy?",
      "noTemplates": "无已保存策略模板",
      "sourceCode": "策略 Source",
      "copyAll": "复制 All"
    },
    "live": {
      "stopSuccess": "策略 stopped",
      "stopFailed": "失败 to stop",
      "runId": "运行 ID",
      "account": "账户",
      "symbol": "品种",
      "timeframe": "周期",
      "mode": "模式",
      "signals": "信号",
      "errors": "错误",
      "startedAt": "已启动",
      "watchSignals": "Watch 信号",
      "confirmStop": "确定停止此策略？",
      "status": "状态",
      "totalSignals": "总计 信号",
      "stoppedAt": "已停止",
      "error": "错误",
      "title": "实盘策略监控",
      "activeTab": "活跃运行",
      "noActive": "无活跃策略",
      "historyTab": "运行历史",
      "noRuns": "无策略运行记录",
      "schedulesTab": "调度",
      "time": "时间",
      "signalType": "类型",
      "volume": "交易量",
      "price": "价格",
      "sl": "SL",
      "tp": "TP",
      "reason": "原因",
      "signalLog": "信号日志",
      "waitingSignals": "等待信号..."
    },
    "schedule": {
      "maxPositionsPlaceholder": "不限"
    },
    "ai": {
      "reviseHint": "先编写代码，然后让AI优化。",
      "explainHint": "编写代码以查看AI解释。",
      "settingsHint": "配置 AI 提供商和模型"
    },
    "validate": {
      "running": "校验运行中...",
      "errors": "错误",
      "warnings": "警告",
      "fixWithAI": "提交错误至 AI 修正",
      "parameters": "参数",
      "hints": "建议",
      "allClear": "所有检查通过 — 未发现问题。",
      "passed": "Validation passed — 保存 is now unlocked."
    },
    "importEA": {
      "writeTab": "策略 Code",
      "importTab": "导入EA",
      "codeTooShort": "请粘贴完整的EA/指标源码。",
      "pastePlaceholder": "粘贴MQL4/MQL5 EA代码...",
      "migration": "策略导入",
      "aiTranslate": "AI 翻译",
      "bridge": "盲区桥接",
      "analyze": "分析策略结构",
      "confirmImport": "确认导入",
      "tryAI": "AI 翻译补充",
      "apply": "应用到编辑器",
      "importSuccess": "MQL 源码已导入，点击「Apply to Editor」写入编辑器",
      "hint": "粘贴MQL4/MQL5代码并点击分析",
      "translate": "翻译为Go",
      "translating": "AI翻译中...",
      "bridgeBtn": "盲区桥接翻译",
      "bridgeSuccess": "桥接成功",
      "bridgeFailedTag": "桥接失败",
      "bridging": "AI 正在桥接盲区…",
      "bridgeFailedMsg": "Agent 无法自动桥接所有盲区",
      "noBridgeNeeded": "覆盖率 100%，无需桥接",
      "bridgeHint": "粘贴 MQL4/MQL5 EA 代码，AI 将自动翻译盲区为 Python 子集"
    },
    "version": {
      "loadFailed": "失败 to load versions",
      "rollbackFailed": "回滚失败",
      "loadVersionFailed": "失败 to load version",
      "loadDiffFailed": "失败 to load diff",
      "colVersion": "版本",
      "colSummary": "变更摘要",
      "colLang": "语言",
      "colHash": "哈希",
      "colDate": "日期",
      "colActions": "操作",
      "title": "Version 历史",
      "diff": "差异",
      "empty": "暂无版本历史",
      "history": "Version 历史"
    }
  },
  "accounts": {
    "bind": {
      "fields": {
        "alias": "账户 Alias"
      },
      "placeholders": {
        "alias": "可选自定义名称"
      },
      "messages": {
        "changeCredentials": "修改凭证"
      }
    },
    "messages": {
      "shareLinkCopied": "分享链接已复制到剪贴板",
      "shareLinkFailed": "失败 to create share link"
    }
  },
  "sre": {
    "breakers": {
      "columns": {
        "strategyId": "策略 ID",
        "state": "状态",
        "totalPnl": "总盈亏",
        "lossPercent": "亏损率",
        "tradeCount": "交易数",
        "trippedAt": "熔断时间",
        "tripReason": "熔断原因"
      },
      "title": "策略断路器",
      "stateClosed": "正常",
      "stateOpen": "已熔断",
      "stateHalfOpen": "半开（探测中）",
      "confirmReset": "重置此断路器？",
      "description": "策略 breaker status overview — auto-detects abnormal losses and trips",
      "noBreakers": "无已注册断路器"
    },
    "canary": {
      "columns": {
        "strategyId": "策略 ID",
        "versionTag": "版本标签",
        "accounts": "金丝雀账户",
        "startAt": "开始时间",
        "days": "天数",
        "status": "状态"
      },
      "promoted": "已晋升",
      "canarying": "金丝雀",
      "confirmDelete": "删除 this canary config?",
      "title": "金丝雀 Configuration",
      "description": "新策略版本先在少量账户上运行N天，再晋升至全部",
      "newCanary": "新建金丝雀",
      "noCanaries": "无金丝雀配置",
      "newCanaryTitle": "新建金丝雀",
      "accountIdsLabel": "金丝雀 账户 IDs (comma or newline separated)",
      "durationDays": "金丝雀 天数"
    },
    "killSwitch": {
      "description": "一键停止所有交易 — 需要输入 KILL 确认；5 分钟内可撤销",
      "engaged": "熔断开关 engaged — all trading stopped",
      "disarmed": "熔断开关 disarmed — trading normal",
      "status": "状态",
      "reason": "原因",
      "operator": "操作人",
      "engagedAt": "启用时间",
      "undo": "Undo 熔断开关",
      "disengage": "Disengage 熔断开关",
      "engage": "启用熔断开关",
      "confirmTitle": "启用 熔断开关 — Confirmation",
      "confirmEngage": "确认 启用",
      "confirmWarning": "此操作将立即停止所有账户的所有交易活动，包括挂单和已提交订单。输入原因并键入 KILL 确认。",
      "reasonLabel": "原因（必填）",
      "reasonPlaceholder": "例如：检测到市场异常波动，紧急停止所有交易",
      "typeKill": "键入 KILL 确认",
      "typeKillPlaceholder": "键入 KILL（大写）"
    }
  },
  "marketplace": {
    "publish": {
      "priceModel": {
        "free": "免费",
        "monthly": "按月订阅",
        "once": "One-时间 Purchase",
        "label": "定价方式"
      },
      "assetClass": {
        "label": "资产类别"
      },
      "riskLevel": {
        "label": "风险等级"
      },
      "return": "收益率",
      "winRate": "胜率",
      "trades": "交易数",
      "title": "发布到市场",
      "titleLabel": "标题",
      "titlePlaceholder": "e.g. Golden Cross 策略",
      "descriptionLabel": "描述",
      "descriptionPlaceholder": "描述策略逻辑、开平仓规则...",
      "priceAmount": "金额",
      "tags": "标签",
      "tagsPlaceholder": "输���后按回车添加标签",
      "codeSnippet": "策略 Preview (public)",
      "codeSnippetPlaceholder": "可选：分享策略代码片段或思路（所有人可见）",
      "includeBacktestSnapshot": "包含最新回测结果"
    },
    "author": {
      "avgRating": "平均评分",
      "empty": "暂无已发布策略。前往策略库发布一个。",
      "published": "已发布",
      "myStrategies": "My Published 策略",
      "publishNew": "Publish New 策略",
      "monthlyRevenue": "月度收入",
      "totalRevenue": "总收入",
      "goToLibrary": "Go to 策略 Library"
    },
    "card": {
      "by": "由",
      "free": "免费",
      "owned": "购买日期",
      "subscribers": "订阅者",
      "winRate": "胜率",
      "yourStrategy": "Your 策略"
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
      "yourRating": "我的评分",
      "runBacktest": "运行回测"
    },
    "messages": {
      "commentFailed": "评论失败",
      "commentPosted": "评论已发布",
      "loginFirst": "请先登录",
      "paymentComingSoon": "支付功能即将上线",
      "rateFailed": "评分失败",
      "rated": "评分已提交",
      "subscribeFailed": "失败",
      "subscribed": "已添加到您的购买",
      "published": "策略 published to marketplace!",
      "publishFailed": "失败 to publish strategy"
    },
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
    "purchases": {
      "empty": "暂无购买记录。前往市场发现策略。",
      "status": "状态",
      "strategy": "策略",
      "runBacktest": "运行回测"
    },
    "sort": {
      "newest": "最新",
      "performance": "最佳表现",
      "popular": "最热门",
      "priceAsc": "价格：从低到高",
      "priceDesc": "价格：从高到低",
      "rating": "最高评分",
      "score": "综合评分"
    },
    "tabs": {
      "author": "作者中心",
      "marketplace": "策略市场",
      "purchases": "我的购买",
      "subscriptions": "我的订阅"
    },
    "backtest": {
      "title": "策略 Backtest",
      "capital": "资金",
      "commission": "佣金",
      "leverage": "杠杆",
      "completed": "已完成",
      "totalReturn": "总计 Return",
      "maxDrawdown": "最大回撤",
      "sharpe": "夏普比率",
      "winRate": "胜率",
      "totalTrades": "总计 交易数",
      "equityCurve": "权益曲线",
      "protected": "策略 code is protected. Backtest runs on our servers.",
      "run": "运行回测",
      "idle": "设置参数并运行回测"
    },
    "empty": "暂无已发布策略",
    "filterByClass": "按资产类别筛选",
    "noSubscriptions": "暂无订阅",
    "searchPlaceholder": "搜索策略...",
    "subtitle": "发现、购买和使用社区策略",
    "title": "策略市场"
  },
  "onboarding": {
    "step1": {
      "title": "连接您的账户",
      "desc": "绑定您的 MT4/MT5 交易账户以开始。",
      "action": "Bind 账户"
    },
    "step2": {
      "title": "创建您的第一个策略",
      "desc": "使用 AI 从自然语言生成交易策略。",
      "action": "打开工作区"
    },
    "step3": {
      "title": "升级您的计划",
      "desc": "解锁更多 AI 代币、策略和实盘交易功能。",
      "action": "查看方案"
    },
    "subtitle": "3 个简单步骤即可开始",
    "dismiss": "知道了，忽略"
  },
  "auth": {
    "fields": {
      "confirmPassword": "确认密码",
      "email": "邮箱",
      "password": "密码",
      "login": "邮箱/账号"
    },
    "forgotPassword": {
      "backToLogin": "返回登录",
      "hint": "请联系管理员或支持人员重置密码。",
      "title": "重置密码"
    },
    "login": {
      "forgotPassword": "忘记密码？",
      "login": "立即登录",
      "noAccount": "没有账户？",
      "registerNow": "立即注册",
      "rememberMe": "记住我",
      "signingIn": "登录中...",
      "subtitle": "这是一个测试不具备责任能力"
    },
    "messages": {
      "fetchMeFailed": "加载用户信息失败",
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
      "passwordRequired": "请输入密码",
      "loginRequired": "请输入邮箱或账号"
    }
  },
  "common": {
    "months": {
      "jan": "1月",
      "jul": "7月"
    },
    "time": {
      "day": "{{n}}天",
      "hour": "{{n}}时",
      "lessThanMinute": "<1分钟",
      "minute": "{{n}}分"
    },
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
    "next": "下一步",
    "no": "否",
    "noData": "暂无数据",
    "noOpenPositionsForSymbol": "{{symbol}} 暂无持仓",
    "none": "无",
    "ok": "确定",
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
    "totalItems": "共 {{count}} 项",
    "translate": "翻译",
    "unexpectedError": "发生了意外错误",
    "unknown": "未知",
    "updated": "已更新",
    "viewOriginal": "查看原文",
    "viewTranslation": "查看译文",
    "yes": "是",
    "you": "你",
    "unsaved": "未保存",
    "saved": "已保存",
    "unknownError": "未知错误",
    "duplicateName": "名称已存在",
    "step1Label": "经纪商",
    "step2Label": "凭证",
    "step3Label": "确认",
    "unit": "单位",
    "action": "操作",
    "on": "开",
    "off": "关",
    "true": "是",
    "false": "否",
    "success": "成功",
    "failed": "失败",
    "reset": "重置",
    "saving": "保存中…"
  },
  "errors": {
    "ai": {
      "api_key_required": "API Key 不能为空",
      "base_url_required": "Base URL 不能为空",
      "base_url_scheme_invalid": "Base URL 必须以 http:// 或 https:// 开头",
      "base_url_should_not_end_with_chat_completions": "Base URL 不应以 /chat/completions 结尾",
      "config_service_not_initialized": "AI 配置服务未初始化",
      "config_valid": "AI 配置有效",
      "failed_to_create_request": "创建请求失败",
      "forbidden_quota": "配额超限",
      "free_tier_exhausted": "AI 模型免费额度已耗尽：请在模型供应商管理后台关闭“use free tier only”或更换付费 Key。",
      "invalid_base_url": "Base URL 无效",
      "invalid_provider": "服务商无效",
      "no_trade_data_available": "暂无可用交易数据",
      "not_configured": "AI 未配置：请先到 AI 设置中启用并配置。",
      "probe_ok": "正常",
      "probe_ok_no_models": "正常（未返回 models）",
      "provider_required": "请先选择服务商",
      "provider_returned_empty_message": "AI 服务返回空消息",
      "rate_limited": "AI 服务触发限流/额度不足（429/资源耗尽）。请稍后重试或更换可用的 API Key/模型配置。",
      "request_failed": "API 请求失败",
      "insufficient_balance_title": "Insufficient 余额",
      "insufficient_balance": "AI钱包余额不足，请充值后继续。"
    },
    "connection_failed": {
      "content": "无法连接到服务器，请检查网络后重试。",
      "title": "连接失败"
    },
    "access_denied": "无权限访问",
    "account_connected": "连接成功",
    "account_connection_failed": "无法连接到交易服务器",
    "account_not_found": "账户不存在",
    "auto_trading_disabled": "自动交易已关闭",
    "auto_trading_enabled": "自动交易已开启",
    "email_already_registered": "邮箱已注册",
    "invalid_credentials": "账号或密码错误",
    "not_authenticated": "未登录",
    "schedule_service_not_available": "调度服务不可用",
    "translate_failed": "翻译失败",
    "user_not_found": "用户不存在"
  },
  "symbolDetection": {
    "tradeMode": {
      "disabled": "已禁用",
      "longOnly": "仅做多",
      "longShort": "多空双向",
      "shortOnly": "仅做空",
      "unknown": "未知"
    },
    "label": "识别到的交易品种",
    "loading": "正在解析…",
    "noSymbols": "未识别到交易品种，请尝试包含具体的品种名称（如\"比特币\"、\"EURUSD\"、\"黄金\"）",
    "resolvedTooltip": "broker: {{broker}} | 模式: {{mode}}",
    "unresolvedTooltip": "尚未绑定交易账户，无法解析"
  },
  "subscription": {
    "feature": {
      "aiTokens": "{{count}} AI Token/月",
      "strategies": "{{count}} 个策略",
      "backtests": "{{count}} 次回测/天",
      "liveStrategies": "{{count}} 个实盘策略",
      "symbols": "{{count}} 个品种/策略"
    },
    "title": "订阅方案",
    "subscribeSuccess": "订阅激活成功！",
    "charged": "已扣费: {{amount}}, 余额: {{balance}}",
    "insufficientBalance": "钱包余额不足，请先充值。",
    "subscribeFailed": "订阅失败，请重试。",
    "cancelSuccess": "自动续订已取消。您的订阅在当前周期结束前仍然有效。",
    "cancelFailed": "取消失败，请重试。",
    "changeSuccess": "方案切换成功！",
    "changeFailed": "方案切换失败，请重试。",
    "billingCycle": "计费",
    "autoRenew": "自动续订",
    "period": "当前周期",
    "cancelAutoRenew": "取消自动续订",
    "usageTitle": "本月使用量",
    "aiTokens": "AI Token",
    "activeStrategies": "活跃策略",
    "runtimeMinutes": "运行时长（分钟）",
    "walletBalance": "钱包余额",
    "month": "月",
    "year": "年",
    "freeForever": "永久免费",
    "currentPlan": "当前方案",
    "choosePlan": "选择方案",
    "noPlans": "暂无可用方案",
    "changePlanTitle": "切换方案",
    "subscribeTitle": "订阅方案",
    "selectBillingCycle": "计费周期",
    "monthly": "月付",
    "yearly": "年付",
    "chargeNotice": "付费方案将从钱包扣款。免费方案不扣费。"
  },
  "agent": {
    "analysis": {
      "title": "回测分析",
      "sharpe": "夏普",
      "drawdown": "最大回撤",
      "winrate": "胜率",
      "consistency": "一致性",
      "risk_adj": "风险调整收益",
      "overfitting": "过拟合风险",
      "observations": "关键观察",
      "suggestions": "改进建议",
      "detailed": "详细分析"
    },
    "semantic_diff": {
      "title": "策略 Changes",
      "effect": "影响"
    },
    "profile": {
      "title": "策略 Profile",
      "timeframe": "时间周期",
      "regime": "市场状态",
      "indicators": "指标",
      "entry": "入场",
      "exit": "出场",
      "risk": "Risk 管理",
      "coverage": "覆盖范围",
      "strengths": "优势",
      "weaknesses": "劣势",
      "blind_spots": "盲点"
    }
  },
  "importAnalysis": {
    "execution": {
      "onBar": "K线收盘驱动",
      "onTick": "逐笔驱动",
      "onInitGrid": "初始化网格"
    },
    "sizing": {
      "fixed": "固定手数",
      "martingale": "马丁格尔",
      "percentBalance": "余额百分比"
    },
    "analyzing": "正在分析策略结构...",
    "tradeLogicComplete": "交易逻辑已全部识别",
    "guiNoiseDesc": "以下盲区属于图表显示/按钮功能，服务端执行时跳过，不影响交易结果。可以安全导入。",
    "cannotImport": "无法自动导入",
    "incompleteCoverage": "交易逻辑覆盖不完整",
    "goodCoverage": "导入覆盖率良好",
    "goodCoverageDesc": "策略 main logic recognized. Safe to import. Check parameter list before use.",
    "coverageTitle": "导入覆盖率",
    "location": "位置",
    "handling": "处理方式",
    "userActionRequired": "需要您操作",
    "noBlindSpots": "无需确认逻辑",
    "noBlindSpotsDesc": "所有策略逻辑已自动识别，可以安全导入。"
  },
  "dashboard": {
    "quickActions": {
      "aiStrategy": "AI 策略"
    }
  },
  "logs": {
    "triggerSource": {
      "manual": "手动",
      "strategy": "策略",
      "recovery": "恢复"
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
    "mtSessionLost": "⚠ MT 会话丢失 — 正在重连…",
    "noSymbolSelected": "选择一个品种以查看行情数据",
    "noSymbolsFound": "未找到品种",
    "popularSymbols": "热门品种",
    "searchPlaceholder": "搜索品种（如 EURUSD, XAUUSD）",
    "searchSymbol": "搜索品种...",
    "selectAccount": "选择交易账户",
    "selectSymbol": "选择品种",
    "spread": "点差",
    "watchlist": "自选"
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
    "strategyLive": "实盘监控",
    "strategyWorkspace": "策略工作台",
    "subscription": "订阅",
    "trading": "交易",
    "wallet": "钱包"
  },
  "profile": {
    "lastLogin": "最后登录",
    "nickname": "昵称",
    "registered": "已注册",
    "role": "角色",
    "status": "状态",
    "title": "个人信息"
  },
  "share": {
    "actions": "操作",
    "createNew": "创建新分享链接",
    "createdAt": "创建时间",
    "deleteConfirm": "删除此分享链接？",
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
    "footer": "由 AlphaForge 生成",
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
    "worstTrade": "最差交易",
    "countUnit": "笔"
  },
  "topbar": {
    "logout": "退出登录",
    "profile": "个人信息",
    "settings": "设置",
    "switchToAdmin": "切换到管理",
    "systemOk": "系统正常运行",
    "user": "普通用户"
  },
  "theme": {
    "switchToDark": "切换到深色模式",
    "switchToLight": "切换到浅色模式"
  },
  "monitoring": {
    "unknown": "未知",
    "healthy": "正常",
    "title": "系统监控",
    "sseConnected": "SSE 已连接",
    "disconnected": "已断开",
    "streamError": "Stream 错误",
    "waitingData": "等待数据...",
    "serviceHealth": "服务健康",
    "uptime": "运行时长",
    "database": "数据库",
    "diskUsage": "磁盘使用",
    "goRuntime": "Go运行时",
    "goroutines": "Goroutines",
    "gcCount": "GC次数",
    "gcPauseAvg": "GC平均暂停",
    "stackUsage": "栈使用",
    "heapMemory": "堆内存",
    "dbPool": "数据库连接池",
    "totalConns": "总计",
    "idle": "空闲",
    "acquired": "已获取",
    "mdGateway": "行情网关",
    "spillFiles": "溢出文件",
    "droppedBars": "丢弃 K 线",
    "droppedSignals": "丢弃信号",
    "consumerLag": "消费者延迟",
    "staleAccounts": "过期账户",
    "deadAccounts": "死账户",
    "avgGapSec": "平均间隔 (秒)",
    "maxGapSec": "最大间隔 (秒)",
    "dlq": "死信队列 (DLQ)",
    "parseErrors": "解析错误",
    "bidGtAsk": "买价>卖价",
    "nonPositive": "非正数",
    "pushInterval": "推送间隔：5秒",
    "lastUpdate": "最后更新"
  }
} as const;
export default Base;
