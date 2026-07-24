// Auto-generated supplementary keys for strategy
// TODO: Translate to zh-tw
const StrategyExtra = {
  "strategy": {
    "backtest": {
      "lotSize": "手數",
      "strategyParameters": "策略引數"
    },
    },
    "workspace": {
      "chartIndicators": {
        "subPane": "副圖指標"
      },
      },
      "tour": {
        "codeDesc": "在此編寫或貼上 MQL 策略程式碼，也可匯入 .mq4/.mq5 檔案。",
        "ai": "AI 助手",
        "aiDesc": "讓 AI 生成、最佳化或除錯策略，生成的程式碼會即時顯示在編輯器中。",
        "backtestDesc": "使用可配置引數執行回測，檢視資金曲線、交易統計和風險指標。",
        "save": "儲存併發布",
        "saveDesc": "將策略儲存為模板、釋出到市場或部署到實盤排程。"
      }
      }
    },
    "codeAssist": {
    },
    },
    "chat": {
      "codeGenerated": "程式碼已生成，使用下方按鈕執行策略審查和回測。"
    },
    },
    "templates": {
      "saveCurrent": "儲存當前策略",
      "chatEdit": "對話編輯",
      "confirmDelete": "刪除此策略？",
      "noTemplates": "暫無儲存的策略模板",
      "sourceCode": "策略原始碼",
      "gallery": {
      "gallery": {
        "fork": "復刻並編輯",
        "aiGenerate": "AI 生成",
        "searchPlaceholder": "搜尋策略...",
        "empty": "未找到策略",
        "deleteFailed": "刪除失敗"
      },
      },
      "scheduleLaunch": {
        "metrics": {
          "maxDrawdown": "最大回撤",
          "sharpe": "夏普比率"
        }
        }
      },
      "detail": {
        "notFound": "未找到策略",
        "noDescription": "暫無描述",
        "equityCurve": "資金曲線",
        "tradeStats": "交易統計"
      },
      },
      "table": {
      },
      },
      "messages": {
        "publishFailed": "釋出失敗"
      },
      },
      "actions": {
      },
      },
    },
    },
    "live": {
      "stopSuccess": "策略已停止",
      "stopFailed": "停止失敗",
      "runId": "執行 ID",
      "account": "帳號",
      "symbol": "品種",
      "timeframe": "週期",
      "mode": "模式",
      "signals": "訊號",
      "errors": "錯誤",
      "startedAt": "開始時間",
      "watchSignals": "檢視訊號",
      "confirmStop": "確認停止此策略？",
      "status": "狀態",
      "totalSignals": "總訊號數",
      "stoppedAt": "停止時間",
      "error": "錯誤",
      "title": "實盤策略監控",
      "activeTab": "活躍執行",
      "noActive": "無活躍策略",
      "historyTab": "執行歷史",
      "noRuns": "無策略執行記錄",
      "schedulesTab": "排程",
      "signalLog": "訊號日誌",
      "waitingSignals": "等待訊號...",
      "time": "時間",
      "signalType": "型別",
      "volume": "手數",
      "price": "價格",
      "sl": "止損",
      "tp": "止盈",
      "reason": "原因"
    },
    "ai": {
      "explainHint": "編寫程式碼後檢視 AI 解釋。",
      "settingsHint": "配置 AI 提供商和模型"
    },
    },
    "validate": {
      "fixWithAI": "傳送錯誤到 AI 修訂",
      "allClear": "所有檢查透過 — 未發現問題。",
      "passed": "驗證透過 — 現在可以儲存。"
    },
    },
    "importEA": {
      "importTab": "匯入 EA",
      "codeTooShort": "請貼上完整的 EA/指標原始碼。",
      "pastePlaceholder": "貼上 MQL4/MQL5 EA 程式碼...",
      "migration": "策略匯入",
      "migration": "策略匯入",
      "aiTranslate": "AI 翻譯",
      "bridge": "盲區橋接",
      "analyze": "分析策略結構",
      "confirmImport": "確認匯入",
      "tryAI": "AI 翻譯補充",
      "importSuccess": "MQL 原始碼已匯入，點選「應用到編輯器」寫入編輯器",
      "importSuccess": "MQL 原始碼已匯入，點選「Apply to Editor」寫入編輯器",
      "translate": "翻譯為 Go",
      "translating": "貼上 MQL4/MQL5 程式碼並點選翻譯",
      "bridgeBtn": "盲區橋接翻譯",
      "bridgeBtn": "盲區橋接翻譯",
      "bridging": "AI bridging blind spots...",
      "bridgeFailedMsg": "Agent 無法自動橋接所有盲區",
      "noBridgeNeeded": "覆蓋率 100%，無需橋接",
      "tooltip": "匯入 MQL4/MQL5 原始碼",
      "button": "匯入 MQL",
      "title": "匯入 MQL 策略"
    },
    },
    "version": {
      "rollbackSuccess": "已回滾到版本 {{n}}",
      "rollbackFailed": "回滾失敗",
      "loadVersionFailed": "載入版本失敗",
      "loadDiffFailed": "載入差異失敗",
      "colSummary": "變更摘要",
      "rollbackConfirm": "回滾到 v{{n}}？",
      "title": "版本歷史",
      "empty": "暫無版本歷史",
      "history": "版本歷史"
    }
    }
  }
} as const;
export default StrategyExtra;
