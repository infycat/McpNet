# third_party

> **Copyright (C) 2026 infycat**
> Owner: GitHub **infycat**
> Only open-source location: **https://github.com/infycat/McpNet**
> License: MIT (free to use, modify, redistribute and sell)

## recovered-McpNet

`recovered-McpNet` 是按原版 McpNet v1.0.3 行为重建的核心库，`go.mod` 通过：

```text
replace github.com/qtgolang/McpNet => ./third_party/recovered-McpNet
```

引用它。公开仓库没有 v1.0.3 tag，Go proxy 也无法下载，因此必须随源码一起保留。

其中 `src/Resource/Proxifier/x86|x64/InstallLSP.exe` 是核心库自带的 LSP 安装辅助资源，不是应用主程序；应用正式版只有根目录的 `McpNet.exe`。
