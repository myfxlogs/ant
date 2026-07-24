// Auto-generated from proto/ant/v1/i18n/ai_wizard_zh-tw.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const AiWizard = {
  "ai": {
    "wizard": {
      "generate": {
        "modals": {
          "final": {
            "title": "程式碼已生成，建議點選「驗證程式碼」確認透過校驗"
          }
        },
        "status": {
          "running": {
            "code": "程式碼生成中",
            "generic": "{{title}}中",
            "risk": "風控與執行約束中",
            "signals": "訊號與指標設計中",
            "style": "市場狀態/風格推薦中"
          },
          "done": "完成",
          "error": "失敗",
          "idle": "等待中",
          "inProgress": "進行中"
        },
        "actions": {
          "abort": "中止",
          "goValidate": "去校驗",
          "hide": "隱藏",
          "regenerateSummary": "重新生成總結",
          "rerun": "重新生成",
          "runAgents": "多個專家分析 + 程式碼生成"
        },
        "cards": {
          "resultsTitle": "多位專家\\\\\\\\\\\\\\\\"
        },
        "hints": {
          "afterGenerated": "生成完成後進入下一步進行驗證/回測/上線"
        },
        "labels": {
          "elapsed": "耗時"
        },
        "sections": {
          "output": "模型返回（Output）",
          "prompt": "傳送給模型（Prompt）",
          "spec": "規格（Spec）"
        }
      },
      "publishBacktest": {
        "modals": {
          "score": {
            "title": "評分確認"
          },
          "status": {
            "title": "回測進行中"
          }
        },
        "actions": {
          "close": "關閉",
          "confirm": "確認",
          "inProgress": "進行中",
          "retry": "重試",
          "runInBackground": "後臺執行",
          "startBacktest": "開始回測",
          "succeeded": "成功"
        },
        "cards": {
          "backtestTitle": "回測",
          "scoreCardTitle": "評分卡"
        },
        "labels": {
          "confirmed": "已確認",
          "elapsed": "耗時",
          "overallScore": "綜合評分",
          "scoringProgress": "評分進度",
          "status": "狀態"
        },
        "draftName": "回測 {{datetime}} {{symbol}} {{timeframe}}",
        "draftNameShort": "回測 {{symbol}} {{timeframe}}"
      },
      "setup": {
        "modals": {
          "deleteDataset": {
            "content": "確定刪除目前選中的凍結資料集嗎？",
            "ok": "刪除",
            "title": "刪除資料集"
          }
        },
        "actions": {
          "deleteCurrentDataset": "刪除目前資料集",
          "freezeFromCurrentRange": "從目前範圍凍結",
          "refreshDataset": "重新整理"
        },
        "cards": {
          "constraintsAndGoalTitle": "約束與目標",
          "hardConstraintsTitle": "硬約束",
          "hintsTitle": "提示",
          "tradeAndDataTitle": "交易與資料"
        },
        "dataModes": {
          "dataset": "凍結資料集",
          "klineRange": "歷史K線範圍"
        },
        "hints": {
          "nextWillGenerateCode": "下一步將開始生成策略程式碼。",
          "tradeDataNextStep": "填寫完成後點選「下一步」，進入約束與目標設定。"
        },
        "labels": {
          "account": "帳號",
          "backtestRange": "回測範圍",
          "dataset": "凍結資料集",
          "historicalData": "歷史資料",
          "intent": "策略目標/想法",
          "macroEvents": "宏觀事件",
          "macroModule": "宏觀模組",
          "maxDrawdownPct": "最大回撤(%)",
          "maxTradesPerDay": "日內最多交易次數",
          "riskPerTradePct": "單筆風險(%)",
          "symbol": "品種",
          "timeframe": "週期"
        },
        "macro": {
          "off": "關閉",
          "on": "開"
        },
        "messages": {
          "datasetDeleted": "資料集已刪除"
        },
        "placeholders": {
          "intentExample": "範例：突破趨勢跟隨；避開高波動；偏好更高勝率...",
          "macroExample": "示例：\\\\\\\\\\\\\\\\n2024-01-03 21:15 FOMC 會議紀要\\\\\\\\\\\\\\\\n2024-01-05 20:30 非農就業資料",
          "selectAccount": "選擇帳號",
          "selectFrozenDataset": "選擇凍結資料集",
          "selectSymbol": "選擇品種",
          "selectTimeframe": "選擇週期"
        },
        "validations": {
          "enterIntent": "請輸入策略目標/想法",
          "selectAccount": "請選擇帳號",
          "selectDataset": "請選擇資料集",
          "selectSymbol": "請選擇品種",
          "selectTimeframe": "請選擇週期"
        }
      },
      "prompts": {
        "base": {
          "account": "帳號: {{accountId}}",
          "constraints": "約束: 最大回撤={{maxDrawdownPct}}% 單筆風險={{riskPerTradePct}}% 日內最多交易={{maxTradesPerDay}} 次",
          "data": "資料: {{dataSpec}}",
          "empty": "(空)",
          "macroDisabled": "宏觀事件: 不使用",
          "macroEnabled": "宏觀事件（使用者提供）：\\\\\\\\\\\\\\\\n{{text}}",
          "params": "引數（定義+當前值；執行時注入 context[\"params\"]）：\\\\\\\\\\\\\\\\n{{params}}",
          "symbol": "品種: {{symbol}}",
          "timeframe": "週期: {{timeframe}}",
          "userIntent": "使用者策略目標（自然語言）：\\\\\\\\\\\\\\\\n{{intent}}"
        },
        "dataSpec": {
          "dataset": "使用凍結資料集 datasetId={{datasetId}}",
          "klineRange": "使用歷史K線範圍 from={{from}} to={{to}}"
        },
        "summary": {
          "codeTitle": "程式碼如下：",
          "intro": "你是量化策略解釋助手。請用簡潔中文（要點形式，最多 12 行）解釋這段 AlphaForge Python 策略程式碼的核心思路。",
          "mustInclude1": "1) 策略型別/正規化",
          "mustInclude2": "2) 主要入場條件（2~4 條要點）",
          "mustInclude3": "3) 主要出場/止損止盈/風控約束（2~4 條要點）",
          "mustInclude4": "4) 適用/不適用場景各 1 條",
          "mustIncludeTitle": "必須包含：",
          "userIntent": "使用者期望（自然語言）：\\\\\\\\\\\\\\\\n{{intent}}"
        },
        "upstream": {
          "risk": "【風控結論】\\\\\\\\\\\\\\\\n{{text}}",
          "sectionTitle": "【上游 Agent 結論（原樣提供）】",
          "signals": "【訊號設計結論】\\\\\\\\\\\\\\\\n{{text}}",
          "style": "【市場狀態/風格結論】\\\\\\\\\\\\\\\\n{{text}}"
        }
      },
      "publish": {
        "actions": {
          "publishTemplate": "發布樣板",
          "startBacktest": "回測（非同步任務）",
          "validateCode": "驗證程式碼"
        },
        "cards": {
          "codeTitle": "1) 策略程式碼（可編輯）",
          "launchTitle": "3) 上線排程",
          "scoreCardTitle": "2) 回測評分卡"
        },
        "messages": {
          "validateFailed": "validate 未透過",
          "validateOk": "validate 透過"
        },
        "placeholders": {
          "codeEditable": "這裡會自動填入 AI 生成的程式碼，你也可以手動修改。"
        }
      },
      "strategyParams": {
        "actions": {
          "addParam": "新增引數",
          "delete": "刪除",
          "exportJson": "匯出 JSON",
          "importJson": "匯入 JSON"
        },
        "hints": {
          "intro": "這些引數會：",
          "line1": "1) 儲存到樣板 parameters",
          "line2": "2) 建立排程時寫入 schedule.parameters",
          "line3Prefix": "3) 執行時系統會把引數注入到 Python 策略的"
        },
        "labels": {
          "default": "預設值",
          "description": "說明",
          "label": "標籤",
          "max": "最大值",
          "min": "最小值",
          "name": "名稱",
          "options": "options（select 可用，逗號分隔）",
          "step": "步長",
          "type": "型別",
          "value": "value（排程目前值）"
        },
        "messages": {
          "copied": "已複製",
          "copyFailed": "複製失敗",
          "importFormatInvalid": "匯入格式錯誤",
          "importMissingName": "匯入失敗：存在缺少 name 的引數",
          "imported": "已匯入 {{count}} 個引數",
          "jsonParseFailed": "JSON 解析失敗"
        },
        "modals": {
          "copyAndClose": "複製並關閉",
          "exportTitle": "匯出引數 JSON",
          "importOk": "匯入",
          "importTitle": "匯入引數 JSON"
        },
        "placeholders": {
          "defaultExample": "例如：10",
          "description": "說明",
          "importJson": "貼上引數 JSON",
          "label": "展示名",
          "nameExample": "例如：fast",
          "optionsExample": "例如：low,medium,high",
          "value": "留空則使用 default"
        },
        "types": {
          "bool": "布林",
          "number": "數字",
          "select": "選擇",
          "string": "字串"
        },
        "validations": {
          "nameRequired": "name 必填",
          "typeRequired": "type 必填"
        },
        "empty": "暫無引數。你可以新增如 fast/slow/risk_per_trade 等引數。",
        "paramCardTitle": "引數 #{{index}}",
        "title": "策略引數（可選）"
      },
      "actions": {
        "cancel": "取消",
        "next": "下一步",
        "prev": "上一步"
      },
      "agents": {
        "codeTitle": "程式碼生成",
        "riskTitle": "風控與執行約束",
        "signalsTitle": "訊號與指標設計",
        "styleTitle": "市場狀態/風格推薦"
      },
      "messages": {
        "agentFailed": "{{title}} 失敗",
        "aiRequestTimeout": "AI 請求逾時（>{{seconds}}s）",
        "backtestCreated": "回測任務已建立",
        "backtestNotDoneWait": "回測尚未完成，請等待評分卡狀態變為成功/失敗/已取消後再繼續",
        "chatAborted": "已中止與模型對話",
        "codeInvalidFixAndContinue": "程式碼驗證未透過，請修復後再繼續",
        "confirmScoreFirst": "請先在評分彈窗中確認評分結果",
        "createBacktestFailed": "建立回測失敗",
        "createDraftFailed": "建立草稿失敗",
        "createScheduleFailed": "建立排程失敗",
        "datasetFrozenCreated": "已凍結建立 dataset",
        "draftNotCreated": "草稿未建立",
        "draftSaved": "草稿已儲存",
        "fillRequired": "請先補全必填項",
        "fillRequiredWithFields": "請先補全必填項：{{fields}}",
        "freezeDatasetFailed": "凍結 dataset 失敗",
        "generateCodeFirst": "請先生成策略程式碼",
        "inputIntentFirst": "請先輸入策略目標/想法",
        "loadAccountsFailed": "載入帳號失敗",
        "loadDatasetFailed": "載入 dataset 失敗",
        "loadSymbolsFailed": "載入品種失敗",
        "modelReturnedEmpty": "模型返回為空",
        "noCodeToBacktest": "暫無程式碼可回測",
        "noCodeToValidate": "暫無程式碼可驗證",
        "noPythonCodeBlock": "程式碼 Agent 未輸出 ```python 程式碼塊```，請在結果中檢查",
        "publishFailed": "發布失敗",
        "publishTemplateFirst": "請先發布樣板",
        "publishedNoId": "已發布，但未拿到返回 id",
        "saveFailed": "儲存失敗",
        "scheduleAlreadyExists": "該帳號下已存在相同策略排程，請勿重複建立。",
        "scheduleCreated": "排程已建立",
        "scheduleCreatedAndEnabled": "排程已建立並啟用",
        "startBacktestFirst": "請先點選「回測（非同步任務）」啟動回測",
        "templatePublished": "樣板已發布",
        "userAborted": "使用者已中止",
        "validateCodeFirst": "請先點選「驗證程式碼」",
        "validateError": "驗證失敗",
        "validateFailed": "驗證未透過",
        "validateOk": "驗證透過",
        "watchBacktestRunFailed": "watchBacktestRun 失敗"
      },
      "schedule": {
        "defaultName": "AI 排程 {{symbol}} {{timeframe}}"
      },
      "steps": {
        "analyze": "分析",
        "backtest": "回測",
        "compliance": "合規",
        "generate": "生成策略",
        "plan": "規劃",
        "publishBacktest": "回測上線-回測",
        "publishCode": "回測上線-程式碼",
        "publishLaunch": "回測上線-上線",
        "setup": "基礎資訊"
      },
      "template": {
        "defaultDescription": "AI 嚮導生成",
        "defaultName": "AI 策略 {{title}}"
      },
      "currentModel": "目前模型：{{model}}",
      "subtitle": "每步一個頁面，可前進/後退",
      "title": "AI 策略嚮導"
    }
  }
} as const;
export default AiWizard;
