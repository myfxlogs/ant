// Auto-generated from proto/ant/v1/i18n/admin_ja.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const Admin = {
  "admin": {
    "strategy": {
      "actions": {
        "archive": "アーカイブ",
        "archiveConfirm": "この戦略をアーカイブしますか？",
        "code": "コード",
        "disable": "無効化",
        "disableConfirm": "すべてのスケジュールを停止しますか？",
        "enable": "有効化",
        "flag": "フラグ",
        "publish": "公開",
        "unflag": "フラグ解除",
        "unpublish": "公開取消"
      },
      "all": {
        "allActive": "すべてのアクティブ",
        "archived": "アーカイブされました",
        "disabled": "無効",
        "flagFilter": "フラグフィルター",
        "flagged": "フラグ付き",
        "searchPlaceholder": "名前で検索...",
        "total": "合計 {{count}} 件"
      },
      "columns": {
        "actions": "操作",
        "code": "コード",
        "description": "説明",
        "flag": "フラグ",
        "name": "名前",
        "no": "否",
        "owner": "所有者",
        "preset": "プリセット",
        "public": "公開",
        "schedules": "スケジュール",
        "status": "ステータス",
        "system": "— システム —",
        "tags": "タグ",
        "tagsPlaceholder": "トレンドフォロー, MA",
        "type": "タイプ",
        "user": "ユーザー",
        "uses": "使用回数",
        "yes": "はい"
      },
      "messages": {
        "archiveFailed": "アーカイブに失敗しました",
        "archiveSuccess": "アーカイブされました",
        "deleteFailed": "削除に失敗しました",
        "disableFailed": "無効化に失敗しました",
        "disableSuccess": "無効化されました — すべてのスケジュールが停止しました",
        "enableFailed": "有効化に失敗しました",
        "enableSuccess": "有効化されました",
        "flagFailed": "フラグに失敗しました",
        "flagSuccess": "戦略にフラグを付けました",
        "loadPresetFailed": "プリセット戦略の読み込みに失敗しました",
        "loadStrategiesFailed": "戦略リストの読み込みに失敗しました",
        "presetCreated": "プリセットが作成されました",
        "presetDeleted": "プリセットが削除されました",
        "presetUpdated": "プリセットが更新されました",
        "publishFailed": "发布失败",
        "publishSuccess": "已发布",
        "saveFailed": "保存に失敗しました",
        "unflagFailed": "フラグ解除に失敗しました",
        "unflagSuccess": "フラグが解除されました",
        "unpublishFailed": "取消发布失败",
        "unpublishSuccess": "已取消发布"
      },
      "preset": {
        "add": "プリセット追加",
        "create": "プリセット作成",
        "deleteConfirm": "このプリセットを削除しますか？",
        "edit": "プリセット編集"
      },
      "tabs": {
        "allStrategies": "すべての戦略",
        "preset": "プリセット戦略"
      },
      "title": "戦略管理"
    },
    "sweep": {
      "aboveThreshold": "閾値超過",
      "address": "アドレス",
      "addressId": "アドレス ID",
      "batchExport": "一括エクスポート",
      "batchExportSuccess": "一括エクスポート完了",
      "builtAt": "構築日時",
      "bundleId": "バンドル ID",
      "bundleStatus": "ステータス",
      "dashboard": "ダッシュボード",
      "derivationIndex": "導出インデックス",
      "export": "エクスポート",
      "exportSuccess": "エクスポート完了",
      "import": "インポート",
      "importHint": "署名済みスイープバンドル (.bin) をアップロードしてインポート・ブロードキャスト。",
      "importSuccess": "インポート完了",
      "importTitle": "署名済みバンドルをインポート",
      "pendingBundles": "保留中バンドル",
      "pendingSignBundles": "署名待ちバンドル",
      "sweepStatus": "スイープステータス",
      "threshold": "閾値",
      "title": "スイープ管理",
      "totalUnswept": "未スイープ総量",
      "undelegate": "委任解除",
      "undelegateSuccess": "委任解除バンドルをエクスポート",
      "unswept": "未スイープ",
      "uploadHint": "クリックまたはドラッグしてアップロード",
      "uploadXpub": "XPUB アップロード",
      "xpubFpNotSet": "フィンガープリント未確認",
      "xpubFpVerified": "フィンガープリント確認済み",
      "xpubHint": "XPUB ファイルをアップロードして入金アドレスを導出。インポート時にフィンガープリントを検証します。",
      "xpubImported": "XPUB をインポートしました",
      "xpubTitle": "XPUB 管理"
    }
  }
} as const;
export default Admin;
