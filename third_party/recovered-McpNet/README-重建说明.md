# recovered-McpNet（重构的 McpNet 依赖库 v1.0.3）

> **Copyright (C) 2026 infycat**
> Owner: GitHub **infycat**
> Only open-source location: **https://github.com/infycat/McpNet**
> License: MIT (free to use, modify, redistribute and sell)

本目录是 **`github.com/qtgolang/McpNet` v1.0.3** 的**重构版本**，用于补全
`McpNet.exe` 主程序（`qtgolang/McpNetTools`）的依赖。

## 来源
- 基础源码 = 从 Go 模块代理取得的**真实 v1.0.0**（`github.com/qtgolang/McpNet v1.0.0`）。
- 在此基础上**重建**了 v1.0.3 相对 v1.0.0 多出的两个包：
  - `src/GoWinHttp/`（`main.go` + `MyReader.go`）
  - `src/JsCall/`（`JsCall.go`）

## 为什么需要重建
v1.0.3 已从 Go 模块代理撤回（v1.0.1–v1.0.5 全部 404），其 GitHub tag 也已删除，
本地克隆仓库在 2025-04 被 force-push 重写历史，源码与提交均不可得。
而主程序**实际 import** 了上述两个包，因此直接用 v1.0.0 仍无法构建。

## 重建说明（重要）
- **`GoWinHttp`**：原版绑定 Windows WinHTTP API（`winhttp.dll`）。本重构版改用标准库
  `net/http` 实现 `Open / SetHeader / SetOutTime / Send / Do`，并**原生实现 SOCKS5
  `ConnectS5` 握手**，对主程序用途（经本地 McpNet 代理打通隧道）行为等价，且可跨平台编译。
  这不是 winhttp.dll 绑定的逐行还原（原始源码已不可得）。
- **`JsCall.ToGBK`**：使用 `golang.org/x/text/encoding/simplifiedchinese`（v1.0.0 已依赖）
  实现 UTF-8 → GBK 转换；转换失败时原样返回。

两个包均通过 `go build` 与调用点签名校验（见主报告第九节）。

## 接入方式
主工程 `deliverable/source/go.mod` 已通过
`replace github.com/qtgolang/McpNet => ../recovered-McpNet` 接入本目录。
构建主程序前请先在主工程目录执行 `go mod tidy`（需要 Go 1.23+ 与完整依赖树 /
原生工具链 CGO、NetFilter 驱动 `netfilter2.sys`）。

> 著作权归原作者（qtgolang / 秦天）所有，本目录仅用于学习与安全研究。
