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
            "klineClose": "K 線收盤觸發"
          },
          "account": "帳號",
          "accountPlaceholder": "選擇賬戶",
          "defaultVolume": "預設手數",
          "defaultVolumeTip": "每個訊號的預設下單量",
          "enableAfterCreate": "建立後立即啟用",
          "hfCooldownMs": "高頻冷卻(毫秒)",
          "hfCooldownMsTip": "報價驅動執行間的冷卻時間",
          "intervalMs": "間隔(毫秒)",
          "intervalMsTip": "非高頻模式最小1000ms",
          "investorTag": "投資者(唯讀)",
          "maxDrawdownPct": "最大回撤%",
          "maxDrawdownPctTip": "回撤超過此閾值自動停止",
          "maxPositions": "最大持倉數",
          "maxPositionsTip": "同時持有的最大倉位元數量",
          "riskSection": "風控設定",
          "scheduleName": "計劃名稱",
          "scheduleNameMax": "最多64字元",
          "scheduleNamePlaceholder": "例如：EURUSD M5 早盤策略",
          "scheduleType": "排程型別",
          "stopLossOffset": "止損偏移",
          "stopLossOffsetTip": "距入場價的止損距離(點)",
          "strategyParamsSection": "策略引數",
          "symbol": "品種",
          "symbolPlaceholder": "選擇商品",
          "symbolPlaceholderEmpty": "未配置商品",
          "takeProfitOffset": "止盈偏移",
          "takeProfitOffsetTip": "距入場價的止盈距離(點)",
          "timeframe": "週期"
        },
        "actions": {
          "addAccount": "新增賬戶",
          "create": "建立排程",
          "createAndEnable": "建立並啟用",
          "createScheduleNoEnable": "新建排程任務",
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
        "backtestRunningHint": "回測正在執行，請稍候。",
        "errorInvestorAccount": "無法使用投資者賬戶啟動計劃。請更新交易密碼以啟用交易。",
        "investorWarningBody": "此賬戶為投資者(唯讀)模式，需要交易許可權才能啟動計劃。",
        "investorWarningTitle": "投資者賬戶",
        "keyMetrics": "關鍵指標",
        "launchSection": "上線排程",
        "newPasswordPlaceholder": "輸入新的交易密碼",
        "noAccountBody": "啟動計劃前需要先繫結MT賬戶。",
        "noAccountTitle": "無賬戶",
        "noRun": "暫無回測執行",
        "score": "評分",
        "title": "上線排程",
        "tradePermissionOk": "交易許可權驗證透過",
        "updatePasswordFailed": "更新交易密碼失敗",
        "updatePasswordHint": "輸入此賬戶的交易密碼以啟用交易。",
        "updatePasswordOk": "交易密碼已更新",
        "updatePasswordStillInvestor": "密碼更新成功但賬戶仍為投資者模式，請聯絡客服。",
        "updatePasswordTitle": "更新交易密碼",
        "verifyingPermission": "驗證交易許可權中..."
      },
      "backtest": {
        "fields": {
          "account": "帳號",
          "extraSymbols": "額外品種 (多選)",
          "initialCapital": "初始本金",
          "range": "範圍",
          "symbol": "品種",
          "timeframe": "週期",
          "title": "標題"
        },
        "parameters": {
          "title": "策略引數"
        },
        "placeholders": {
          "account": "選擇帳號",
          "extraSymbols": "可選，適用於配對/輪動策略",
          "range": "選擇日期範圍",
          "symbol": "選擇品種"
        },
        "quickRange": {
          "custom": "自定義"
        },
        "tooltips": {
          "extraSymbols": "額外獲取 K 線的品種 (同帳戶、同週期)。策略可透過 context[\"closes_by_symbol\"] 訪問。"
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
          "createSchedule": "新建排程任務",
          "launchSchedule": "檢視評分",
          "view": "檢視"
        },
        "status": {
          "canceled": "已取消",
          "canceling": "取消中",
          "completed": "已完成",
          "failed": "失敗",
          "queued": "佇列中",
          "running": "執行中"
        },
        "table": {
          "actions": "操作",
          "createdAt": "建立時間",
          "status": "狀態",
          "symbol": "品種",
          "timeframe": "週期",
          "title": "標題"
        },
        "batchDelete": "刪除 {{count}} 條",
        "batchDeleteConfirm": "刪除 {{count}} 條回測報告？",
        "batchDeleteSuccess": "已刪除 {{count}} 條回測報告",
        "deleteConfirm": "刪除此記錄？",
        "empty": "暫無回測記錄",
        "title": "回測記錄"
      },
      "codeModal": {
        "actions": {
          "copy": "複製"
        },
        "title": "策略程式碼"
      },
      "editTemplateModal": {
        "actions": {
          "validateCode": "驗證程式碼"
        },
        "fields": {
          "code": "策略程式碼",
          "description": "描述",
          "name": "名稱",
          "publicShare": "公開"
        },
        "placeholders": {
          "codeSample": "輸入Python策略程式碼...",
          "description": "可選：策略說明",
          "name": "例如：均線交叉策略"
        },
        "title": {
          "create": "新建範本",
          "edit": "編輯模板"
        },
        "validation": {
          "codeRequired": "程式碼不能為空",
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
        "launchSchedule": "上線排程",
        "viewCode": "檢視程式碼"
      },
      "badges": {
        "preset": "預設"
      },
      "messages": {
        "backtestCancelFailed": "取消回測失敗",
        "backtestCancelRequested": "已請求取消回測",
        "backtestRangeInvalid": "回測日期範圍無效",
        "backtestReportDeleted": "回測報告已刪除",
        "backtestReportNotFound": "未找到回測報告",
        "backtestRunNoPublishedTemplate": "回測執行沒有已發布範本",
        "backtestRunningCannotPublish": "回測正在執行，無法釋出。",
        "backtestSubmitFailed": "提交回測失敗",
        "backtestSubmitted": "回測已提交",
        "cannotPublishAndCreateDraftFailed": "無法釋出，草稿建立失敗。",
        "codeCopied": "程式碼已複製",
        "codeValidationFailed": "程式碼驗證失敗",
        "codeValidationNotPassed": "程式碼驗證未透過",
        "codeValidationPassed": "程式碼驗證透過",
        "copyFailed": "複製失敗，請手動複製",
        "createScheduleFailed": "建立排程失敗",
        "deepLinkNavigate": "已從外部連結開啟範本及最新執行詳情",
        "enterStrategyCode": "請輸入策略程式碼",
        "fetchTemplateListFailed": "載入範本列表失敗",
        "missingDraftIdCannotPublish": "缺少草稿 ID，無法釋出。",
        "missingScheduleInfo": "缺少排程資訊",
        "publishFailed": "釋出失敗",
        "publishedButNoTemplateId": "已發布，但缺少範本 ID。",
        "readStrategyCodeFailed": "讀取策略程式碼失敗",
        "readTemplateStatusFailed": "讀取範本狀態失敗",
        "republishedButNoTemplateId": "已重新發布，但缺少範本 ID。",
        "scheduleCreated": "排程已建立",
        "scheduleCreatedAndEnabled": "排程已建立並啟用",
        "selectBacktestRange": "請選擇回測日期範圍",
        "strategyCodeEmptyCannotBacktest": "策略程式碼為空，無法回測。",
        "strategyCodeEmptyCannotPublish": "策略程式碼為空，請先儲存程式碼再發布。",
        "systemTemplateReadOnly": "系統範本為只讀，請克隆後編輯。",
        "templateAlreadyPublished": "範本已發布",
        "templateCreated": "範本已建立",
        "templateDeleted": "範本已刪除",
        "templateNotDraftUnknownPublishStatus": "範本非草稿，釋出狀態未知。",
        "templateNotPublishedCannotCreateSchedule": "範本未釋出，無法建立排程。",
        "templatePublished": "範本已發布",
        "templateRepublished": "範本已重新發布",
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
        "emptyUser": "暫無使用者範本，點選上方“新建範本”開始。",
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
        "user": "使用者範本"
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
        "parameters": "引數"
      },
      "copySuffix": " (副本)",
      "defaultDraftName": "草稿範本",
      "deleteConfirm": "刪除此範本？",
      "scheduleName": "{{symbol}} {{timeframe}} {{name}}",
      "title": "策略模板"
    }
  }
} as const;
export default StrategyTemplates;
