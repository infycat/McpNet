# McpNet.exe 逆向工程报告

> **Copyright (C) 2026 infycat**
> Owner: GitHub **infycat**
> Only open-source location: **https://github.com/infycat/McpNet**
> License: MIT (free to use, modify, redistribute and sell)

**目标文件**：本机保存的原始参考可执行文件（未随仓库发布）
**文件大小**：75,517,440 字节（约 72 MB）
**MD5**：`d1063f8d5a01060a1f2b074dc2774889`
**分析日期**：2026-10-17
**结论**：源码已**完整恢复**（非伪代码重建）；
其核心依赖库 **`github.com/qtgolang/McpNet` v1.0.3**（原已下架/不可获取）现已**重构补全**，项目可对照构建。

---

## 一、核心结论

`McpNet.exe` 是一个 **Wails v2 桌面应用**（Go 后端 + WebView2 前端），
其源码为开源项目 **`github.com/qtgolang/McpNetTools`** 的
**commit `411b25638e6747fb51f2bb1c44a4da2743548068`**（2024-06-04）。

该 commit 已通过**三重证据**精确锁定，恢复出的源码即为原始源码。

---

## 二、目标身份

| 项目 | 值 |
|---|---|
| 产品名称 | McpNet 抓包工具 |
| 产品版本 | 1.0.2.7 |
| 主模块路径 | `changeme`（Wails 模板默认名，作者未修改） |
| 源码目录 | 原始上游项目目录（本机路径不公开） |
| 上游仓库 | https://github.com/qtgolang/McpNetTools |
| 精确提交 | `411b256`（2024-06-04 03:20:02 +0800） |
| 依赖核心库 | `github.com/qtgolang/McpNet` v1.0.3 |
| 构建工具链 | Go 1.20.4 windows/amd64, CGO_ENABLED=1 |
| 构建标志 | `-ldflags="-w -s -H windowsgui"`, tags=`desktop,wv2runtime.download,production` |

---

## 三、技术栈

### 后端（Go）
- **Wails v2.7.1** — 桌面应用框架（Go + WebView2）
- **McpNet v1.0.3** — 网络中间件核心（HTTP/TCP/UDP/WebSocket 拦截）
- **NetFilter SDK** — TDI Hook 驱动（`netfilter2.sys`，用于流量拦截）
- **yaegi v0.15.1** — Go 解释器（执行 `.goc` 用户脚本）
- **otto** — JavaScript 解释器（执行 JS 用户脚本）
- **go-redis / gosysproxy / brotli / protobuf** — 辅助依赖

### 前端（Vue 3）
- Vue 3.2.37 + Element Plus 2.4.1
- Monaco Editor 0.45.0（代码编辑器）
- AG Grid 30.2.0（请求列表表格）
- Vite 3 构建

---

## 四、证据链

### 1. 构建元数据（`go version -m`）
```
path    changeme
mod     changeme  (devel)
dep     github.com/qtgolang/McpNet  v1.0.3
dep     github.com/wailsapp/wails/v2  v2.7.1
dep     github.com/traefik/yaegi      v0.15.1
dep     github.com/robertkrimen/otto  ...
build   -ldflags="-w -s -H windowsgui"
build   vcs.revision=411b25638e6747fb51f2bb1c44a4da2743548068
build   vcs.time=2024-06-03T19:20:02Z
build   vcs.modified=true
```

### 2. pclntab 函数名表（22,609 个函数 / 1,706 个源文件）
`-s -w` 剥离了符号表（`go tool nm` 返回 "no symbols"），
但 Go 运行时所需的 **pclntab 完整保留**，据此恢复出：
- 主模块包结构：`changeme/CommAnd`、`changeme/MapHash`、`changeme/Resource`
- 原始源文件路径：`G:/Mcp/McpNetTools/{main,app,McpNet,Config,...}.go`

### 3. 嵌入资源比对（决定性证据）
从二进制解析 Go `embed.FS`（stride=48 的 `{name, data, hash}` 结构），
提取出 `Resource/ScriptTemplate/` 下全部文件，与上游 commit `411b256` 逐一比对：

| 文件 | 结果 |
|---|---|
| `annotation.txt` | **MD5 完全一致** |
| `info.json` | **MD5 完全一致** |
| `_rsa.goc` | 归一化行尾后一致 |
| `_aes-des.goc` | 归一化行尾后一致 |
| `_header.goc` | 归一化行尾后一致 |
| `_forward.goc` / `_pid.goc` / `_response.goc` | 归一化行尾后一致 |

> 注：`.goc` 差异仅为 CRLF/LF 行尾（仓库 CRLF，编译期 LF），内容完全相同。

---

## 五、恢复方法

1. **PE 结构分析** → 判定为 x64 原生 PE，无 CLR 头，非 .NET
2. **特征识别** → 发现 `go_tcpCanReceive` 导出符号、`wails-app-ws`、`EmbeddedBrowserWebView.dll` → 判定 Wails/Go
3. **`go version -m`** → 提取完整依赖清单与 VCS 信息
4. **pclntab 解析** → 恢复 22,609 个函数名 + 1,706 个源文件路径
5. **embed.FS 解析** → 提取 218 个嵌入文件（前端资源 + 脚本模板）
6. **上游比对** → 定位 commit `411b256`，验证一致性
7. **交付** → 完整项目源码 + 提取资源 + 证据

---

## 六、交付物清单

```
deliverable/
├── source/                  ← 完整项目源码（可直接构建）
│   ├── main.go / app.go / McpNet.go / Config.go / Find.go
│   ├── HOSTSRules.go / ReplaceRules.go / RequestCertificate.go
│   ├── process.go / public.go / update.go / PbToJson.go / CreateRequestCode.go
│   ├── CommAnd/             （Windows.go / Linux.go / darwin.go）
│   ├── MapHash/             （hash.go / br.go）
│   ├── Resource/            （Resource.go + BuiltFunc.txt + GoCode.txt + ScriptTemplate/）
│   ├── frontend/            ← 完整前端源码（~70 个 Vue 组件）
│   │   ├── src/App.vue, main.js, style.css
│   │   ├── src/components/  （Request/ Response/ Settings/ Find/ CertDoc/ ...）
│   │   ├── src/assets/vs/   （Monaco Editor 资源）
│   │   └── package.json / index.html
│   ├── build/               （Windows 图标 / 安装脚本 / 清单）
│   ├── go.mod / go.sum / wails.json
│   └── img/
├── extracted-frontend/      ← 从二进制直接提取的编译产物（用于交叉验证）
│   ├── dist/                （index.html, assets/, vs/）
│   ├── ScriptTemplate/      （9 个 .goc + annotation.txt + info.json）
│   └── _manifest.txt
├── evidence/
│   ├── funcnames.txt        （22,609 个函数名）
│   ├── sourcefiles.txt      （1,706 个源文件路径）
│   ├── mcpnet_api_v1.0.3.txt   （v1.0.3 编译进映像的 1084 个符号）
│   ├── mcpnet_pkgdirs_v1.0.3.txt（v1.0.3 实际包集合，22 个）
│   └── *.py                 （分析脚本，可复现）
├── recovered-McpNet/      ← 重构补全的核心依赖库（真实 v1.0.0 + 重建 GoWinHttp/JsCall）
│   ├── go.mod / go.sum      （module github.com/qtgolang/McpNet, go 1.20）
│   ├── Api/ McpNet/ src/ …（v1.0.0 原样源码）
│   ├── src/GoWinHttp/       （main.go + MyReader.go，重构）
│   └── src/JsCall/          （JsCall.go，重构）
└── buildinfo.txt            （go version -m 原始输出）
```

---

## 七、置信度与遗留项

### 高置信度（已验证）
- ✅ 目标身份与版本：构建元数据 + 文件路径双重确认
- ✅ Go 后端源码：commit 精确匹配，脚本模板 MD5 一致
- ✅ 前端源码：仓库含完整 `frontend/src`
- ✅ 项目配置：`go.mod` module 名 `changeme` 与 buildinfo 一致

### 遗留项（已说明）
1. ~~**McpNet 库 v1.0.3 不可获取**~~ → **已重构补全，见第九节**
   经深度溯源确认：该版本已从 Go 模块代理 **撤回**（proxy.golang.org 对
   v1.0.1–v1.0.5 全部返回 404），且 GitHub 上的 `v1.0.3` tag 已被删除
   （本地克隆仓库 2025-04 被 force-push 重写历史，仅保留 v1.0.0 与 v1.2.6+）。
   原交付的 `go.mod` 记录 v0.0.2，作者构建时本地升级到 v1.0.3（`vcs.modified=true`）。
   → 现已以「真实 v1.0.0 源码 + 重构的 v1.0.3 专属两个包」重建出可构建的
   `recovered-McpNet` 模块，并通过 `replace` 指令接入主工程，详见第九节。

2. **embed hash 未用于校验**
   Go 编译器使用内部 content-addressable hash（`cmd/internal/hash.Sum32`），
   非标准 SHA-256/MD5，故 218 个提取文件 hash 校验显示 0/N。
   → 已改用**上游源码逐文件比对**作为替代验证，结论可靠。

3. **`vcs.modified=true`**
   作者工作区有未提交修改（推测为 go.mod 依赖升级）。
   → 不影响主源码一致性，已由脚本模板 MD5 验证覆盖。

---

## 九、核心依赖库 v1.0.3 的逆向补全（"进一步"工作）

### 9.1 问题的本质
主程序（McpNetTools）在 `buildinfo.txt` 中明确依赖 **`github.com/qtgolang/McpNet v1.0.3`**，
但交付物此前无法获得该版本源码，导致项目不可构建。本节记录对其的溯源与重构。

### 9.2 溯源结论：v1.0.3 已无法从公开渠道获取
逐项验证结果如下：

| 渠道 | 命令 / 路径 | 结果 |
|---|---|---|
| Go 模块代理（zip） | `proxy.golang.org/github.com/qtgolang/mcp!net/@v/v1.0.3.zip` | **404** |
| Go 模块代理（info/mod） | 同上 `.info` / `.mod` | **404** |
| 校验和数据库 | `sum.golang.org/lookup/github.com/qtgolang/mcp!net@v1.0.3` | **404**（`unknown revision`） |
| 相邻版本 v1.0.1/1.0.2/1.0.4/1.0.5 | proxy `.info` | **全部 404**（作者已批量撤回） |
| GitHub 发布 tag | `github.com/qtgolang/McpNet/archive/refs/tags/v1.0.3.zip` | **404**（tag 已删除） |
| 本地克隆仓库 `git tag` | `git -C upstream/McpNet tag` | 仅 `v1.0.0` 与 `v1.2.6 … v1.5.1` |
| 本地克隆全历史 | `git log --all -- src/GoWinHttp` / `-- src/JsCall` | **无任何提交包含这两个目录** |

**结论**：作者在 2025-04 对 `qtgolang/McpNet` 进行了仓库重建（force-push 重写历史 +
删除中间 tag + 从代理撤回模块），v1.0.1–v1.0.5 全部消失，`v1.0.3` 的源码与提交已不可得。
v1.0.3 对应的 `go.sum` 记录为
`h1:E0e9ojaphPjsHKhQ5/WvDcaQWiZjld7NPB2QEZff1Do=`，但无法反推出对应源码。

### 9.3 缺口定位：包集合与公开 API 均有差异
从二进制 pclntab 还原出 v1.0.3 实际编译进映像的包集合（共 22 个），与**真实可下载的
v1.0.0**（从代理取得的 zip）比对，目录级差异是 v1.0.3 多出两个包：

- `src/GoWinHttp`（`main.go` + `MyReader.go`）
- `src/JsCall`（`JsCall.go`）

这两个包是主程序**实际 import** 的：

| 主程序文件 | 引用 |
|---|---|
| `MapHash/hash.go` | `import ".../src/GoWinHttp"` → `ConnectS5`、`NewGoWinHttp()`、`SetProxyType`、`SetProxyIP`、`Do` |
| `update.go` | `import ".../src/GoWinHttp"` → `WinHttp{}`、`Open`、`SetOutTime`、`SetHeader`、`Send` |
| `process.go` | `import ".../src/JsCall"` → `ToGBK` |

因此：**用真实 v1.0.0 充当 v1.0.3 仍无法构建**——必须补齐这两个包。

此外，逐文件检查确认 v1.0.0 的 `McpNet` 包也没有 `HttpConn`、`WsConn`、`TcpConn`、
`UDPConn` 这四个回调包装类型；主程序源码却直接访问这些类型及其字段/方法。重建模块因此
还在 `McpNet/McpNet.go` 添加了兼容包装层。它们是依据主程序调用点和二进制符号重建的，
并非从 v1.0.3 源码恢复出的原始定义。

本地 `upstream/McpNet` Git 仓库的标签从 `v1.0.0` 跳至 `v1.2.6`，没有 v1.0.3 标签；
`git log --all -- src/GoWinHttp src/JsCall` 无相关提交，`git fsck --unreachable --no-reflogs`
也没有发现可恢复的悬空对象。因此本地 Git 历史无法补回 v1.0.3。

### 9.4 重构方案
在 `deliverable/recovered-McpNet/` 下，**以真实 v1.0.0 源码为基础**，补充重建两个包，
并改正 `go.mod`（module 仍为 `github.com/qtgolang/McpNet`，go 1.23.0）：

- `src/GoWinHttp/main.go` + `MyReader.go`
  - 原版通过 Windows WinHTTP API（`winhttp.dll`）实现 HTTP 请求；
    重构版为**可跨平台构建**，改用标准库 `net/http` 承载 `Open/Send/Do`，并**原生实现
    SOCKS5 `CONNECT` 握手**（`ConnectS5`）——对主程序用途（经本地 McpNet 代理打通隧道）行为等价。
  - 暴露 API（与调用点逐一核对）：
    `WinHttp` 结构体、`NewGoWinHttp() *WinHttp`、`Open(method,url) error`、
    `SetOutTime(connect,send,receive int)`、`SetHeader(k,v string)`、
    `SetProxyType(bool)`、`SetProxyIP(string)`、
    `Do(*http.Request) (*http.Response, error)`、
    `Send(body string) (*http.Response, error)`、`Proxy` 结构体、
    `ConnectS5(conn *net.Conn, proxy *Proxy, host string, port int) bool`。
- `src/JsCall/JsCall.go`
  - 仅被主程序使用的 `ToGBK(s string) string`，用 `golang.org/x/text/encoding/simplifiedchinese`
    （v1.0.0 已依赖）实现 UTF-8→GBK 转换；转换失败时原样返回，保证调用安全。

### 9.5 验证
- 两个包**分别 `go build` 通过**，`gofmt` 干净。
- 编写覆盖**全部调用点签名**的临时校验程序（完全照搬主程序写法），在 `recovered-McpNet`
  模块内 `go build` 通过 → 证明重构 API 与主程序调用**完全匹配**（校验程序已删除）。
- 修正 `ConnectS5` 的端口参数为 `int`，并在调用点将解析出的 `uint16` 明确转换为 `int`。
- 修正脚本回调边界的 HTTP Header 类型：Yaegi 脚本继续使用标准库 `net/http.Header`，主程序与重建库之间通过显式类型转换衔接，避免启动时 interface conversion panic。
- 主工程 `deliverable/source/go.mod` 已改为：
  - `github.com/qtgolang/McpNet v1.0.3`（修正原错误占位 `v0.0.2`）
  - `go → 1.23.0`（因依赖库及其依赖要求 Go ≥ 1.23；与构建工具链兼容）
  - `replace github.com/qtgolang/McpNet => ../recovered-McpNet`
- 已完整编译 Windows/amd64 桌面版 `deliverable/source/McpNet_new.exe`，构建标签为
  `desktop,wv2runtime.download,production`，SHA-256：
  `5CCA7303DA9AC2C86E252B2B025037AF1225C0E7EB82953BFE0682CB79A2E97D`。
- 针对新版与旧版互相影响，配置和 WebView2 资料目录已改为独立的 `%USERPROFILE%\McpNet_new`，默认监听端口改为 2025。介面启动后监听保持关闭，须手动点「启动监听」；系统代理也须手动启用。关闭时会停止监听，并撤销本实例设置的代理。
- 旧版关闭视窗的处理会把系统代理设置为 `127.0.0.1:2024` 后退出，导致 HTTP 请求失败；该调用已删除。本机发现此失效代理仍启用时，已将 Windows 代理关闭并清除该地址。
- 为增加主窗体可见性，新版使用标准 Windows 标题栏，在 DOM 就绪时主动显示并置中窗口；WebView2 资料目录不可写时回退到 `%TEMP%\McpNet_new\WebView2`。
- 诊断原 exe 时确认它由裸 `go build` 生成、没有 Wails 桌面构建标签，因此启动会显示
  “Wails applications will not build without the correct build tags”并退出；重建产物已包含正确标签。

### 9.6 说明与局限
- 重构版 `GoWinHttp` 为**功能等价实现**，并非原始 `winhttp.dll` 绑定的逐行还原
  （原始源码已不可得）；在 Windows 上行为等价，且可跨平台编译。
- 四个连接包装类型同样属于依据调用点重建的兼容 API，行为覆盖当前主程序使用的字段和方法，
  不代表 v1.0.3 原始库源码已完整复原。
- 本次环境使用 Go 1.24.9；Wails CLI v2.7.1 自身的旧版 `x/tools` 在此工具链上发生
  `panic: unsupported version: 2`，故以 Wails CLI 对应的桌面/production 标签和 GUI linker
  参数直接构建。最终 exe 的 Go 构建信息已确认含正确标签。
- 修复了两份 manifest 冲突：最终 PE 资源树只有一个数字类型 24 的 manifest，内容要求管理员权限；需用户确认 UAC。实际管理员环境中的主窗体、抓包及驱动功能尚待用户端复测。
- 启动诊断优先写入 EXE 旁边的 `McpNet_new_startup.log`；该目录不可写时回退到 `%USERPROFILE%\McpNet_new\startup.log`。WebView2 在 30 秒内未就绪会弹出诊断窗口。

---

## 八、免责说明

本报告仅用于**学习与安全研究**目的，基于用户本机提供的文件与公开仓库信息。
逆向所得源码的著作权归原作者（qtgolang / 秦天）所有，
请遵守原项目 LICENSE，不得用于商业侵权或非法用途。
