const aiCore = {
  ai: {
    client: {
      errors: {
        edgeGatewayTimeout: '網站最外層閘道逾時（常見為 Cloudflare 524）：請求未到應用程式即被中斷，「產生程式碼」等長步驟較易觸發。請在辯論程式碼步驟按「重新嘗試產生程式碼」，或先返回上一步再進入產生程式碼；仍失敗時需請維運調大閘道／來源站逾時。',
        requestFailed: '請求失敗，請重試。',
        insufficientBalance: '供應商回報餘額不足/逾期付款。請在供應商控制台充值後重試。',
        rateLimited: '供應商正在對您的請求進行速率限制。請稍候再試。',
        unauthorized: '供應商拒絕API金鑰（401）。請檢查金鑰值及其對所選模型的存取權限。',
        forbidden: '供應商拒絕請求（403）。請檢查金鑰權限、IP白名單及帳戶狀態。',
        invalidModelId: '模型{{model}}不可用——可能名稱錯誤、已棄用或超出您的權限範圍。請從下拉選單中選擇其他模型，或從供應商控制台複製標準ID。',
        contextTooLong: '請求超過模型上下文視窗大小。請縮短對話/輸入內容，或選擇上下文更大的模型。',
        contentBlocked: '供應商安全過濾器已阻止回應。請重新措辭提示後重試。',
        regionNotSupported: '所選供應商在您的地區/國家不可用。請切換至其他供應商。',
        providerInternalError: '供應商返回伺服器端錯誤（5xx）。請稍候或切換至其他供應商。',
        networkUnreachable: '閘道逾時或無法連線。請檢查基礎URL、網路連線，或稍後重試。',
        gatewayTimeoutOrUnreachable: '閘道逾時或無法連線。',
        gatewayUnauthorized401: '閘道未經授權（401）。',
        gatewayForbidden403: '閘道禁止存取（403）。',
        gatewayRateLimited429: '閘道速率限制（429）。'
      }
    },
    'agent提示詞s': {
      style: {
        title: '市場狀態/風格推薦',
        prompt: `你是資深量化投研分析師。請基於以下資訊，推薦策略範式：趨勢/均值回歸/短線，並說明理由、適用條件與不適用場景。

輸出要求：用 Markdown，必須包含：
1) 推理過程：你如何從資料/約束/目標推導（分點）
2) 結論：主推薦（只能選一個主範式）+ 備選（可選）+ 適用/不適用條件
3) 風險提示：至少 3 條

{{baseInfo}}`
      },
      signals: {
        title: '訊號與指標設計',
        prompt: `你是量化因子與訊號工程師。請在不依賴外部資料（除非使用者提供宏觀事件表）的前提下，設計可實作的交易訊號。

要求：明確入場/出場/過濾條件，盡量參數化，避免過度擬合。

輸出要求：用 Markdown，必須包含：
1) 推理過程：為何選擇這些指標/閾值/過濾條件（分點）
2) 結論：可執行的規則清單（入場/出場/過濾），並給出參數建議（含預設值/範圍）
3) 邊界與風險：至少 3 條（例如：震盪/跳空/高波動/消息面等）

{{baseInfo}}`
      },
      risk: {
        title: '風控與執行約束',
        prompt: `你是交易風控與執行專家。請根據以下資訊，設計倉位管理、止損止盈、最大回撤控制、冷卻期/交易頻率限制等規則。

輸出要求：用 Markdown，必須包含：
1) 推理過程：為何這些風控能匹配目標/約束（分點）
2) 結論：硬約束（必須遵守）+ 預設參數（建議值/範圍）+ 觸發後的動作
3) 失敗模式：至少 3 條（例如：連續虧損、滑點擴大、點差異常等）

{{baseInfo}}`
      },
      code: {
        title: '程式碼生成',
        prompt: `你是 AntTrader Python 策略程式碼工程師。請生成一份可執行的 AntTrader Python 策略程式碼，要求：
- 必須通過 validate 校驗（禁止 import、禁止 dunder、遵守沙箱約束）
- 使用 on_tick / on_kline 等平台提供的 API（不要自訂網路/檔案存取）
- run 必須且只能接收一個參數：context（參數名必須是 context；不允許 run(ctx)、run(context, data) 等）
- run(context) 回傳一個 dict，至少包含：signal(buy/sell/hold)、symbol、confidence(0~1)、risk_level(low/medium/high)、reason
- 必須從 context["params"] 讀取參數（它是由調度參數注入的 dict）；參數缺失時使用參數表中的 default 值
- 使用上文的訊號設計與風控建議（若未提供，也請給出合理預設）
- 直接輸出完整程式碼，並用 \`\`\`python 包裹
- 嚴格輸出：只允許輸出 1 個 \`\`\`python 程式碼塊\`\`\`，除此之外不要輸出任何解釋文字
- 程式碼塊內必須是純 Python 程式碼：禁止出現 Markdown 符號（例如 "- ", "* ", "###"）、禁止出現中文全形標點、禁止出現三個反引號圍欄 \`\`\`

【必須照抄的入口模板（不要改函式名/參數個數/參數名）】
\`\`\`python
def run(context):
    params = context.get("params") or {}
    symbol = context.get("symbol") or params.get("symbol") or ""
    # TODO: 在這裡實作訊號/風控邏輯
    return {
        "signal": "hold",
        "symbol": symbol,
        "confidence": 0.5,
        "risk_level": "low",
        "reason": "",
    }
\`\`\`

{{baseInfo}}

【附：上游分析（若有）】
請你將市場/訊號/風控三個分析結論落到程式碼中（若上游結論未提供，也請給出合理預設）。`
      }
    },
    systemAI: {
      taglines: {
        openai: 'GPT 系列 · 官方',
        anthropic: 'Claude 系列',
        deepseek: 'DeepSeek · 高性價比',
        moonshot: 'Kimi · 長上下文',
        qwen: '阿里雲 · 中文最佳化',
        zhipu: '清華系 · 通用',
        openai_compatible: '任意相容端點'
      },
      pageTitle: 'AI 助手設定',
      pageSubtitle: '設定 AI 大腦 — 選擇模型廠商、管理 API 金鑰與可用模型，並指定全站兜底使用的「預設主模型」。',
      emptyConfigs: '暫無 AI 廠商 設定（系統啟動時會自動建立預設 廠商）',
      section1: {
        title: '選擇模型廠商',
        subtitle: '卡片直接展示每個廠商的設定與就緒狀態，點擊即可選擇'
      },
      statusBar: {
        enabled: '已啟用',
        disabled: '未啟用',
        keyReady: '金鑰就緒',
        checking: '連線檢測中…',
        connected: '連線正常'
      },
      status: {
        noprovider: '尚未選擇廠商',
        noProviderDesc: '請從下方卡片挑選一個模型廠商開始設定',
        error: '存在異常',
        ready: '運行就緒',
        readyDesc: '已啟用並連線正常',
        notEnabled: '連線正常，尚未啟用',
        notEnabledDesc: '打開「啟用」開關即可投入使用',
        configReady: '設定已就緒',
        configReadyDesc: '新增可用模型後系統將自動完成連線檢測',
        checkUrl: '請檢查 基礎網址',
        checkUrlDesc: 'API 金鑰已就緒，但地址似乎無效',
        needKey: '請完成金鑰設定',
        needKeyDesc: '填寫 API 金鑰後將自動發現模型清單',
        connectionFailed: '連線異常，請檢查上方提示',
        noProvider: '尚未選擇供應商'
      },
      cardState: {
        noKey: '未設定',
        noModel: '待選模型',
        enabled: '已啟用',
        readyDisabled: '已就緒 · 未啟用'
      },
      cardTags: {
        current: '目前',
        hasKey: '已配金鑰',
        noKey: '未配金鑰',
        noModels: '未設定可用模型',
        enabledButUnavailable: '啟用但不可用'
      },
      fields: {
        autoFetching: '自動拉取中',
        baseUrlCustomHint: '輸入 OpenAI 相容端點，例如 https://model.example.com/v1',
        baseUrlReadonlyHint: '官方地址由系統維護，不可修改',
        baseUrlCustomPlaceholder: '例如: https://model.example.com/v1',
        baseUrlReadonlyPlaceholder: '官方地址（唯讀）',
        httpWarning: '目前為 HTTP，生產環境建議使用 HTTPS',
        apiKeyHint: '輸入後將自動加密儲存，無需手動提交',
        apiKeyPastePlaceholder: '貼上 API 金鑰，將自動預儲存',
        enabledHint: '關閉後該廠商不參與系統路由',
        temperatureHint: '越高越發散，越低越穩定',
        timeoutHint: '單次請求最長等待時間',
        maxTokensHint: '單次回應最大權杖數',
        primaryFor: '主要用途',
        primaryForHint: '僅用於服務內部路由：對話 / 向量嵌入 / 摘要 / 推理'
      },
      messages: {
        loadConfigFailed: '載入配置失敗',
        secretSavedAutoDiscover: '金鑰已儲存，正在自動探索模型...',
        secretAutoSaveFailed: '金鑰自動儲存失敗',
        autoDiscoveredModels: '自動探索到{{count}}個模型（僅供參考）',
        autoValidatedModels: '自動驗證：找到{{count}}個模型',
        configSaved: '配置已儲存',
        configSaveFailed: '配置儲存失敗',
        toggleEnabledFailed: '切換啟用狀態失敗',
        secretDeletedConfigReset: '金鑰已刪除，供應商配置已重設為預設值',
        deleteSecretFailed: '刪除金鑰失敗',
        validationPassedModels: '驗證通過：找到{{count}}個模型',
        validationFailedNeedApiKey: '驗證失敗：此供應商通常需要API金鑰。請先填寫並儲存金鑰，然後重試。'
      }
    },
    tabs: {
      settings: '設定',
      agentSettings: '專家設定',
      gate: 'AI閘道'
    },
    agentPrompts: {
      style: {
        title: '市場狀況/風格推薦',
        prompt: `您是一位資深量化策略分析師。請根據以下資訊推薦策略範式：趨勢/均值回歸/短線，並說明理由、適用條件及不適用場景。

輸出要求：使用Markdown，必須包含：
1）推理過程：如何從資料/約束/目標推導（要點式）
2）結論：主要推薦（僅一種主要範式）+ 替代方案 + 適用/不適用條件
3）風險提示：至少3項

{{baseInfo}}`
      },
      signals: {
        title: '訊號與指標設計',
        prompt: `您是一位量化因子與訊號工程師。在不依賴外部資料（除非使用者提供宏觀事件表）的情況下，設計可執行的交易訊號。

要求：明確定義入場/出場/過濾條件，最好參數化，避免過度擬合。

輸出要求：使用Markdown，必須包含：
1）推理過程：為何選擇這些指標/閾值/過濾條件（要點式）
2）結論：可執行規則列表（入場/出場/過濾），附參數建議（預設值/範圍）
3）邊界與風險：至少3項（例如：盤整/跳空/高波動/新聞事件）

{{baseInfo}}`
      },
      risk: {
        title: '風險控制與執行約束',
        prompt: `您是一位交易風險與執行專家。請根據以下資訊設計倉位管理、止損/止盈、最大回撤控制、冷卻期/交易頻率限制等。

輸出要求：使用Markdown，必須包含：
1）推理過程：為何這些控制措施符合目標/約束（要點式）
2）結論：硬約束 + 預設參數（建議值/範圍）+ 觸發後的操作
3）失效模式：至少3項（例如：連續虧損、滑點擴大、點差異常）

{{baseInfo}}`
      },
      code: {
        title: '程式碼生成代理',
        prompt: `您是一位AntTrader Python策略程式碼工程師。請生成可執行的AntTrader Python策略程式碼，需滿足：
- 通過驗證檢查（無import、無dunder、沙箱約束）
- 使用平台API如on_tick / on_kline（無自訂網路/檔案存取）
- run()必須恰好接收一個參數：context（必須命名為context；不接受run(ctx)、run(context, data)等）
- run(context)返回dict，至少包含：signal(buy/sell/hold)、symbol、confidence(0~1)、risk_level(low/medium/high)、reason
- 從context["params"]讀取參數（來自排程注入）；若缺失則使用預設值
- 使用上游訊號設計與風險控制（未提供時給出合理預設值）
- 輸出完整程式碼，包裹在\`\`\`python\`\`\`中
- 嚴格輸出：僅一個\`\`\`python\`\`\`區塊，無解釋文字
- 程式碼區塊必須是純Python：無Markdown符號、無中文標點、無巢狀程式碼圍欄

[強制入口模板（請勿更改函數名稱/參數數量/參數名稱）]
\`\`\`python
def run(context):
    params = context.get("params") or {}
    symbol = context.get("symbol") or params.get("symbol") or ""
    # TODO: 在此實現訊號/風險邏輯
    return {
        "signal": "hold",
        "symbol": symbol,
        "confidence": 0.5,
        "risk_level": "low",
        "reason": "",
    }
\`\`\`

{{baseInfo}}

[注意：上游分析結論——應用至程式碼（缺失時提供合理預設值）]`
      }
    },
    consensus: {
      title: '共識與討論',
      actions: {
        refresh: '重新整理'
      },
      fields: {
        account: '帳戶',
        symbol: '品種',
        timeframe: '時間框架'
      },
      panel: {
        title: '目標評分',
        decision: '決策',
        overallScore: '總體',
        technicalScore: '技術'
      },
      signals: {
        rsi: {
          value: 'RSI',
          flag: '訊號'
        },
        macd: {
          value: 'MACD',
          signalLine: '訊號線',
          hist: '柱狀圖',
          flag: '訊號',
          trend: '形態'
        },
        ma: {
          trend: 'MA趨勢'
        }
      }
    },
    conversation: {
      defaultTitle: '新對話'
    },
    chatBox: {
      emptyDescription: '開始與AI助手對話',
      thinking: '思考中...',
      truncated: '內容過長，已截斷',
      expandAll: '全部展開',
      collapse: '收起'
    },
    reports: {
      tradeAnalysis: {
        title: 'AI交易分析報告',
        riskAssessmentPrefix: '風險評估：'
      }
    },
    signalCard: {
      status: {
        pending: '待處理',
        confirmed: '已確認',
        executed: '已執行',
        cancelled: '已取消'
      },
      labels: {
        price: '價格',
        volume: '手數',
        confidence: '信心度',
        stopLoss: '止損',
        takeProfit: '止盈',
        analysisReason: '分析理由'
      },
      actions: {
        confirm: '確認',
        cancel: '取消',
        executeTrade: '執行交易'
      },
      confirmCancel: {
        title: '確定要取消此訊號嗎？'
      },
      confirmExecute: {
        title: '確定要執行此交易訊號嗎？',
        description: '將立即下單'
      }
    },
    assistant: {
      messages: {
        noCodeBlockFound: '未找到程式碼區塊（\`\`\`...\`\`\`）'
      }
    },
    strategyCard: {
      status: {
        active: '啟用中',
        inactive: '未啟用',
        paused: '已暫停'
      },
      actionType: {
        buy: '買入',
        sell: '賣出',
        closeLong: '平多',
        closeShort: '平空',
        alert: '警報'
      },
      labels: {
        triggeredCount: '已觸發{{count}}次',
        lastTriggeredAt: '上次觸發：{{time}}'
      },
      sections: {
        conditions: '觸發條件',
        actions: '操作'
      },
      tooltips: {
        createdAt: '建立時間',
        lastTriggeredAt: '上次觸發'
      },
      actions: {
        start: '啟動',
        stop: '停止'
      },
      confirmDelete: {
        title: '確定要刪除此策略嗎？',
        description: '刪除後無法恢復'
      }
    },
    requireConfig: {
      title: '尚未配置LLM',
      description: '請先前往設定頁面配置AI供應商、模型及API金鑰，然後再使用策略精靈或聊天功能。',
      actions: {
        goSettings: '前往設定'
      }
    },
    riskEval: {
      failed: '風險評估失敗'
    },
    workflowRuns: {
      title: 'AI工作流程',
      defaultTitle: 'AI工作流程',
      hints: {
        selectToViewDetail: '從左側選擇一個執行項目以檢視詳情'
      },
      messages: {
        loadListFailed: '載入執行列表失敗',
        loadDetailFailed: '載入詳情失敗'
      }
    },
    backtestScoreCard: {
      title: '回測評分卡',
      stateLabel: '狀態',
      status: {
        succeeded: '成功',
        running: '執行中',
        pending: '佇列中',
        failed: '失敗',
        cancelRequested: '取消中',
        canceled: '已取消'
      },
      recommendation: {
        loading: '風險評估進行中，請等待完成後再上線。',
        recommended: '建議上線：風險可控，指標健康。',
        cautious: '謹慎上線：建議先小資金/手動驗證一段時間。',
        notRecommended: '不建議直接上線：高風險或不可靠，請先優化後再嘗試。'
      },
      backendRiskScore: {
        title: '後端風險評分',
        loading: '計算中...',
        unknown: '未知',
        reliable: '可靠',
        unreliable: '不可靠',
        reasons: '原因',
        warnings: '警告',
        empty: '無（請先儲存模板，回測完成後會自動計算）'
      },
      score: {
        empty: '暫無評分（等待回測或無指標資料）',
        title: '總體評分（啟發式）'
      },
      level: {
        excellent: '優秀',
        good: '良好',
        fair: '一般',
        poor: '較差'
      },
      metrics: {
        totalReturn: '總收益率',
        annualReturn: '年化收益率',
        maxDrawdown: '最大回撤',
        sharpe: '夏普比率',
        winRate: '勝率',
        totalTrades: '總交易次數',
        equityPoints: '權益點數'
      },
      chart: {
        title: '權益曲線'
      }
    },
    gate: {
      title: 'AI閘道進度',
      pipelineDesc: '6階段閘道管線：合規性 → 前瞻偏差 → 前進式驗證 → 縮減夏普 → 紙交易 → 相關性',
      labels: {
        compliance: '合規性',
        lookahead: '前瞻偏差',
        walkforward: '前進式驗證',
        deflated_sharpe: '縮減夏普比率',
        paper: '紙交易',
        correlation: '相關性'
      },
      descriptions: {
        compliance: 'DSL表達式非空驗證',
        lookahead: '未來函數引用掃描（close[t+N]、ref負偏移）',
        walkforward: '淨化前進式交叉驗證',
        deflated_sharpe: 'Lopez de Prado縮減夏普比率',
        paper: '≥14天紙交易驗證',
        correlation: '與現有策略的訊號相關性檢查'
      },
      status: {
        evaluating: '評估中...'
      },
      strategyParams: '策略參數',
      dslExpression: 'DSL表達式',
      dailyReturns: '每日收益率（逗號或換行分隔）',
      numAttempts: '策略嘗試次數',
      paperMetrics: '紙交易指標',
      paperDays: '紙交易天數',
      paperNetPnL: '紙交易淨損益',
      paperNetReturn: '紙交易淨收益率',
      paperTradeCount: '紙交易次數',
      backtestNetReturn: '回測淨收益率',
      backtestGrossReturn: '回測總收益率',
      runPipeline: '執行閘道管線',
      retry: '重試',
      gateProgress: '閘道評估進度',
      pipelineResult: '管線結果',
      allPassed: '全部6個閘道已通過——策略符合推廣至上線評估條件',
      failed: '失敗：{{gate}}',
      details: '詳情'
    }
  }
} as const;

export default aiCore;
