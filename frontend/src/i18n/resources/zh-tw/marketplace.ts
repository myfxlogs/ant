const Marketplace = {
  "marketplace": {
    "title": "策略市場",
    "subtitle": "發現和訂閱交易策略",
    "tabs": { "marketplace": "市場", "purchases": "我的購買", "author": "作者中心", "bundles": "捆綁包", "optimization": "AI 優化", "fees": "階梯費率" },
    "searchPlaceholder": "搜尋策略...",
    "empty": "暫無策略",
    "filter": { "all": "全部", "free": "免費", "paid": "付費" },
    "sort": { "score": "評分", "newest": "最新", "popular": "最熱", "rating": "評價", "priceAsc": "價格從低到高", "priceDesc": "價格從高到低" },
    "card": { "free": "免費", "owned": "已擁有", "winRate": "勝率", "pnl": "總盈虧", "users": "訂閱者", "rent": "¥{{amount}}/月", "buy": "¥{{amount}}" },
    "messages": { "loginFirst": "請先登入", "subscribed": "訂閱成功", "subscribeFailed": "訂閱失敗", "published": "策略已發布到市場", "publishFailed": "發布失敗", "rated": "已評分！" },
    "detail": { "author": "作者", "price": "價格", "assetClass": "資產類別", "riskLevel": "風險等級", "subscribers": "訂閱者", "avgRating": "評分", "description": "描述", "tags": "標籤", "yourRating": "你的評分", "comments": "評論", "noComments": "暫無評論", "commentPlaceholder": "寫評論...", "getFree": "免費獲取", "buyNow": "立即購買", "owned": "已擁有", "freePrice": "免費", "rentPrice": "¥{{amount}} / 月", "buyPrice": "¥{{amount}} 買斷", "runBacktest": "執行回測" },
    "purchases": { "empty": "暫無購買", "strategy": "策略", "date": "購買時間", "status": "狀態", "actions": "操作", "runBacktest": "執行回測" },
    "author": { "empty": "暫無已發布策略", "noPublished": "暫無已發布策略", "published": "已發布", "subscribers": "訂閱者", "avgRating": "平均評分", "myStrategies": "我的已發布策略", "publishNew": "發布新策略", "goToLibrary": "前往策略庫" },
    "payment": { "purchaseSuccess": "購買成功！策略已添加到你的庫中。", "purchaseFailed": "購買失敗，請重試。", "insufficientBalance": "餘額不足", "alreadyPurchased": "你已經擁有這個策略。", "title": "確認購買", "balance": "錢包餘額", "confirm": "確認支付", "cancel": "取消" },
    "publish": {
      "title": "發布到市場",
      "titleLabel": "標題",
      "titlePlaceholder": "如：金叉策略",
      "descriptionLabel": "描述",
      "descriptionPlaceholder": "描述你的策略邏輯、入場/出場規則...",
      "assetClass": { "label": "資產類別", "forex": "外匯", "crypto": "加密貨幣", "commodity": "大宗商品", "index": "指數", "stock": "股票" },
      "riskLevel": { "label": "風險等級", "low": "低", "medium": "中", "high": "高" },
      "priceModel": { "label": "定價", "free": "免費", "subscription": "按月訂閱", "once": "一次性購買" },
      "priceAmount": "金額",
      "tags": "標籤",
      "tagsPlaceholder": "輸入並按回車添加標籤",
      "codeSnippet": "策略預覽（公開）",
      "codeSnippetPlaceholder": "可選：分享策略的高層思路或程式碼片段（所有人可見）",
      "includeBacktestSnapshot": "附帶最新回測成績"
    },
    "backtest": { "title": "策略回測", "protected": "策略程式碼受保護，回測在伺服器端執行。", "run": "執行回測", "idle": "設定參數並執行回測" }
  }
};
export default Marketplace;
