// Auto-generated from proto/ant/v1/i18n/strategy_templates_zh-cn.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const StrategyTemplates = {
  "strategy": {
    "templates": {
      "scheduleLaunch": {
        "form": {
          "scheduleTypes": {
            "hfQuote": "高频报价",
            "interval": "定时执行",
            "klineClose": "K 线收盘触发"
          },
          "account": "账号",
          "accountPlaceholder": "选择账户",
          "defaultVolume": "默认手数",
          "defaultVolumeTip": "每个信号的默认下单量",
          "enableAfterCreate": "创建后立即启用",
          "hfCooldownMs": "高频冷却(毫秒)",
          "hfCooldownMsTip": "报价驱动执行间的冷却时间",
          "intervalMs": "间隔(毫秒)",
          "intervalMsTip": "非高频模式最小1000ms",
          "investorTag": "投资者(只读)",
          "maxDrawdownPct": "最大回撤%",
          "maxDrawdownPctTip": "回撤超过此阈值自动停止",
          "maxPositions": "最大持仓数",
          "maxPositionsTip": "同时持有的最大仓位数量",
          "riskSection": "风控设置",
          "scheduleName": "计划名称",
          "scheduleNameMax": "最多64字符",
          "scheduleNamePlaceholder": "例如：EURUSD M5 早盘策略",
          "scheduleType": "计划类型",
          "stopLossOffset": "止损偏移",
          "stopLossOffsetTip": "距入场价的止损距离(点)",
          "strategyParamsSection": "策略参数",
          "symbol": "品种",
          "symbolPlaceholder": "选择品种",
          "symbolPlaceholderEmpty": "未配置品种",
          "takeProfitOffset": "止盈偏移",
          "takeProfitOffsetTip": "距入场价的止盈距离(点)",
          "timeframe": "周期"
        },
        "actions": {
          "addAccount": "添加账户",
          "create": "创建调度",
          "createAndEnable": "创建并启用",
          "createScheduleNoEnable": "新建调度任务",
          "publishTemplate": "发布模板",
          "updateTradingPassword": "更新交易密码"
        },
        "metrics": {
          "annualReturn": "年化收益",
          "maxDrawdown": "最大回撤",
          "sharpe": "夏普比率",
          "totalReturn": "总收益",
          "totalTrades": "交易次数",
          "winRate": "胜率"
        },
        "backtestRunningHint": "回测正在运行，请稍候。",
        "errorInvestorAccount": "无法使用投资者账户启动计划。请更新交易密码以启用交易。",
        "investorWarningBody": "此账户为投资者(只读)模式，需要交易权限才能启动计划。",
        "investorWarningTitle": "投资者账户",
        "keyMetrics": "关键指标",
        "launchSection": "上线调度",
        "newPasswordPlaceholder": "输入新的交易密码",
        "noAccountBody": "启动计划前需要先绑定MT账户。",
        "noAccountTitle": "无账户",
        "noRun": "暂无回测运行",
        "score": "评分",
        "title": "上线调度",
        "tradePermissionOk": "交易权限验证通过",
        "updatePasswordFailed": "更新交易密码失败",
        "updatePasswordHint": "输入此账户的交易密码以启用交易。",
        "updatePasswordOk": "交易密码已更新",
        "updatePasswordStillInvestor": "密码更新成功但账户仍为投资者模式，请联系客服。",
        "updatePasswordTitle": "更新交易密码",
        "verifyingPermission": "验证交易权限中..."
      },
      "backtest": {
        "fields": {
          "account": "账号",
          "extraSymbols": "额外品种 (多选)",
          "initialCapital": "初始本金",
          "range": "范围",
          "symbol": "品种",
          "timeframe": "周期",
          "title": "标题"
        },
        "parameters": {
          "title": "策略参数"
        },
        "placeholders": {
          "account": "选择账号",
          "extraSymbols": "可选，适用于配对/轮动策略",
          "range": "选择日期范围",
          "symbol": "选择品种"
        },
        "quickRange": {
          "custom": "自定义"
        },
        "tooltips": {
          "extraSymbols": "额外获取 K 线的品种 (同账户、同周期)。策略可通过 context[\"closes_by_symbol\"] 访问。"
        },
        "validation": {
          "accountRequired": "请选择账号",
          "initialCapitalRequired": "请输入初始本金",
          "rangeRequired": "请选择日期范围",
          "symbolRequired": "请选择品种",
          "timeframeRequired": "请选择周期"
        },
        "accountDisabledSuffix": "（已禁用）",
        "modalTitleWithName": "回测: {{name}}",
        "title": "回测"
      },
      "backtestRuns": {
        "actions": {
          "createSchedule": "新建调度任务",
          "launchSchedule": "查看评分",
          "view": "查看"
        },
        "status": {
          "canceled": "已取消",
          "canceling": "取消中",
          "completed": "已完成",
          "failed": "失败",
          "queued": "排队中",
          "running": "运行中"
        },
        "table": {
          "actions": "操作",
          "createdAt": "创建时间",
          "status": "状态",
          "symbol": "品种",
          "timeframe": "周期",
          "title": "标题"
        },
        "batchDelete": "删除 {{count}} 条",
        "batchDeleteConfirm": "删除 {{count}} 条回测报告？",
        "batchDeleteSuccess": "已删除 {{count}} 条回测报告",
        "deleteConfirm": "删除此记录？",
        "empty": "暂无回测记录",
        "title": "回测记录"
      },
      "codeModal": {
        "actions": {
          "copy": "复制"
        },
        "title": "策略代码"
      },
      "editTemplateModal": {
        "actions": {
          "validateCode": "验证代码"
        },
        "fields": {
          "code": "策略代码",
          "description": "描述",
          "name": "名称",
          "publicShare": "公开"
        },
        "placeholders": {
          "codeSample": "输入Python策略代码...",
          "description": "可选：策略说明",
          "name": "例如：均线交叉策略"
        },
        "title": {
          "create": "新建模板",
          "edit": "编辑模板"
        },
        "validation": {
          "codeRequired": "代码不能为空",
          "nameRequired": "请输入名称"
        }
      },
      "actions": {
        "backtest": "回测",
        "copy": "复制",
        "create": "新建模板",
        "createTemplate": "新建模板",
        "delete": "删除",
        "edit": "编辑",
        "launchSchedule": "上线调度",
        "viewCode": "查看代码"
      },
      "badges": {
        "preset": "预设"
      },
      "messages": {
        "backtestCancelFailed": "取消回测失败",
        "backtestCancelRequested": "已请求取消回测",
        "backtestRangeInvalid": "回测日期范围无效",
        "backtestReportDeleted": "回测报告已删除",
        "backtestReportNotFound": "未找到回测报告",
        "backtestRunNoPublishedTemplate": "回测运行没有已发布模板",
        "backtestRunningCannotPublish": "回测正在运行，无法发布。",
        "backtestSubmitFailed": "提交回测失败",
        "backtestSubmitted": "回测已提交",
        "cannotPublishAndCreateDraftFailed": "无法发布，草稿创建失败。",
        "codeCopied": "代码已复制",
        "codeValidationFailed": "代码验证失败",
        "codeValidationNotPassed": "代码验证未通过",
        "codeValidationPassed": "代码验证通过",
        "copyFailed": "复制失败，请手动复制",
        "createScheduleFailed": "创建调度失败",
        "deepLinkNavigate": "已从外部链接打开模板及最新运行详情",
        "enterStrategyCode": "请输入策略代码",
        "fetchTemplateListFailed": "加载模板列表失败",
        "missingDraftIdCannotPublish": "缺少草稿 ID，无法发布。",
        "missingScheduleInfo": "缺少调度信息",
        "publishFailed": "发布失败",
        "publishedButNoTemplateId": "已发布，但缺少模板 ID。",
        "readStrategyCodeFailed": "读取策略代码失败",
        "readTemplateStatusFailed": "读取模板状态失败",
        "republishedButNoTemplateId": "已重新发布，但缺少模板 ID。",
        "scheduleCreated": "调度已创建",
        "scheduleCreatedAndEnabled": "调度已创建并启用",
        "selectBacktestRange": "请选择回测日期范围",
        "strategyCodeEmptyCannotBacktest": "策略代码为空，无法回测。",
        "strategyCodeEmptyCannotPublish": "策略代码为空，请先保存代码再发布。",
        "systemTemplateReadOnly": "系统模板为只读，请克隆后编辑。",
        "templateAlreadyPublished": "模板已发布",
        "templateCreated": "模板已创建",
        "templateDeleted": "模板已删除",
        "templateNotDraftUnknownPublishStatus": "模板非草稿，发布状态未知。",
        "templateNotPublishedCannotCreateSchedule": "模板未发布，无法创建调度。",
        "templatePublished": "模板已发布",
        "templateRepublished": "模板已重新发布",
        "templateUpdated": "模板已更新"
      },
      "status": {
        "draft": "草稿",
        "published": "已发布"
      },
      "table": {
        "actions": "操作",
        "createdAt": "创建时间",
        "defaultHint": "默认值",
        "description": "描述",
        "emptyUser": "暂无用户模板，点击上方“新建模板”开始。",
        "loadingDefault": "正在加载默认模板...",
        "name": "名称",
        "status": "状态",
        "tags": "标签",
        "updatedAt": "更新时间",
        "useCount": "使用次数",
        "visibility": "可见性"
      },
      "tabs": {
        "system": "系统模板",
        "user": "用户模板"
      },
      "visibility": {
        "private": "私有",
        "public": "公开"
      },
      "copySuffix": " (副本)",
      "defaultDraftName": "草稿模板",
      "deleteConfirm": "删除此模板？",
      "scheduleName": "{{symbol}} {{timeframe}} {{name}}",
      "title": "策略模板"
    }
  }
} as const;
export default StrategyTemplates;
