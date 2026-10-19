# 專案與架構指南

**繁體中文（預設）** · [简体中文](#简体中文) · [English](#english)

## 繁體中文

### 系統定位

McpNet 是 Windows 桌面網路分析工具。Vue 3 呈現介面；Wails v2 將前端命令和事件連到 Go 應用；Go 負責規則、插件、MCP 與使用者設定；本機重建的 McpNet 核心負責代理和協定處理。程式啟動會顯示 GUI 並啟動綁定 `127.0.0.1` 的 MCP 服務；流量監聽與系統代理則有各自的啟停控制。

### 元件與請求流程

```mermaid
flowchart LR
  UI[Vue 3 UI] <-->|Wails Bind / Events| APP[Go App]
  APP --> RULES[Rules and config]
  APP --> PLUG[Yaegi plugin hooks]
  APP --> MCP[MCP HTTP service]
  APP --> CORE[McpNet core]
  CORE <--> DRV[Windows network driver / proxy path]
  DRV <--> NET[Target application and network]
  MCP <--> CLIENT[Local MCP client]
  CORE --> EVT[HTTP / WS / TCP / UDP events]
  EVT --> APP
```

典型資料流程：

```mermaid
flowchart TD
  A[使用者明確啟動監聽] --> B[Windows 代理或驅動取得流量]
  B --> C{協定辨識}
  C -->|HTTP / HTTPS| D[請求、回應與規則處理]
  C -->|WebSocket| E[訊息事件與插件 hook]
  C -->|TCP / UDP| F[連線與資料事件]
  D --> G[使用者規則／啟用的插件]
  E --> G
  F --> G
  G --> H[顯示、修改或轉送]
  H --> I[目標端／用戶端]
```

### 腦圖

```mermaid
mindmap
  root((McpNet))
    Desktop
      Wails v2
      Go backend
      WebView2
      Vue 3 UI
    Capture
      HTTP and HTTPS
      WebSocket
      TCP
      UDP
      Windows driver and proxy
    Extensibility
      Rules and scripts
      Yaegi Go plugins
      User data directory
      Public examples only
    Integration
      MCP over local HTTP
      JSON-RPC tools
      Config and automation
    Delivery
      Go module
      Embedded frontend
      Recovered core dependency
      Windows executable
```

### 部署／分佈圖

```mermaid
flowchart TB
  subgraph REPO[GitHub repository]
    GO[Go app source]
    FE[frontend/src + frontend/dist]
    CORE[third_party/recovered-McpNet]
    DOC[docs and scripts]
    EX[plugins/examples only]
  end
  GO -->|embed at build time| FE
  GO -->|local replace dependency| CORE
  GO --> BUILD[Windows build]
  BUILD --> EXE[McpNet.exe - ignored build output]
  EX -->|manual review and copy| USER[User data: %USERPROFILE%\\McpNet_new\\plugins]
  USER -->|runtime scan| GO
  GO -->|127.0.0.1:2026| MCPCLIENT[MCP client]
```

### 路徑與責任

```text
repository root/
├─ main.go, app.go, mcp.go, Config.go, ...   Go app and Wails entry points
├─ CommAnd/                                  UI command/event dispatch
├─ MapHash/                                  request/session lookup helpers
├─ Resource/                                 embedded script templates and metadata
├─ frontend/
│  ├─ src/                                   Vue UI source
│  ├─ wailsjs/                               generated Wails bindings
│  └─ dist/                                  embedded production assets (build input)
├─ third_party/recovered-McpNet/             local core module from go.mod replace
├─ build/                                    Windows resources and application icons
├─ scripts/                                  Windows build and cleanup scripts
├─ docs/                                     architecture and reverse-engineering notes
└─ plugins/examples/                          safe, non-personal sample plugins only
```

Keep these module-relative paths stable: `main.go` embeds `frontend/dist`; `go.mod` replaces `github.com/qtgolang/McpNet` with `./third_party/recovered-McpNet`; build scripts locate the module root relative to `scripts/`. Moving any of these requires updating the embed directive, `replace`, and scripts together. User configuration and installed plugins belong under `%USERPROFILE%\McpNet_new\`, never in the repository.

### 協定與支援功能

| 協定／介面 | 已實作能力 | 條件與備註 |
|---|---|---|
| HTTP | 請求／回應檢視、規則處理、插件 hook、內容編輯 | 必須經過已啟用的監聽路徑 |
| HTTPS / TLS | TLS/HTTPS 流量處理與憑證管理 | 解密依 CA 信任、應用程式憑證釘選及設定而異；不保證所有應用均可解密 |
| WebSocket | 連線／訊息檢視、訊息 hook、編解碼規則 | 插件可讀寫訊息本文；依連線是否經代理及規則而定 |
| TCP | 連線與資料事件、轉送／腳本處理 | 依 Windows 驅動或代理設定；不是背景自動全攔截 |
| UDP | 收送資料事件及插件 hook | Windows 擷取依驅動和程序設定；部分路徑需 SOCKS5 等代理 |
| MCP | `POST /mcp` JSON-RPC、`GET /mcp` SSE、`/health` | 預設 `127.0.0.1:2026`；可查詢捕獲會話和 WebSocket/TCP/UDP 訊息，也可控制監聽與系統代理 |

MCP 服務支援 `initialize`、`ping`、`tools/list`、`tools/call`，並回覆 `resources/list`、`prompts/list`（目前為空集合）。捕獲工具為 `mcpnet_list_captures`、`mcpnet_get_capture`、`mcpnet_get_messages`；其餘工具提供狀態、啟停監聽、系統代理、通用 UI 命令與說明。捕獲清單預設只回傳中繼資料；URL 查詢值、使用者資訊和片段會遮蔽。標頭與封包本文必須以 `include_body=true` 明確要求，本文以 Base64 傳回且每段最多 64 KiB。捕獲工具只讀取介面已顯示的會話，不會自行啟動監聽。`mcpnet_set_system_proxy` 可能同時啟動監聽；通用 UI 命令可能改變設定或流量處理狀態。MCP 只綁定 loopback，但本機其他程式仍可能連線，封包內容仍需視為敏感資料。

### 插件與公開邊界

- 倉庫只允許 `plugins/examples/` 下的公開範例；`.gitignore` 將其餘 `plugins/` 內容排除。
- 個人完成品插件（包含協定專用 XXTEA/XOR 插件）、金鑰、抓包與使用者設定不應提交。
- 執行時插件從 `%USERPROFILE%\McpNet_new\plugins\<id>\` 載入，資料夾包含 `plugin.json` 和 `main.go`；啟用插件會在流量事件中執行，僅安裝可信任的程式碼。
- hook：`OnHTTPRequest`、`OnHTTPResponse`、`OnWebSocket`、`OnUDP`、`OnURLDisplay`。可用 API 包含 `Direction`、`Host`、`Path`、`URL`、`SetURL`、`Method`、`PID`、`Process`、`Body`、`SetBody`、Hex/Base64、XOR/XXTEA 工具與 `Log`。`OnURLDisplay` 與 `SetURL` 僅修改 UI 顯示，不變更實際請求 URL。
- `SetBody` 會標記資料已變更；不要假設只讀插件具有沙箱隔離。
- 個人協定插件（WebSocket XXTEA/XOR、URL 顯示解碼與 UDP 解碼）安裝於 `%USERPROFILE%\McpNet_new\plugins\personal-local-codecs\`，只存在本機，不隨 GitHub 源碼發布。

### 建置與品質檢查

前端先執行 `npm ci` 與 `npm run build`，再於模組根目錄執行 Go 建置或 `scripts/build.ps1`。建置產物 `McpNet.exe` 不納入 Git。`frontend/dist`、`app.syso` 是可重現建置所需的輸入資源，除非同步調整建置流程，否則勿忽略。第三方恢復核心的來源與限制請看 [third_party/recovered-McpNet/README.md](../third_party/recovered-McpNet/README.md)。

---

## 简体中文

### 系统定位与流程

McpNet 是 Windows 桌面网络分析工具，由 Vue 3 前端、Wails v2/Go 应用和本地重建的 McpNet 核心组成。应用启动会显示 GUI 并启动绑定 `127.0.0.1` 的 MCP 服务；流量监听与系统代理分别控制，启动应用不等于开始全系统抓包。

```mermaid
flowchart LR
  UI[Vue 3 界面] <-->|Wails 绑定与事件| APP[Go 应用]
  APP --> RULE[规则与配置]
  APP --> PLUG[Yaegi 插件钩子]
  APP --> MCP[MCP HTTP 服务]
  APP --> CORE[McpNet 核心]
  CORE <--> DRV[Windows 驱动或代理路径]
  DRV <--> NET[目标应用与网络]
  MCP <--> CLIENT[本机 MCP 客户端]
  CORE --> EVT[HTTP / WS / TCP / UDP 事件]
  EVT --> APP
```

```mermaid
mindmap
  root((McpNet))
    桌面端
      Wails v2
      Go 后端
      WebView2
      Vue 3
    流量
      HTTP 与 HTTPS
      WebSocket
      TCP
      UDP
    扩展
      规则与脚本
      Yaegi Go 插件
      用户数据目录
      仅公开示例
    集成
      本机 HTTP MCP
      JSON-RPC 工具
    发布
      Go 模块
      嵌入式前端
      本地核心依赖
      Windows 程序
```

```mermaid
flowchart TB
  subgraph REPO[GitHub 仓库]
    GO[Go 应用源码]
    FE[frontend/src 与 frontend/dist]
    CORE[third_party/recovered-McpNet]
    DOC[文档与脚本]
    EX[仅 plugins/examples]
  end
  GO -->|构建时嵌入| FE
  GO -->|本地 replace 依赖| CORE
  GO --> BUILD[Windows 构建]
  BUILD --> EXE[McpNet.exe，忽略的产物]
  EX -->|审阅后手动复制| USER[用户数据目录插件]
  USER -->|运行时扫描| GO
  GO -->|127.0.0.1:2026| MCPCLIENT[MCP 客户端]
```

### 目录、协议与发布

主要目录：Go/Wails 文件位于仓库根目录；`CommAnd/` 为命令分发，`MapHash/` 为请求会话辅助，`Resource/` 为嵌入脚本模板；`frontend/src` 为 Vue 源码，`frontend/dist` 为 Go embed 构建输入；`third_party/recovered-McpNet` 是 `go.mod` 的本地 replace 核心；`build/` 放 Windows 图标和资源，`scripts/` 放构建脚本，`docs/` 放项目文档，`plugins/examples/` 仅放公开示例。请勿随意移动 `frontend/dist`、第三方核心或 `scripts`，否则要同步更新 embed、replace 和脚本路径。

| 协议／接口 | 功能与限制 |
|---|---|
| HTTP | 查看请求／响应、规则处理、插件钩子与内容编辑；流量须经过已启用的监听路径。 |
| HTTPS/TLS | 支持 TLS/HTTPS 与证书管理；解密受 CA 信任、证书固定和应用设置影响。 |
| WebSocket | 查看连接与消息、运行消息钩子和编解码规则。 |
| TCP | 连接与数据事件、转发和脚本处理；取决于驱动或代理设置。 |
| UDP | 收发事件和插件钩子；Windows 捕获可能需要驱动、进程配置或代理。 |
| MCP | `POST /mcp` JSON-RPC、`GET /mcp` SSE、`/health`；默认只监听 `127.0.0.1:2026`。支持查询捕获会话和 WebSocket/TCP/UDP 消息，以及状态、监听和代理控制。 |

MCP 方法包括 `initialize`、`ping`、`tools/list`、`tools/call`；资源和提示列表目前为空。捕获工具为 `mcpnet_list_captures`、`mcpnet_get_capture` 和 `mcpnet_get_messages`。默认仅返回元数据，URL 查询参数值、用户信息和片段会自动遮蔽；读取请求头和数据体必须显式设置 `include_body=true`，数据体以 Base64 返回且每段最多 64 KiB。查询不会自动开启监听；代理控制可能同时启动监听，通用命令可能改变应用状态。MCP 绑定回环地址不代表本机程序无法访问，捕获数据仍应视作敏感信息。

插件运行目录为 `%USERPROFILE%\McpNet_new\plugins\<id>\`，含 `plugin.json` 和 `main.go`；钩子为 `OnHTTPRequest`、`OnHTTPResponse`、`OnWebSocket`、`OnUDP`、`OnURLDisplay`。`SetBody` 可修改流量，`SetURL` 只修改界面显示，因此只安装可信插件。GitHub 仓库仅允许 `plugins/examples/`，忽略个人插件、密钥、抓包和用户配置；示例默认禁用。个人 XXTEA/XOR、URL 显示及 UDP 解码插件保留在本机用户数据目录，不发布到 GitHub。

构建前在 `frontend/` 执行 `npm ci`、`npm run build`，再在 Go 模块根目录运行 `go build` 或 `scripts/build.ps1`。`McpNet.exe` 是忽略的构建产物；`frontend/dist` 和 `app.syso` 仍是构建输入。HTTPS 解密、驱动捕获和系统代理仅用于获得授权的设备与流量。

---

## English

### System overview and flow

McpNet is a Windows desktop network analysis tool built from a Vue 3 frontend, a Wails v2/Go application, and a locally reconstructed McpNet core. Startup opens the GUI and starts an MCP service bound to `127.0.0.1`; capture and system-proxy controls are separate. Launching the app does not itself enable system-wide interception.

```mermaid
flowchart LR
  UI[Vue 3 UI] <-->|Wails bindings and events| APP[Go application]
  APP --> RULE[Rules and config]
  APP --> PLUG[Yaegi plugin hooks]
  APP --> MCP[MCP HTTP service]
  APP --> CORE[McpNet core]
  CORE <--> DRV[Windows driver or proxy path]
  DRV <--> NET[Target app and network]
  MCP <--> CLIENT[Local MCP client]
  CORE --> EVT[HTTP / WS / TCP / UDP events]
  EVT --> APP
```

```mermaid
mindmap
  root((McpNet))
    Desktop
      Wails v2
      Go backend
      WebView2
      Vue 3
    Traffic
      HTTP and HTTPS
      WebSocket
      TCP
      UDP
    Extensions
      Rules and scripts
      Yaegi Go plugins
      User data directory
      Public examples only
    Integration
      Local HTTP MCP
      JSON-RPC tools
    Delivery
      Go module
      Embedded frontend
      Local core dependency
      Windows executable
```

```mermaid
flowchart TB
  subgraph REPO[GitHub repository]
    GO[Go app source]
    FE[frontend/src and frontend/dist]
    CORE[third_party/recovered-McpNet]
    DOC[Docs and scripts]
    EX[plugins/examples only]
  end
  GO -->|embedded at build time| FE
  GO -->|local replace dependency| CORE
  GO --> BUILD[Windows build]
  BUILD --> EXE[McpNet.exe, ignored output]
  EX -->|review and copy manually| USER[Plugins in user data]
  USER -->|runtime scan| GO
  GO -->|127.0.0.1:2026| MCPCLIENT[MCP client]
```

### Repository layout and supported protocols

The repository root contains the Go/Wails app. `CommAnd/` dispatches UI commands; `MapHash/` contains request/session helpers; `Resource/` holds embedded script templates. `frontend/src` is the Vue source and `frontend/dist` is the Go embed input. `third_party/recovered-McpNet` is the local module selected by `go.mod`; `build/` contains Windows resources, `scripts/` contains build scripts, `docs/` contains project documentation, and `plugins/examples/` contains public sample plugins only. Keep the embed, local `replace`, and build-script paths in sync if any of these directories move.

| Protocol / interface | Implemented capability | Conditions |
|---|---|---|
| HTTP | Request/response inspection, rules, plugin hooks, and body editing | Traffic must traverse an enabled capture path. |
| HTTPS / TLS | TLS/HTTPS handling and certificate management | Decryption depends on CA trust, certificate pinning, and application behavior. |
| WebSocket | Connection/message inspection, message hooks, codec rules | Depends on the connection traversing the proxy and applicable rules. |
| TCP | Connection/data events, forwarding, and scripts | Depends on Windows driver or proxy configuration. |
| UDP | Send/receive events and plugin hooks | Windows capture may require a driver, process setup, or proxy. |
| MCP | JSON-RPC `POST /mcp`, SSE `GET /mcp`, and `/health`; query captured sessions and socket messages | Bound to `127.0.0.1:2026` by default. |

MCP methods include `initialize`, `ping`, `tools/list`, and `tools/call`; resource and prompt lists are currently empty. `mcpnet_list_captures`, `mcpnet_get_capture`, and `mcpnet_get_messages` expose read-only capture monitoring. Capture listings return metadata only; URL query values, user info, and fragments are redacted. Headers and payloads require explicit `include_body=true`, payloads are Base64 and capped at 64 KiB each. Querying does not start capture. Other tools expose status, listener start/stop, system proxy control, generic UI commands, and help. Proxy control may start capture, and generic commands may change application state. Loopback binding does not prevent other local processes from connecting, so captured traffic remains sensitive.

Runtime plugins are loaded from `%USERPROFILE%\McpNet_new\plugins\<id>\`, containing `plugin.json` and `main.go`. Hooks are `OnHTTPRequest`, `OnHTTPResponse`, `OnWebSocket`, `OnUDP`, and `OnURLDisplay`. `SetBody` can change traffic; `SetURL` changes display only. Install only code you trust. The GitHub allowlist keeps only `plugins/examples/`; personal finished plugins, keys, captures, and user configuration must not be committed. Personal WebSocket XXTEA/XOR, URL display, and UDP decoding stay under the local user-data directory. The example is disabled by default.

Build the frontend with `npm ci` and `npm run build` in `frontend/`, then run `go build` from the module root or use `scripts/build.ps1`. `McpNet.exe` is ignored build output; `frontend/dist` and `app.syso` remain build inputs. Use HTTPS decryption, driver capture, and system proxy features only on devices and traffic you are authorized to inspect.
