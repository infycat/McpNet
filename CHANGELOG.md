# Release Notes / 版本紀錄

Current stable release / 目前正式版本：**v1.1.3**. Previous releases remain available. GitHub's automatically generated source archives are source code, not executable builds. / 舊版仍保留下載；GitHub 自動產生的 Source code 壓縮檔是原始碼，不是可執行成品。

## v1.1.3 — 2026-10-23 — Startup Fix / 啟動修復

### 繁體中文

- 完整修正已確認的啟動失敗根因：為內建腳本檢查器補上舊版 SunnyNet `public` 與 `RSA` 匯入路徑相容映射，避免程式初始化時因找不到模組而在 UI 顯示前崩潰。
- 修正版已在本機以正式 GUI 模式啟動並確認主視窗正常顯示；啟動診斷日誌仍會記錄初始化流程，方便排查其他電腦特定環境問題。
- 本版本適用 Windows x64；首次啟動仍可能需要 WebView2 Runtime。此修正解決本次確認的程式碼相容性問題，但不代表已在所有 Windows 裝置組合上完成測試。

### English

- Fully fixes the confirmed startup failure root cause: the bundled script checker now maps legacy SunnyNet `public` and `RSA` import paths, preventing a missing-module panic before the UI appears.
- The corrected build was launched in GUI mode on the development machine and its main window was confirmed visible. Startup diagnostics remain available to investigate any machine-specific environment issue.
- For Windows x64. WebView2 Runtime may still be required on first launch. This resolves the confirmed code-compatibility failure; it does not claim testing across every Windows device configuration.

- **Platform / 平台:** Windows x64
- **File / 檔案:** `McpNet-v1.1.3-windows-x64.exe`
- **Release / 發布:** [GitHub Release](https://github.com/infycat/McpNet/releases/tag/v1.1.3)

## v1.1.2 — 2026-10-23 — Startup Compatibility Fix / 啟動相容性修正

### 繁體中文

- 以已驗證可啟動的 v1.1.1 初始化路徑為基礎，恢復 Monaco 編輯器的原始載入順序，移除可能令部分環境啟動競速或把啟動轉送到隱藏程序的改動。
- 啟動超過 30 秒時只記錄等待進度，不再提前顯示錯誤；首次建立 WebView2 資料或較慢電腦可繼續等待。
- 每次啟動建立帶時間與 PID 的診斷日誌，保留最近 50 份；優先使用 `%LOCALAPPDATA%\McpNet_new\logs\`，無法寫入時回退至使用者設定目錄或 `%TEMP%`。
- 記錄 WebView2 執行階段、資料目錄、Wails/前端初始化與主視窗顯示狀態，方便追查「程序存在但 UI 未出現」的情況。
- Windows 發布成品內嵌 WebView2 Evergreen bootstrapper；缺少執行環境時可選擇安裝，下載 Runtime 本身仍需網路連線。

### English

- Restored the Monaco editor initialization order from the known-working v1.1.1 startup path and removed changes that could race startup or redirect a launch to a hidden process on some systems.
- After 30 seconds, startup now logs continued progress instead of showing a premature failure dialog; first-run WebView2 profile creation and slower PCs can continue waiting.
- Each launch gets a timestamped, PID-specific diagnostic log; only the newest 50 are retained. Logs use `%LOCALAPPDATA%\McpNet_new\logs\` first, then the user config directory or `%TEMP%` if needed.
- Logs record the WebView2 runtime and profile path, Wails/frontend initialization, and main-window display state to diagnose cases where a process exists but the UI does not appear.
- Windows release builds embed the WebView2 Evergreen bootstrapper and can offer to install a missing runtime; downloading the runtime itself still requires internet access.

- **Platform / 平台:** Windows x64
- **File / 檔案:** `McpNet-v1.1.2-windows-x64.exe`
- **Release / 發布:** [GitHub Release](https://github.com/infycat/McpNet/releases/tag/v1.1.2)

## v1.1.1 — 2026-10-20 — Patch Release / 修補版

### 繁體中文

- 源碼頁整理為單一 GitHub 入口，集中提供源碼、SDK 與文件。
- 將舊 QQ 群資訊替換為 Discord 社群邀請連結。
- 縮小「作者」頁的個人資料圖片，改善視窗中的顯示比例。
- 公開版本不包含個人本機插件或使用者設定。

### English

- Consolidated the source, SDK, and documentation links into one GitHub entry.
- Replaced the old QQ group details with the Discord community invite.
- Reduced the profile image size on the Author tab for a better fit in the window.
- Personal local plugins and user settings are not included in the public build.

- **Platform / 平台:** Windows x64
- **File / 檔案:** `McpNet-v1.1.1-windows-x64.exe`
- **Release / 發布:** [GitHub Release](https://github.com/infycat/McpNet/releases/tag/v1.1.1)
- File size and SHA-256 are added to the release after the automated Windows build completes. / 自動 Windows 建置完成後，會將檔案大小與 SHA-256 寫入 Release。

## v1.1.0 — 2026-10-20 — Second Release / 第二版

### 中文

- 修正程式設定最大化時頂部覆蓋主視窗工具列，並避免右上角視窗控制按鈕重疊，確保仍可移動主視窗。
- 修正最大化設定視窗底部未延伸至視窗邊緣、露出主畫面與捲軸的問題。
- README 功能截圖已放在腦圖之前，方便先了解程式功能。
- 本版本面向 Windows x64；公開成品不包含個人本機插件及設定。

### English

- Fixed maximized Program Settings covering the main window toolbar and overlapping the top-right window controls, so the main window remains movable.
- Fixed the maximized settings view ending above the bottom edge and exposing underlying content and scrollbars.
- Moved the README feature screenshots before the mind maps to make the app's features easier to discover.
- Built for Windows x64. Personal local plugins and settings are not included in the public build.

- **Platform / 平台:** Windows x64
- **File / 檔案:** `McpNet-v1.1.0-windows-x64.exe`
- **Size / 大小:** 91,961,856 bytes (~87.7 MiB)
- **SHA-256:** `6939D6EB999FE8869E0C89B987FE22C5EBC01446874429746B8D03F57F2E93CD`
- **Download / 下載:** [GitHub Release](https://github.com/infycat/McpNet/releases/tag/v1.1.0)

## v1.0.0 — 2026-10-20 — Initial Public Release / 首個正式公開版本

### 中文

- 更新 Windows、應用程式與文件預覽所使用的品牌圖示。
- 修復切換主題或彈窗重繪後部分視窗背景透明、底層畫面透出的問題。
- 修復設定視窗最大化時遮住頂部工具列或露出底部主畫面與捲軸的問題。
- 將「打賞作者」改為「作者」，並替換作者頁圖片。
- 首版保留 McpNet 的 Windows 桌面網路分析、HTTP(S)/WebSocket/TCP/UDP 檢視與處理、規則、本機插件及 MCP 服務功能。

### English

- Refreshed the brand icon used by the Windows executable, app, and documentation preview.
- Fixed transparent themed surfaces that exposed underlying content after theme changes or dialog redraws.
- Fixed maximized settings covering the top toolbar or exposing the underlying footer and scrollbars.
- Renamed the “Support the Author” tab to “Author” and refreshed its image.
- The initial release includes the Windows network analysis UI, HTTP(S)/WebSocket/TCP/UDP inspection and processing, rules, local plugins, and MCP service.

- **Platform / 平台:** Windows x64
- **File / 檔案:** `McpNet-v1.0.0-windows-x64.exe`
- **Size / 大小:** 91,961,856 bytes (~87.7 MiB)
- **SHA-256:** `BFC3825775BCE65E29169A6DE613294BB3D5B987B12E7425EDD83FF1218C7630`
- **Download / 下載:** [GitHub Release](https://github.com/infycat/McpNet/releases/tag/v1.0.0)

On first launch, WebView2 Runtime may be required. Opening the app does not automatically start traffic capture or enable the system proxy. / 首次啟動可能需要 WebView2 Runtime。開啟程式不會自動啟動流量監聽或系統代理。
