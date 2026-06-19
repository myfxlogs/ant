// Auto-generated from proto/ant/v1/i18n/strategy_schedules_zh-tw.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const Schedules = {
  "strategy": {
    "schedules": {
      "actions": {
        "create": "新建調度",
        "healthCheck": "健康檢查",
        "logs": "執行日誌",
        "runNow": "立即執行"
      },
      "createSchedule": "建立調度",
      "deleteConfirm": {
        "title": "删除此排程？"
      },
      "editModal": {
        "advanced": {
          "fixedIntervalSeconds": "固定間隔(秒)",
          "fixedIntervalSecondsExtra": "可選。填寫後將按固定間隔執行（不會再自動跟隨週期）。例如：60 表示每 60 秒執行一次",
          "hfCooldownMs": "高頻模式：最小觸發間隔(ms)",
          "hfCooldownMsExtra": "用於去抖：兩次評估/下單之間的最小間隔",
          "parametersJson": "參數(JSON對象)",
          "parametersJsonExtra": "策略的 JSON 參數",
          "stableOverrideIntervalSeconds": "穩定模式高級：間隔(秒)",
          "stableOverrideIntervalSecondsExtra": "可選。預設綁定週期(timeframe)。填寫後將覆蓋穩定模式的觸發間隔",
          "timeframe": "週期",
          "timeframeExtra": "預設即可。僅用於K線與指標計算，不影響EA本質（策略驅動交易）",
          "title": "高級設定",
          "triggerMode": "觸發模式",
          "triggerModeExtra": "穩定：按K線/週期觸發（信號更穩但有延遲）；高頻：報價流觸發（更快但噪聲大，需要去抖）",
          "triggerModeOptions": {
            "hf": "高频信号流",
            "stable": "穩定（K線/週期）"
          }
        },
        "autoName": {
          "strategy": "策略"
        },
        "fields": {
          "account": "帳號",
          "cronExpression": "Cron 表達式",
          "cronExtra": "標準 5 段：分鐘 小時 日 月 週。例如：*/5 * * * * 每5分鐘；0 9 * * 1-5 工作日9點",
          "enableExtra": "创建后启用排程",
          "intervalSeconds": "間隔(秒)",
          "intervalSecondsExtra": "自動跟隨週期(timeframe)，無需修改",
          "lot": "手數(Lot)",
          "lotExtra": "下單手數，建議從 0.01 開始",
          "name": "名稱",
          "runFrequency": "運行頻率",
          "symbol": "品種",
          "template": "模板",
          "templateExtra": "來自「策略管理」中保存的模板"
        },
        "placeholders": {
          "name": "例如：EURUSD M5 早盤策略",
          "selectAccountFirst": "先選帳號",
          "symbol": "選擇品種"
        },
        "runFrequencyExtra": {
          "byTimeframe": "按時間周期執行",
          "cron": "高級：使用 Cron 精確控制執行時間"
        },
        "runFrequencyOptions": {
          "byTimeframe": "按週期觸發（推薦）",
          "cron": "Cron 表达式"
        },
        "title": {
          "create": "新建調度任務",
          "edit": "編輯調度任務"
        },
        "validation": {
          "accountRequired": "請選擇帳號",
          "cronRequired": "請輸入 cron",
          "lotRequired": "請輸入手數",
          "nameRequired": "請輸入名稱",
          "runFrequencyRequired": "請選擇運行頻率",
          "symbolRequired": "請輸入品種",
          "templateRequired": "請選擇模板",
          "timeframeRequired": "請選擇週期",
          "triggerModeRequired": "請選擇触发模式"
        }
      },
      "enableCount": "啟用次數",
      "format": {
        "cron": "定時: {{expr}}",
        "interval": "每 {{s}}秒"
      },
      "health": {
        "fields": {
          "configKey": "設定鍵",
          "failedRuns": "執行失敗次數",
          "grade": "健康級別",
          "lastRunAt": "最後運行時間",
          "latestError": "最近錯誤",
          "latestProfit": "最近成交盈虧",
          "latestTicket": "最近成交 Ticket",
          "rule": "判定依據",
          "successOverTotal": "執行成功/總次數",
          "thresholds": "當前門檻"
        },
        "grade": {
          "alert": "警报",
          "healthy": "健康",
          "noSample": "無樣本",
          "pending": "待檢測",
          "watch": "關注"
        },
        "messages": {
          "clickRefresh": "點選重新整理載入健康資料",
          "loadFailed": "載入健康檢查資料失敗"
        },
        "notes": {
          "alert": "成功率低。请立即检查策略/帳戶状况。",
          "healthy": "成功率高且失敗次數可控。",
          "noSample": "樣本不足，至少需要 {{minSampleSize}} 筆運行記錄。",
          "pending": "請先執行健康檢查。",
          "watch": "成功率達到關注門檻（>= {{yellowSuccessRate}}%），建議持續觀察。"
        },
        "runLogs": {
          "signalType": "信號(用於下單)"
        },
        "sections": {
          "orders": "最近訂單记录",
          "runLogs": "最近執行日誌"
        },
        "summaryBanner": "健康分級：{{grade}}；最近樣本 {{totalRuns}} 次，成功率 {{successRate}}%",
        "thresholdsSummary": "min_sample_size={{minSampleSize}}；綠色：成功率>={{greenSuccessRate}}% 且失敗次數<={{greenMaxFailedRuns}}；黃色：成功率>={{yellowSuccessRate}}%",
        "title": "策略健康檢查 {{name}}"
      },
      "messages": {
        "defaultTemplateNotFound": "預設模板不存在，請刷新頁面重試",
        "executeFailed": "執行失敗",
        "importDefaultTemplateFailedNoId": "匯入預設模板失敗：未返回模板ID",
        "noOrderableSignal": "沒有可下單的信號",
        "orderFailed": "下單失敗",
        "orderSubmitted": "已提交下單",
        "parametersParseFailed": "參數解析失敗",
        "signalHoldCannotOrder": "當前信號為 hold/無交易動作，不能下單",
        "strategyExecuteFailed": "策略執行失敗",
        "templateCodeEmptyCannotExecute": "模板 code 為空，無法執行",
        "volumeInvalid": "下單手數無效（volume 必須 > 0）"
      },
      "nextRunAt": "下次運行",
      "status": {
        "disabled": "已停用",
        "running": "運行中"
      },
      "table": {
        "account": "帳號",
        "actions": "操作",
        "lastRun": "最後運行時間",
        "name": "名稱",
        "schedule": "計劃",
        "status": "狀態",
        "template": "模板",
        "tradeParams": "交易參數"
      },
      "templateVisibility": {
        "private": "私有",
        "public": "公開"
      },
      "title": "策略調度",
      "triggerModal": {
        "actions": {
          "confirmOrder": "確認下單",
          "rerun": "重新執行"
        },
        "cards": {
          "logs": "執行日誌",
          "signal": "信號(用於下單)"
        },
        "confirmOrder": {
          "ok": "确认",
          "title": "確認下單"
        },
        "emptyLogs": "(無日誌)",
        "emptySignal": "无信号",
        "messages": {
          "signalNotOrderable": "信号不可下单"
        },
        "summary": {
          "account": "帳號",
          "scheduleName": "調度名稱",
          "symbol": "品種",
          "timeframe": "週期"
        },
        "title": "立即執行(直接下單)"
      },
      "validation": {
        "parametersMustBeJsonObject": "參數必须为 JSON 对象"
      }
    }
  }
} as const;
export default Schedules;
