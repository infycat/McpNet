# 版本紀錄

本專案採用 `MAJOR.MINOR.PATCH` 版本號。正式版依時間由新到舊列出；每個版本的實際成品、校驗碼與完整說明以連結的 GitHub Release 頁面為準。GitHub 自動產生的 Source code ZIP/TAR.GZ 是原始碼，不是可執行成品。

## 正式版本

| 版本 | 日期 | 狀態 | 平台 | 檔案 | 發布說明 |
|---|---|---|---|---|---|
| [v1.2.1](https://github.com/infycat/McpNet/releases/tag/v1.2.1) | 2026-10-20 | MCP URL 隱私修正 | Windows x64 | `McpNet-v1.2.1-windows-x64.exe` | [下載與校驗碼](https://github.com/infycat/McpNet/releases/tag/v1.2.1) |
| [v1.2.0](https://github.com/infycat/McpNet/releases/tag/v1.2.0) | 2026-10-20 | MCP 捕獲監控更新 | Windows x64 | `McpNet-v1.2.0-windows-x64.exe` | [下載與校驗碼](https://github.com/infycat/McpNet/releases/tag/v1.2.0) |
| [v1.1.0](https://github.com/infycat/McpNet/releases/tag/v1.1.0) | 2026-10-20 | 最新版／介面與插件功能更新 | Windows x64 | `McpNet-v1.1.0-windows-x64.exe` | [下載與校驗碼](https://github.com/infycat/McpNet/releases/tag/v1.1.0) |
| [v1.0.0](https://github.com/infycat/McpNet/releases/tag/v1.0.0) | 2026-10-19 | 首個公開版 | Windows x64 | `McpNet-v1.0.0-windows-x64.exe` | [下載與校驗碼](https://github.com/infycat/McpNet/releases/tag/v1.0.0) |

## v1.2.1 — 2026-10-20

安全修正：MCP 捕獲清單與詳情中的 URL 會遮蔽所有查詢參數值、使用者資訊及片段，避免認證 token 等資料隨中繼資料回傳。封包標頭與本文仍僅在客戶端明確設定 `include_body=true` 時提供。

- **平台：** Windows x64
- **檔案：** `McpNet-v1.2.1-windows-x64.exe`
- **檔案大小：** 91,819,520 bytes（約 87.6 MiB）
- **SHA-256：** `0B30D2D66E2F336E8CA9133807F9E95CC1E05975F9967ECF6354000A17790676`
- **下載：** [GitHub Release 成品](https://github.com/infycat/McpNet/releases/download/v1.2.1/McpNet-v1.2.1-windows-x64.exe)

## v1.2.0 — 2026-10-20

補足 MCP 的捕獲監控能力。MCP 客戶端現在可查詢已顯示的 HTTP(S)、WebSocket、TCP、UDP 會話及分頁讀取連線訊息；封包本文與標頭採明確 opt-in，預設不回傳，並以 Base64 與每段 64 KiB 上限傳輸。系統代理仍由使用者控制，不會自動開啟。

- **MCP 工具：** `mcpnet_list_captures`、`mcpnet_get_capture`、`mcpnet_get_messages`；原有監聽、系統代理與狀態工具保留。
- **隱私與大小限制：** 清單預設僅回傳中繼資料；讀取標頭／本文需 `include_body=true`，單段本文最多 64 KiB。
- **平台：** Windows x64
- **檔案：** `McpNet-v1.2.0-windows-x64.exe`
- **檔案大小：** 91,820,032 bytes（約 87.6 MiB）
- **SHA-256：** `951CB36062BD7E6E0D01E3074F53B29E603BD29F8B6B2144A61F6C6530E8B32D`
- **下載：** [GitHub Release 成品](https://github.com/infycat/McpNet/releases/download/v1.2.0/McpNet-v1.2.0-windows-x64.exe)

## v1.1.0 — 2026-10-20

第二版重點是介面整理與本機插件／編解碼流程完善，保留原有協定分析、MCP 整合及個人插件本機化設計。

- **介面與佈局：** 統一明暗及漸變主題，改善設定視窗、進程列表與主請求表格；表格欄寬可調，窄視窗時欄位按比例收縮、文字以省略號呈現，並縮小表頭與列高。
- **進程列表：** 顯示進程圖標；「附近應用進程」模式列出有視窗的應用，關閉後可查看完整進程清單。
- **本機插件：** 顯示插件讀取狀態並可啟用／停用；個人插件仍留在本機資料目錄，公開倉庫僅保留通用範例。
- **協定處理：** 修正 HTTP、WebSocket、UDP 插件 hook 的顯示／線上資料分離；改善 WebSocket 雙向解碼、編碼與 JSON 顯示及 JSON 主動發送，並增加回歸測試。
- **平台：** Windows x64
- **檔案：** `McpNet-v1.1.0-windows-x64.exe`
- **SHA-256：** `B4ACE65F887F56D4B03B74570952D71AB999353E4D3F31AC3F6FB546EB882A22`
- **檔案大小：** 91,793,408 bytes（約 87.5 MiB）
- **下載：** [GitHub Release 成品](https://github.com/infycat/McpNet/releases/download/v1.1.0/McpNet-v1.1.0-windows-x64.exe)

本版不會因啟動而自動啟用全系統監聽或系統代理。使用者需在介面內明確開啟。

## v1.0.0 — 2026-10-19

首個公開版本，包含 Windows 桌面 UI、HTTP(S)/WebSocket/TCP/UDP 流量檢視、規則與插件擴充、本機 MCP 服務，以及繁體中文／簡體中文／英文專案文件。

- **平台：** Windows x64
- **檔案：** `McpNet-v1.0.0-windows-x64.exe`
- **SHA-256：** `30C7A398560D47EFD6C288A9D9C9E6F0C8F89F2DCA42E12BCEFF7D9D7F369B9A`
- **檔案大小：** 91,674,112 bytes（約 87.4 MiB）
- **下載：** [GitHub Release 成品](https://github.com/infycat/McpNet/releases/download/v1.0.0/McpNet-v1.0.0-windows-x64.exe)

首次啟動可能需要 WebView2 Runtime。流量監聽與系統代理不會因為程式開啟而自動啟用；請在應用程式內確認狀態後自行開始。個人插件不包含在公開成品或倉庫中。
