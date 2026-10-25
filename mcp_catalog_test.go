package main

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func TestMCPCommandCatalogCoversLiteralFrontendCommands(t *testing.T) {
	root := filepath.Join("frontend", "src")
	available := make(map[string]bool)
	for _, command := range mcpUICommandCatalog() {
		if strings.TrimSpace(command.Name) == "" || strings.TrimSpace(command.Description) == "" {
			t.Fatalf("command catalog contains an empty entry: %#v", command)
		}
		if command.ArgumentFields == nil {
			t.Errorf("command %q has no Args-field discovery metadata", command.Name)
		}
		if available[command.Name] {
			t.Fatalf("command catalog contains duplicate command %q", command.Name)
		}
		available[command.Name] = true
	}

	callPattern := regexp.MustCompile(`CallGoDo\s*\(\s*["']([^"']+)["']`)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == "assets" || entry.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".vue" && ext != ".js" && ext != ".ts" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, match := range callPattern.FindAllSubmatch(data, -1) {
			name := string(match[1])
			// This legacy UI call is an argument-like no-op, not a backend command.
			if name == "CancelMustTcp" {
				continue
			}
			if !available[name] {
				t.Errorf("frontend command %q is missing from mcpUICommandCatalog", name)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, dynamic := range []string{"启动监听", "停止监听", "枚举进程图标", "枚举全部进程图标"} {
		if !available[dynamic] {
			t.Errorf("dynamic UI command %q is missing from mcpUICommandCatalog", dynamic)
		}
	}
	for _, pluginCommand := range []string{"插件源码", "插件清單", "保存插件", "删除插件", "導入插件ZIP", "導入插件資料夾", "插件界面動作"} {
		if !available[pluginCommand] {
			t.Errorf("plugin workflow command %q is missing from mcpUICommandCatalog", pluginCommand)
		}
	}
	for _, command := range mcpUICommandCatalog() {
		if strings.HasPrefix(command.Name, "插件") || strings.Contains(command.Name, "插件") || strings.Contains(command.Name, "插件資料夾") {
			if strings.Contains(command.Name, "源码") || strings.Contains(command.Name, "清單") || strings.Contains(command.Name, "保存插件") || strings.Contains(command.Name, "删除插件") || strings.Contains(command.Name, "插件界面動作") {
				if strings.Contains(command.Description, "existing McpNet UI/backend command") {
					t.Errorf("plugin command %q has no actionable Args guidance", command.Name)
				}
			}
		}
	}
}

func TestEveryNativeFileDialogHasAnMCPPathOverride(t *testing.T) {
	source, err := os.ReadFile("public.go")
	if err != nil {
		t.Fatal(err)
	}
	dialogPattern := regexp.MustCompile(`runtime\.(?:OpenFileDialog|SaveFileDialog)\s*\(`)
	dialogCalls := dialogPattern.FindAllIndex(source, -1)
	if len(dialogCalls) != 3 {
		t.Fatalf("found %d native file dialogs, expected the audited 3 dialog call sites", len(dialogCalls))
	}
	pathOverrides := make(map[string]bool)
	for _, command := range mcpUICommandCatalog() {
		pathOverrides[command.Name] = command.PathOverride
	}
	casePattern := regexp.MustCompile(`"([^"]+)"`)
	for _, call := range dialogCalls {
		prefix := source[:call[0]]
		caseStart := strings.LastIndex(string(prefix), "case ")
		if caseStart < 0 {
			t.Fatalf("file dialog at byte %d is not inside a command case", call[0])
		}
		caseHeader := string(prefix[caseStart:])
		caseEnd := strings.Index(caseHeader, ":")
		if caseEnd < 0 {
			t.Fatalf("could not find command case header before file dialog at byte %d", call[0])
		}
		commands := casePattern.FindAllStringSubmatch(caseHeader[:caseEnd], -1)
		mapped := false
		for _, command := range commands {
			if pathOverrides[command[1]] {
				mapped = true
				break
			}
		}
		if !mapped {
			t.Errorf("native file dialog at byte %d has no MCP path override; preceding case=%q", call[0], caseHeader[:caseEnd])
		}
	}
}

func TestMCPCommandCatalogIsAdvertised(t *testing.T) {
	found := false
	for _, tool := range mcpTools() {
		if tool.Name == "mcpnet_list_commands" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("mcpnet_list_commands is not advertised")
	}
	if len(mcpUICommandCatalog()) < 70 {
		t.Fatalf("command catalog unexpectedly small: %d", len(mcpUICommandCatalog()))
	}
}

func TestMCPCommandCatalogMapsToBackendEventCases(t *testing.T) {
	casePattern := regexp.MustCompile(`case\s+["']([^"']+)["']`)
	backendCases := make(map[string]bool)
	for _, path := range []string{"public.go", "process.go", "RequestCertificate.go", "ReplaceRules.go", "HOSTSRules.go"} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, match := range casePattern.FindAllSubmatch(data, -1) {
			backendCases[string(match[1])] = true
		}
	}
	for _, command := range mcpUICommandCatalog() {
		if !backendCases[command.Name] {
			t.Errorf("catalog command %q has no backend event case", command.Name)
		}
	}
}

func TestBackendUICommandsAreDiscoverableOverMCP(t *testing.T) {
	casePattern := regexp.MustCompile(`(?m)^\tcase\s+([^\n:]+):`)
	namePattern := regexp.MustCompile(`"([^"]+)"`)
	available := make(map[string]bool)
	for _, command := range mcpUICommandCatalog() {
		available[command.Name] = true
	}
	for _, path := range []string{"public.go", "process.go", "RequestCertificate.go", "ReplaceRules.go", "HOSTSRules.go"} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, clause := range casePattern.FindAllSubmatch(data, -1) {
			for _, name := range namePattern.FindAllSubmatch(clause[1], -1) {
				command := string(name[1])
				if command == "" {
					continue
				}
				if !available[command] {
					t.Errorf("backend UI command %q from %s is missing from the MCP discovery catalog", command, path)
				}
			}
		}
	}
}

func TestMCPCommandCatalogIncludesLiteralUIArgumentKeys(t *testing.T) {
	callPattern := regexp.MustCompile(`(?s)CallGoDo\s*\(\s*["']([^"']+)["']\s*,\s*\{([^{}]*)\}`)
	keyPattern := regexp.MustCompile(`(?:^|,)\s*([A-Za-z_$][\w$]*)\s*:`)
	fieldsByCommand := make(map[string]map[string]bool)
	for _, command := range mcpUICommandCatalog() {
		fields := make(map[string]bool)
		for _, field := range command.ArgumentFields {
			fields[field] = true
		}
		fieldsByCommand[command.Name] = fields
	}
	err := filepath.WalkDir(filepath.Join("frontend", "src"), func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == "assets" || entry.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".vue" && ext != ".js" && ext != ".ts" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, call := range callPattern.FindAllSubmatch(data, -1) {
			command := string(call[1])
			if command == "CancelMustTcp" {
				continue
			}
			fields, ok := fieldsByCommand[command]
			if !ok {
				t.Errorf("frontend command %q with literal Args is missing from the MCP catalog", command)
				continue
			}
			for _, key := range keyPattern.FindAllSubmatch(call[2], -1) {
				field := string(key[1])
				if !fields[field] {
					t.Errorf("MCP command %q omits literal UI Args field %q used in %s", command, field, path)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestMCPCommandCatalogIncludesSettingsAndPluginManagementCommands(t *testing.T) {
	want := []string{"主题信息", "设置插件状态", "保存主题", "监听状态", "插件列表", "mcp设置路径", "mcp信息", "mcp重启", "mcp設定權限", "mcp權限"}
	available := make(map[string]bool)
	for _, command := range mcpUICommandCatalog() {
		available[command.Name] = true
	}
	for _, name := range want {
		if !available[name] {
			t.Errorf("command catalog omitted backend command %q", name)
		}
	}
}

func TestMCPCommandCatalogPublishesUIArgumentKeys(t *testing.T) {
	byName := make(map[string]mcpUICommand)
	for _, command := range mcpUICommandCatalog() {
		byName[command.Name] = command
	}
	want := map[string][]string{
		"查找":      {"Options", "Value", "Type", "Range", "Color", "ProtoSkip"},
		"主动发送":    {"direction", "SendType", "wsType", "Data", "Theology", "IsWs", "IsTCP", "IsUDP"},
		"保存修改数据":  {"Theology", "Type", "Tabs", "Data", "UTF8"},
		"mcp設定權限": {"Mode", "AllowPrivateRead", "AllowSettings", "AllowPluginCode", "AllowNetworkWrites"},
		"选择文件":    {"Title", "Filters", "Path"},
		"保存文件对话框": {"Title", "Filters", "Path"},
		"导出默认证书":  {"Path"},
		"创建证书":    {"domain", "country", "company", "department", "province", "city", "outTime"},
		"導入插件資料夾": {"Manifest", "Source"},
	}
	for name, fields := range want {
		command, ok := byName[name]
		if !ok {
			t.Errorf("catalog omitted %q", name)
			continue
		}
		if !reflect.DeepEqual(command.ArgumentFields, fields) {
			t.Errorf("%q argument_fields = %#v, want %#v", name, command.ArgumentFields, fields)
		}
	}
}

func TestMCPUICommandPermissionCategories(t *testing.T) {
	tests := []struct {
		command  string
		args     map[string]any
		category string
	}{
		{command: "HTTP请求获取", category: "private_read"},
		{command: "枚举进程", category: "private_read"},
		{command: "选择文件", category: "private_read"},
		{command: "保存文件对话框", category: "private_read"},
		{command: "导出默认证书", category: "settings"},
		{command: "插件源码", category: "plugin_code"},
		{command: "保存Go脚本代码", category: "plugin_code"},
		{command: "主动发送", category: "network_write"},
		{command: "全部放行", category: "network_write"},
		{command: "启动监听", category: "settings"},
		{command: "mcp設定權限", category: "settings"},
		{command: "socket点击右键菜单", args: map[string]any{"Type": "empty"}, category: "settings"},
		{command: "socket点击右键菜单", args: map[string]any{"Type": "AllHEX"}, category: "private_read"},
		{command: "unrecognized-command", category: "generic"},
	}
	for _, test := range tests {
		category, _ := mcpUICommandPermission(test.command, test.args)
		if category != test.category {
			t.Errorf("mcpUICommandPermission(%q) = %q, want %q", test.command, category, test.category)
		}
	}
	custom := mcpAccessSettings{Mode: mcpPermissionCustom, AllowPrivateRead: true, AllowSettings: true, AllowPluginCode: true, AllowNetworkWrites: true}
	for _, category := range []string{"private_read", "settings", "plugin_code", "network_write"} {
		if !custom.permits(category) {
			t.Errorf("custom profile with all feature grants does not permit %q", category)
		}
	}
	for _, command := range mcpUICommandCatalog() {
		category, _ := mcpUICommandPermission(command.Name, nil)
		if category == "generic" {
			t.Errorf("catalog command %q has no permission category; custom access profiles could not authorize it", command.Name)
		}
		if command.PermissionCategory == "" || command.Authorization == "" {
			t.Errorf("catalog command %q is missing authorization metadata: %#v", command.Name, command)
		}
		if command.Name == "socket点击右键菜单" && command.PermissionCategory != "context_dependent" {
			t.Errorf("context-dependent command advertises permission category %q", command.PermissionCategory)
		}
		if command.Name == "mcp設定權限" && command.Authorization != "always_requires_direct_confirmation" {
			t.Errorf("MCP permission command authorization = %q", command.Authorization)
		}
		if command.Name != "mcp設定權限" && command.Authorization != "requires_permission_category_or_direct_confirmation" {
			t.Errorf("command %q has unexpected authorization metadata %q", command.Name, command.Authorization)
		}
	}
}

func TestMCPUICommandCannotSelfElevateMCPPermissions(t *testing.T) {
	t.Setenv("USERPROFILE", t.TempDir())
	if err := saveMCPAccessSettings(mcpAccessSettings{Mode: mcpPermissionUnrestricted}); err != nil {
		t.Fatal(err)
	}
	server := &mcpServer{app: &App{}}
	if server.authorizeMCPUICommand("mcp設定權限", "settings", "permission change") {
		t.Fatal("MCP permission changes must require a direct confirmation even in unrestricted mode")
	}
	if !server.authorizeMCPUICommand("启动监听", "settings", "start listener") {
		t.Fatal("unrestricted mode should grant ordinary settings commands")
	}
}
