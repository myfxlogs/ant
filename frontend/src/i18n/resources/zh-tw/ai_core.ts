// Auto-generated from proto/ant/v1/i18n/ai_core_zh-tw.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const AiCore = {
  "ai": {
    "agentPrompts": {
      "code": {
        "title": "程式碼生成 Agent"
      },
      "risk": {
        "title": "风控与执行约束"
      },
      "signals": {
        "title": "信号与指标设计"
      },
      "style": {
        "title": "市场狀態/风格推荐"
      }
    },
    "assistant": {
      "messages": {
        "noCodeBlockFound": "No code block found (\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\`\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\`\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\`...\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\`\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\`\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\`)"
      }
    },
    "backtestScoreCard": {
      "backendRiskScore": {
        "empty": "无（先儲存模板，回测完成後自動計算）",
        "loading": "計算中...",
        "reasons": "原因",
        "reliable": "可靠",
        "title": "策略風險評分",
        "unknown": "未知",
        "unreliable": "不可靠",
        "warnings": "警告"
      },
      "chart": {
        "title": "净值曲线"
      },
      "level": {
        "excellent": "優秀",
        "fair": "一般",
        "good": "良好",
        "poor": "差"
      },
      "metrics": {
        "annualReturn": "年化收益率",
        "equityPoints": "净值点数",
        "maxDrawdown": "最大回撤",
        "sharpe": "夏普比率",
        "totalReturn": "總收益率",
        "totalTrades": "總交易次數",
        "winRate": "勝率"
      },
      "recommendation": {
        "cautious": "謹慎上線：建議先小資金/手動驗證一段時間。",
        "loading": "風險評估進行中，請等待完成後再上線。",
        "notRecommended": "Not recommended for direct live: high risk or unreliable, optimize before trying.",
        "recommended": "建議上線：風險可控，指標健康。"
      },
      "score": {
        "empty": "暫無評分（等待回測或無指標資料）",
        "title": "綜合評分（啟發式）"
      },
      "stateLabel": "狀態",
      "status": {
        "cancelRequested": "取消中",
        "canceled": "已取消",
        "failed": "失敗",
        "pending": "佇列中",
        "running": "執行中",
        "succeeded": "成功"
      },
      "title": "回測評分卡"
    },
    "chatBox": {
      "collapse": "摺疊",
      "emptyDescription": "開始與AI助手對話",
      "expandAll": "全部展開",
      "thinking": "思考中...",
      "truncated": "內容過長，已截斷"
    },
    "client": {
      "errors": {
        "contentBlocked": "供應商安全過濾器已阻止回應。請重新措辭提示後重試。",
        "contextTooLong": "請求超過模型上下文視窗大小。請縮短對話/輸入內容，或選擇上下文更大的模型。",
        "edgeGatewayTimeout": "網站最外層閘道逾時（常見為 Cloudflare 524）：請求未到應用程式即被中斷，「產生程式碼」等長步驟較易觸發。請在辯論程式碼步驟按「重新嘗試產生程式碼」，或先返回上一步再進入產生程式碼；仍失敗時需請維運調大閘道／來源站逾時。",
        "forbidden": "供應商拒絕請求（403）。請檢查金鑰權限、IP白名單及帳戶狀態。",
        "gatewayForbidden403": "閘道禁止存取（403）。",
        "gatewayRateLimited429": "網關速率受限 (429)。",
        "gatewayTimeoutOrUnreachable": "閘道逾時或無法連線。",
        "gatewayUnauthorized401": "閘道未經授權（401）。",
        "insufficientBalance": "供應商回報餘額不足/逾期付款。請在供應商控制台充值後重試。",
        "invalidModelId": "模型{{model}}不可用——可能名稱錯誤、已棄用或超出您的權限範圍。請從下拉選單中選擇其他模型，或從供應商控制台複製標準ID。",
        "networkUnreachable": "閘道逾時或無法連線。請檢查基礎URL、網路連線，或稍後重試。",
        "providerInternalError": "供應商返回伺服器端錯誤（5xx）。請稍候或切換至其他供應商。",
        "rateLimited": "供應商正在對您的請求進行速率限制。請稍候再試。",
        "regionNotSupported": "所選供應商在您的地區/國家不可用。請切換至其他供應商。",
        "requestFailed": "請求失敗，請重試。",
        "unauthorized": "供應商拒絕API金鑰（401）。請檢查金鑰值及其對所選模型的存取權限。"
      }
    },
    "consensus": {
      "actions": {
        "refresh": "重新整理"
      },
      "fields": {
        "account": "帳戶",
        "symbol": "品種",
        "timeframe": "周期"
      },
      "panel": {
        "decision": "決策",
        "overallScore": "總體",
        "technicalScore": "技術面",
        "title": "目標評分"
      },
      "signals": {
        "ma": {
          "trend": "均線趨勢"
        },
        "macd": {
          "flag": "訊號",
          "hist": "柱狀圖",
          "signalLine": "訊號線",
          "trend": "形態",
          "value": "MACD"
        },
        "rsi": {
          "flag": "訊號",
          "value": "RSI"
        }
      },
      "title": "共識與討論"
    },
    "conversation": {
      "defaultTitle": "新對話"
    },
    "gate": {
      "allPassed": "全部6個閘道已通過——策略符合推廣至上線評估條件",
      "backtestGrossReturn": "回測總收益率",
      "backtestNetReturn": "回測淨收益率",
      "dailyReturns": "每日收益率（逗號或換行分隔）",
      "descriptions": {
        "compliance": "DSL表達式非空驗證",
        "correlation": "与现有策略的訊號相關性檢查",
        "deflated_sharpe": "Lopez de Prado縮減夏普比率",
        "lookahead": "未來函數引用掃描（close[t+N]、ref負偏移）",
        "paper": "≥14天紙交易驗證",
        "walkforward": "淨化前進式交叉驗證"
      },
      "details": "詳情",
      "dslExpression": "DSL表達式",
      "evaluating": "評估中...",
      "fail": "失敗",
      "failed": "失敗：{{gate}}",
      "gateProgress": "閘道評估進度",
      "labels": {
        "compliance": "合規性",
        "correlation": "相關性",
        "deflated_sharpe": "縮減夏普比率",
        "lookahead": "前瞻偏差",
        "paper": "紙交易",
        "walkforward": "前進式驗證"
      },
      "noData": "無數據",
      "numAttempts": "策略嘗試次數",
      "paperDays": "紙交易天數",
      "paperMetrics": "紙交易指標",
      "paperNetPnL": "紙交易淨損益",
      "paperNetReturn": "紙交易淨收益率",
      "paperTradeCount": "紙交易次數",
      "pass": "通过",
      "pipelineDesc": "6階段閘道管線：合規性 → 前瞻偏差 → 前進式驗證 → 縮減夏普 → 紙交易 → 相關性",
      "pipelineResult": "管線結果",
      "retry": "重試",
      "runHint": "请先執行回测，然后點選\"執行质量门\"評估策略质量。",
      "runPipeline": "執行閘道管線",
      "selectRun": "選擇回測運行...",
      "skipped": "已跳过",
      "status": {
        "evaluating": "評估中..."
      },
      "strategyParams": "策略參數",
      "title": "AI閘道進度",
      "unknown": "未知"
    },
    "gateway": {
      "balance": "錢包餘額",
      "modelPlaceholder": "選擇 AI 模型",
      "monthlyCost": "本月費用",
      "monthlyTokens": "本月 Token",
      "noModels": "暫無可用模型",
      "selectModel": "選擇模型",
      "title": "AI 網關",
      "usageByFeature": "按功能用量",
      "useGateway": "AI 網關",
      "useGatewayDesc": "扣錢包餘額 · 按 Token 計費",
      "useOwnKey": "我的 API Key",
      "useOwnKeyDesc": "直付廠商 · 自行管理",
      "useOwnKeyHint": "使用你自己的 API Key，直接向所選廠商付費。在下方選擇廠商卡片進行配置。"
    },
    "reports": {
      "tradeAnalysis": {
        "riskAssessmentPrefix": "風險評估:",
        "title": "AI交易分析報告"
      }
    },
    "requireConfig": {
      "actions": {
        "goSettings": "前往設定"
      },
      "description": "請先前往設定頁面配置AI供應商、模型及API金鑰，然後再使用策略精靈或聊天功能。",
      "title": "尚未配置LLM"
    },
    "riskEval": {
      "failed": "風險評估失敗"
    },
    "signalCard": {
      "actions": {
        "cancel": "取消",
        "confirm": "確認",
        "executeTrade": "執行交易"
      },
      "confirmCancel": {
        "title": "确定要取消此訊號？"
      },
      "confirmExecute": {
        "description": "将立即下单",
        "title": "確定要執行此交易訊號嗎？"
      },
      "labels": {
        "analysisReason": "分析理由",
        "confidence": "信心度",
        "price": "價格",
        "stopLoss": "止損",
        "takeProfit": "止盈",
        "volume": "手數"
      },
      "status": {
        "cancelled": "已取消",
        "confirmed": "已確認",
        "executed": "已執行",
        "pending": "待處理"
      }
    },
    "strategyCard": {
      "actionType": {
        "alert": "警報",
        "buy": "買入",
        "closeLong": "平多",
        "closeShort": "平空",
        "sell": "賣出"
      },
      "actions": {
        "start": "啟動",
        "stop": "停止"
      },
      "confirmDelete": {
        "description": "刪除后無法恢复",
        "title": "確定要刪除此策略嗎？"
      },
      "labels": {
        "lastTriggeredAt": "最近觸發: {{time}}",
        "triggeredCount": "已觸發{{count}}次"
      },
      "sections": {
        "actions": "操作",
        "conditions": "觸發條件"
      },
      "status": {
        "active": "啟用中",
        "inactive": "未啟用",
        "paused": "已暫停"
      },
      "tooltips": {
        "createdAt": "建立時間",
        "lastTriggeredAt": "最近觸發"
      }
    },
    "systemAI": {
      "cardState": {
        "enabled": "已啟用",
        "noKey": "未設定",
        "noModel": "待選模型",
        "readyDisabled": "就緒 · 已停用"
      },
      "cardTags": {
        "current": "目前",
        "enabledButUnavailable": "已啟用但不可用",
        "hasKey": "已配金鑰",
        "noKey": "未配金鑰",
        "noModels": "未設定可用模型"
      },
      "customProvider": {
        "deleted": "自訂提供者已刪除",
        "fillNameFirst": "请先填写名稱",
        "nameHint": "用于识别此提供者的唯一名稱",
        "nameLabel": "提供者名稱",
        "namePlaceholder": "我的自訂提供者",
        "nameRequired": "服务商名称不能為空"
      },
      "emptyConfigs": "暫無 AI 廠商 設定（系統啟動時會自動建立預設 廠商）",
      "fields": {
        "apiKeyHint": "輸入後將自動加密儲存，無需手動提交",
        "apiKeyPastePlaceholder": "貼上 API 金鑰，將自動預儲存",
        "autoFetching": "自動拉取中",
        "baseUrlCustomHint": "輸入 OpenAI 相容端點，例如 https://model.example.com/v1",
        "baseUrlCustomPlaceholder": "例如: https://model.example.com/v1",
        "baseUrlReadonlyHint": "官方地址由系統維護，不可修改",
        "baseUrlReadonlyPlaceholder": "官方地址（唯讀）",
        "enabledHint": "關閉後該廠商不參與系統路由",
        "httpWarning": "目前為 HTTP，生產環境建議使用 HTTPS",
        "maxTokensHint": "單次回應最大權杖數",
        "primaryFor": "主要用途",
        "primaryForHint": "用于內部分发：對話/嵌入/摘要/推理",
        "temperatureHint": "越高越發散，越低越穩定",
        "timeoutHint": "單次請求最長等待時間"
      },
      "messages": {
        "autoDiscoveredModels": "自動探索到{{count}}個模型（僅供參考）",
        "autoValidatedModels": "自動驗證：找到{{count}}個模型",
        "configSaveFailed": "配置儲存失敗",
        "configSaved": "配置已儲存",
        "deleteSecretFailed": "刪除金鑰失敗",
        "loadConfigFailed": "載入配置失敗",
        "secretAutoSaveFailed": "金鑰自動儲存失敗",
        "secretDeletedConfigReset": "金鑰已刪除，供應商配置已重設為預設值",
        "secretSavedAutoDiscover": "金鑰已儲存，正在自動探索模型...",
        "toggleEnabledFailed": "切換啟用狀態失敗",
        "validationFailedNeedApiKey": "驗證失敗：此服務商通常需要 API Key，請先填寫並儲存 Key 後重試。",
        "validationPassedModels": "驗證通過：找到{{count}}個模型"
      },
      "pageSubtitle": "設定 AI 大腦 — 選擇模型廠商、管理 API 金鑰與可用模型，並指定全站兜底使用的「預設主模型」。",
      "pageTitle": "AI 助手設定",
      "section1": {
        "subtitle": "Cards show each provider's configuration and readiness; click to select",
        "title": "選擇模型廠商"
      },
      "status": {
        "checkUrl": "請檢查 基礎網址",
        "checkUrlDesc": "API 金鑰已就緒，但地址似乎無效",
        "configReady": "設定已就緒",
        "configReadyDesc": "新增可用模型後系統將自動完成連線檢測",
        "connectionFailed": "連接錯誤，请檢查上方提示",
        "error": "存在異常",
        "needKey": "請完成金鑰設定",
        "needKeyDesc": "填寫 API 金鑰後將自動發現模型清單",
        "noProvider": "尚未選擇供應商",
        "noProviderDesc": "請從下方卡片挑選一個模型廠商開始設定",
        "notEnabled": "連線正常，尚未啟用",
        "notEnabledDesc": "打開「啟用」開關即可投入使用",
        "ready": "運行就緒",
        "readyDesc": "已啟用並連線正常"
      },
      "statusBar": {
        "checking": "連線檢測中…",
        "connected": "已連接",
        "disabled": "未啟用",
        "enabled": "已啟用",
        "keyReady": "金鑰就緒"
      },
      "taglines": {
        "anthropic": "Claude 系列",
        "deepseek": "DeepSeek · 高性價比",
        "moonshot": "Kimi · 長上下文",
        "openai": "GPT 系列 · 官方",
        "openai_compatible": "任意相容端點",
        "qwen": "阿里雲 · 中文最佳化",
        "zhipu": "清華系 · 通用"
      }
    },
    "tabs": {
      "agentSettings": "專家設定",
      "gate": "AI 质量门",
      "settings": "設定"
    },
    "workflowRuns": {
      "defaultTitle": "AI工作流程",
      "hints": {
        "selectToViewDetail": "从左侧選擇執行记录查看詳情"
      },
      "messages": {
        "loadDetailFailed": "載入詳情失敗",
        "loadListFailed": "載入執行列表失敗"
      },
      "title": "AI工作流程"
    }
  }
} as const;
export default AiCore;
