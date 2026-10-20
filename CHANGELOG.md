# Release Notes / 版本紀錄

Current stable release / 目前正式版本：**v1.1.0**. The first release, v1.0.0, remains available. GitHub's automatically generated source archives are source code, not executable builds. / 首版 v1.0.0 仍保留下載；GitHub 自動產生的 Source code 壓縮檔是原始碼，不是可執行成品。

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
