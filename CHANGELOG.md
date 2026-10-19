# 版本紀錄

本專案採用 `MAJOR.MINOR.PATCH` 版本號。正式版依時間由新到舊列出；每個版本的實際成品、校驗碼與完整說明以連結的 GitHub Release 頁面為準。GitHub 自動產生的 Source code ZIP/TAR.GZ 是原始碼，不是可執行成品。

## 正式版本

| 版本 | 日期 | 狀態 | 平台 | 檔案 | 發布說明 |
|---|---|---|---|---|---|
| [v1.1.0](https://github.com/infycat/McpNet/releases/tag/v1.1.0) | 2026-10-20 | 最新版／介面與插件功能更新 | Windows x64 | `McpNet-v1.1.0-windows-x64.exe` | [下載與校驗碼](https://github.com/infycat/McpNet/releases/tag/v1.1.0) |
| [v1.0.0](https://github.com/infycat/McpNet/releases/tag/v1.0.0) | 2026-10-19 | 首個公開版 | Windows x64 | `McpNet-v1.0.0-windows-x64.exe` | [下載與校驗碼](https://github.com/infycat/McpNet/releases/tag/v1.0.0) |

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
