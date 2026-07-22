const Marketplace = {
  "marketplace": {
    "title": "マーケットプレイス",
    "subtitle": "トレーディング戦略を見つけて購読する",
    "tabs": { "marketplace": "マーケット", "purchases": "購入済み", "author": "作者センター", "bundles": "バンドル", "optimization": "AI最適化", "fees": "手数料ティア" },
    "searchPlaceholder": "戦略を検索...",
    "empty": "戦略が見つかりません",
    "filter": { "all": "すべて", "free": "無料", "paid": "有料" },
    "sort": { "score": "スコア", "newest": "新着", "popular": "人気", "rating": "評価", "priceAsc": "価格：低→高", "priceDesc": "価格：高→低" },
    "card": { "free": "無料", "owned": "所有済み", "winRate": "勝率", "pnl": "総損益", "users": "購読者", "rent": "¥{{amount}}/月", "buy": "¥{{amount}}" },
    "messages": { "loginFirst": "ログインしてください", "subscribed": "購読しました", "subscribeFailed": "購読に失敗しました", "published": "戦略を公開しました", "publishFailed": "公開に失敗しました", "rated": "評価しました！" },
    "detail": { "author": "作者", "price": "価格", "assetClass": "資産クラス", "riskLevel": "リスクレベル", "subscribers": "購読者", "avgRating": "評価", "description": "説明", "tags": "タグ", "yourRating": "あなたの評価", "comments": "コメント", "noComments": "コメントなし", "commentPlaceholder": "コメントを書く...", "getFree": "無料で取得", "buyNow": "今すぐ購入", "owned": "所有済み", "freePrice": "無料", "rentPrice": "¥{{amount}} / 月", "buyPrice": "¥{{amount}} 買い切り", "runBacktest": "バックテスト実行" },
    "purchases": { "empty": "購入履歴なし", "strategy": "戦略", "date": "購入日", "status": "ステータス", "actions": "操作", "runBacktest": "バックテスト実行" },
    "author": { "empty": "公開済み戦略なし", "noPublished": "公開済み戦略なし", "published": "公開済み", "subscribers": "購読者", "avgRating": "平均評価", "myStrategies": "公開した戦略", "publishNew": "新規公開", "goToLibrary": "戦略ライブラリへ" },
    "payment": { "purchaseSuccess": "購入完了！戦略がライブラリに追加されました。", "purchaseFailed": "購入に失敗しました。再試行してください。", "insufficientBalance": "残高不足", "alreadyPurchased": "すでに所有しています。", "title": "購入確認", "balance": "ウォレット残高", "confirm": "支払い確定", "cancel": "キャンセル" },
    "publish": {
      "title": "マーケットに公開",
      "titleLabel": "タイトル",
      "titlePlaceholder": "例：ゴールデンクロス戦略",
      "descriptionLabel": "説明",
      "descriptionPlaceholder": "戦略ロジック、エントリー/イグジットルールを記述...",
      "assetClass": { "label": "資産クラス", "forex": "外国為替", "crypto": "暗号通貨", "commodity": "コモディティ", "index": "指数", "stock": "株式" },
      "riskLevel": { "label": "リスクレベル", "low": "低", "medium": "中", "high": "高" },
      "priceModel": { "label": "価格設定", "free": "無料", "subscription": "月額購読", "once": "買い切り" },
      "priceAmount": "金額",
      "tags": "タグ",
      "tagsPlaceholder": "入力してEnterで追加",
      "codeSnippet": "戦略プレビュー（公開）",
      "codeSnippetPlaceholder": "任意：戦略の概要やコードスニペット（全員に表示）",
      "includeBacktestSnapshot": "最新のバックテスト結果を含める"
    },
    "backtest": { "title": "戦略バックテスト", "protected": "戦略コードは保護されています。バックテストはサーバー上で実行されます。", "run": "バックテスト実行", "idle": "パラメータを設定してバックテストを実行" }
  }
};
export default Marketplace;
