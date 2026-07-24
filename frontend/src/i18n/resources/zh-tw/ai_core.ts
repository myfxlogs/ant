// Auto-generated from proto/ant/v1/i18n/ai_core_zh-cn.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const AiCore = {
  "ai": {
    "consensus": {
      "signals": {
        "ma": {
          "trend": "均線趨勢"
        },
        "macd": {
          "flag": "訊號",
          "hist": "柱體",
          "signalLine": "訊號線",
          "trend": "形態",
          "value": "MACD"
        },
        "rsi": {
          "flag": "訊號",
          "value": "RSI"
        }
      },
      "actions": {
        "refresh": "重新整理"
      },
      "fields": {
        "account": "賬號",
        "symbol": "品種",
        "timeframe": "週期"
      },
      "panel": {
        "decision": "決策",
        "overallScore": "總體分",
        "technicalScore": "技術面",
        "title": "客觀評分"
      },
      "title": "共識與對話"
    },
    "agentPrompts": {
      "code": {
        "title": "程式碼生成 Agent"
      },
      "risk": {
        "title": "風控與執行約束"
      },
      "signals": {
        "title": "訊號與指標設計"
      },
      "style": {
        "title": "市場狀態/風格推薦"
      }
    },
    "assistant": {
      "messages": {
        "noCodeBlockFound": "未找到程式碼塊 (\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\`\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\`\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\`...\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\`\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\`\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\`)"
      }
    },
    "backtestScoreCard": {
      "backendRiskScore": {
        "empty": "無（先儲存模板，回測完成後自動計算）",
        "loading": "計算中...",
        "reasons": "原因",
        "reliable": "可靠",
        "title": "策略風險評分",
        "unknown": "未知",
        "unreliable": "不可靠",
        "warnings": "警告"
      },
      "chart": {
        "title": "淨值曲線"
      },
      "level": {
        "excellent": "優秀",
        "fair": "一般",
        "good": "良好",
        "poor": "差"
      },
      "metrics": {
        "annualReturn": "年化收益",
        "equityPoints": "淨值點數",
        "maxDrawdown": "最大回撤",
        "sharpe": "夏普",
        "totalReturn": "總收益",
        "totalTrades": "交易次數",
        "winRate": "勝率"
      },
      "recommendation": {
        "cautious": "謹慎上線：建議先小資金/手動確認執行一段時間。",
        "loading": "風險評估計算中，建議先等待完成再上線。",
        "notRecommended": "不建議直接上線：高風險或不可靠，請先最佳化後再試。",
        "recommended": "推薦上線：風險可控，指標整體健康。"
      },
      "score": {
        "empty": "暫無評分（等待回測完成或無 metrics）",
        "title": "綜合評分（啟發式）"
      },
      "status": {
        "cancelRequested": "取消中",
        "canceled": "已取消",
        "failed": "失敗",
        "pending": "排隊中",
        "running": "執行中",
        "succeeded": "成功"
      },
      "stateLabel": "狀態",
      "title": "回測評分卡"
    },
    "client": {
      "errors": {
        "contentBlocked": "服務商安全過濾器阻止了響應。請改寫提示詞後重試。",
        "contextTooLong": "請求超出模型上下文視窗。請縮短對話/輸入，或選擇上下文更大的模型。",
        "edgeGatewayTimeout": "邊緣閘道器超時 (通常為 Cloudflare HTTP 524)：瀏覽器未收到應用響應，長時執行操作常見。請重試；如持續出現，請聯絡運維提高代理/源站超時。",
        "forbidden": "服務商拒絕了請求 (403)。請檢查 Key 許可權、IP 白名單和賬戶狀態。",
        "gatewayForbidden403": "閘道器被禁止 (403)。",
        "gatewayRateLimited429": "閘道器速率受限 (429)。",
        "gatewayTimeoutOrUnreachable": "閘道器超時或不可達。",
        "gatewayUnauthorized401": "閘道器未授權 (401)。",
        "insufficientBalance": "服務商報告餘額為空/欠費。請在服務商控制檯充值後重試。",
        "invalidModelId": "模型不可用{{model}} — 可能輸入錯誤、已棄用或超出您的套餐。請從下拉選單中選擇其他模型，或從服務商控制檯複製標準 ID。",
        "networkUnreachable": "閘道器超時或不可達。請檢查 Base URL、網路連線，或稍後重試。",
        "providerInternalError": "服務商返回伺服器錯誤 (5xx)。請稍候或切換到其他服務商。",
        "rateLimited": "服務商正在限制您的請求頻率，請稍候再試。",
        "regionNotSupported": "所選服務商在您所在地區/國家不可用，請切換到其他服務商。",
        "requestFailed": "請求失敗，請重試。",
        "unauthorized": "服務商拒絕了 API Key (401)。請檢查 Key 值及其是否有權訪問所選模型。"
      }
    },
    "gate": {
      "descriptions": {
        "compliance": "DSL 表示式非空驗證",
        "correlation": "與現有策略的訊號相關性檢查",
        "deflated_sharpe": "Lopez de Prado 緊縮夏普比率",
        "lookahead": "掃描未來函式引用 (close[t+N], ref 負偏移)",
        "paper": "≥14 天模擬交易驗證",
        "walkforward": "Purged Walk-Forward 交叉驗證"
      },
      "labels": {
        "compliance": "合規檢查",
        "correlation": "相關性",
        "deflated_sharpe": "通縮夏普比率",
        "lookahead": "前視偏差",
        "paper": "模擬交易",
        "walkforward": "前向分析"
      },
      "status": {
        "evaluating": "評估中..."
      },
      "allPassed": "所有 6 個 Gate 透過，策略可進入 PromoteToLive 評估",
      "backtestGrossReturn": "回測毛收益",
      "backtestNetReturn": "回測淨收益",
      "dailyReturns": "日收益率 (逗號或換行分隔)",
      "details": "詳細結果",
      "dslExpression": "DSL 表示式",
      "evaluating": "評估中...",
      "fail": "失敗",
      "failed": "未透過: {{gate}}",
      "gateProgress": "Gate 評估進度",
      "noData": "無資料",
      "numAttempts": "策略嘗試次數",
      "paperDays": "模擬天數",
      "paperMetrics": "模擬交易指標",
      "paperNetPnL": "模擬 Net P&L",
      "paperNetReturn": "模擬淨收益",
      "paperTradeCount": "模擬交易數",
      "pass": "透過",
      "pipelineDesc": "6 級 Gate 管道: Compliance → LookAhead → Walk-Forward → DeflatedSharpe → Paper → Correlation",
      "pipelineResult": "管道結果",
      "retry": "重試",
      "runHint": "請先執行回測，然後點選\"執行質量門\"評估策略質量。",
      "runPipeline": "執行 Gate 管道",
      "selectRun": "選擇回測執行...",
      "skipped": "已跳過",
      "strategyParams": "策略引數",
      "title": "AI Gate 進度面板",
      "unknown": "未知"
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
      "description": "請先到設定頁配置 AI 提供商、模型與 API Key，然後再使用策略嚮導或聊天。",
      "title": "尚未配置大模型"
    },
    "signalCard": {
      "actions": {
        "cancel": "取消",
        "confirm": "確認",
        "executeTrade": "執行交易"
      },
      "confirmCancel": {
        "title": "確定要取消此訊號？"
      },
      "confirmExecute": {
        "description": "將立即下單",
        "title": "確定要執行這個交易訊號嗎?"
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
        "pending": "待確認"
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
        "description": "刪除後無法恢復",
        "title": "確定要刪除這個策略嗎?"
      },
      "labels": {
        "lastTriggeredAt": "最近觸發: {{time}}",
        "triggeredCount": "觸發 {{count}} 次"
      },
      "sections": {
        "actions": "操作",
        "conditions": "觸發條件"
      },
      "status": {
        "active": "執行中",
        "inactive": "已停止",
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
        "noKey": "未配置",
        "noModel": "待選模型",
        "readyDisabled": "就緒 · 已禁用"
      },
      "cardTags": {
        "current": "當前",
        "enabledButUnavailable": "已啟用但不可用",
        "hasKey": "已配金鑰",
        "noKey": "未配金鑰",
        "noModels": "未配置可用模型"
      },
      "customProvider": {
        "deleted": "自定義提供商已刪除",
        "fillNameFirst": "請先填寫名稱",
        "nameHint": "用於識別此提供商的唯一名稱",
        "nameLabel": "提供商名稱",
        "namePlaceholder": "我的自定義提供商",
        "nameRequired": "服務商名稱不能為空"
      },
      "fields": {
        "apiKeyHint": "輸入後將自動加密儲存，無需手動提交",
        "apiKeyPastePlaceholder": "貼上 API Key，將自動預儲存",
        "autoFetching": "自動拉取中",
        "baseUrlCustomHint": "輸入 OpenAI 相容端點，例如 https://model.example.com/v1",
        "baseUrlCustomPlaceholder": "例如: https://model.example.com/v1",
        "baseUrlReadonlyHint": "官方地址由系統維護，不可修改",
        "baseUrlReadonlyPlaceholder": "官方地址（只讀）",
        "enabledHint": "關閉後該廠商不參與系統路由",
        "httpWarning": "當前為 HTTP，生產環境建議使用 HTTPS",
        "maxTokensHint": "單次響應最大 token 數",
        "primaryFor": "主要用途（Primary For）",
        "primaryForHint": "用於內部分發：對話/嵌入/摘要/推理",
        "temperatureHint": "越高越發散，越低越穩定",
        "timeoutHint": "單次請求最長等待時間"
      },
      "messages": {
        "autoDiscoveredModels": "已自動發現 {{count}} 個模型（僅作選擇建議）",
        "autoValidatedModels": "已自動驗證：發現 {{count}} 個模型",
        "configSaveFailed": "配置儲存失敗",
        "configSaved": "配置已儲存",
        "deleteSecretFailed": "刪除金鑰失敗",
        "loadConfigFailed": "載入配置失敗",
        "secretAutoSaveFailed": "金鑰自動儲存失敗",
        "secretDeletedConfigReset": "金鑰已刪除，廠商配置已恢復預設初始化",
        "secretSavedAutoDiscover": "金鑰已儲存，正在自動發現模型...",
        "toggleEnabledFailed": "更新啟用狀態失敗",
        "validationFailedNeedApiKey": "驗證失敗：此服務商通常需要 API Key，請先填寫並儲存 Key 後重試。",
        "validationPassedModels": "驗證透過：發現 {{count}} 個模型"
      },
      "section1": {
        "subtitle": "卡片展示每個提供商的配置和就緒狀態，點選選擇",
        "title": "選擇模型廠商"
      },
      "status": {
        "checkUrl": "請檢查 Base URL",
        "checkUrlDesc": "API Key 已就緒，但地址似乎無效",
        "configReady": "配置已就緒",
        "configReadyDesc": "新增可用模型後系統將自動完成連通性檢測",
        "connectionFailed": "連線錯誤，請檢查上方提示",
        "error": "存在異常",
        "needKey": "請完成金鑰配置",
        "needKeyDesc": "填寫 API Key 後將自動發現模型列表",
        "noProvider": "尚未選擇廠商",
        "noProviderDesc": "請從下方卡片挑選一個模型廠商開始配置",
        "notEnabled": "連線正常，尚未啟用",
        "notEnabledDesc": "開啟「啟用」開關即可投入使用",
        "ready": "執行就緒",
        "readyDesc": "已啟用並連線正常"
      },
      "statusBar": {
        "checking": "連通性檢測中…",
        "connected": "已連線",
        "disabled": "未啟用",
        "enabled": "已啟用",
        "keyReady": "金鑰就緒"
      },
      "taglines": {
        "anthropic": "Claude 系列",
        "deepseek": "深度求索 · 高價效比",
        "moonshot": "Kimi · 長上下文",
        "openai": "GPT 系列 · 官方",
        "openai_compatible": "任意相容端點",
        "qwen": "阿里雲 · 中文最佳化",
        "zhipu": "清華系 · 通用"
      },
      "emptyConfigs": "暫無 AI Provider 配置（系統啟動時會自動建立預設 Provider）",
      "pageSubtitle": "配置 AI 大腦 — 選擇模型廠商、管理 API 金鑰與可用模型，並指定全站兜底使用的「預設主模型」。",
      "pageTitle": "AI 助手設定"
    },
    "workflowRuns": {
      "hints": {
        "selectToViewDetail": "從左側選擇執行記錄檢視詳情"
      },
      "messages": {
        "loadDetailFailed": "載入詳情失敗",
        "loadListFailed": "載入執行記錄失敗"
      },
      "defaultTitle": "AI 工作流",
      "title": "AI 工作流"
    },
    "chatBox": {
      "collapse": "收起",
      "emptyDescription": "開始與AI助手對話",
      "expandAll": "展開全部",
      "thinking": "思考中...",
      "truncated": "內容過長，已截斷"
    },
    "conversation": {
      "defaultTitle": "新對話"
    },
    "gateway": {
      "balance": "錢包餘額",
      "modelPlaceholder": "選擇 AI 模型",
      "monthlyCost": "本月費用",
      "monthlyTokens": "本月 Token",
      "noModels": "暫無可用模型",
      "selectModel": "選擇模型",
      "title": "AI 閘道器",
      "usageByFeature": "按功能用量",
      "useGateway": "AI 閘道器",
      "useGatewayDesc": "扣錢包餘額 · 按 Token 計費",
      "useOwnKey": "我的 API Key",
      "useOwnKeyDesc": "直付廠商 · 自行管理",
      "useOwnKeyHint": "使用你自己的 API Key，直接向所選廠商付費。在下方選擇廠商卡片進行配置。",
      "groupMyKeys": "我的 API 金鑰",
      "groupGateway": "AI 閘道器",
      "groupCurrent": "當前選擇"
    },
    "riskEval": {
      "failed": "風險評估失敗"
    },
    "tabs": {
      "agentSettings": "專家設定",
      "gate": "AI 質量門",
      "settings": "設定"
    }
  }
} as const;
export default AiCore;
