# 版本紀錄

本專案目前以 **v1.0.0** 作為首個正式下載版本。舊版 Release 可執行成品已移除；Git 原始碼提交歷史仍保留。正式下載與校驗資訊以 [GitHub Release](https://github.com/infycat/McpNet/releases/tag/v1.0.0) 為準。

## v1.0.0 — 2026-10-20

首個正式公開版本。保留 McpNet 的 Windows 桌面網路分析、HTTP(S)/WebSocket/TCP/UDP 檢視與處理、規則與本機插件，以及 MCP 服務功能，並修復以下體驗問題：

- **品牌圖示：** Windows EXE、工作列、應用程式與文件預覽統一採用新的透明方形 Net/MCP/封包圖示。
- **主題底板：** 修復切換主題或彈窗重繪後部分視窗背景透明、底層畫面透出的問題；明暗與漸變配色使用一致的表面色彩變數。
- **作者頁：** 將「打賞作者」改為「作者」，並替換為新的作者 GitHub 個人頁圖片。
- **發布內容：** 對外只保留這個正式版本的 Windows x64 EXE；個人插件及本機設定不包含於公開倉庫或成品。

- **平台：** Windows x64
- **檔案：** `McpNet-v1.0.0-windows-x64.exe`
- **檔案大小：** 91,961,856 bytes（約 87.7 MiB）
- **SHA-256：** `1B8246FB3EF16AE77BC55656C37CB5C4F7C5DF3E6F04841F2300AAD51D895C41`
- **下載：** [GitHub Release 成品](https://github.com/infycat/McpNet/releases/tag/v1.0.0)

首次啟動可能需要 WebView2 Runtime。流量監聽與系統代理不會因為程式開啟而自動啟用；請在應用程式內確認狀態後自行開始。
