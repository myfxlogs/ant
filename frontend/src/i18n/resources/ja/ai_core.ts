// Auto-generated from proto/ant/v1/i18n/ai_core_ja.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const AiCore = {
  "ai": {
    "agentPrompts": {
      "code": {
        "title": "コード生成エージェント"
      },
      "risk": {
        "title": "リスク管理と執行制約"
      },
      "signals": {
        "title": "シグナルとインジケーター設計"
      },
      "style": {
        "title": "相場環境・スタイル推奨"
      }
    },
    "assistant": {
      "messages": {
        "noCodeBlockFound": "No code block found (\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\`\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\`\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\`...\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\`\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\`\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\`)"
      }
    },
    "backtestScoreCard": {
      "backendRiskScore": {
        "empty": "无（先保存模板，回测完成后自动计算）",
        "loading": "計算中...",
        "reasons": "理由",
        "reliable": "信頼できる",
        "title": "ストラテジーリスクスコア",
        "unknown": "不明",
        "unreliable": "信頼できない",
        "warnings": "警告"
      },
      "chart": {
        "title": "净值曲线"
      },
      "level": {
        "excellent": "優秀",
        "fair": "普通",
        "good": "良好",
        "poor": "差"
      },
      "metrics": {
        "annualReturn": "年換算収益率",
        "equityPoints": "净值点数",
        "maxDrawdown": "最大ドローダウン",
        "sharpe": "シャープレシオ",
        "totalReturn": "総収益率",
        "totalTrades": "総取引数",
        "winRate": "勝率"
      },
      "recommendation": {
        "cautious": "本番注意：まずは少額または手動確認でしばらく運用してください。",
        "loading": "リスク評価中です。完了するまでお待ちください。",
        "notRecommended": "Not recommended for direct live: high risk or unreliable, optimize before trying.",
        "recommended": "本番推奨：リスク管理可能、指標良好。"
      },
      "score": {
        "empty": "スコア未評価（バックテスト待ちまたは指標なし）",
        "title": "综合评分（启发式）"
      },
      "stateLabel": "状態",
      "status": {
        "cancelRequested": "キャンセル中",
        "canceled": "已取消",
        "failed": "失敗",
        "pending": "キュー待ち",
        "running": "実行中",
        "succeeded": "成功"
      },
      "title": "バックテストスコアカード"
    },
    "chatBox": {
      "collapse": "收起",
      "emptyDescription": "开始与AI助手对话",
      "expandAll": "展开全部",
      "thinking": "思考中...",
      "truncated": "内容过长，已截断"
    },
    "client": {
      "errors": {
        "contentBlocked": "内容がプロバイダーのセーフティポリシーによりブロックされました。質問の言い回しを調整して再度お試しください。",
        "contextTooLong": "リクエストがモデルの最大コンテキスト長を超えています。会話履歴・入力を短縮するか、より大きなコンテキストウィンドウのモデルを選んでください。",
        "edgeGatewayTimeout": "フロントの前段ゲートウェイがタイムアウトしました（Cloudflare の HTTP 524 など）。長時間の「コード生成」で起きやすいです。ディベートのコード画面で「コード生成を再試行」するか、一つ戻ってから再度進めてください。改善しない場合はプロキシ／オリジンのタイムアウト延長が必要です。",
        "forbidden": "プロバイダーから 403 アクセス拒否：Key の権限、IP ホワイトリスト、アカウント状態を確認してください。",
        "gatewayForbidden403": "ゲートウェイアクセス拒否（403）。",
        "gatewayRateLimited429": "网关速率受限 (429)。",
        "gatewayTimeoutOrUnreachable": "ゲートウェイ接続タイムアウト/到達不可。",
        "gatewayUnauthorized401": "ゲートウェイ未認証（401）。",
        "insufficientBalance": "プロバイダーからの返答：残高不足または支払い延滞。プロバイダーコンソールで残高・請求を確認し、再度お試しください。",
        "invalidModelId": "モデル利用不可{{model}}：存在しない、廃止された、または権限外の可能性があります。ドロップダウンから選ぶか、プロバイダーコンソールから正しい モデル ID をコピーしてください。",
        "networkUnreachable": "モデルゲートウェイへの接続がタイムアウトまたは到達不可。AI 設定の 基底 URL がアクセス可能か、ネットワークが正常かを確認し、またはしばらくしてから再度お試しください。",
        "providerInternalError": "プロバイダーサービスが一時的に利用不可（5xx）。しばらくしてから再度お試しください、または別のプロバイダーに切り替えてください。",
        "rateLimited": "プロバイダーによるレート制限（リクエストが頻繁すぎます）。しばらくしてから再度お試しください。",
        "regionNotSupported": "現在の地域・国はこのプロバイダーではサポートされていません。地域を変更するか、別のプロバイダーを選択してください。",
        "requestFailed": "リクエストに失敗しました。もう一度お試しください。",
        "unauthorized": "プロバイダーから 401 未認証：API キー が正しいか、モデル権限があるかを確認してください。"
      }
    },
    "consensus": {
      "actions": {
        "refresh": "刷新"
      },
      "fields": {
        "account": "口座",
        "symbol": "銘柄",
        "timeframe": "周期"
      },
      "panel": {
        "decision": "判定",
        "overallScore": "総合",
        "technicalScore": "技术面",
        "title": "目標スコア"
      },
      "signals": {
        "ma": {
          "trend": "均线趋势"
        },
        "macd": {
          "flag": "シグナル",
          "hist": "ヒストグラム",
          "signalLine": "シグナル線",
          "trend": "形态",
          "value": "MACD"
        },
        "rsi": {
          "flag": "シグナル",
          "value": "RSI"
        }
      },
      "title": "コンセンサスと議論"
    },
    "conversation": {
      "defaultTitle": "新对话"
    },
    "gate": {
      "allPassed": "全6ゲートを通過 — ストラテジーは本番昇格評価の対象です",
      "backtestGrossReturn": "バックテスト総収益率",
      "backtestNetReturn": "バックテスト純収益率",
      "dailyReturns": "日次リターン（カンマまたは改行区切り）",
      "descriptions": {
        "compliance": "DSL式の空チェック",
        "correlation": "与现有策略的信号相关性检查",
        "deflated_sharpe": "Lopez de Prado 収縮シャープレシオ",
        "lookahead": "未来関数参照のスキャン（close[t+N]、ref負のオフセット）",
        "paper": "14日以上のペーパートレーディング検証",
        "walkforward": "パージド・ウォークフォワード交差検証"
      },
      "details": "詳細",
      "dslExpression": "DSL式",
      "evaluating": "评估中...",
      "fail": "失败",
      "failed": "不合格：{{gate}}",
      "gateProgress": "ゲート評価進捗",
      "labels": {
        "compliance": "コンプライアンス",
        "correlation": "相关性",
        "deflated_sharpe": "収縮シャープレシオ",
        "lookahead": "先読みバイアス",
        "paper": "ペーパートレーディング",
        "walkforward": "ウォークフォワード"
      },
      "noData": "データがありません",
      "numAttempts": "ストラテジー試行回数",
      "paperDays": "ペーパー日数",
      "paperMetrics": "ペーパートレーディング指標",
      "paperNetPnL": "ペーパー純損益",
      "paperNetReturn": "ペーパー純収益率",
      "paperTradeCount": "ペーパー取引回数",
      "pass": "通过",
      "pipelineDesc": "6段階ゲートパイプライン：コンプライアンス → 先読みバイアス → ウォークフォワード → 収縮シャープ → ペーパー → 相関",
      "pipelineResult": "パイプライン結果",
      "retry": "再試行",
      "runHint": "请先运行回测，然后点击\"运行质量门\"评估策略质量。",
      "runPipeline": "ゲートパイプライン実行",
      "selectRun": "バックテスト実行を選択...",
      "skipped": "已跳过",
      "status": {
        "evaluating": "评估中..."
      },
      "strategyParams": "ストラテジーパラメーター",
      "title": "AIゲート進捗",
      "unknown": "不明"
    },
    "gateway": {
      "balance": "ウォレット残高",
      "modelPlaceholder": "AI モデルを選択",
      "monthlyCost": "今月の費用",
      "monthlyTokens": "今月のトークン",
      "noModels": "利用可能なモデルがありません",
      "selectModel": "モデル選択",
      "title": "AI ゲートウェイ",
      "usageByFeature": "機能別使用量",
      "useGateway": "AI ゲートウェイ",
      "useGatewayDesc": "ウォレット課金 · トークン単位",
      "useOwnKey": "自分の API Key",
      "useOwnKeyDesc": "直接課金 · 自己管理",
      "useOwnKeyHint": "自分の API Key を使用してプロバイダーに直接支払います。下のプロバイダーカードを選択して設定してください。"
    },
    "reports": {
      "tradeAnalysis": {
        "riskAssessmentPrefix": "风险评估:",
        "title": "AI取引分析レポート"
      }
    },
    "requireConfig": {
      "actions": {
        "goSettings": "前往设置"
      },
      "description": "先に設定画面でAIプロバイダー、モデル、APIキーを設定してください。その後、ストラテジーウィザードまたはチャットをご利用いただけます。",
      "title": "LLMがまだ設定されていません"
    },
    "riskEval": {
      "failed": "风险评估失败"
    },
    "signalCard": {
      "actions": {
        "cancel": "キャンセル",
        "confirm": "確認",
        "executeTrade": "执行交易"
      },
      "confirmCancel": {
        "title": "确定要取消此信号？"
      },
      "confirmExecute": {
        "description": "将立即下单",
        "title": "この取引シグナルを執行してもよろしいですか？"
      },
      "labels": {
        "analysisReason": "分析理由",
        "confidence": "確信度",
        "price": "価格",
        "stopLoss": "ストップロス",
        "takeProfit": "テイクプロフィット",
        "volume": "ロット"
      },
      "status": {
        "cancelled": "已取消",
        "confirmed": "確認済み",
        "executed": "執行済み",
        "pending": "保留中"
      }
    },
    "strategyCard": {
      "actionType": {
        "alert": "警报",
        "buy": "買い",
        "closeLong": "買い決済",
        "closeShort": "売り決済",
        "sell": "売り"
      },
      "actions": {
        "start": "開始",
        "stop": "停止"
      },
      "confirmDelete": {
        "description": "删除后无法恢复",
        "title": "このストラテジーを削除してもよろしいですか？"
      },
      "labels": {
        "lastTriggeredAt": "最近触发: {{time}}",
        "triggeredCount": "{{count}}回トリガー"
      },
      "sections": {
        "actions": "操作",
        "conditions": "トリガー条件"
      },
      "status": {
        "active": "稼働中",
        "inactive": "停止中",
        "paused": "已暂停"
      },
      "tooltips": {
        "createdAt": "作成日時",
        "lastTriggeredAt": "最近触发"
      }
    },
    "systemAI": {
      "cardState": {
        "enabled": "已启用",
        "noKey": "未配置",
        "noModel": "待选模型",
        "readyDisabled": "就绪 · 已禁用"
      },
      "cardTags": {
        "current": "当前",
        "enabledButUnavailable": "已启用但不可用",
        "hasKey": "已配密钥",
        "noKey": "未配密钥",
        "noModels": "未配置可用模型"
      },
      "customProvider": {
        "deleted": "自定义提供商已删除",
        "fillNameFirst": "请先填写名称",
        "nameHint": "用于识别此提供商的唯一名称",
        "nameLabel": "提供商名称",
        "namePlaceholder": "我的自定义提供商",
        "nameRequired": "服务商名称不能为空"
      },
      "emptyConfigs": "暂无 AI Provider 配置（系统启动时会自动创建默认 Provider）",
      "fields": {
        "apiKeyHint": "输入后将自动加密保存，无需手动提交",
        "apiKeyPastePlaceholder": "粘贴 API Key，将自动预保存",
        "autoFetching": "自动拉取中",
        "baseUrlCustomHint": "输入 OpenAI 兼容端点，例如 https://model.example.com/v1",
        "baseUrlCustomPlaceholder": "例如: https://model.example.com/v1",
        "baseUrlReadonlyHint": "官方地址由系统维护，不可修改",
        "baseUrlReadonlyPlaceholder": "官方地址（只读）",
        "enabledHint": "关闭后该厂商不参与系统路由",
        "httpWarning": "当前为 HTTP，生产环境建议使用 HTTPS",
        "maxTokensHint": "单次响应最大 token 数",
        "primaryFor": "主要用途（Primary For）",
        "primaryForHint": "用于内部分发：对话/嵌入/摘要/推理",
        "temperatureHint": "越高越发散，越低越稳定",
        "timeoutHint": "单次请求最长等待时间"
      },
      "messages": {
        "autoDiscoveredModels": "已自动发现 {{count}} 个模型（仅作选择建议）",
        "autoValidatedModels": "已自动验证：发现 {{count}} 个模型",
        "configSaveFailed": "配置保存失败",
        "configSaved": "配置已保存",
        "deleteSecretFailed": "删除密钥失败",
        "loadConfigFailed": "加载配置失败",
        "secretAutoSaveFailed": "密钥自动保存失败",
        "secretDeletedConfigReset": "密钥已删除，厂商配置已恢复默认初始化",
        "secretSavedAutoDiscover": "密钥已保存，正在自动发现模型...",
        "toggleEnabledFailed": "更新启用状态失败",
        "validationFailedNeedApiKey": "検証失敗：このプロバイダーは通常APIキーが必要です。キーを入力・保存してから再試行してください。",
        "validationPassedModels": "验证通过：发现 {{count}} 个模型"
      },
      "pageSubtitle": "配置 AI 大脑 — 选择模型厂商、管理 API 密钥与可用模型，并指定全站兜底使用的「默认主模型」。",
      "pageTitle": "AI 助手设置",
      "section1": {
        "subtitle": "Cards show each provider's configuration and readiness; click to select",
        "title": "选择模型厂商"
      },
      "status": {
        "checkUrl": "请检查 Base URL",
        "checkUrlDesc": "API Key 已就绪，但地址似乎无效",
        "configReady": "配置已就绪",
        "configReadyDesc": "添加可用模型后系统将自动完成连通性检测",
        "connectionFailed": "连接错误，请检查上方提示",
        "error": "存在异常",
        "needKey": "请完成密钥配置",
        "needKeyDesc": "填写 API Key 后将自动发现模型列表",
        "noProvider": "尚未选择厂商",
        "noProviderDesc": "请从下方卡片挑选一个模型厂商开始配置",
        "notEnabled": "连接正常，尚未启用",
        "notEnabledDesc": "打开「启用」开关即可投入使用",
        "ready": "运行就绪",
        "readyDesc": "已启用并连接正常"
      },
      "statusBar": {
        "checking": "连通性检测中…",
        "connected": "已连接",
        "disabled": "未启用",
        "enabled": "已启用",
        "keyReady": "密钥就绪"
      },
      "taglines": {
        "anthropic": "Claude 系列",
        "deepseek": "深度求索 · 高性价比",
        "moonshot": "Kimi · 长上下文",
        "openai": "GPT 系列 · 官方",
        "openai_compatible": "任意兼容端点",
        "qwen": "阿里云 · 中文优化",
        "zhipu": "清华系 · 通用"
      }
    },
    "tabs": {
      "agentSettings": "专家设置",
      "gate": "AI 质量门",
      "settings": "设置"
    },
    "workflowRuns": {
      "defaultTitle": "AIワークフロー",
      "hints": {
        "selectToViewDetail": "从左侧选择运行记录查看详情"
      },
      "messages": {
        "loadDetailFailed": "加载详情失败",
        "loadListFailed": "実行一覧の読み込みに失敗しました"
      },
      "title": "AIワークフロー"
    }
  }
} as const;
export default AiCore;
