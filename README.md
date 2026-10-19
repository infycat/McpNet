# McpNet

> Windows 桌面網路分析與協定除錯工具，整合流量檢視、可編輯規則、腳本／插件與本機 MCP 服務。

**繁體中文（預設）** · [简体中文](docs/ARCHITECTURE.md#简体中文) · [English](docs/ARCHITECTURE.md#english)

## 下載與版本

**目前穩定版：v1.0.0**（首個公開版本，Windows x64）

- [下載 McpNet v1.0.0（Windows x64）](https://github.com/infycat/McpNet/releases/download/v1.0.0/McpNet-v1.0.0-windows-x64.exe)
- [查看 v1.0.0 發布說明與校驗碼](https://github.com/infycat/McpNet/releases/tag/v1.0.0)
- [瀏覽所有版本與歷史下載](https://github.com/infycat/McpNet/releases)

| 版本 | 發布日期 | 平台 | 下載 | 說明 |
|---|---|---|---|---|
| **v1.0.0**（最新） | 2026-10-19 | Windows x64 | [EXE](https://github.com/infycat/McpNet/releases/download/v1.0.0/McpNet-v1.0.0-windows-x64.exe) | 首個公開版本；功能與限制見[發布說明](https://github.com/infycat/McpNet/releases/tag/v1.0.0) |

下載後可在 PowerShell 使用 `Get-FileHash .\McpNet-v1.0.0-windows-x64.exe -Algorithm SHA256` 核對檔案雜湊。請只從本專案的 Releases 頁面下載；新版本發布後，版本表會保留歷史版本並標示最新版本。

GitHub 專案根目錄請使用本資料夾 `deliverable/source/`；不要從整個工作區根目錄發布，因為上層另含逆向研究中間產物與封存檔。

McpNet 由 Go/Wails 桌面後端、Vue 3 前端與 SunnyNet 核心組成，可在明確啟動監聽後檢視和處理 HTTP(S)、WebSocket、TCP、UDP 流量。MCP 服務僅綁定本機回環位址；插件由使用者資料目錄載入，GitHub 範例目錄不放個人完成品插件。

## 專案導覽

- [系統架構、流程圖、腦圖與部署分佈圖](docs/ARCHITECTURE.md)
- [簡體中文與 English 完整說明](docs/ARCHITECTURE.md#简体中文)
- [核心 SunnyNet 重建與相容性說明](third_party/recovered-McpNet/README-重建说明.md)
- [逆向研究背景](docs/REVERSE_ENGINEERING.md)
- [插件範例與發布邊界](plugins/examples/README.md)

## 功能概覽

| 範圍 | 能力 |
|---|---|
| 流量 | HTTP/HTTPS、WebSocket、TCP、UDP；請求／回應檢視、內容搜尋與資料檢視 |
| 規則 | Hosts、轉送／替換、請求與回應處理、憑證及腳本設定 |
| 擴充 | 使用者 Go 插件（Yaegi 解譯）、腳本範本、可讀寫的協定內容處理介面 |
| 自動化 | MCP over HTTP，提供狀態、監聽控制、系統代理設定與 UI 命令工具 |
| 桌面 | Windows、Wails v2、Vue 3、WebView2；介面支援繁中、簡中與英文 |

實際可攔截範圍取決於 Windows 權限、驅動與憑證設定、應用程式流量路徑及使用者啟用的規則；啟動程式不等於已開始全系統攔截。HTTPS 解密可能需要安裝並信任本機 CA，請只在你有權分析的裝置與流量上操作。

## 快速建置

需求：Windows、Go 1.23+、Node.js/npm（如需重建前端）、WebView2 Runtime；重新產生 Windows 資源時另需 MinGW `windres`。

```powershell
cd <repository-root>
npm --prefix frontend ci
npm --prefix frontend run build
go build -tags "desktop,production,wv2runtime.download" -ldflags "-H windowsgui -s -w" -o McpNet.exe .
```

或直接使用專案腳本（需已有 `frontend/dist`）：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\build.ps1
```

前端 `dist/` 是 Go `embed` 的建置輸入，須保留在 Git 或先由 npm 產生。Git 忽略規則會排除編譯後的主程式和個人插件，只允許 `plugins/examples/` 下的公開範例。

## 執行與資料

- 使用者資料：`%USERPROFILE%\McpNet_new\`；設定、MCP 設定、WebView2 資料及使用者插件彼此分開保存。
- MCP 預設端點：`http://127.0.0.1:2026/mcp`；健康檢查：`http://127.0.0.1:2026/health`。
- MCP 會隨應用程式啟動；網路監聽及系統代理有獨立控制，MCP 工具呼叫也可能改變這些狀態。
- 使用者插件：`%USERPROFILE%\McpNet_new\plugins\<plugin-id>\`。個人插件只在本機安裝並載入；倉庫僅保留通用範例。

## GitHub 發布原則

提交前確認 `.gitignore` 規則有效；`plugins/` 中僅保留 `plugins/examples/`。個人 XXTEA/XOR、URL 顯示與 UDP 解碼插件位於本機使用者資料目錄，不在 GitHub 源碼樹中。請勿將個人插件、用戶設定、日誌、私有憑證、抓包資料或發布用二進位檔加入索引。範例插件預設停用，匯入前請先檢閱程式碼。

授權條款請見 [LICENSE](LICENSE)。

版本紀錄與版本號規則見 [CHANGELOG.md](CHANGELOG.md)。每個正式版本均提供獨立 Release 頁面、變更摘要、平台／相容性資訊和下載成品；原始碼 ZIP 由 GitHub 自動提供，並非已編譯的應用程式。

核心與前端依賴包含第三方元件；其各自授權與版權聲明仍適用，請一併保留第三方授權檔。
