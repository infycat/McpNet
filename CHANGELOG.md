# Release Notes / 版本紀錄

## v1.2.5 — 2026-10-27 — Process Capture Repair / 進程封包捕獲修復

### 繁體中文

- 修復部分遊戲進程的 HTTP 與 WebSocket 流量無法進入捕獲的問題：進程攔截改用 WFP/NFAPI 路徑，並在載入驅動時自動啟動僅綁定 `127.0.0.1` 的本機捕獲接收端。
- 此流程不會自動設定 Windows 系統代理，也不會新增防火牆規則；程式設定中的監聽狀態會反映本機接收端是否運行。
- 已用 MiniGameApp 實際驗證，捕獲到 HTTP 200 與 WebSocket 101 流量。

### English

- Fixed HTTP and WebSocket traffic from some game processes missing from capture. Process interception now uses the WFP/NFAPI path and automatically starts a capture endpoint bound only to `127.0.0.1` when the driver is loaded.
- This does not configure the Windows system proxy or add a firewall rule. The app's listener status reflects whether the local endpoint is running.
- Verified against MiniGameApp with captured HTTP 200 and WebSocket 101 traffic.

- **Platform / 平台:** Windows x64
- **File / 檔案:** `McpNet-v1.2.5-windows-x64.exe`
- **Release / 發布:** [GitHub Release](https://github.com/infycat/McpNet/releases/tag/v1.2.5)

## v1.2.4 — 2026-10-25 — Automatic Local CA Generation / 本機 CA 自動產生

### 繁體中文

- CA 設定新增「自動生成並應用證書」：產生適用於本機 HTTPS 解密的 RSA-2048 自簽根 CA，保存於使用者設定目錄並立即套用至 McpNet；私鑰不會上傳。此功能不會自動安裝至 Windows 受信任根憑證存放區，系統信任仍需使用者明確執行既有安裝操作。

### English

- Added “Generate and apply a certificate” to CA settings. It creates an RSA-2048 self-signed root CA for local HTTPS inspection, stores the certificate and private key in the user's configuration directory, and applies it to McpNet. The private key is never uploaded. The feature does not install the CA into the Windows trusted root store; that remains a separate, explicit user action.

## v1.2.3 — 2026-10-25 — MCP Tools and Live Process Filters / MCP 工具與進程規則即時同步

### 繁體中文

- 新增 `mcpnet_list_commands`，讓 AI 客戶端可探索前端使用的後端命令，再透過 `mcpnet_command` 以 UI 相同的 `Args` 格式操作。
- 新增直接網路 MCP 工具：可建立 HTTP/HTTPS、WebSocket 請求及 TCP/UDP 連線並傳送 Base64 資料；網路發送與讀取響應分別使用網路寫入和私密資料權限。回應大小限制為 64 KiB，HTTP 不跟隨重新導向。`mcpnet_send_session_data` 仍用於既有 WebSocket/TCP/UDP 連線。
- 新增具名插件 MCP 工具，可讀取插件清單／原始碼、建立或更新插件、刪除插件，以及執行插件自訂 UI 動作；保存操作強制維持停用，必須另行授權啟用。
- 插件清單、匯入、啟用、管理與自訂 UI 動作改為直接呼叫 Go 後端，不依賴前端視窗事件轉接；啟用／停用後同步清除插件快取。
- 修正注入式 MCP 確認回呼被本機已保存的完全允許設定略過的測試隔離問題；擴充前端及後端命令目錄覆蓋測試，並補入受私密資料權限保護的「枚舉進程」命令。
- MCP 與健康檢查 HTTP 端點拒絕非 loopback Host/Origin，移除通配 CORS 與無狀態服務虛構的 session ID；新增 2025-11-25 等協議版本協商、未知版本拒絕及 JSON-RPC 版本檢查。
- 通用 UI 命令現在按私密資料、設定、插件程式碼及網路寫入分類，讓部分授權模式可精細使用；命令目錄提供每個操作的 `Args` 欄位、權限類別及授權方式提示，並標明依參數變化權限的命令；檔案選擇與儲存視窗可由 MCP 傳入絕對路徑，避免呼叫原生視窗；默認 CA 證書可指定路徑匯出且不覆寫既有檔案。MCP 權限設定變更一律須使用者直接確認。
- 修正封包 Hex 檢視器在滑鼠座標尚未初始化時呼叫 `elementFromPoint`，造成前端 JavaScript 錯誤的問題。
- 修正 MCP HTTP 重播：補回捕獲請求本文長度與 Host，讓 POST 等請求可正確重播；一般重播現在等待代理回應並回報 HTTP 狀態碼，失敗會回傳錯誤，不再把背景排隊當成已成功。
- 修正 MCP 以進程名稱新增或移除攔截規則後，程序設定左右清單未同步的問題；名稱規則會即時顯示於左側，右側也會勾選所有匹配的執行中進程。

### English

- Added `mcpnet_list_commands` so AI clients can discover backend commands used by the UI and invoke them through `mcpnet_command` with the same `Args` shape.
- Added direct network MCP tools for new HTTP/HTTPS requests, WebSocket messages, and TCP/UDP connections with Base64 payloads. Network writes and response reads use separate permission categories; responses are capped at 64 KiB and HTTP redirects are not followed. `mcpnet_send_session_data` remains available for existing WebSocket/TCP/UDP sessions.
- Added named plugin MCP tools to read plugin manifests/source, create or update plugins, delete plugins, and invoke custom plugin UI actions. Saving always leaves a plugin disabled; enabling is a separate authorized operation.
- Routed plugin listing, import, enablement, management, and custom UI actions directly through the Go backend instead of relying on frontend event forwarding; enabling/disabling now invalidates the plugin cache.
- Fixed test isolation where a locally persisted unrestricted policy could bypass an injected MCP confirmation callback; expanded frontend/backend catalog coverage checks and added the private-read protected `枚舉進程` command.
- The MCP and health-check HTTP endpoints now reject non-loopback Host/Origin values. The MCP service removes wildcard CORS and synthetic session IDs, negotiates protocol versions including 2025-11-25, rejects unsupported version headers, and validates the JSON-RPC version.
- Categorized generic UI commands by private-read, settings, plugin-code, and network-write permissions so custom grants can use them selectively. Command discovery now lists each operation's `Args` fields, permission category, and authorization behavior, including context-dependent commands; file-picker commands accept explicit absolute paths without opening native dialogs; default CA certificate export accepts a path and refuses to overwrite existing files; changing MCP permissions always requires direct user confirmation.
- Fixed a Hex packet-viewer JavaScript error caused by calling `elementFromPoint` before mouse coordinates were initialized.
- Fixed MCP HTTP replay by preserving the captured body length and Host. Normal replay now waits for the proxy response and returns its HTTP status; transport failures are reported instead of treating background queuing as success.
- Fixed process settings not refreshing after MCP adds or removes an executable-name filter. Name rules now appear in the left list immediately, and matching running processes are checked in the right list.

Current stable release / 目前正式版本：**v1.2.5**. Previous releases remain available. GitHub's automatically generated source archives are source code, not executable builds. / 舊版仍保留下載；GitHub 自動產生的 Source code 壓縮檔是原始碼，不是可執行成品。

- **Platform / 平台:** Windows x64
- **File / 檔案:** `McpNet-v1.2.5-windows-x64.exe`
- **Release / 發布:** [GitHub Release](https://github.com/infycat/McpNet/releases/tag/v1.2.5)

## v1.2.2 — 2026-10-24 — MCP Permission Profiles and Live Plugins / MCP 權限設定與插件即時更新

### 繁體中文

- 插件功能清單高度增加，並在面板展開期間即時檢查插件變更；透過 MCP 導入或管理插件後，約 1.5 秒內更新，不必重啟程式。
- MCP 新增三種本機權限模式：逐次詢問（預設）、部分允許與完全允許。部分允許可分別授權讀取完整封包資料、變更設定／監聽代理、導入或啟用插件程式碼、發送或重播網路資料。
- 完全允許會略過所有 MCP 操作確認，包括私密封包資料讀取與網路／插件操作；只連接可信任的 MCP 客戶端。權限設定保存在本機，原有設定預設仍是逐次詢問。
- MCP 健康檢查與伺服器資訊改為使用同一個應用程式版本來源。
- MCP 可依精確的執行檔名稱查找本機進程，並在授權後直接載入進程攔截驅動、增刪進程名稱／PID 規則或清除規則；不必透過滑鼠鍵盤操作介面。查找 PID 屬私密資料權限；驅動與篩選規則使用設定權限，載入驅動仍需 Windows 管理員權限。
- Windows 建置改用 Go 內部 PE linker，保留應用程式 UAC manifest，避免 MinGW 重複插入 manifest 導致連結失敗。

### English

- Enlarged the plugin list and added live change detection while the panel is open. Plugins imported or managed through MCP appear within about 1.5 seconds without restarting the app.
- Added three local MCP permission modes: ask per operation (default), custom/partial grants, and unrestricted access. Partial grants independently control full capture-data reads, settings/capture/proxy controls, plugin import or execution, and network sends/replays.
- Unrestricted access skips all MCP confirmations, including private capture reads and network/plugin actions. Connect trusted MCP clients only. Permission settings are stored locally, and existing installations remain in ask-per-operation mode by default.
- MCP health and server information now use the application's shared version value.
- MCP can find local processes by exact executable name and, after authorization, load the process-interception driver, add/remove executable-name or PID filters, or clear filters directly without UI automation. PID lookup uses the private-read permission; driver/filter operations use settings permission, and driver loading still requires Windows administrator privileges.
- Windows builds now use Go's internal PE linker to preserve the application's UAC manifest and avoid duplicate-manifest failures from MinGW.

- **Platform / 平台:** Windows x64
- **File / 檔案:** `McpNet-v1.2.2-windows-x64.exe`
- **Release / 發布:** [GitHub Release](https://github.com/infycat/McpNet/releases/tag/v1.2.2)

## v1.2.1 — 2026-10-24 — MCP Integration and Privacy / MCP 整合與隱私

### 繁體中文

- 擴充本機 MCP 工具，AI 客戶端可查詢捕獲清單、詳情與訊息，查看狀態和插件，並在使用者確認後匯入／啟用插件、發送或重播封包、控制監聽與系統代理，以及呼叫通用插件介面命令。
- 涉及封包發送、插件程式碼、監聽、系統代理或插件介面操作時，逐次顯示確認；MCP 連線不會自動啟動攔截或代理。
- URL 查詢值、認證資訊與片段預設遮蔽；封包本文、標頭與其他敏感欄位須明確選擇，並經使用者同意才會提供。
- 修正 Windows 原生確認視窗的回傳值判讀：按下「是(Y)」會正確授權該次操作，「否(N)」或取消仍維持拒絕；「否」設為安全預設值。
- 更新 MCP 安全與捕獲檢視測試、架構說明及插件範例文件。

### English

- Expanded local MCP tools so AI clients can inspect captured sessions, details, and messages; check status and plugins; and, after user confirmation, import/enable plugins, send or replay packets, control capture/system proxy, and invoke generic plugin UI actions.
- Packet sends, plugin code, capture/proxy controls, and plugin UI actions require per-operation confirmation. Connecting an MCP client does not automatically start interception or configure the system proxy.
- URL query values, credentials, and fragments are redacted by default. Payloads, headers, and other sensitive fields require an explicit MCP request and user consent before they are shared.
- Fixed native Windows consent handling: selecting “Yes” now authorizes that operation, while “No” or cancel still denies it; “No” remains the safe default.
- Updated MCP privacy/capture tests, architecture documentation, and plugin examples.

- **Platform / 平台:** Windows x64
- **File / 檔案:** `McpNet-v1.2.1-windows-x64.exe`
- **Release / 發布:** [GitHub Release](https://github.com/infycat/McpNet/releases/tag/v1.2.1)

## v1.2.0 — 2026-10-23 — Open Plugin Platform / 開源插件平台

### 繁體中文

- **大更新：以開源與社群擴充為核心。** 使用者可自行開發並分享插件，按自己的協定和工作流程組裝功能，不再只受內建解碼器限制。
- 支援 ZIP／資料夾一鍵匯入，套件需包含 `main.go` 與一個 JSON 清單；匯入後預設停用，重複 ID 不會覆蓋既有插件。
- Go hooks 可在 HTTP 請求／回應、WebSocket、UDP 等事件中自由組合協定、方向、URL、字串、正則、位元組模式和自訂條件，並按插件邏輯解碼、編碼或修改封包。
- 插件 Go hook 可呼叫 `mcpnet.OpenWindow(title, html)` 或 `mcpnet.OpenExternalURL(url)`，依插件自己的條件開啟工具窗口或外部頁面；客戶端只提供通用窗口容器，不加固定插件按鈕。
- ZIP／資料夾選擇由內置 WebView 處理，取消或驗證錯誤不會關閉主程式；插件匯入、啟用、封包發送、監聽／系統代理更改及通用 MCP UI 命令均逐次要求使用者確認。敏感封包本文、URL、用戶端 IP 與 PID 需同意後才會提供給 MCP 客戶端。
- README、架構文件與公開範例補充插件套件格式、開發和分享方式、hook API、自訂 UI 與安全邊界。
- **安全提醒：** Go 插件在本機執行且沒有安全沙箱隔離；只導入可信來源。個人插件、金鑰、設定與抓包仍留在本機，公開倉庫只收錄通用範例。

### English

- **Major update centered on open source and community extensibility.** Users can build and share plugins around their own protocols and workflows instead of being limited to built-in decoders.
- Added one-click ZIP/folder import for packages containing `main.go` and one JSON manifest. Imported plugins are disabled by default; duplicate IDs never overwrite installed plugins.
- Go hooks can combine protocol, direction, URL, text, regexp, byte-pattern, and custom conditions across HTTP request/response, WebSocket, UDP, and other events, then decode, encode, or modify packets according to plugin logic.
- Go hooks can call `mcpnet.OpenWindow(title, html)` or `mcpnet.OpenExternalURL(url)` based on plugin logic; the client provides a generic host instead of a fixed per-plugin button.
- ZIP/folder selection uses the embedded WebView, and cancellation or validation errors no longer close the app. Plugin import/enable, packet sends, listener/proxy changes, and generic MCP UI commands require per-action user approval. Private payloads, full URLs, client IPs, and PIDs are shared with MCP only after direct user consent.
- Expanded the README, architecture guide, and public examples with package format, plugin development/sharing, hook APIs, custom UI, and security boundaries.
- **Security note:** Go plugins run locally without a security sandbox; import trusted sources only. Personal plugins, keys, settings, and captures stay local; the public repository contains generic examples only.

- **Platform / 平台:** Windows x64
- **File / 檔案:** `McpNet-v1.2.0-windows-x64.exe`
- **Release / 發布:** [GitHub Release](https://github.com/infycat/McpNet/releases/tag/v1.2.0)

## v1.1.4 — 2026-10-23 — Plugin Extensibility / 插件功能擴充

### 繁體中文

- 新增社群插件一鍵導入，可選 ZIP 或資料夾；套件須包含 `main.go` 與一個 JSON 清單。導入後預設停用，重複 ID 不會覆蓋既有插件。
- 「解碼與編碼」擴充為「插件功能」。Go hooks 可在 HTTP 請求／回應、WebSocket、UDP 等事件中自行組合協議、URL、字串、正則與位元組格式判斷，並可改寫封包後繼續轉送。
- 插件清單可選擇提供 `ui.title` 和 `ui.html`，建立 HTML/JavaScript 小視窗；透過 `OnPluginAction` 與 `mcpnet.Action()`、`mcpnet.Payload()`、`mcpnet.SetUIResult()` 和插件 Go 邏輯互通。
- HTML/JavaScript 視窗使用限制網路與頂層導覽的 iframe sandbox；Go 插件程式仍在本機執行且不具安全沙箱隔離，只啟用可信來源。
- 更新 README、架構文件與插件範例，說明安裝格式、封包處理 hook、自訂介面及安全限制。

### English

- Added one-click community plugin import from a ZIP or folder containing `main.go` and one JSON manifest. Imports are disabled by default, and duplicate IDs never overwrite an installed plugin.
- Expanded “Encoding & Decoding” into “Plugin Features.” Go hooks can combine protocol, URL, text, regexp, and byte-pattern checks across HTTP request/response, WebSocket, UDP, and other events, and can rewrite packets before forwarding.
- Plugin manifests can optionally define `ui.title` and `ui.html` for a custom HTML/JavaScript window. `OnPluginAction` connects the UI to Go code through `mcpnet.Action()`, `mcpnet.Payload()`, and `mcpnet.SetUIResult()`.
- The HTML/JavaScript UI runs in an iframe sandbox with network access and top-level navigation restricted. Go plugin code still runs locally without a security sandbox; enable trusted code only.
- Updated the README, architecture guide, and plugin examples with import format, packet hooks, custom UI, and security limitations.

- **Platform / 平台:** Windows x64
- **File / 檔案:** `McpNet-v1.1.4-windows-x64.exe`
- **Release / 發布:** [GitHub Release](https://github.com/infycat/McpNet/releases/tag/v1.1.4)

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
