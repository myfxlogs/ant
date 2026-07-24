// Auto-generated from proto/ant/v1/i18n/admin_zh-tw.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const Admin = {
  "admin": {
    "strategy": {
      "actions": {
        "archive": "歸檔",
        "archiveConfirm": "歸檔此策略？",
        "code": "程式碼",
        "disable": "禁用",
        "disableConfirm": "停止所有排程？",
        "enable": "啟用",
        "flag": "標記",
        "publish": "發布",
        "unflag": "取消標記",
        "unpublish": "取消發布"
      },
      "all": {
        "allActive": "全部活躍",
        "archived": "已歸檔",
        "disabled": "已禁用",
        "flagFilter": "標記篩選",
        "flagged": "已標記",
        "searchPlaceholder": "搜尋名稱...",
        "total": "共 {{count}} 條"
      },
      "columns": {
        "actions": "操作",
        "code": "程式碼",
        "description": "描述",
        "flag": "標記",
        "name": "名稱",
        "no": "否",
        "owner": "所有者",
        "preset": "預設",
        "public": "公開",
        "schedules": "排程數",
        "status": "狀態",
        "system": "— 系統 —",
        "tags": "標籤",
        "tagsPlaceholder": "趨勢追蹤, MA",
        "type": "型別",
        "user": "使用者",
        "uses": "使用次數",
        "yes": "是"
      },
      "messages": {
        "archiveFailed": "歸檔失敗",
        "archiveSuccess": "已歸檔",
        "deleteFailed": "刪除失敗",
        "disableFailed": "禁用失敗",
        "disableSuccess": "已禁用 — 所有排程已停止",
        "enableFailed": "啟用失敗",
        "enableSuccess": "已啟用",
        "flagFailed": "標記失敗",
        "flagSuccess": "策略已標記",
        "loadPresetFailed": "載入預設策略失敗",
        "loadStrategiesFailed": "載入策略列表失敗",
        "presetCreated": "預設已建立",
        "presetDeleted": "預設已刪除",
        "presetUpdated": "預設已更新",
        "publishFailed": "釋出失敗",
        "publishSuccess": "已釋出",
        "saveFailed": "儲存失敗",
        "unflagFailed": "取消標記失敗",
        "unflagSuccess": "標記已取消",
        "unpublishFailed": "取消釋出失敗",
        "unpublishSuccess": "已取消釋出"
      },
      "preset": {
        "add": "新增預設",
        "create": "建立預設",
        "deleteConfirm": "確認刪除此預設？",
        "edit": "編輯預設"
      },
      "tabs": {
        "allStrategies": "所有策略",
        "preset": "預設策略"
      },
      "title": "策略管理"
    },
    "sweep": {
      "aboveThreshold": "超過閾值",
      "address": "地址",
      "addressId": "地址 ID",
      "batchExport": "批次匯出",
      "batchExportSuccess": "批次匯出完成",
      "builtAt": "構建時間",
      "bundleId": "批次 ID",
      "bundleStatus": "狀態",
      "dashboard": "儀表盤",
      "derivationIndex": "派生索引",
      "export": "匯出",
      "exportSuccess": "匯出完成",
      "import": "匯入",
      "importHint": "上傳已簽名的歸集包 (.bin) 以匯入並廣播。",
      "importSuccess": "匯入完成",
      "importTitle": "匯入已簽名批次",
      "pendingBundles": "待簽名批次",
      "pendingSignBundles": "待簽名批次",
      "sweepStatus": "歸集狀態",
      "threshold": "閾值",
      "title": "歸集管理",
      "totalUnswept": "未歸集總量",
      "undelegate": "取消委託",
      "undelegateSuccess": "取消委託批次已匯出",
      "unswept": "未歸集",
      "uploadHint": "點選或拖拽檔案上傳",
      "uploadXpub": "上傳 XPUB",
      "xpubFpNotSet": "指紋未驗證",
      "xpubFpVerified": "指紋已驗證",
      "xpubHint": "上傳 XPUB 檔案以派生充值地址。匯入時將驗證指紋。",
      "xpubImported": "XPUB 已匯入",
      "xpubTitle": "XPUB 管理"
    }
  }
} as const;
export default Admin;
