// Auto-generated supplementary keys for admin
const AdminExtra = {
  "admin": {
    "aiGateway": {
      "errors": {
        "loadProviders": "加载供应商失败",
        "toggleFailed": "切换失败",
        "loadModels": "加载模型失败"
      },
      "addProviderPending": "添加供应商功能待后端支持",
      "title": "AI 网关管理",
      "description": "管理 AI 供应商、模型和定价。用户从可用模型中选择，按 token 从钱包计费。",
      "addProvider": "添加供应商",
      "columns": {
        "baseUrl": "基础 URL",
        "apiKey": "API 密钥"
      },
      "configured": "未配置",
      "editProvider": "添加供应商",
      "providerId": "请输入供应商 ID",
      "providerIdPlaceholder": "deepseek / openai / qwen ...",
      "displayName": "显示名称",
      "displayNamePlaceholder": "DeepSeek",
      "baseUrl": "请输入基础 URL",
      "apiKeyLabel": "API 密钥，加密存储",
      "apiKeyEditPlaceholder": "留空则保持不变",
      "editModel": "添加模型",
      "modelName": "模型名称",
      "priceInput": "输入价格（$/1M）",
      "priceOutput": "输出价格（$/1M）",
      "addModel": "添加模型",
      "confirmDeleteModel": "删除此模型？",
      "noModels": "暂无模型"
    },
    "account": {
      "errors": {
        "loadFailed": "加载账户失败",
        "freezeFailed": "冻结失败",
        "unfreezeFailed": "解冻失败"
      },
      "frozen": "账户已冻结",
      "unfrozen": "账户已解冻",
      "columns": {
        "createdAt": "创建时间"
      },
      "confirmFreeze": "冻结此账户？",
      "title": "账户管理",
      "searchPlaceholder": "搜索账户",
      "detail": "账户详情",
      "auditLogs": "审计日志"
    },
    "settings": {
      "saveSuccess": "保存成功",
      "saveFailed": "保存失败",
      "deleteFailed": "删除失败",
      "actionFailed": "操作失败",
      "columns": {
        "key": "设置键"
      },
      "confirmDelete": "确认删除？",
      "title": "Agent 管理设置",
      "addSetting": "添加设置",
      "permissionRules": "权限规则 (permission.rule.N)",
      "permissionFormat": "格式：",
      "permissionExample": "示例：",
      "permissionAddRule": "添加规则：创建设置键",
      "addManagedSetting": "添加托管设置",
      "settingKey": "设置键",
      "keyPlaceholder": "例如：allowed_models, disable_live_trading, permission.rule.1",
      "valuePlaceholder": "例如：claude-sonnet-5,deepseek-v4"
    },
    "autogen": {
      "approved": "任务已批准并发布",
      "rejected": "任务已拒绝",
      "enqueued": "{{count}} 个任务已入队",
      "confirmApprove": "批准并发布？",
      "confirmReject": "拒绝此任务？",
      "title": "AI 策略生成任务",
      "allStatus": "全部状态",
      "triggerBatch": "触发批量生成",
      "symbols": "品种（逗号分隔）",
      "timeframes": "时间周期（逗号分隔）",
      "strategyTypes": "策略类型（逗号分隔）"
    },
    "billing": {
      "columns": {
        "autoRenew": "自动续费",
        "periodStart": "周期开始",
        "periodEnd": "周期结束",
        "createdAt": "创建时间",
        "balanceBefore": "交易前余额",
        "balanceAfter": "交易后余额"
      },
      "title": "账单管理",
      "monthlyRevenue": "月收入",
      "totalRevenue": "总收入",
      "activeSubs": "活跃订阅",
      "planRevenue": "套餐收入明细",
      "filterByPlan": "按套餐筛选",
      "filterByStatus": "按状态筛选",
      "walletTransactions": "钱包交易",
      "filterByType": "按类型筛选",
      "txPlatformFee": "平台手续费"
    },
    "coupon": {
      "loadFailed": "加载优惠券失败",
      "fillRequired": "请填写必填字段",
      "created": "优惠券已创建",
      "createFailed": "创建优惠券失败",
      "disabled": "优惠券已禁用",
      "disableFailed": "禁用优惠券失败",
      "colMinPurchase": "最低消费",
      "create": "创建优惠券",
      "createTitle": "创建优惠券",
      "codePlaceholder": "优惠券码（如 SUMMER20）",
      "valuePlaceholder": "折扣值（如 20 表示 20% 或 50 表示 $50）",
      "minPurchasePlaceholder": "最低消费金额（0 = 无限制）",
      "maxUsesPlaceholder": "最大使用次数（0 = 无限）",
      "expiresPlaceholder": "过期时间（ISO 8601，空 = 永不过期）"
    },
    "dashboard": {
      "errors": {
        "loadFailed": "加载仪表盘数据失败"
      },
      "title": "管理仪表盘",
      "totalUsers": "总用户数",
      "activeUsers": "活跃用户",
      "verifiedUsers": "已验证用户",
      "mtAccounts": "MT 账户",
      "onlineAccounts": "在线账户",
      "todayTrades": "今日交易",
      "todayProfit": "今日盈亏",
      "activeSubs": "活跃订阅",
      "monthlyRevenue": "月收入",
      "totalRevenue": "总收入",
      "marketStrategies": "市场策略",
      "marketSales": "市场销售",
      "marketRevenue": "市场收入",
      "recentLogs": "最近日志"
    },
    "logs": {
      "modules": {
        "userManagement": "用户管理",
        "accountManagement": "账户管理",
        "systemConfig": "系统配置"
      },
      "columns": {
        "actionType": "操作类型",
        "ip": "IP 地址"
      },
      "errors": {
        "loadFailed": "加载日志失败"
      },
      "title": "操作日志",
      "filterModule": "按模块筛选",
      "filterAction": "按操作筛选"
    },
    "depositAddresses": {
      "importFailed": "导入失败",
      "user": "用户 ID",
      "received": "已收到 USDT",
      "assignedAt": "分配时间",
      "importHint": "在离线机器上使用 hdgen 工具生成 deposit_addresses.bin，然后在此上传。",
      "all": "全部状态",
      "import": "导入地址",
      "availablePool": "池中可用",
      "total": "总地址数"
    },
    "deposit": {
      "table": {
        "user": "用户 ID",
        "amount": "USDT 金额",
        "txHash": "交易哈希"
      },
      "title": "充值管理"
    },
    "analytics": {
      "platformRev": "平台收入",
      "providerRev": "供应商收入",
      "activeBuyers": "活跃买家",
      "refundRate": "退款率",
      "newSubs": "新订阅者",
      "totalStrategies": "总策略数",
      "newStrategies": "新增策略",
      "topByRevenue": "收入最高策略",
      "topBySubs": "订阅最多策略",
      "topProvidersRev": "收入最高供应商",
      "topProvidersStrat": "策略最多供应商"
    },
    "marketplace": {
      "loadFailed": "加载策略失败",
      "featureSuccess": "策略已设为推荐",
      "featureFailed": "设置推荐失败",
      "unfeatureSuccess": "已取消推荐",
      "unfeatureFailed": "取消推荐失败",
      "unfeature": "取消推荐",
      "filterStatus": "全部状态",
      "searchPlaceholder": "按标题搜索...",
      "featureTitle": "推荐策略",
      "featureDesc": "设置推荐展示优先级。数值越高越显著。"
    },
    "refund": {
      "loadFailed": "加载退款请求失败",
      "approved": "退款已批准并执行",
      "rejected": "退款请求已拒绝",
      "processFailed": "处理退款失败",
      "approve": "批准并执行",
      "filterStatus": "全部状态",
      "approveTitle": "批准退款",
      "rejectTitle": "拒绝退款",
      "reviewNotePlaceholder": "审核备注（拒绝时可选，批准时建议填写）..."
    },
    "sidebar": {
      "shareManagement": "分享分析"
    },
    "walletCalculator": {
      "title": "Token ↔ USD 计算器",
      "selectModel": "选择模型（计价基准）",
      "usdAmount": "USD 金额",
      "fillResult": "填充结果"
    },
    "wallet": {
      "errors": {
        "noUserSelected": "未选择用户"
      },
      "messages": {
        "adjustSuccess": "余额调整成功",
        "adjustFailed": "调整失败"
      },
      "columns": {
        "walletNumber": "钱包编号",
        "balanceAfter": "交易后余额"
      },
      "title": "钱包管理",
      "tabWallets": "用户钱包",
      "userList": "用户列表",
      "searchPlaceholder": "搜索钱包/邮箱/昵称",
      "noMatch": "无用户",
      "walletDetail": "钱包详情",
      "adjustBalance": "调整余额",
      "tabDepositAddresses": "充值地址"
    },
    "config": {
      "apiKey": "API 密钥"
    },
    "userManagement": {
      "form": {
        "accountNumber": "账户编号",
        "accountNumberInvalid": "5-6 位数字，不以 0 开头，不含 4 或 7"
      },
      "messages": {
        "loadUsersFailed": "加载用户失败"
      }
    }
  }
} as const;
export default AdminExtra;
