// Auto-generated from proto/ant/v1/i18n/strategy_templates_zh-tw.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const StrategyTemplates = {
  "strategy": {
    "templates": {
      "scheduleLaunch": {
        "form": {
          "scheduleTypes": {
            "hfQuote": "高頻報價",
            "interval": "定時執行",
            "klineClose": "K 线收盘触发"
          },
          "account": "帳號",
          "accountPlaceholder": "選擇账戶",
          "defaultVolume": "默认手數",
          "defaultVolumeTip": "每個信号的默认下單量",
          "enableAfterCreate": "建立后立即啟用",
          "hfCooldownMs": "高頻冷卻(毫秒)",
          "hfCooldownMsTip": "报价驱动执行间的冷却時間",
          "intervalMs": "間隔(毫秒)",
          "intervalMsTip": "非高頻模式最小1000ms",
          "investorTag": "投資者(唯讀)",
          "maxDrawdownPct": "最大回撤%",
          "maxDrawdownPctTip": "回撤超過此閾值自动停止",
          "maxPositions": "最大持仓數",
          "maxPositionsTip": "同時持有的最大仓位元數量",
          "riskSection": "風控設定",
          "scheduleName": "计划名稱",
          "scheduleNameMax": "最多64字元",
          "scheduleNamePlaceholder": "例如：EURUSD M5 早盤策略",
          "scheduleType": "排程類型",
          "stopLossOffset": "止損偏移",
          "stopLossOffsetTip": "距入場價的止損距離(點)",
          "strategyParamsSection": "策略参數",
          "symbol": "品種",
          "symbolPlaceholder": "選擇商品",
          "symbolPlaceholderEmpty": "未配置商品",
          "takeProfitOffset": "止盈偏移",
          "takeProfitOffsetTip": "距入場價的止盈距離(點)",
          "timeframe": "週期"
        },
        "actions": {
          "addAccount": "新增账戶",
          "create": "建立調度",
          "createAndEnable": "建立並啟用",
          "createScheduleNoEnable": "新建調度任務",
          "publishTemplate": "發布模板",
          "updateTradingPassword": "更新交易密碼"
        },
        "metrics": {
          "annualReturn": "年化收益",
          "maxDrawdown": "最大回撤",
          "sharpe": "夏普比率",
          "totalReturn": "總收益",
          "totalTrades": "交易次數",
          "winRate": "勝率"
        },
        "backtestRunningHint": "回測正在運行，請稍候。",
        "errorInvestorAccount": "无法使用投資者账戶啟動计划。请更新交易密碼以啟用交易。",
        "investorWarningBody": "此账戶为投資者(唯讀)模式，需要交易權限才能啟動计划。",
        "investorWarningTitle": "投資者账戶",
        "keyMetrics": "關鍵指標",
        "launchSection": "上線調度",
        "newPasswordPlaceholder": "输入新的交易密码",
        "noAccountBody": "啟動计划前需要先绑定MT账戶。",
        "noAccountTitle": "无账戶",
        "noRun": "暫無回測運行",
        "score": "評分",
        "title": "上線調度",
        "tradePermissionOk": "交易權限驗證通過",
        "updatePasswordFailed": "更新交易密碼失敗",
        "updatePasswordHint": "輸入此账戶的交易密碼以啟用交易。",
        "updatePasswordOk": "交易密碼已更新",
        "updatePasswordStillInvestor": "密碼更新成功但账戶仍为投資者模式，請聯絡客服。",
        "updatePasswordTitle": "更新交易密碼",
        "verifyingPermission": "驗證交易權限中..."
      },
      "backtest": {
        "fields": {
          "account": "帳號",
          "extraSymbols": "额外品種 (多选)",
          "initialCapital": "初始本金",
          "range": "範圍",
          "symbol": "品種",
          "timeframe": "週期",
          "title": "标题"
        },
        "parameters": {
          "title": "策略参數"
        },
        "placeholders": {
          "account": "選擇帳號",
          "extraSymbols": "可选，适用于配对/輪动策略",
          "range": "選擇日期範圍",
          "symbol": "選擇品種"
        },
        "quickRange": {
          "custom": "自定义"
        },
        "tooltips": {
          "extraSymbols": "额外获取 K 线的品種 (同帳戶、同周期)。策略可通过 context[\"closes_by_symbol\"] 访问。"
        },
        "validation": {
          "accountRequired": "請選擇帳號",
          "initialCapitalRequired": "請輸入初始本金",
          "rangeRequired": "請選擇日期範圍",
          "symbolRequired": "請輸入品種",
          "timeframeRequired": "請選擇週期"
        },
        "accountDisabledSuffix": "（已禁用）",
        "modalTitleWithName": "回測: {{name}}",
        "title": "回測"
      },
      "backtestRuns": {
        "actions": {
          "createSchedule": "新建調度任務",
          "launchSchedule": "查看評分",
          "view": "查看"
        },
        "status": {
          "canceled": "已取消",
          "canceling": "取消中",
          "completed": "已完成",
          "failed": "失敗",
          "queued": "佇列中",
          "running": "運行中"
        },
        "table": {
          "actions": "操作",
          "createdAt": "建立時間",
          "status": "狀態",
          "symbol": "品種",
          "timeframe": "週期",
          "title": "标题"
        },
        "batchDelete": "删除 {{count}} 条",
        "batchDeleteConfirm": "删除 {{count}} 条回測报告？",
        "batchDeleteSuccess": "已刪除 {{count}} 条回測报告",
        "deleteConfirm": "删除此记录？",
        "empty": "暫無回測记录",
        "title": "回測记录"
      },
      "codeModal": {
        "actions": {
          "copy": "複製"
        },
        "title": "策略代碼"
      },
      "editTemplateModal": {
        "actions": {
          "validateCode": "驗證程式碼"
        },
        "fields": {
          "code": "策略代碼",
          "description": "描述",
          "name": "名稱",
          "publicShare": "公開"
        },
        "placeholders": {
          "codeSample": "輸入Python策略代碼...",
          "description": "可選：策略說明",
          "name": "例如：均線交叉策略"
        },
        "title": {
          "create": "新建範本",
          "edit": "編輯模板"
        },
        "validation": {
          "codeRequired": "程式碼不能为空",
          "nameRequired": "請輸入名稱"
        }
      },
      "actions": {
        "backtest": "回測",
        "copy": "複製",
        "create": "新建模板",
        "createTemplate": "新建範本",
        "delete": "刪除",
        "edit": "編輯",
        "launchSchedule": "上線調度",
        "viewCode": "查看代碼"
      },
      "badges": {
        "preset": "預設"
      },
      "messages": {
        "backtestCancelFailed": "取消回測失敗",
        "backtestCancelRequested": "已请求取消回測",
        "backtestRangeInvalid": "回測日期範圍无效",
        "backtestReportDeleted": "回測报告已刪除",
        "backtestReportNotFound": "未找到回測报告",
        "backtestRunNoPublishedTemplate": "回測執行没有已發布範本",
        "backtestRunningCannotPublish": "回測正在執行，无法发布。",
        "backtestSubmitFailed": "提交回測失敗",
        "backtestSubmitted": "回測已提交",
        "cannotPublishAndCreateDraftFailed": "无法发布，草稿创建失敗。",
        "codeCopied": "程式碼已複製",
        "codeValidationFailed": "程式碼驗證失敗",
        "codeValidationNotPassed": "程式碼驗證未通过",
        "codeValidationPassed": "程式碼驗證通过",
        "copyFailed": "複製失敗，請手動複製",
        "createScheduleFailed": "创建排程失敗",
        "deepLinkNavigate": "已从外部链接打开範本及最新執行详情",
        "enterStrategyCode": "請輸入策略代碼",
        "fetchTemplateListFailed": "載入範本列表失敗",
        "missingDraftIdCannotPublish": "缺少草稿 ID，无法发布。",
        "missingScheduleInfo": "缺少排程信息",
        "publishFailed": "发布失敗",
        "publishedButNoTemplateId": "已發布，但缺少範本 ID。",
        "readStrategyCodeFailed": "读取策略程式碼失敗",
        "readTemplateStatusFailed": "读取範本狀態失敗",
        "republishedButNoTemplateId": "已重新发布，但缺少範本 ID。",
        "scheduleCreated": "排程已建立",
        "scheduleCreatedAndEnabled": "排程已建立并启用",
        "selectBacktestRange": "請選擇回測日期範圍",
        "strategyCodeEmptyCannotBacktest": "策略程式碼为空，无法回測。",
        "strategyCodeEmptyCannotPublish": "策略程式碼为空，请先儲存程式碼再发布。",
        "systemTemplateReadOnly": "系统範本为只读，请克隆后编辑。",
        "templateAlreadyPublished": "範本已發布",
        "templateCreated": "範本已建立",
        "templateDeleted": "範本已刪除",
        "templateNotDraftUnknownPublishStatus": "範本非草稿，发布狀態未知。",
        "templateNotPublishedCannotCreateSchedule": "範本未发布，无法创建排程。",
        "templatePublished": "範本已發布",
        "templateRepublished": "範本已重新发布",
        "templateUpdated": "範本已更新"
      },
      "status": {
        "draft": "草稿",
        "published": "已發布"
      },
      "table": {
        "actions": "操作",
        "createdAt": "建立時間",
        "defaultHint": "預設值",
        "description": "描述",
        "emptyUser": "暫無用户範本，點選上方“新建範本”开始。",
        "loadingDefault": "正在載入預設模板...",
        "name": "名稱",
        "status": "狀態",
        "tags": "標籤",
        "updatedAt": "更新時間",
        "useCount": "使用次數",
        "visibility": "可見性"
      },
      "tabs": {
        "system": "系統模板",
        "user": "用户範本"
      },
      "visibility": {
        "private": "私有",
        "public": "公開"
      },
      "gallery": {
        "title": "策略",
        "aiGenerate": "AI 生成",
        "searchPlaceholder": "搜尋策略...",
        "filterAll": "全部",
        "filterMine": "我的",
        "filterSystem": "系統",
        "sortRecent": "最新",
        "sortReturn": "收益",
        "sortRisk": "風險",
        "sortUsage": "使用量",
        "empty": "未找到策略",
        "system": "系統",
        "shared": "共享",
        "deploy": "部署",
        "fork": "分支",
        "publish": "發布",
        "unpublish": "取消發布",
        "unpublishSuccess": "已取消發布",
        "unpublishFailed": "取消發布失敗",
        "deleteFailed": "刪除失敗"
      },
      "detail": {
        "notFound": "未找到策略",
        "overview": "概覽",
        "noDescription": "暫無描述",
        "equityCurve": "權益曲線",
        "tradeStats": "交易統計",
        "profitFactor": "盈利因子",
        "parameters": "參數"
      },
      "copySuffix": " (副本)",
      "defaultDraftName": "草稿範本",
      "deleteConfirm": "删除此範本？",
      "scheduleName": "{{symbol}} {{timeframe}} {{name}}",
      "title": "策略模板"
    }
  }
} as const;
export default StrategyTemplates;
