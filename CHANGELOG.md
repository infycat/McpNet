# 版本紀錄

本專案採用 `MAJOR.MINOR.PATCH` 版本號。正式版依時間由新到舊列出；每個版本的實際成品、校驗碼與完整說明以連結的 GitHub Release 頁面為準。GitHub 自動產生的 Source code ZIP/TAR.GZ 是原始碼，不是可執行成品。

## 正式版本

| 版本 | 日期 | 狀態 | 平台 | 成品與發布說明 |
|---|---|---|---|---|
| [v1.0.0](https://github.com/infycat/McpNet/releases/tag/v1.0.0) | 2026-10-19 | 最新／首個公開版 | Windows x64 | [下載 EXE](https://github.com/infycat/McpNet/releases/download/v1.0.0/McpNet-v1.0.0-windows-x64.exe) |

## v1.0.0 — 2026-10-19

首個公開版本，包含 Windows 桌面 UI、HTTP(S)/WebSocket/TCP/UDP 流量檢視、規則與插件擴充、本機 MCP 服務，以及繁體中文／簡體中文／英文專案文件。

- **平台：** Windows x64
- **檔案：** `McpNet-v1.0.0-windows-x64.exe`
- **SHA-256：** `30C7A398560D47EFD6C288A9D9C9E6F0C8F89F2DCA42E12BCEFF7D9D7F369B9A`
- **檔案大小：** 91,674,112 bytes（約 87.4 MiB）
- **下載：** [GitHub Release 成品](https://github.com/infycat/McpNet/releases/download/v1.0.0/McpNet-v1.0.0-windows-x64.exe)

首次啟動可能需要 WebView2 Runtime。流量監聽與系統代理不會因為程式開啟而自動啟用；請在應用程式內確認狀態後自行開始。個人插件不包含在公開成品或倉庫中。
