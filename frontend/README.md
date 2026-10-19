# Frontend

繁體中文為主的 McpNet 桌面介面，使用 Vue 3、Vite、Element Plus 與 Monaco Editor，透過 Wails bindings/events 與 Go 後端互動。

## 開發

```powershell
npm ci
npm run dev
```

## 正式建置

```powershell
npm ci
npm run build
```

輸出至 `dist/`，由 Go `main.go` 的 `//go:embed all:frontend/dist` 嵌入桌面應用。變更介面後請重新建置前端，再建置 Go 應用。請勿將使用者設定或插件放在此前端目錄。

詳細架構與建置方式請參閱[專案指南](../docs/ARCHITECTURE.md)及[根目錄說明](../README.md)。
